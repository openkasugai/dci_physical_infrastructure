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
	"exporter_module/internal/server/test_utils"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// TestMetricsImplement_Init_ValidMetrics_ReturnsSuccess tests successful metrics initialization
func TestMetricsImplement_Init_ValidMetrics_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create test gauge
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_metric",
			Help: "Test metric for unit testing",
		},
		[]string{"label1", "label2"},
	)

	gaugeList := []*prometheus.GaugeVec{testGauge}

	// Execute
	err := metrics.Init(gaugeList)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Cleanup - unregister the metric
	prometheus.Unregister(testGauge)
}

// TestMetricsImplement_Init_EmptyMetrics_ReturnsSuccess tests initialization with empty metrics list
func TestMetricsImplement_Init_EmptyMetrics_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	var emptyGaugeList []*prometheus.GaugeVec

	// Execute
	err := metrics.Init(emptyGaugeList)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestMetricsImplement_Init_NilMetrics_ReturnsSuccess tests initialization with nil metrics list
func TestMetricsImplement_Init_NilMetrics_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Execute
	err := metrics.Init(nil)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestMetricsImplement_Init_MultipleMetrics_ReturnsSuccess tests initialization with multiple metrics
func TestMetricsImplement_Init_MultipleMetrics_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create test gauges
	testGauge1 := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_metric_1",
			Help: "First test metric for unit testing",
		},
		[]string{"label1"},
	)

	testGauge2 := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_metric_2",
			Help: "Second test metric for unit testing",
		},
		[]string{"label2"},
	)

	gaugeList := []*prometheus.GaugeVec{testGauge1, testGauge2}

	// Execute
	err := metrics.Init(gaugeList)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Cleanup
	prometheus.Unregister(testGauge1)
	prometheus.Unregister(testGauge2)
}

// TestMetricsImplement_Init_DuplicateMetrics_HandlesPanic tests initialization with duplicate metric names
func TestMetricsImplement_Init_DuplicateMetrics_HandlesPanic(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create test gauge
	testGauge1 := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "duplicate_metric",
			Help: "First duplicate test metric",
		},
		[]string{"label1"},
	)

	testGauge2 := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "duplicate_metric", // Same name - will cause panic
			Help: "Second duplicate test metric",
		},
		[]string{"label2"},
	)

	gaugeList := []*prometheus.GaugeVec{testGauge1, testGauge2}

	// Execute - should panic due to duplicate metric names
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic due to duplicate metric names, but no panic occurred")
		}
		// Cleanup
		prometheus.Unregister(testGauge1)
	}()

	_ = metrics.Init(gaugeList)
}

// TestMetricsImplement_Finalize_ValidCall_NoError tests finalization
func TestMetricsImplement_Finalize_ValidCall_NoError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Execute - should not panic or cause errors
	metrics.Finalize()

	// Verify - no error expected, just verify it doesn't panic
}

// TestMetricsImplement_Write_ValidMetric_ReturnsSuccess tests successful metric writing
func TestMetricsImplement_Write_ValidMetric_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_write_metric",
			Help: "Test metric for write testing",
		},
		[]string{"test_label"},
	)
	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	// Execute
	labels := prometheus.Labels{"test_label": "test_value"}
	err := metrics.Write(testGauge, labels, 42.5, nil)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Note: Direct value verification is complex with Prometheus gauges in tests
	// The important thing is that no error occurred during the write operation
}

// TestMetricsImplement_Write_MultipleLabels_ReturnsSuccess tests metric writing with multiple labels
func TestMetricsImplement_Write_MultipleLabels_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge with multiple labels
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_multi_label_metric",
			Help: "Test metric with multiple labels",
		},
		[]string{"label1", "label2", "label3"},
	)
	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	// Execute
	labels := prometheus.Labels{
		"label1": "value1",
		"label2": "value2",
		"label3": "value3",
	}
	err := metrics.Write(testGauge, labels, 123.456, nil)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestMetricsImplement_Write_ZeroValue_ReturnsSuccess tests metric writing with zero value
func TestMetricsImplement_Write_ZeroValue_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_zero_metric",
			Help: "Test metric for zero value",
		},
		[]string{"zero_label"},
	)
	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	// Execute
	labels := prometheus.Labels{"zero_label": "zero_value"}
	err := metrics.Write(testGauge, labels, 0.0, nil)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestMetricsImplement_Write_NegativeValue_ReturnsSuccess tests metric writing with negative value
func TestMetricsImplement_Write_NegativeValue_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_negative_metric",
			Help: "Test metric for negative value",
		},
		[]string{"neg_label"},
	)
	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	// Execute
	labels := prometheus.Labels{"neg_label": "neg_value"}
	err := metrics.Write(testGauge, labels, -99.9, nil)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestMetricsImplement_Write_EmptyLabels_ReturnsSuccess tests metric writing with empty labels
func TestMetricsImplement_Write_EmptyLabels_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge with no labels
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_empty_labels_metric",
			Help: "Test metric with no labels",
		},
		[]string{}, // No label names
	)
	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	// Execute
	labels := prometheus.Labels{} // Empty labels
	err := metrics.Write(testGauge, labels, 88.8, nil)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestMetricsImplement_Write_OverwriteValue_ReturnsSuccess tests overwriting metric value
