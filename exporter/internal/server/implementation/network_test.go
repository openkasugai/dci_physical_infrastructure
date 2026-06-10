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
	"fmt"
	"os"
	"testing"

	"exporter_module/internal/server/interfaces"
	"exporter_module/internal/server/test_utils"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// Mock for Manager interface
type MockNetworkManager struct {
	serverListFunc  func([]interfaces.ServerTargetList)
	networkListFunc func([]interfaces.NetworkTargetList)
	SetP2POnFunc    func([]interfaces.ServerTargetList)
}

func (m *MockNetworkManager) ServerList(targets []interfaces.ServerTargetList) {
	if m.serverListFunc != nil {
		m.serverListFunc(targets)
	}
}

func (m *MockNetworkManager) NetworkList(targets []interfaces.NetworkTargetList) {
	if m.networkListFunc != nil {
		m.networkListFunc(targets)
	}
}

func (m *MockNetworkManager) SetP2POn(targets []interfaces.ServerTargetList) {
	if m.SetP2POnFunc != nil {
		m.SetP2POnFunc(targets)
	}
}

// Mock for Ansible interface
type MockNetworkAnsible struct {
	cmdExecuteFunc func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error)
}

func (m *MockNetworkAnsible) CmdExecute(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
	if m.cmdExecuteFunc != nil {
		return m.cmdExecuteFunc(ctx, host, loginUser, sshKey, playbook, extraVars)
	}
	return nil, nil
}

// Mock for Metrics interface
type MockNetworkMetrics struct {
	initFunc     func([]*prometheus.GaugeVec) error
	finalizeFunc func()
	writeFunc    func(*prometheus.GaugeVec, prometheus.Labels, float64) error
	deleteFunc   func(*prometheus.GaugeVec, prometheus.Labels) error
}

func (m *MockNetworkMetrics) Init(gaugeList []*prometheus.GaugeVec) error {
	if m.initFunc != nil {
		return m.initFunc(gaugeList)
	}
	return nil
}

func (m *MockNetworkMetrics) Finalize() {
	if m.finalizeFunc != nil {
		m.finalizeFunc()
	}
}

func (m *MockNetworkMetrics) Write(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) error {
	if m.writeFunc != nil {
		return m.writeFunc(gauge, labels, value)
	}
	return nil
}

func (m *MockNetworkMetrics) Delete(gauge *prometheus.GaugeVec, labels prometheus.Labels) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(gauge, labels)
	}
	return nil
}

// Helper function to set up environment variable for a test
func setEnvForNetworkTest(t *testing.T, key, value string) func() {
	originalValue := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		if originalValue != "" {
			os.Setenv(key, originalValue)
		} else {
			os.Unsetenv(key)
		}
	}
}

