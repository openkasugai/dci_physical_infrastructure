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
