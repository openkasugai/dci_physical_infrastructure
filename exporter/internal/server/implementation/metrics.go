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
	"exporter_module/internal/server/interfaces" // import for interface
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

var (
	// Metrics for network
	rxOkGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_rx_ok",
			Help: "Number of received packets",
		},
		[]string{"host", "interface"},
	)
	txOkGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_tx_ok",
			Help: "Number of transmitted packets",
		},
		[]string{"host", "interface"},
	)
	rxBpsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_rx_bps",
			Help: "Received bits per second",
		},
		[]string{"host", "interface"},
	)
	txBpsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_tx_bps",
			Help: "Transmitted bits per second",
		},
		[]string{"host", "interface"},
	)
	rxUtilGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_rx_util",
			Help: "Receive utilization percentage",
		},
		[]string{"host", "interface"},
	)
	txUtilGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_tx_util",
			Help: "Transmit utilization percentage",
		},
		[]string{"host", "interface"},
	)
	rxErrGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_rx_err",
			Help: "Number of received errors",
		},
		[]string{"host", "interface"},
	)
	txErrGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_tx_err",
			Help: "Number of transmitted errors",
		},
		[]string{"host", "interface"},
	)
	rxDrpGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_rx_drp",
			Help: "Number of received dropped packets",
		},
		[]string{"host", "interface"},
	)
	txDrpGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_tx_drp",
			Help: "Number of transmitted dropped packets",
		},
		[]string{"host", "interface"},
	)
	rxOvrGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_rx_ovr",
			Help: "Number of received overflows",
		},
		[]string{"host", "interface"},
	)
	txOvrGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "interface_tx_ovr",
			Help: "Number of transmitted overflows",
		},
		[]string{"host", "interface"},
	)

	// Metrics for server
	cpuGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "server_cpu_usage",
			Help: "CPU usage",
		},
		[]string{"serverId"},
	)
	memoryGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "server_memory_usage",
			Help: "Memory usage",
		},
		[]string{"serverId"},
	)
	p2pEnableGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "server_p2p_enable",
			Help: "P2P Enable Status",
		},
		[]string{"serverId"},
	)
)

// struct of Metrics
type MetricsImplement struct {
	Logger klog.Logger
}

func (l MetricsImplement) Init(metrics []*prometheus.GaugeVec) (err error) {
	l.Logger.V(2).Info("start Init", "metricsCount", len(metrics))
	defer func() {
		l.Logger.V(2).Info("end Init", "err", err)
	}()

	l.Logger.Info("initializing Prometheus metrics")

	// register metrics to prometheus registory
	for i, m := range metrics {
		l.Logger.V(2).Info("registering metric", "index", i)
		prometheus.MustRegister(m)
	}

	l.Logger.Info("Prometheus metrics initialization completed", "registeredCount", len(metrics))
	return nil
}

func (l MetricsImplement) Finalize() {
	l.Logger.V(2).Info("start Finalize")
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()

	l.Logger.Info("finalizing Prometheus metrics")
	l.Logger.V(2).Info("branch: no cleanup required for Prometheus metrics")
	// nothing to do
}

func (l MetricsImplement) Write(metrics *prometheus.GaugeVec, label prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) (err error) {
	l.Logger.V(2).Info("start Write", "label", label, "value", value)
	defer func() {
		l.Logger.V(2).Info("end Write", "err", err)
	}()

	// update metrics
	metrics.With(label).Set(value)
	l.Logger.V(2).Info("branch: metric value set successfully", "label", label, "value", value)

    // append to writed metrics list if provided
    if writedMetrics != nil {
        *writedMetrics = append(*writedMetrics, interfaces.MetricLabel{
            Metrics: metrics,
            Label:   label,
        })
        l.Logger.V(2).Info("branch: metric added to tracking list", "label", label, "listSize", len(*writedMetrics))
    } else {
        l.Logger.V(2).Info("branch: metric tracking not enabled", "label", label)
    }

	return nil
}

func (l MetricsImplement) Delete(metrics *prometheus.GaugeVec, label prometheus.Labels) (err error) {
	l.Logger.V(2).Info("start Delete", "label", label)
	defer func() {
		l.Logger.V(2).Info("end Delete", "err", err)
	}()

	// delete metrics
	metrics.Delete(label)
	l.Logger.V(2).Info("branch: metric deleted successfully", "label", label)
	return nil
}