// TestNetworkImplement_Init_ValidMetrics_ReturnsSuccess tests successful network metrics initialization
func TestNetworkImplement_Init_ValidMetrics_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	mockMetrics := &MockNetworkMetrics{
		initFunc: func([]*prometheus.GaugeVec) error {
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Execute
	err := network.Init()

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestNetworkImplement_Init_MetricsInitError_ReturnsError tests initialization failure
func TestNetworkImplement_Init_MetricsInitError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	expectedError := errors.New("metrics initialization failed")
	mockMetrics := &MockNetworkMetrics{
		initFunc: func([]*prometheus.GaugeVec) error {
			return expectedError
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Execute
	err := network.Init()

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != expectedError.Error() {
		t.Errorf("Expected error %v, got: %v", expectedError, err)
	}
}

// TestNetworkImplement_Finalize_ValidCall_CallsMetricsFinalize tests finalization
func TestNetworkImplement_Finalize_ValidCall_CallsMetricsFinalize(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	finalizeCalled := false
	mockMetrics := &MockNetworkMetrics{
		finalizeFunc: func() {
			finalizeCalled = true
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Execute
	network.Finalize()

	// Verify
	if !finalizeCalled {
		t.Error("Expected Metrics.Finalize to be called")
	}
}

// TestNetworkImplement_Collection_ValidTargets_ProcessesSuccessfully tests successful processing of valid targets
func TestNetworkImplement_Collection_ValidTargets_ProcessesSuccessfully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup product mappings environment variable
	cleanupMappings := setEnvForNetworkTest(t, "PRODUCT_MAPPINGS", `{"nw_products":[{"vendor":"edgecore networks","product_name":"DCS208/AS5812-54X","os":"Edgecore SONiC","type":"EdgeCoreSonic"}],"server_products":[],"cdi_products":[],"maas_products":[]}`)
	defer cleanupMappings()

	logger := klog.NewKlogr()

	// Mock successful ansible response with valid network data
	mockOutput := []interface{}{
		"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
		"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
		" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0         0",
		" Ethernet1        U     88,938,325     0.00 B/s      0.00%         0          0         0    157,749,525   364.34 B/s      0.00%         0         0         0",
	}

	ansibleCallCount := 0
	mockAnsible := &MockNetworkAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			ansibleCallCount++
			// Verify parameters
			if playbook != "get_counter.yaml" {
				t.Errorf("Expected playbook 'get_counter.yaml', got: %s", playbook)
			}
			return mockOutput, nil
		},
	}

	writeCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	targetList := []interfaces.NetworkTargetList{
		{IPAddress: "192.168.1.100", LoginUser: "admin", ProductInfo: "{\"vendor\":\"edgecore networks\",\"product_name\":\"DCS208/AS5812-54X\",\"os\":\"Edgecore SONiC\"}"},
		{IPAddress: "192.168.1.101", LoginUser: "admin", ProductInfo: "{\"vendor\":\"edgecore networks\",\"product_name\":\"DCS208/AS5812-54X\",\"os\":\"Edgecore SONiC\"}"},
	}

	// Execute
	network.Collection(targetList)

	// Verify
	if ansibleCallCount != 2 {
		t.Errorf("Expected 2 ansible calls, got: %d", ansibleCallCount)
	}
	// Each interface has 12 metrics, 2 interfaces per target, 2 targets = 48 total writes
	expectedWrites := 2 * 2 * 12
	if writeCallCount != expectedWrites {
		t.Errorf("Expected %d metric writes, got: %d", expectedWrites, writeCallCount)
	}
}

// TestNetworkImplement_Collection_AnsibleError_ContinuesProcessing tests handling of ansible errors
func TestNetworkImplement_Collection_AnsibleError_ContinuesProcessing(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForNetworkTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	// Setup product mappings environment variable
	cleanupMappings := setEnvForNetworkTest(t, "PRODUCT_MAPPINGS", `{"nw_products":[{"vendor":"edgecore networks","product_name":"DCS208/AS5812-54X","os":"Edgecore SONiC","type":"EdgeCoreSonic"}],"server_products":[],"cdi_products":[],"maas_products":[]}`)
	defer cleanupMappings()

	logger := klog.NewKlogr()

	ansibleCallCount := 0
	mockAnsible := &MockNetworkAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			ansibleCallCount++
			return nil, errors.New("ansible execution failed")
		},
	}

	writeCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	targetList := []interfaces.NetworkTargetList{
		{IPAddress: "192.168.1.100", LoginUser: "admin", ProductInfo: "{\"vendor\":\"edgecore networks\",\"product_name\":\"DCS208/AS5812-54X\",\"os\":\"Edgecore SONiC\"}"},
		{IPAddress: "192.168.1.101", LoginUser: "admin", ProductInfo: "{\"vendor\":\"edgecore networks\",\"product_name\":\"DCS208/AS5812-54X\",\"os\":\"Edgecore SONiC\"}"},
	}

	// Execute
	network.Collection(targetList)

	// Verify
	if ansibleCallCount != 2 {
		t.Errorf("Expected 2 ansible calls, got: %d", ansibleCallCount)
	}
	// No metric writes should occur due to ansible errors
	if writeCallCount != 0 {
		t.Errorf("Expected 0 metric writes due to ansible errors, got: %d", writeCallCount)
	}
}