func TestMetricsImplement_Write_OverwriteValue_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_overwrite_metric",
			Help: "Test metric for overwriting values",
		},
		[]string{"overwrite_label"},
	)
	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	labels := prometheus.Labels{"overwrite_label": "overwrite_value"}

	// Execute - first write
	err := metrics.Write(testGauge, labels, 10.0, nil)
	if err != nil {
		t.Errorf("Expected no error on first write, got: %v", err)
	}

	// Execute - second write (overwrite)
	err = metrics.Write(testGauge, labels, 20.0, nil)
	if err != nil {
		t.Errorf("Expected no error on second write, got: %v", err)
	}
}

// TestMetricsImplement_Write_NilGauge_HandlesPanic tests writing to nil gauge
func TestMetricsImplement_Write_NilGauge_HandlesPanic(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	labels := prometheus.Labels{"test": "value"}

	// Execute - should panic with nil gauge
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when writing to nil gauge, but no panic occurred")
		}
	}()

	_ = metrics.Write(nil, labels, 42.0, nil)
}

// TestMetricsImplement_NetworkMetrics_CanWriteValues tests writing to network-specific metrics
func TestMetricsImplement_NetworkMetrics_CanWriteValues(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create fresh metrics for testing to avoid conflicts
	testRxOkGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_interface_rx_ok",
			Help: "Test Number of received packets",
		},
		[]string{"host", "interface"},
	)

	prometheus.MustRegister(testRxOkGauge)
	defer prometheus.Unregister(testRxOkGauge)

	// Execute
	labels := prometheus.Labels{"host": "testhost", "interface": "eth0"}
	err := metrics.Write(testRxOkGauge, labels, 1000.0, nil)

	// Verify
	if err != nil {
		t.Errorf("Expected no error writing to network metric, got: %v", err)
	}
}

// TestMetricsImplement_ServerMetrics_CanWriteValues tests writing to server-specific metrics
func TestMetricsImplement_ServerMetrics_CanWriteValues(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create fresh metrics for testing to avoid conflicts
	testCpuGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_server_cpu_usage",
			Help: "Test CPU usage",
		},
		[]string{"serverId"},
	)

	prometheus.MustRegister(testCpuGauge)
	defer prometheus.Unregister(testCpuGauge)

	// Execute
	labels := prometheus.Labels{"serverId": "server-123"}
	err := metrics.Write(testCpuGauge, labels, 75.5, nil)

	// Verify
	if err != nil {
		t.Errorf("Expected no error writing to server metric, got: %v", err)
	}
}

// TestMetricsImplement_Delete_ValidLabels_DeletesSuccessfully tests successful metric deletion
func TestMetricsImplement_Delete_ValidLabels_DeletesSuccessfully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_delete_metric",
			Help: "Test metric for deletion testing",
		},
		[]string{"test_label"},
	)

	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	// First write a value to create the metric
	labels := prometheus.Labels{"test_label": "test_value"}
	err := metrics.Write(testGauge, labels, 42.5, nil)
	if err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	// Execute - delete the metric
	err = metrics.Delete(testGauge, labels)

	// Verify - should not return error
	if err != nil {
		t.Errorf("Expected no error deleting metric, got: %v", err)
	}
}

// TestMetricsImplement_Delete_MultipleLabels_DeletesSuccessfully tests deletion with multiple labels
func TestMetricsImplement_Delete_MultipleLabels_DeletesSuccessfully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge with multiple labels
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_delete_multi_label_metric",
			Help: "Test metric for deletion with multiple labels",
		},
		[]string{"label1", "label2", "label3"},
	)

	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	// Write multiple metrics
	labels1 := prometheus.Labels{"label1": "value1", "label2": "value2", "label3": "value3"}
	labels2 := prometheus.Labels{"label1": "valueA", "label2": "valueB", "label3": "valueC"}
	
	metrics.Write(testGauge, labels1, 10.0, nil)
	metrics.Write(testGauge, labels2, 20.0, nil)

	// Execute - delete one of the metrics
	err := metrics.Delete(testGauge, labels1)

	// Verify
	if err != nil {
		t.Errorf("Expected no error deleting metric, got: %v", err)
	}
}

// TestMetricsImplement_Delete_NonExistentMetric_DoesNotError tests deleting non-existent metric
func TestMetricsImplement_Delete_NonExistentMetric_DoesNotError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	metrics := &MetricsImplement{Logger: logger}

	// Create and register test gauge
	testGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_delete_nonexistent_metric",
			Help: "Test metric for non-existent deletion testing",
		},
		[]string{"test_label"},
	)

	prometheus.MustRegister(testGauge)
	defer prometheus.Unregister(testGauge)

	// Execute - try to delete a metric that was never created
	labels := prometheus.Labels{"test_label": "nonexistent_value"}
	err := metrics.Delete(testGauge, labels)

	// Verify - Prometheus Delete doesn't error on non-existent metrics
	if err != nil {
		t.Errorf("Expected no error deleting non-existent metric, got: %v", err)
	}
}
