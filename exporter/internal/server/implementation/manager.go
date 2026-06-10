// Copyright 2026 NTT, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package implementation

import (
    "context"
	"encoding/json"
	"fmt"
    "errors"
    "strings"
	"common/models/extra_parameters"
	"exporter_module/internal/server/utils"
	"exporter_module/internal/server/interfaces" // import for interface
	"exporter_module/internal/server/implementation/pg_cdi"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// struct of manager
type ManagerImplement struct {
	Logger     klog.Logger
	Ansible    interfaces.Ansible
	CdiAnsible interfaces.CdiAnsible // CDI Ansible for machine status and P2P operations
	Metrics    interfaces.Metrics
	serverList []interfaces.ServerTargetList
	networkList []interfaces.NetworkTargetList
	p2pSettingList []interfaces.ServerTargetList
}

func (m *ManagerImplement) ServerList(serverList []interfaces.ServerTargetList) {
	defer func() {
		m.Logger.V(2).Info("end ServerList", "server_count", len(serverList))
	}()
	m.Logger.V(2).Info("start ServerList", "server_count", len(serverList))

	// clearn up metrics for servers that are no longer monitored
    cleanupMetrics(
		m,
        m.serverList,
        serverList,
        func(target interfaces.ServerTargetList) string { return target.ServerID },
        func(target interfaces.ServerTargetList) []interfaces.MetricLabel { return target.WritedMetrics },
        "server",
    )

	// update current server list
	m.serverList = serverList
}

func (m *ManagerImplement) NetworkList(networkList []interfaces.NetworkTargetList) {
	defer func() {
		m.Logger.V(2).Info("end NetworkList", "network_count", len(networkList))
	}()
	m.Logger.V(2).Info("start NetworkList", "network_count", len(networkList))

	// clearn up metrics for networks that are no longer monitored
    cleanupMetrics(
		m,
        m.networkList,
        networkList,
        func(target interfaces.NetworkTargetList) string { return target.IPAddress },
        func(target interfaces.NetworkTargetList) []interfaces.MetricLabel { return target.WritedMetrics },
        "network",
    )

	// update current network list
	m.networkList = networkList
}

// cleanupMetrics removes metrics for targets that are no longer monitored
func cleanupMetrics[T any](
    m *ManagerImplement,
    previousList []T,
    currentList []T,
    getID func(T) string,
    getWritedMetrics func(T) []interfaces.MetricLabel,
    targetType string,
) {
    m.Logger.V(2).Info("start cleanupMetrics", "type", targetType, "previous_count", len(previousList), "current_count", len(currentList))

	// create a map for current targets for quick lookup
    currentMap := make(map[string]T)
    for _, current := range currentList {
        currentMap[getID(current)] = current
    }

	// iterate over previous targets to find removed ones
    for _, previous := range previousList {
        id := getID(previous)
        current, existsInCurrent := currentMap[id]

		// if target no longer exists, delete all its metrics
        if !existsInCurrent {
            previousMetrics := getWritedMetrics(previous)
            m.Logger.V(2).Info("branch: target removed, deleting all metrics", "type", targetType, "id", id, "metrics_count", len(previousMetrics))
            for _, metric := range previousMetrics {
                err := m.Metrics.Delete(metric.Metrics, metric.Label)
                if err != nil {
                    m.Logger.V(2).Info("branch: metric deletion failed", "type", targetType, "id", id, "label", metric.Label, "error", err.Error())
                } else {
                    m.Logger.V(2).Info("branch: metric deleted", "type", targetType, "id", id, "label", metric.Label)
                }
            }
		// if target still exists, check for removed metrics
        } else {
            m.Logger.V(2).Info("branch: target exists, checking for removed metrics", "type", targetType, "id", id)

            previousMetrics := getWritedMetrics(previous)
            currentMetrics := getWritedMetrics(current)

			// create a map for current metrics for quick lookup
            currentMetricsMap := make(map[string]bool)
            for _, currentMetric := range currentMetrics {
                key := m.getMetricKey(currentMetric)
                currentMetricsMap[key] = true
            }

			// iterate over previous metrics to find removed ones
            for _, previousMetric := range previousMetrics {
                key := m.getMetricKey(previousMetric)
                if !currentMetricsMap[key] {
                    m.Logger.V(2).Info("branch: metric removed, deleting", "type", targetType, "id", id, "label", previousMetric.Label)
                    err := m.Metrics.Delete(previousMetric.Metrics, previousMetric.Label)
                    if err != nil {
                        m.Logger.V(2).Info("branch: metric deletion failed", "type", targetType, "id", id, "label", previousMetric.Label, "error", err.Error())
                    } else {
                        m.Logger.V(2).Info("branch: metric deleted successfully", "type", targetType, "id", id, "label", previousMetric.Label)
                    }
                }
            }
        }
    }

    m.Logger.V(2).Info("end cleanupMetrics", "type", targetType)
}

// getMetricKey generates a unique key for a metric label map
func (m *ManagerImplement) getMetricKey(metric interfaces.MetricLabel) string {
    var labelStr string
    for k, v := range metric.Label {
        labelStr += k + "=" + v + ","
    }
    return labelStr
}

func (m *ManagerImplement) SetP2POn(serverList []interfaces.ServerTargetList) {
    defer func() {
        m.Logger.V(2).Info("end SetP2POn", "server_count", len(serverList))
    }()
    m.Logger.V(2).Info("start SetP2POn", "server_count", len(serverList))
    
    // create a map for previous P2P setting list for quick lookup
    previousP2PMap := make(map[string]interfaces.ServerTargetList)
    for _, previous := range m.p2pSettingList {
        previousP2PMap[previous.ServerID] = previous
    }

    // list to store updated P2P settings
    updatedP2PList := []interfaces.ServerTargetList{}

    for _, server := range serverList {
        m.Logger.V(2).Info("P2P checking server", "server_id", server.ServerID, "ip_address", server.HostIPAddress)
        
        // skip if P2P is disabled
        if !server.P2PEnable {
            m.Logger.V(2).Info("P2P is disabled, skipping check", "server_id", server.ServerID)
            continue
        }

        m.Logger.V(2).Info("P2P is enabled, performing check", "server_id", server.ServerID)

        // get machine status (power ON and P2P enable status)
        isPowerON, p2pEnable, err := m.getMachineStatus(server)
        if err != nil {
            m.Logger.V(2).Info("Failed to get machine status, skipping check", "server_id", server.ServerID, "error", err.Error())
            continue
        }

        // get uptime seconds via Ansible
        var uptimeSeconds int64
        uptimeSeconds, err = m.getUptimeSeconds(server)
        if err != nil {
            m.Logger.V(2).Info("Failed to get uptime seconds, skipping check", "server_id", server.ServerID, "error", err.Error())
            continue
        }

        // search for previous information by ServerID
        previous, previousExists := previousP2PMap[server.ServerID]
        if !previousExists {
            m.Logger.V(2).Info("No previous P2P setting found", "server_id", server.ServerID)
        }

        // determine if P2P setting should be triggered
        // Conditions:
        // 1. No previous record (first run)
        // 2. Power OFF to Power ON transition
        // 3. Reboot detection (UptimeSeconds decreased)
        shouldSetP2P := !previousExists || 
                       (!previous.PowerON && isPowerON) ||
                       (isPowerON && previous.UptimeSeconds > uptimeSeconds)

        if shouldSetP2P {
            m.Logger.V(2).Info("P2P setting trigger condition met", "server_id", server.ServerID, 
                "previousExists", previousExists, "previous_power_on", previous.PowerON, "current_power_on", isPowerON,
                "previous_uptime", previous.UptimeSeconds, "current_uptime", uptimeSeconds)

            // if current P2P is enabled, turn off P2P first
            if p2pEnable {
                m.Logger.V(2).Info("Turning off P2P before turning on", "server_id", server.ServerID)
                err = m.setMachineP2P(server, "off")
                if err != nil {
                    m.Logger.V(2).Info("Failed to turn off P2P (ignored)", "server_id", server.ServerID, "error", err.Error())
                }
            }

            // turn on P2P
            m.Logger.V(2).Info("Turning on P2P", "server_id", server.ServerID)
            err = m.setMachineP2P(server, "on")
            if err != nil {
                m.Logger.V(2).Info("Failed to turn on P2P (ignored)", "server_id", server.ServerID, "error", err.Error())
            } else {
                // update p2pEnable status after successful operation
                p2pEnable = true
            }
        } else {
            m.Logger.V(2).Info("P2P setting not needed", "server_id", server.ServerID)
        }

        // copy server and update with current information
        updatedServer := server
        updatedServer.PowerON = isPowerON
        updatedServer.P2PEnable = p2pEnable
        updatedServer.UptimeSeconds = uptimeSeconds

        // write P2P enable metrics
        p2pEnableFloat := 0.0
        if p2pEnable {
            p2pEnableFloat = 1.0
        }
        err = m.Metrics.Write(p2pEnableGauge, prometheus.Labels{"serverId": server.ServerID}, p2pEnableFloat, &updatedServer.WritedMetrics)
        if err != nil {
            m.Logger.V(2).Info("branch: p2p_enable metrics write failed", "server_id", server.ServerID, "error", err.Error())
        }

        // add to updated P2P setting list
        updatedP2PList = append(updatedP2PList, updatedServer)
    }

    // update p2pSettingList with current information
    m.p2pSettingList = updatedP2PList
    m.Logger.V(2).Info("Updated p2pSettingList", "count", len(m.p2pSettingList))
}

// getMachineStatus retrieves the power and P2P status of a server via Ansible module
func (m *ManagerImplement) getMachineStatus(server interfaces.ServerTargetList) (isPowerON bool, p2pEnable bool, err error) {
    m.Logger.V(2).Info("start getMachineStatus", "server_id", server.ServerID)
    defer func() {
        m.Logger.V(2).Info("end getMachineStatus", "server_id", server.ServerID, "is_power_on", isPowerON, "p2p_enable", p2pEnable)
    }()

    // parse extra parameters
    extraParams := extra_parameters.PgCDIExtraParameters{}
    if err = json.Unmarshal([]byte(server.CdiInfo.ExtraParameters), &extraParams); err != nil {
        m.Logger.V(2).Info("branch: ansible command failed", "server_id", server.ServerID, "error", err.Error())
        m.Logger.Error(err, err.Error())
        return
    }

	// generate extra option
	extra := fmt.Sprintf("cdi_user=%s cdi_password=%s cdi_guest=%s group_name=%s machine_name=%s",
		extraParams.CDIUser,
		extraParams.CDIPassword,
		extraParams.CDIGuest,
		server.CdiInfo.GroupName,
		server.CdiInfo.MachineName,
	)

    cdiAnsible := m.CdiAnsible
	if cdiAnsible == nil {
		// fallback to default implementation if not injected
		cdiAnsible = &pg_cdi.PgCDIAnsibleImple{Logger: m.Logger}
	}

    // execute Ansible module to get machine status
    playbook := "machine_show.yaml"
	sshKey := ""
	if config := utils.GetConfig(); config != nil {
		sshKey = config.SshKey
	}
    m.Logger.V(2).Info("branch: executing ansible command", "server_id", server.ServerID, "playbook", playbook)
    errMsg, data := cdiAnsible.CmdExecute(context.Background(), server.CdiInfo.RemoteHost, server.CdiInfo.RemoteUser, sshKey, playbook, extra)
	if errMsg != nil {
        err = errors.New(*errMsg)
        m.Logger.V(2).Info("branch: ansible command failed", "server_id", server.ServerID, "error", err.Error())
        m.Logger.Error(err, err.Error())
		return
	}

	// parse p2p and power status from machine data
	isPowerON, p2pEnable, err = m.parseP2PStatus(data)
	if err != nil {
		m.Logger.Error(err, "Failed to parse P2P status")
		return
	}

    return
}

// parseP2PStatus extracts p2p and power status from machine JSON data
func (m *ManagerImplement) parseP2PStatus(data map[string]interface{}) (isPowerON bool, p2pEnable bool, err error) {
	m.Logger.V(2).Info("start parseP2PStatus")
	defer func() {
		m.Logger.V(2).Info("end parseP2PStatus", "is_power_on", isPowerON, "p2p_enable", p2pEnable, "err", err)
	}()

	// extract necessary data
	var extractData map[string]interface{}
	var machines []interface{}
	var ok bool
	if extractData, ok = data["data"].(map[string]interface{}); ok {
		machines, ok = extractData["machines"].([]interface{})
	}
	if !ok {
		err = errors.New("ansible output is unexpected")
		m.Logger.Error(err, "Ansible cmd is failed")
		return
	}
    
	// extract data object and create json string
	jsonData, err := json.Marshal(machines[0])
	if err != nil {
		m.Logger.Error(err, "Ansible output is unexpected")
		return
	}

	// unmarshal json data to extract machine information
	var machineInfo map[string]interface{}
	err = json.Unmarshal(jsonData, &machineInfo)
	if err != nil {
		m.Logger.Error(err, "Failed to unmarshal machine data")
		return
	}

	// extract p2p status
	p2pValue, ok := machineInfo["p2p"].(string)
	if !ok {
		err = errors.New("p2p field not found or invalid type in machine data")
		m.Logger.Error(err, err.Error())
		return
	}
	p2pEnable = (p2pValue == "on")
	m.Logger.V(2).Info("branch: extracted p2p status", "p2p_value", p2pValue, "p2p_enable", p2pEnable)

	// extract mach_status_detail
	machStatusDetail, ok := machineInfo["mach_status_detail"].(string)
	if !ok {
		err = errors.New("mach_status_detail field not found or invalid type in machine data")
		m.Logger.Error(err, err.Error())
		return
	}
	m.Logger.V(2).Info("branch: extracted mach_status_detail", "mach_status_detail", machStatusDetail)

	// determine power status from mach_status_detail
    trimmedStatus := strings.TrimSpace(machStatusDetail)
    isPowerON = strings.Contains(trimmedStatus, "PON")
	m.Logger.V(2).Info("branch: determined power status", "is_power_on", isPowerON)

    return
}

// getUptimeSeconds retrieves the uptime seconds of a server via Ansible module
func (m *ManagerImplement) getUptimeSeconds(server interfaces.ServerTargetList) (uptimeSeconds int64, err error) {
    m.Logger.V(2).Info("start getUptimeSeconds", "server_id", server.ServerID)
    defer func() {
        m.Logger.V(2).Info("end getUptimeSeconds", "server_id", server.ServerID, "uptime_seconds", uptimeSeconds)
    }()

    // execute Ansible module to get uptime seconds
    playbook := "get_uptime_seconds.yaml"
	sshKey := ""
	if config := utils.GetConfig(); config != nil {
		sshKey = config.SshKey
	}
    m.Logger.V(2).Info("branch: executing ansible command", "server_id", server.ServerID, "playbook", playbook)
    output, err := m.Ansible.CmdExecute(context.Background(), server.HostIPAddress, server.LoginUser, sshKey, playbook, "")
    if err != nil {
        m.Logger.V(2).Info("branch: ansible command failed", "server_id", server.ServerID, "error", err.Error())
        m.Logger.Error(err, err.Error())
        return
    }
    
    // parse uptime seconds from Ansible output
    uptimeSeconds, err = m.parseUptimeSeconds(output)
    if err != nil {
        m.Logger.V(2).Info("branch: parsing uptimeSeconds failed", "server_id", server.ServerID, "error", err.Error())
        m.Logger.Error(err, err.Error())
        return
    }

    return
}

// parseUptimeSeconds extracts the uptime seconds from the Ansible module output
func (m *ManagerImplement) parseUptimeSeconds(input interface{}) (uptimeSeconds int64, err error) {
    m.Logger.V(2).Info("start parseUptimeSeconds")
    defer func() {
        m.Logger.V(2).Info("end parseUptimeSeconds", "uptime_seconds", uptimeSeconds)
    }()

	m.Logger.V(2).Info("branch: parsing uptimeSeconds response")
	fixedErrorMsg := "invalid uptimeSeconds of server response: "
	json, ok := input.(map[string]interface{})
	if !ok {
		m.Logger.V(2).Info("branch: JSON unmarshaling failed")
		err = errors.New("failed to unmarshal JSON")
		m.Logger.Error(err, fixedErrorMsg+err.Error())
		return
	}

	// ket exist check
	v, ok := json["uptime_seconds"]
	if !ok {
		err = errors.New("Key uptime_seconds is not found in inputJson")
		return
	}

	// type check
	floatVal, ok := v.(float64)
	if !ok {
		err = errors.New("Value for key uptime_seconds is not a float64")
		return
	}
	uptimeSeconds = int64(floatVal)

    return
}


func (m *ManagerImplement) setMachineP2P(server interfaces.ServerTargetList, power string) (err error) {
    m.Logger.V(2).Info("start setMachineP2P", "server_id", server.ServerID, "power", power)
    defer func() {
        m.Logger.V(2).Info("end setMachineP2P", "err", err)
    }()

    // parse extra parameters
    extraParams := extra_parameters.PgCDIExtraParameters{}
    if err = json.Unmarshal([]byte(server.CdiInfo.ExtraParameters), &extraParams); err != nil {
        m.Logger.V(2).Info("branch: ansible command failed", "server_id", server.ServerID, "error", err.Error())
        m.Logger.Error(err, err.Error())
        return
    }

	// generate extra option
	extra := fmt.Sprintf("cdi_user=%s cdi_password=%s cdi_guest=%s group_name=%s machine_name=%s p2p=%s",
		extraParams.CDIUser,
		extraParams.CDIPassword,
		extraParams.CDIGuest,
		server.CdiInfo.GroupName,
		server.CdiInfo.MachineName,
		power,
	)

    cdiAnsible := m.CdiAnsible
	if cdiAnsible == nil {
		// fallback to default implementation if not injected
		cdiAnsible = &pg_cdi.PgCDIAnsibleImple{Logger: m.Logger}
	}

    // execute Ansible module to set machine p2p
    playbook := "machine_p2p.yaml"
	sshKey := ""
	if config := utils.GetConfig(); config != nil {
		sshKey = config.SshKey
	}
    m.Logger.V(2).Info("branch: executing ansible command", "server_id", server.ServerID, "playbook", playbook)
    errMsg, _ := cdiAnsible.CmdExecute(context.Background(), server.CdiInfo.RemoteHost, server.CdiInfo.RemoteUser, sshKey, playbook, extra)
	if errMsg != nil {
        err = errors.New(*errMsg)
        m.Logger.V(2).Info("branch: ansible command failed", "server_id", server.ServerID, "error", err.Error())
        m.Logger.Error(err, err.Error())
		return
    }

    return
}