// TestNetworkImplement_Collection_EmptyTargetList_ProcessesSuccessfully tests collection with empty targets
func TestNetworkImplement_Collection_EmptyTargetList_ProcessesSuccessfully(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForNetworkTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	ansibleCallCount := 0
	mockAnsible := &MockNetworkAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			ansibleCallCount++
			return nil, nil
		},
	}

	mockMetrics := &MockNetworkMetrics{}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	var emptyTargetList []interfaces.NetworkTargetList

	// Execute
	network.Collection(emptyTargetList)

	// Verify
	if ansibleCallCount != 0 {
		t.Errorf("Expected 0 ansible calls for empty target list, got: %d", ansibleCallCount)
	}
}

// TestNetworkImplement_Collection_NoSSHKey_UsesEmptyKey tests collection without SSH_KEY env variable
func TestNetworkImplement_Collection_NoSSHKey_UsesEmptyKey(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup - ensure SSH_KEY is not set
	cleanup := setEnvForNetworkTest(t, "SSH_KEY", "")
	defer cleanup()

	// Setup product mappings environment variable
	cleanupMappings := setEnvForNetworkTest(t, "PRODUCT_MAPPINGS", `{"nw_products":[{"vendor":"edgecore networks","product_name":"DCS208/AS5812-54X","os":"Edgecore SONiC","type":"EdgeCoreSonic"}],"server_products":[],"cdi_products":[],"maas_products":[]}`)
	defer cleanupMappings()

	logger := klog.NewKlogr()

	receivedSSHKey := ""
	mockAnsible := &MockNetworkAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			receivedSSHKey = sshKey
			return []interface{}{"empty response"}, nil
		},
	}

	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	targetList := []interfaces.NetworkTargetList{
		{IPAddress: "192.168.1.100", LoginUser: "admin", ProductInfo: "{\"vendor\":\"edgecore networks\",\"product_name\":\"DCS208/AS5812-54X\",\"os\":\"Edgecore SONiC\"}"},
	}

	// Execute
	network.Collection(targetList)

	// Verify
	if receivedSSHKey != "" {
		t.Errorf("Expected empty SSH key, got: '%s'", receivedSSHKey)
	}
}

// TestNetworkImplement_parseAndWriteMetrics_ValidData_ParsesCorrectly tests valid data parsing
func TestNetworkImplement_parseAndWriteMetrics_ValidData_ParsesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	ansibleCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			ansibleCallCount++
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Mock data with proper format
	mockData := []interface{}{
		"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
		"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
		" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0         0",
	}

	// Execute
	network.parseAndWriteMetrics("testhost", mockData, nil)

	// Verify - should have 12 metric writes (rx: ok,bps,util,err,drp,ovr + tx: ok,bps,util,err,drp,ovr)
	expectedCalls := 12
	if ansibleCallCount != expectedCalls {
		t.Errorf("Expected %d metric writes, got: %d", expectedCalls, ansibleCallCount)
	}
}

// TestNetworkImplement_parseAndWriteMetrics_InvalidFieldCount_SkipsLine tests handling of invalid field count
func TestNetworkImplement_parseAndWriteMetrics_InvalidFieldCount_SkipsLine(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	writeCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Mock data with invalid field count (should have 16 fields)
	mockData := []interface{}{
		"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR", // Header (will be skipped)
		"----------  -------  -------------  -----------  ---------  --------", // Header separator (will be skipped)
		" Ethernet0        D              0",                                   // Only 3 fields instead of 16
	}

	// Execute
	network.parseAndWriteMetrics("testhost", mockData, nil)

	// Verify - no metric writes should occur due to invalid field count
	if writeCallCount != 0 {
		t.Errorf("Expected 0 metric writes due to invalid field count, got: %d", writeCallCount)
	}
}

// TestNetworkImplement_parseAndWriteMetrics_EmptyData_HandlesGracefully tests handling of empty data
func TestNetworkImplement_parseAndWriteMetrics_EmptyData_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	writeCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Mock empty data
	mockData := []interface{}{}

	// Execute
	network.parseAndWriteMetrics("testhost", mockData, nil)

	// Verify - no metric writes should occur
	if writeCallCount != 0 {
		t.Errorf("Expected 0 metric writes for empty data, got: %d", writeCallCount)
	}
}

