package interfaces

import "github.com/prometheus/client_golang/prometheus"

type MetricLabel struct {
    Metrics *prometheus.GaugeVec
    Label   prometheus.Labels
}

// interface of Metrics
type Metrics interface {
	Init(metrics []*prometheus.GaugeVec) (err error)
	Finalize()
	Write(metrics *prometheus.GaugeVec, label prometheus.Labels, value float64, writedMetrics *[]MetricLabel) (err error)
	Delete(metrics *prometheus.GaugeVec, label prometheus.Labels) (err error)
}
