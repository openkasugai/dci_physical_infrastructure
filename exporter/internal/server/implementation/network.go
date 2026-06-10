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
	"common/models"
	"exporter_module/internal/server/interfaces" // import for interface
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// struct of network
type NetworkImplement struct {
	Logger  klog.Logger
	Ansible interfaces.Ansible
	Metrics interfaces.Metrics
	Manager interfaces.Manager
}

func (l *NetworkImplement) Init() (err error) {
	defer func() {
		l.Logger.V(2).Info("end Init", "error", err)
	}()
	l.Logger.V(2).Info("start Init")

	l.Logger.V(2).Info("branch: initializing network metrics")
	err = l.Metrics.Init([]*prometheus.GaugeVec{
		rxOkGauge, txOkGauge, rxBpsGauge, txBpsGauge,
		rxUtilGauge, txUtilGauge, rxErrGauge, txErrGauge,
		rxDrpGauge, txDrpGauge, rxOvrGauge, txOvrGauge,
	})
	if err != nil {
		l.Logger.V(2).Info("branch: network metrics initialization failed", "error", err.Error())
		return
	}

	l.Logger.V(2).Info("branch: network metrics initialization successful")
	return nil
}

func (l *NetworkImplement) Finalize() {
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()
	l.Logger.V(2).Info("start Finalize")

	l.Logger.V(2).Info("branch: finalizing network metrics")
	l.Metrics.Finalize()
}

func (l *NetworkImplement) Collection(targetList []interfaces.NetworkTargetList) {
	defer func() {
		l.Logger.V(2).Info("end Collection", "target_count", len(targetList))
	}()
	l.Logger.V(2).Info("start Collection", "target_count", len(targetList))

	// get SSH_KEY from env
	sshKey := os.Getenv("SSH_KEY")

	l.Logger.Info("network metrics collection started", "target_count", len(targetList))

	updateTargetList := make([]interfaces.NetworkTargetList, 0)
	for i, target := range targetList {
        func() {
            defer func() {
                updateTargetList = append(updateTargetList, target)
                l.Logger.V(2).Info("branch: target added to update list", "ip_address", target.IPAddress)
            }()
			l.Logger.V(2).Info("branch: processing network target", "index", i, "ip_address", target.IPAddress)

			// determine product type
			product := models.ParseProductTypeFromJSON[models.NWProductType](target.ProductInfo)
			if (product == models.EdgeCoreSonic) || (product == models.BroadcomSonic) {	// EdgeCore Sonic or Broadcom Sonic
				l.Logger.V(2).Info("branch: target is SONiC product", "product", product, "ip_address", target.IPAddress)

				// ansible execute
				playbook := "get_counter.yaml"
				l.Logger.V(2).Info("branch: executing ansible command", "ip_address", target.IPAddress, "playbook", playbook)
				output, err := l.Ansible.CmdExecute(context.Background(), target.IPAddress, target.LoginUser, sshKey, playbook, "")
				if err != nil {
					l.Logger.V(2).Info("branch: ansible command failed", "ip_address", target.IPAddress, "error", err.Error())
					l.Logger.Error(err, err.Error())
					return
				}

				l.Logger.V(2).Info("branch: parsing and writing metrics", "ip_address", target.IPAddress)
				// parse and write metrics
				l.parseAndWriteMetrics(target.IPAddress, output, &target.WritedMetrics)

			} else {
				l.Logger.V(2).Info("branch: ", "target is unsupport product", "ProductInfo", target.ProductInfo)
			}
		}()
	}
	l.Manager.NetworkList(updateTargetList)

	l.Logger.Info("network metrics collection completed", "target_count", len(targetList))
}