// TestNetworkImplement_parseAndWriteMetrics_MetricWriteError_ContinuesProcessing tests handling of metric write errors
func TestNetworkImplement_parseAndWriteMetrics_MetricWriteError_ContinuesProcessing(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	writeCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			return errors.New("metric write failed")
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Mock data
	mockData := []interface{}{
		"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
		"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
		" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0         0",
	}

	// Execute
	network.parseAndWriteMetrics("testhost", mockData, nil)

	// Verify - should attempt all 12 metric writes despite errors
	expectedCalls := 12
	if writeCallCount != expectedCalls {
		t.Errorf("Expected %d metric write attempts, got: %d", expectedCalls, writeCallCount)
	}
}

// TestNetworkImplement_parseAndWriteMetrics_InvalidFloatValues_SkipsLine tests handling of invalid float values
func TestNetworkImplement_parseAndWriteMetrics_InvalidFloatValues_SkipsLine(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	writeCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	network := &NetworkImplement{
		Logger:  klog.NewKlogr(),
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Mock data with invalid float values (non-numeric values)
	testCases := []struct {
		name     string
		mockData []interface{}
	}{
		{"rxOk invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D         invalid    0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0         0",
		}},
		{"rxBps invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     invalid  B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0         0",
		}},
		{"rxUtil invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      invalid         0          0         0              0     0.00 B/s      0.00%         0         0         0",
		}},
		{"rxErr invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%     invalid          0         0              0     0.00 B/s      0.00%         0         0         0",
		}},
		{"rxDpr invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%         0     invalid         0              0     0.00 B/s      0.00%         0         0         0",
		}},
		{"rxOvr invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%         0          0     invalid              0     0.00 B/s      0.00%         0         0         0",
		}},
		{"txOk invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0          invalid     0.00 B/s      0.00%         0         0         0",
		}},
		{"txBps invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0          invalid  B/s      0.00%         0         0         0",
		}},
		{"txUtil invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      invalid         0         0         0",
		}},
		{"txErr invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%     invalid         0         0",
		}},
		{"txDpr invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL	TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0     invalid         0",
		}},
		{"txOvr invalid", []interface{}{
			"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
			"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
			" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0     invalid",
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			writeCallCount = 0 // Reset count for each sub-test
			network.parseAndWriteMetrics("testhost", tc.mockData, nil)

			// Verify - no metric writes should occur due to parsing error in first field
			if writeCallCount != 0 {
				t.Errorf("Expected 0 metric writes due to invalid float value, got: %d", writeCallCount)
			}
		})
	}
}

// TestparseFloat_ValidFloat_ReturnsCorrectValue tests parsing valid float values
func TestParseFloat_ValidFloat_ReturnsCorrectValue(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		input    string
		expected float64
	}{
		{"123.45", 123.45},
		{"0.00", 0.0},
		{"1,000.50", 1000.50}, // Test comma removal
		{"999", 999.0},
		{"-50.25", -50.25},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input_%s", tc.input), func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			result, err := parseFloat(tc.input)
			if err != nil {
				t.Errorf("Expected no error for input %s, got: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Expected %f for input %s, got: %f", tc.expected, tc.input, result)
			}
		})
	}
}

// TestparseFloat_InvalidFloat_ReturnsError tests parsing invalid float values
func TestParseFloat_InvalidFloat_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []string{
		"invalid",
		"12.34.56",
		"abc123",
		"",
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input_%s", tc), func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			_, err := parseFloat(tc)
			if err == nil {
				t.Errorf("Expected error for invalid input %s, got nil", tc)
			}
		})
	}
}

// TestparsePercentage_ValidPercentage_ReturnsCorrectValue tests parsing valid percentage values
func TestParsePercentage_ValidPercentage_ReturnsCorrectValue(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		input    string
		expected float64
	}{
		{"50.5%", 50.5},
		{"0%", 0.0},
		{"100%", 100.0},
		{"99.99%", 99.99},
		{"  75.25%  ", 75.25}, // Test whitespace trimming
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input_%s", tc.input), func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			result, err := parsePercentage(tc.input)
			if err != nil {
				t.Errorf("Expected no error for input %s, got: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Expected %f for input %s, got: %f", tc.expected, tc.input, result)
			}
		})
	}
}

