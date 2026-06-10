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
	"errors"
	"exporter_module/internal/server/interfaces" // import for interface
	"fmt"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// HTTP custom error struct
type CustomError struct {
	StatusCode int
	Message    string
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("<%d> %s", e.StatusCode, e.Message)
}

// struct of Server
type ServerImplement struct {
	Logger  klog.Logger
	Ansible interfaces.Ansible
	API     interfaces.API
	Metrics interfaces.Metrics
	Manager interfaces.Manager
}

func (l *ServerImplement) Init() (err error) {
	defer func() {
		l.Logger.V(2).Info("end Init", "error", err)
	}()
	l.Logger.V(2).Info("start Init")

	l.Logger.V(2).Info("branch: initializing metrics")
	err = l.Metrics.Init([]*prometheus.GaugeVec{cpuGauge, memoryGauge, p2pEnableGauge})
	if err != nil {
		l.Logger.V(2).Info("branch: metrics initialization failed", "error", err.Error())
		return
	}

	l.Logger.V(2).Info("branch: metrics initialization successful")
	return nil
}

func (l *ServerImplement) Finalize() {
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()
	l.Logger.V(2).Info("start Finalize")

	l.Logger.V(2).Info("branch: finalizing metrics")
	l.Metrics.Finalize()
}

func (l *ServerImplement) Colloction(targetList []interfaces.ServerTargetList) {
	defer func() {
		l.Logger.V(2).Info("end Colloction", "target_count", len(targetList))
	}()
	l.Logger.V(2).Info("start Colloction", "target_count", len(targetList))

	// get SSH_KEY from env
	sshKey := os.Getenv("SSH_KEY")
	// ansible playbook
	playbook := "get_metrics.yaml"

	l.Logger.Info("server metrics collection started", "target_count", len(targetList))

	updateTargetList := make([]interfaces.ServerTargetList, 0)
	for i, target := range targetList {
        func() {
            defer func() {
                updateTargetList = append(updateTargetList, target)
                l.Logger.V(2).Info("branch: target added to update list", "server_id", target.ServerID)
            }()
			l.Logger.V(2).Info("branch: processing server target", "index", i, "server_id", target.ServerID)

			// Get metrics(CPU/Memory usage)
			cpuUsage, memoryUsage, err := l.getServerMetrics(target, sshKey, playbook)
			if err == nil {
				l.Logger.V(2).Info("branch: server metrics obtained successfully", "server_id", target.ServerID)
				// write CPU metrics
				err = l.Metrics.Write(cpuGauge, prometheus.Labels{"serverId": target.ServerID}, cpuUsage, &target.WritedMetrics)
				if err != nil {
					l.Logger.V(2).Info("branch: CPU metrics write failed", "server_id", target.ServerID, "error", err.Error())
				}

				// write memory metrics
				err = l.Metrics.Write(memoryGauge, prometheus.Labels{"serverId": target.ServerID}, memoryUsage, &target.WritedMetrics)
				if err != nil {
					l.Logger.V(2).Info("branch: memory metrics write failed", "server_id", target.ServerID, "error", err.Error())
				}
			} else {
				l.Logger.V(2).Info("branch: server metrics collection failed", "server_id", target.ServerID, "error", err.Error())
			}
		}()
	}
	l.Manager.ServerList(updateTargetList)

	l.Logger.Info("server metrics collection completed", "target_count", len(targetList))
}

func (l *ServerImplement) getServerMetrics(target interfaces.ServerTargetList, sshKey string, playbook string) (cpuUsage float64, memoryUsage float64, err error) {
	defer func() {
		l.Logger.V(2).Info("end getServerMetrics",
			"server_id", target.ServerID,
			"cpuUsage", cpuUsage,
			"memoryUsage", memoryUsage,
			"error", err)
	}()
	l.Logger.V(2).Info("start getServerMetrics",
		"server_id", target.ServerID,
		"host_ip", target.HostIPAddress,
		"playbook", playbook)

	l.Logger.V(2).Info("branch: executing ansible command", "server_id", target.ServerID)
	output, err := l.Ansible.CmdExecute(context.Background(), target.HostIPAddress, target.LoginUser, sshKey, playbook, "")
	if err != nil {
		l.Logger.V(2).Info("branch: ansible command failed", "server_id", target.ServerID, "error", err.Error())
		l.Logger.Error(err, err.Error())
		return
	}

	l.Logger.V(2).Info("branch: parsing metrics response", "server_id", target.ServerID)
	fixedErrorMsg := "invalid metrics of server response: "
	metrics, ok := output.(map[string]interface{})
	if !ok {
		l.Logger.V(2).Info("branch: JSON unmarshaling failed", "server_id", target.ServerID)
		err = errors.New("failed to unmarshal JSON")
		l.Logger.Error(err, fixedErrorMsg+err.Error())
		return
	}

	l.Logger.V(2).Info("branch: extracting CPU usage", "server_id", target.ServerID)
	cpuUsage, err = getJsonFloatValue(metrics, "cpu_usage")
	if err != nil {
		l.Logger.V(2).Info("branch: CPU usage extraction failed", "server_id", target.ServerID, "error", err.Error())
		l.Logger.Error(err, fixedErrorMsg+err.Error())
		return
	}

	l.Logger.V(2).Info("branch: extracting memory usage", "server_id", target.ServerID)
	memoryUsage, err = getJsonFloatValue(metrics, "mem_usage")
	if err != nil {
		l.Logger.V(2).Info("branch: memory usage extraction failed", "server_id", target.ServerID, "error", err.Error())
		l.Logger.Error(err, fixedErrorMsg+err.Error())
		return
	}

	l.Logger.V(2).Info("branch: metrics extraction successful", "server_id", target.ServerID)
	return
}

func getValue(inputJson interface{}, key string) (value interface{}, err error) {

	obj, ok := inputJson.(map[string]interface{})
	if !ok {
		err = errors.New("inputJson is not invalid type")
		return
	}

	// ket exist check
	v, ok := obj[key]
	if !ok {
		err = errors.New("Key " + key + " not found in inputJson")
		return
	}
	value = v
	return
}

func getJsonFloatValue(inputJson interface{}, key string) (value float64, err error) {

	v, err := getValue(inputJson, key)
	if err != nil {
		return
	}

	// type check
	floatVal, ok := v.(float64)
	if !ok {
		strVal, ok := v.(string)
		if !ok {
			err = errors.New("Value for key " + key + " is not a float64")
			return
		}
		floatVal, err = strconv.ParseFloat(strVal, 64)
		if err != nil {
			return
		}
	}
	value = floatVal
	return
}