func (l *NetworkImplement) parseAndWriteMetrics(host string, input interface{}, writedMetrics *[]interfaces.MetricLabel) {
	defer func() {
		l.Logger.V(2).Info("end parseAndWriteMetrics", "host", host)
	}()
	l.Logger.V(2).Info("start parseAndWriteMetrics", "host", host)

	headerSkipped := false
	fixedErrorMsg := "invalid nw switch response: "

	for _, msg := range input.([]interface{}) {
		line := msg.(string)
		line = strings.TrimSpace(line)

		// Skip empty lines and header lines
		if line == "" || strings.HasPrefix(line, "IFACE") || strings.HasPrefix(line, "----------") {
			if !headerSkipped {
				l.Logger.V(2).Info("branch: skipping header line", "host", host)
				headerSkipped = true
				continue
			} else {
				continue // Skip the second header line
			}
		}

		fields := strings.Fields(line)

		// Check if the line has enough fields
		if len(fields) != 16 {
			l.Logger.V(2).Info("branch: skipping line with insufficient fields", "host", host, "field_count", len(fields))
			continue
		}

		// Extract data
		iface := fields[0]
		l.Logger.V(2).Info("branch: processing interface", "host", host, "interface", iface)

		rxOKValue := fields[2]
		rxBPSValue := fields[3]
		rxUtilValue := fields[5]
		rxErrValue := fields[6]
		rxDrpValue := fields[7]
		rxOvrValue := fields[8]
		txOKValue := fields[9]
		txBPSValue := fields[10]
		txUtilValue := fields[12]
		txErrValue := fields[13]
		txDrpValue := fields[14]
		txOvrValue := fields[15]

		// Convert string values to float64
		rxOKFloat, err := parseFloat(rxOKValue)
		if err != nil {
			l.Logger.V(2).Info("branch: rx_ok parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"rx_ok for %s: %v", iface, err)
			continue
		}

		rxBPSFloat, err := parseFloat(rxBPSValue)
		if err != nil {
			l.Logger.V(2).Info("branch: rx_bps parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"rx_bps for %s: %v", iface, err)
			continue
		}

		rxUtilFloat, err := parsePercentage(rxUtilValue)
		if err != nil {
			l.Logger.V(2).Info("branch: rx_util parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"rx_util for %s: %v", iface, err)
			continue
		}

		rxErrFloat, err := parseFloat(rxErrValue)
		if err != nil {
			l.Logger.V(2).Info("branch: rx_err parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"rx_err for %s: %v", iface, err)
			continue
		}

		rxDrpFloat, err := parseFloat(rxDrpValue)
		if err != nil {
			l.Logger.V(2).Info("branch: rx_drp parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"rx_drp for %s: %v", iface, err)
			continue
		}

		rxOvrFloat, err := parseFloat(rxOvrValue)
		if err != nil {
			l.Logger.V(2).Info("branch: rx_ovr parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"rx_ovr for %s: %v", iface, err)
			continue
		}

		txOKFloat, err := parseFloat(txOKValue)
		if err != nil {
			l.Logger.V(2).Info("branch: tx_ok parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"tx_ok for %s: %v", iface, err)
			continue
		}

		txBPSFloat, err := parseFloat(txBPSValue)
		if err != nil {
			l.Logger.V(2).Info("branch: tx_bps parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"tx_bps for %s: %v", iface, err)
			continue
		}

		txUtilFloat, err := parsePercentage(txUtilValue)
		if err != nil {
			l.Logger.V(2).Info("branch: tx_util parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"tx_util for %s: %v", iface, err)
			continue
		}

		txErrFloat, err := parseFloat(txErrValue)
		if err != nil {
			l.Logger.V(2).Info("branch: tx_err parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"tx_err for %s: %v", iface, err)
			continue
		}

		txDrpFloat, err := parseFloat(txDrpValue)
		if err != nil {
			l.Logger.V(2).Info("branch: tx_drp parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"tx_drp for %s: %v", iface, err)
			continue
		}

		txOvrFloat, err := parseFloat(txOvrValue)
		if err != nil {
			l.Logger.V(2).Info("branch: tx_ovr parsing failed", "host", host, "interface", iface, "error", err.Error())
			l.Logger.Error(err, fixedErrorMsg+"tx_ovr for %s: %v", iface, err)
			continue
		}

		l.Logger.V(2).Info("branch: writing metrics to Prometheus", "host", host, "interface", iface)
		// Set Prometheus metrics
		err = l.Metrics.Write(rxOkGauge, prometheus.Labels{"host": host, "interface": iface}, rxOKFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: rxOkGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(rxBpsGauge, prometheus.Labels{"host": host, "interface": iface}, rxBPSFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: rxBpsGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(rxUtilGauge, prometheus.Labels{"host": host, "interface": iface}, rxUtilFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: rxUtilGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(txOkGauge, prometheus.Labels{"host": host, "interface": iface}, txOKFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: txOkGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(txBpsGauge, prometheus.Labels{"host": host, "interface": iface}, txBPSFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: txBpsGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(txUtilGauge, prometheus.Labels{"host": host, "interface": iface}, txUtilFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: txUtilGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(rxErrGauge, prometheus.Labels{"host": host, "interface": iface}, rxErrFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: rxErrGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(rxDrpGauge, prometheus.Labels{"host": host, "interface": iface}, rxDrpFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: rxDrpGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(rxOvrGauge, prometheus.Labels{"host": host, "interface": iface}, rxOvrFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: rxOvrGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(txErrGauge, prometheus.Labels{"host": host, "interface": iface}, txErrFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: txErrGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(txDrpGauge, prometheus.Labels{"host": host, "interface": iface}, txDrpFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: txDrpGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}
		err = l.Metrics.Write(txOvrGauge, prometheus.Labels{"host": host, "interface": iface}, txOvrFloat, writedMetrics)
		if err != nil {
			l.Logger.V(2).Info("branch: txOvrGauge write failed", "host", host, "interface", iface, "error", err.Error())
		}

		l.Logger.V(2).Info("branch: metrics written successfully", "host", host, "interface", iface)
	}

	l.Logger.Info("metrics parsing and writing completed", "host", host)
}

// Helper function to parse float values
func parseFloat(value string) (float64, error) {
	value = strings.ReplaceAll(value, ",", "")
	return strconv.ParseFloat(value, 64)
}

// Helper function to parse percentage values
func parsePercentage(value string) (float64, error) {
	value = strings.ReplaceAll(value, "%", "")
	value = strings.TrimSpace(value)
	return strconv.ParseFloat(value, 64)
}