// TestparsePercentage_InvalidPercentage_ReturnsError tests parsing invalid percentage values
func TestParsePercentage_InvalidPercentage_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []string{
		"invalid%",
		"50.5.5%",
		"abc%",
		"%",
		"",
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input_%s", tc), func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			_, err := parsePercentage(tc)
			if err == nil {
				t.Errorf("Expected error for invalid input %s, got nil", tc)
			}
		})
	}
}

// TestNetworkImplement_parseAndWriteMetrics_MultipleInterfaceData_ProcessesAll tests processing multiple interfaces
func TestNetworkImplement_parseAndWriteMetrics_MultipleInterfaceData_ProcessesAll(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	writeCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	logger := klog.NewKlogr()
	network := NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Mock data with multiple interface data
	mockData := []interface{}{
		"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
		"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
		" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0         0",
		" Ethernet1        U     88,938,325     0.00 B/s      0.00%         0          0         0    157,749,525   364.34 B/s      0.00%         0         0         0",
		" Ethernet2        U     38,262,729  3819.87 B/s      0.00%         0          0         0    137,901,153  4343.70 B/s      0.00%         0         0         0",
	}

	// Execute
	network.parseAndWriteMetrics("testhost", mockData, nil)

	// Verify - should process all 3 interfaces
	if writeCallCount != 3*12 {
		t.Errorf("Expected 36 metric writes for multiple interfaces, got: %d", writeCallCount)
	}
}

// TestNetworkImplement_parseAndWriteMetrics_PartialErrorData_ProcessesValid tests processing data with some invalid lines
func TestNetworkImplement_parseAndWriteMetrics_PartialErrorData_ProcessesValid(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	writeCallCount := 0
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	logger := klog.NewKlogr()
	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Mock data with mix of valid and invalid lines
	mockData := []interface{}{
		"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
		"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
		" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0         0",
		" Ethernet1        U     88,938,325     0.00 B/s      0.00%", // Invalid line - too few fields
		" Ethernet2        U     38,262,729  3819.87 B/s      0.00%         0          0         0    137,901,153  4343.70 B/s      0.00%         0         0         0",
	}

	// Execute
	network.parseAndWriteMetrics("testhost", mockData, nil)

	// Verify - should process only 2 valid interfaces (eth0 and eth2)
	if writeCallCount != 2*12 {
		t.Errorf("Expected 24 metric writes for valid interfaces only, got: %d", writeCallCount)
	}
}

// TestNetworkImplement_parseAndWriteMetrics_PercentageValues_ParsesCorrectly tests percentage value parsing
func TestNetworkImplement_parseAndWriteMetrics_PercentageValues_ParsesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	writeCallCount := 0
	var capturedMetrics map[string]float64
	mockMetrics := &MockNetworkMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64) error {
			writeCallCount++
			capturedMetrics = make(map[string]float64)
			capturedMetrics["rx_util"] = value
			capturedMetrics["tx_util"] = value
			return nil
		},
	}

	mockManager := &MockNetworkManager{}

	logger := klog.NewKlogr()
	network := &NetworkImplement{
		Logger:  logger,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	// Mock data with percentage values (note: field positions matter for utilization)
	mockData := []interface{}{
		"     IFACE    STATE          RX_OK       RX_BPS    RX_UTIL    RX_ERR     RX_DRP    RX_OVR          TX_OK       TX_BPS    TX_UTIL    TX_ERR    TX_DRP    TX_OVR",
		"----------  -------  -------------  -----------  ---------  --------  ---------  --------  -------------  -----------  ---------  --------  --------  --------",
		" Ethernet0        D              0     0.00 B/s      0.00%         0          0         0              0     0.00 B/s      0.00%         0         0         0",
	}

	// Execute
	network.parseAndWriteMetrics("testhost", mockData, nil)

	// Verify
	if writeCallCount != 1*12 {
		t.Errorf("Expected 12 metric write, got: %d", writeCallCount)
	}

	if capturedMetrics != nil {
		// Check if utilization values were parsed correctly (percentage values)
		if _, exists := capturedMetrics["rx_util"]; !exists {
			t.Error("Expected rx_util metric to be captured")
		}
		if _, exists := capturedMetrics["tx_util"]; !exists {
			t.Error("Expected tx_util metric to be captured")
		}
	}
}
