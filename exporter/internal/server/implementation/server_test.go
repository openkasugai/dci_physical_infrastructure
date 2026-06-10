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

// Mock for Ansible interface for server tests
type MockServerAnsible struct {
	cmdExecuteFunc func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error)
}

func (m *MockServerAnsible) CmdExecute(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
	if m.cmdExecuteFunc != nil {
		return m.cmdExecuteFunc(ctx, host, loginUser, sshKey, playbook, extraVars)
	}
	return nil, nil
}

// Mock for API interface for server tests
type MockServerAPI struct {
	apiExecuteFunc        func(ctx context.Context, method, host, api, user, password string) (map[string]interface{}, error)
	apiExecuteJWTAuthFunc func(ctx context.Context, method, url, apiname, jwt, queryParameter string) (interface{}, error)
}

func (m *MockServerAPI) APIExecute(ctx context.Context, method, host, api, user, password string) (map[string]interface{}, error) {
	if m.apiExecuteFunc != nil {
		return m.apiExecuteFunc(ctx, method, host, api, user, password)
	}
	return nil, nil
}

func (m *MockServerAPI) APIExecuteUserAuth(ctx context.Context, method, url, apiname, loginUser, loginPass, queryParameter string) (interface{}, error) {
	if m.apiExecuteFunc != nil {
		result, err := m.apiExecuteFunc(ctx, method, url, apiname, loginUser, loginPass)
		return result, err
	}
	return nil, nil
}

func (m *MockServerAPI) APIExecuteJWTAUth(ctx context.Context, method, url, apiname, jwt, queryParameter string) (interface{}, error) {
	if m.apiExecuteJWTAuthFunc != nil {
		return m.apiExecuteJWTAuthFunc(ctx, method, url, apiname, jwt, queryParameter)
	}
	return nil, nil
}

// Mock for Metrics interface for server tests
type MockServerMetrics struct {
	initFunc     func([]*prometheus.GaugeVec) error
	finalizeFunc func()
	writeFunc    func(*prometheus.GaugeVec, prometheus.Labels, float64, *[]interfaces.MetricLabel) error
	deleteFunc   func(*prometheus.GaugeVec, prometheus.Labels) error
}

func (m *MockServerMetrics) Init(gaugeList []*prometheus.GaugeVec) error {
	if m.initFunc != nil {
		return m.initFunc(gaugeList)
	}
	return nil
}

func (m *MockServerMetrics) Finalize() {
	if m.finalizeFunc != nil {
		m.finalizeFunc()
	}
}

func (m *MockServerMetrics) Write(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) error {
	if m.writeFunc != nil {
		return m.writeFunc(gauge, labels, value, writedMetrics)
	}
	return nil
}

func (m *MockServerMetrics) Delete(gauge *prometheus.GaugeVec, labels prometheus.Labels) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(gauge, labels)
	}
	return nil
}

// Mock for Manager interface for server tests
type MockServerManager struct {
	serverListFunc  func([]interfaces.ServerTargetList)
	networkListFunc func([]interfaces.NetworkTargetList)
	SetP2POnFunc    func([]interfaces.ServerTargetList)
}

func (m *MockServerManager) ServerList(targetList []interfaces.ServerTargetList) {
	if m.serverListFunc != nil {
		m.serverListFunc(targetList)
	}
}

func (m *MockServerManager) NetworkList(targetList []interfaces.NetworkTargetList) {
	if m.networkListFunc != nil {
		m.networkListFunc(targetList)
	}
}

func (m *MockServerManager) SetP2POn(targetList []interfaces.ServerTargetList) {
	if m.SetP2POnFunc != nil {
		m.SetP2POnFunc(targetList)
	}
}

// Helper function to set up environment variable for a server test
func setEnvForServerTest(t *testing.T, key, value string) func() {
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

// TestCustomError_Error_ValidError_ReturnsFormattedString tests CustomError error formatting
func TestCustomError_Error_ValidError_ReturnsFormattedString(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	customErr := &CustomError{
		StatusCode: 404,
		Message:    "Not Found",
	}

	// Execute
	result := customErr.Error()

	// Verify
	expected := "<404> Not Found"
	if result != expected {
		t.Errorf("Expected error message '%s', got: '%s'", expected, result)
	}
}

// TestCustomError_Error_ZeroStatusCode_ReturnsFormattedString tests CustomError with zero status code
func TestCustomError_Error_ZeroStatusCode_ReturnsFormattedString(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	customErr := &CustomError{
		StatusCode: 0,
		Message:    "Empty status",
	}

	// Execute
	result := customErr.Error()

	// Verify
	expected := "<0> Empty status"
	if result != expected {
		t.Errorf("Expected error message '%s', got: '%s'", expected, result)
	}
}

// TestServerImplement_Init_ValidMetrics_ReturnsSuccess tests successful server metrics initialization
func TestServerImplement_Init_ValidMetrics_ReturnsSuccess(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	logger := klog.NewKlogr()
	mockMetrics := &MockServerMetrics{
		initFunc: func([]*prometheus.GaugeVec) error {
			return nil
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Metrics: mockMetrics,
	}

	// Execute
	err := server.Init()

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestServerImplement_Init_MetricsInitError_ReturnsError tests initialization failure
func TestServerImplement_Init_MetricsInitError_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	logger := klog.NewKlogr()
	expectedError := errors.New("metrics initialization failed")
	mockMetrics := &MockServerMetrics{
		initFunc: func([]*prometheus.GaugeVec) error {
			return expectedError
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Metrics: mockMetrics,
	}

	// Execute
	err := server.Init()

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != expectedError.Error() {
		t.Errorf("Expected error %v, got: %v", expectedError, err)
	}
}

// TestServerImplement_Finalize_ValidCall_CallsMetricsFinalize tests finalization
func TestServerImplement_Finalize_ValidCall_CallsMetricsFinalize(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	logger := klog.NewKlogr()
	finalizeCalled := false
	mockMetrics := &MockServerMetrics{
		finalizeFunc: func() {
			finalizeCalled = true
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Metrics: mockMetrics,
	}

	// Execute
	server.Finalize()

	// Verify
	if !finalizeCalled {
		t.Error("Expected Metrics.Finalize to be called")
	}
}

// TestServerImplement_Collection_CotsServer_ProcessesSuccessfully tests successful server collection
func TestServerImplement_Collection_CotsServer_ProcessesSuccessfully(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	// Mock successful ansible response
	mockAnsibleOutput := map[string]interface{}{
		"cpu_usage": 75.5,
		"mem_usage": 85.2,
	}

	ansibleCallCount := 0
	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			ansibleCallCount++
			return mockAnsibleOutput, nil
		},
	}

	writeCallCount := 0
	mockMetrics := &MockServerMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockServerManager{
		serverListFunc: func(targetList []interfaces.ServerTargetList) {
			// Do nothing in test
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	targetList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "admin",
			ProductInfo:   "{\"vendor\":\"supermicro\",\"product_name\":\"SuperServer 6029P-TRT\"}",
		},
	}

	// Execute
	server.Colloction(targetList)

	// Verify
	if ansibleCallCount != 1 {
		t.Errorf("Expected 1 ansible call, got: %d", ansibleCallCount)
	}
	// Should have 2 metric writes (CPU, Memory only)
	if writeCallCount != 2 {
		t.Errorf("Expected 2 metric writes, got: %d", writeCallCount)
	}
}

// TestServerImplement_Collection_ComposedServer_ProcessesSuccessfully tests successful server collection
func TestServerImplement_Collection_ComposedServer_ProcessesSuccessfully(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	// Mock successful ansible response
	mockAnsibleOutput := map[string]interface{}{
		"cpu_usage": 65.3,
		"mem_usage": 78.9,
	}

	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			return mockAnsibleOutput, nil
		},
	}

	writeCallCount := 0
	mockMetrics := &MockServerMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockServerManager{
		serverListFunc: func(targetList []interfaces.ServerTargetList) {
			// Do nothing in test
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	targetList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-002",
			HostIPAddress: "192.168.1.101",
			LoginUser:     "admin",
			ProductInfo:   "{\"vendor\":\"supermicro\",\"product_name\":\"SuperServer 6029P-TRT\"}",
		},
	}

	// Execute
	server.Colloction(targetList)

	// Verify - Should have 2 metric writes (CPU, Memory only)
	if writeCallCount != 2 {
		t.Errorf("Expected 2 metric writes, got: %d", writeCallCount)
	}
}

// TestServerImplement_Collection_AnsibleError_SkipsMetrics tests handling of ansible errors
func TestServerImplement_Collection_AnsibleError_SkipsMetrics(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			return nil, errors.New("ansible execution failed")
		},
	}

	writeCallCount := 0
	mockMetrics := &MockServerMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockServerManager{
		serverListFunc: func(targetList []interfaces.ServerTargetList) {
			// Do nothing in test
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	targetList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "admin",
			ProductInfo:   "{\"vendor\":\"supermicro\",\"product_name\":\"SuperServer 6029P-TRT\"}",
		},
	}

	// Execute
	server.Colloction(targetList)

	// Verify - No metrics should be written when ansible fails
	if writeCallCount != 0 {
		t.Errorf("Expected 0 metric writes due to ansible failure, got: %d", writeCallCount)
	}
}

// TestServerImplement_Collection_MetricsWriteError_ContinuesProcessing tests handling of metrics write errors
func TestServerImplement_Collection_MetricsWriteError_ContinuesProcessing(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	mockAnsibleOutput := map[string]interface{}{
		"cpu_usage": 75.5,
		"mem_usage": 85.2,
	}

	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			return mockAnsibleOutput, nil
		},
	}

	writeCallCount := 0
	mockMetrics := &MockServerMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) error {
			writeCallCount++
			// Return error on first write to test error handling
			if writeCallCount == 1 {
				return errors.New("metrics write failed")
			}
			return nil
		},
	}

	mockManager := &MockServerManager{
		serverListFunc: func(targetList []interfaces.ServerTargetList) {
			// Do nothing in test
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	targetList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "admin",
			ProductInfo:   "{\"vendor\":\"supermicro\",\"product_name\":\"SuperServer 6029P-TRT\"}",
		},
	}

	// Execute
	server.Colloction(targetList)

	// Verify - Should have 2 metric write attempts (CPU and Memory)
	if writeCallCount != 2 {
		t.Errorf("Expected 2 metric write attempts, got: %d", writeCallCount)
	}
}

// TestServerImplement_Collection_ValidAnsibleResponse_WritesMetrics tests successful metrics collection
func TestServerImplement_Collection_ValidAnsibleResponse_WritesMetrics(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	mockAnsibleOutput := map[string]interface{}{
		"cpu_usage": 75.5,
		"mem_usage": 85.2,
	}

	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			return mockAnsibleOutput, nil
		},
	}

	writeCallCount := 0
	mockMetrics := &MockServerMetrics{
		writeFunc: func(gauge *prometheus.GaugeVec, labels prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) error {
			writeCallCount++
			return nil
		},
	}

	mockManager := &MockServerManager{
		serverListFunc: func(targetList []interfaces.ServerTargetList) {
			// Do nothing in test
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	targetList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "admin",
			ProductInfo:   "{\"vendor\":\"supermicro\",\"product_name\":\"SuperServer 6029P-TRT\"}",
		},
	}

	// Execute
	server.Colloction(targetList)

	// Verify - Should have 2 metric writes (CPU and Memory)
	if writeCallCount != 2 {
		t.Errorf("Expected 2 metric writes (CPU/Memory), got: %d", writeCallCount)
	}
}

// TestServerImplement_Collection_EmptyTargetList_ProcessesSuccessfully tests collection with empty targets
func TestServerImplement_Collection_EmptyTargetList_ProcessesSuccessfully(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	ansibleCallCount := 0
	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			ansibleCallCount++
			return nil, nil
		},
	}

	mockMetrics := &MockServerMetrics{}

	mockManager := &MockServerManager{
		serverListFunc: func(targetList []interfaces.ServerTargetList) {
			// Do nothing in test
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Metrics: mockMetrics,
		Manager: mockManager,
	}

	var emptyTargetList []interfaces.ServerTargetList

	// Execute
	server.Colloction(emptyTargetList)

	// Verify
	if ansibleCallCount != 0 {
		t.Errorf("Expected 0 ansible calls for empty target list, got: %d", ansibleCallCount)
	}
}

// TestServerImplement_getServerMetrics_ValidOutput_ParsesCorrectly tests valid metrics parsing
func TestServerImplement_getServerMetrics_ValidOutput_ParsesCorrectly(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	mockAnsibleOutput := map[string]interface{}{
		"cpu_usage": 65.7,
		"mem_usage": 82.3,
	}

	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			return mockAnsibleOutput, nil
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
	}

	target := interfaces.ServerTargetList{
		ServerID:      "test-server",
		HostIPAddress: "192.168.1.100",
		LoginUser:     "admin",
	}

	// Execute
	cpuUsage, memoryUsage, err := server.getServerMetrics(target, "test-key", "test-playbook")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if cpuUsage != 65.7 {
		t.Errorf("Expected CPU usage 65.7, got: %f", cpuUsage)
	}
	if memoryUsage != 82.3 {
		t.Errorf("Expected memory usage 82.3, got: %f", memoryUsage)
	}
}

// TestServerImplement_getServerMetrics_InvalidJsonOutput_ReturnsError tests invalid JSON output
func TestServerImplement_getServerMetrics_InvalidJsonOutput_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	// Invalid output - not a map[string]interface{}
	invalidOutput := "invalid json string"

	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			return invalidOutput, nil
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
	}

	target := interfaces.ServerTargetList{
		ServerID:      "test-server",
		HostIPAddress: "192.168.1.100",
		LoginUser:     "admin",
	}

	// Execute
	cpuUsage, memoryUsage, err := server.getServerMetrics(target, "test-key", "test-playbook")

	// Verify
	if err == nil {
		t.Error("Expected error for invalid JSON output, got nil")
	}
	if cpuUsage != 0 {
		t.Errorf("Expected CPU usage 0 for error case, got: %f", cpuUsage)
	}
	if memoryUsage != 0 {
		t.Errorf("Expected memory usage 0 for error case, got: %f", memoryUsage)
	}
}

// TestServerImplement_getServerMetrics_MissingCpuUsage_ReturnsError tests missing CPU usage field
func TestServerImplement_getServerMetrics_MissingCpuUsage_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	// Missing cpu_usage field
	incompleteOutput := map[string]interface{}{
		"mem_usage": 82.3,
	}

	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			return incompleteOutput, nil
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
	}

	target := interfaces.ServerTargetList{
		ServerID:      "test-server",
		HostIPAddress: "192.168.1.100",
		LoginUser:     "admin",
	}

	// Execute
	_, _, err := server.getServerMetrics(target, "test-key", "test-playbook")

	// Verify
	if err == nil {
		t.Error("Expected error for missing cpu_usage field, got nil")
	}
}

// TestServerImplement_getServerMetrics_MissingMemUsage_ReturnsError tests missing memory usage field
func TestServerImplement_getServerMetrics_MissingMemUsage_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cleanup := setEnvForServerTest(t, "SSH_KEY", "test-ssh-key")
	defer cleanup()

	logger := klog.NewKlogr()

	// Missing cpu_usage field
	incompleteOutput := map[string]interface{}{
		"cpu_usage": 82.3,
	}

	mockAnsible := &MockServerAnsible{
		cmdExecuteFunc: func(ctx context.Context, host, loginUser, sshKey, playbook, extraVars string) (interface{}, error) {
			return incompleteOutput, nil
		},
	}

	server := &ServerImplement{
		Logger:  logger,
		Ansible: mockAnsible,
	}

	target := interfaces.ServerTargetList{
		ServerID:      "test-server",
		HostIPAddress: "192.168.1.100",
		LoginUser:     "admin",
	}

	// Execute
	_, _, err := server.getServerMetrics(target, "test-key", "test-playbook")

	// Verify
	if err == nil {
		t.Error("Expected error for missing mem_usage field, got nil")
	}
}

// NOTE: The following tests for getCurrentPowerConsumption have been commented out
// because the function has been removed from the implementation.
// If the function is re-implemented, these tests should be uncommented.

/*
// TestgetCurrentPowerConsumption_CotsServer_ValidData_ParsesCorrectly tests COTS server power consumption parsing
func TestGetCurrentPowerConsumption_CotsServer_ValidData_ParsesCorrectly(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cotsData := map[string]interface{}{
		"PowerControl": []interface{}{
			map[string]interface{}{
				"PowerMetrics": map[string]interface{}{
					"AverageConsumedWatts": 150.0,
				},
			},
		},
	}

	// Execute
	power, err := getCurrentPowerConsumption(cotsData, true)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if power != 150 {
		t.Errorf("Expected power 150, got: %d", power)
	}
}

// TestgetCurrentPowerConsumption_ComposedServer_ValidData_ParsesCorrectly tests composed server power consumption parsing
func TestGetCurrentPowerConsumption_ComposedServer_ValidData_ParsesCorrectly(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	composedData := map[string]interface{}{
		"Oem": map[string]interface{}{
			"ts_fujitsu": map[string]interface{}{
				"ChassisPowerConsumption": map[string]interface{}{
					"AveragePowerW": 120.0,
				},
			},
		},
	}

	// Execute
	power, err := getCurrentPowerConsumption(composedData, false)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if power != 120 {
		t.Errorf("Expected power 120, got: %d", power)
	}
}

// TestgetCurrentPowerConsumption_CotsServer_MissingPowerControl_ReturnsError tests missing PowerControl
func TestGetCurrentPowerConsumption_CotsServer_MissingPowerControl_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	invalidCotsData := map[string]interface{}{}

	// Execute
	_, err := getCurrentPowerConsumption(invalidCotsData, true)

	// Verify
	if err == nil {
		t.Error("Expected error for missing PowerControl, got nil")
	}
}

// TestgetCurrentPowerConsumption_CotsServer_EmptyPowerControl_ReturnsError tests empty PowerControl array
func TestGetCurrentPowerConsumption_CotsServer_EmptyPowerControl_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	emptyPowerControlData := map[string]interface{}{
		"PowerControl": []interface{}{},
	}

	// Execute
	_, err := getCurrentPowerConsumption(emptyPowerControlData, true)

	// Verify
	if err == nil {
		t.Error("Expected error for empty PowerControl, got nil")
	}
}

// TestgetCurrentPowerConsumption_ComposedServer_MissingOem_ReturnsError tests missing Oem field
func TestGetCurrentPowerConsumption_ComposedServer_MissingOem_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	invalidComposedData := map[string]interface{}{}

	// Execute
	_, err := getCurrentPowerConsumption(invalidComposedData, false)

	// Verify
	if err == nil {
		t.Error("Expected error for missing Oem, got nil")
	}
}

// TestGetCurrentPowerConsumption_CotsServer_InvalidPowerConsumptionWatts_ReturnsError tests invalid PowerConsumptionWatts
func TestGetCurrentPowerConsumption_CotsServer_InvalidPowerConsumptionWatts_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	invalidData := map[string]interface{}{
		"PowerControl": []interface{}{
			map[string]interface{}{
				"PowerMetrics": "invalid",
			},
		},
	}

	// Execute
	_, err := getCurrentPowerConsumption(invalidData, true)

	// Verify
	if err == nil {
		t.Error("Expected error for invalid PowerMetrics, got nil")
	}
}

// TestGetCurrentPowerConsumption_ComposedServer_InvalidPowerConsumption_ReturnsError tests invalid power consumption for composed server
func TestGetCurrentPowerConsumption_ComposedServer_InvalidPowerConsumption_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	invalidData := map[string]interface{}{
		"Oem": map[string]interface{}{
			"ts_fujitsu": map[string]interface{}{
				"ChassisPowerConsumption": "invalid",
			},
		},
	}

	// Execute
	_, err := getCurrentPowerConsumption(invalidData, false)

	// Verify
	if err == nil {
		t.Error("Expected error for invalid ChassisPowerConsumption, got nil")
	}
}

// TestGetCurrentPowerConsumption_ComposedServer_MissingPowerConsumptionData_ReturnsError tests missing power consumption data for composed server
func TestGetCurrentPowerConsumption_ComposedServer_MissingPowerConsumptionData_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	invalidComposedData := map[string]interface{}{
		"Oem": map[string]interface{}{
			"ts_fujitsu": map[string]interface{}{
				"ChassisPowerConsumption": map[string]interface{}{
					"wrong_key": 120.0,
				},
			},
		},
	}

	// Execute
	_, err := getCurrentPowerConsumption(invalidComposedData, false)

	// Verify
	if err == nil {
		t.Error("Expected error for missing AveragePowerW, got nil")
	}
}

// TestGetCurrentPowerConsumption_ComposedServer_MissingTsFujitsuKey_ReturnsError tests missing ts_fujitsu key for composed server
func TestGetCurrentPowerConsumption_ComposedServer_MissingTsFujitsuKey_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	invalidComposedData := map[string]interface{}{
		"Oem": map[string]interface{}{
			"wrong_key": map[string]interface{}{
				"ChassisPowerConsumption": map[string]interface{}{
					"AveragePowerW": 120.0,
				},
			},
		},
	}

	// Execute
	_, err := getCurrentPowerConsumption(invalidComposedData, false)

	// Verify
	if err == nil {
		t.Error("Expected error for missing ts_fujitsu key, got nil")
	}
}

// TestGetCurrentPowerConsumption_ComposedServer_InvalidStructure_ReturnsError tests invalid structure for composed server
func TestGetCurrentPowerConsumption_ComposedServer_InvalidStructure_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	composedData := map[string]interface{}{
		"Oem": map[string]interface{}{
			"ts_fujitsu": "invalid_structure",
		},
	}

	// Execute
	_, err := getCurrentPowerConsumption(composedData, false)

	// Verify
	if err == nil {
		t.Error("Expected error for invalid structure, got nil")
	}
}

// TestGetCurrentPowerConsumption_CotsServer_InvalidPowerConsumptionType tests COTS server power consumption parsing
func TestGetCurrentPowerConsumption_CotsServer_InvalidPowerConsumptionType(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	cotsData := map[string]interface{}{
		"PowerControl": []interface{}{
			map[string]interface{}{
				"PowerMetrics": map[string]interface{}{
					"AverageConsumedWatts": "150.0",
				},
			},
		},
	}

	// Execute
	_, err := getCurrentPowerConsumption(cotsData, true)

	// Verify
	if err == nil {
		t.Error("Expected error for invalid PowerConsumptionData, got nil")
	}
}
*/

// TestgetValue_ValidKey_ReturnsValue tests getValue function with valid key
func TestGetValue_ValidKey_ReturnsValue(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	testData := map[string]interface{}{
		"test_key":   "test_value",
		"number_key": 123,
	}

	// Execute
	value, err := getValue(testData, "test_key")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got: %v", value)
	}
}

// TestgetValue_InvalidKey_ReturnsError tests getValue function with invalid key
func TestGetValue_InvalidKey_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	testData := map[string]interface{}{
		"existing_key": "value",
	}

	// Execute
	_, err := getValue(testData, "non_existing_key")

	// Verify
	if err == nil {
		t.Error("Expected error for non-existing key, got nil")
	}
}

// TestgetValue_InvalidInputType_ReturnsError tests getValue function with invalid input type
func TestGetValue_InvalidInputType_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup - not a map[string]interface{}
	invalidInput := "not a map"

	// Execute
	_, err := getValue(invalidInput, "any_key")

	// Verify
	if err == nil {
		t.Error("Expected error for invalid input type, got nil")
	}
}

// TestgetJsonFloatValue_ValidFloatString_ReturnsValue tests float value parsing from string
func TestGetJsonFloatValue_ValidFloatString_ReturnsValue(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	testData := map[string]interface{}{
		"float_string": "75.5",
		"float_value":  85.2,
	}

	testCases := []struct {
		key      string
		expected float64
	}{
		{"float_string", 75.5},
		{"float_value", 85.2},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("key_%s", tc.key), func(t *testing.T) {
			cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanupKlog()

			// Execute
			value, err := getJsonFloatValue(testData, tc.key)

			// Verify
			if err != nil {
				t.Errorf("Expected no error for key %s, got: %v", tc.key, err)
			}
			if value != tc.expected {
				t.Errorf("Expected %f for key %s, got: %f", tc.expected, tc.key, value)
			}
		})
	}
}

// TestgetJsonFloatValue_InvalidFloatString_ReturnsError tests invalid float string parsing
func TestGetJsonFloatValue_InvalidFloatString_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	testData := map[string]interface{}{
		"invalid_float": "not_a_float",
	}

	// Execute
	_, err := getJsonFloatValue(testData, "invalid_float")

	// Verify
	if err == nil {
		t.Error("Expected error for invalid float string, got nil")
	}
}

// NOTE: The following tests for getJsonIntValue have been commented out
// because the function has been removed from the implementation.
// If the function is re-implemented, these tests should be uncommented.

/*
// TestgetJsonIntValue_ValidValues_ReturnsCorrectInt tests integer value parsing
func TestGetJsonIntValue_ValidValues_ReturnsCorrectInt(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	testData := map[string]interface{}{
		"int_value":    42,
		"float_value":  150.0,
		"string_value": "200",
	}

	testCases := []struct {
		key      string
		expected int
	}{
		{"int_value", 42},
		{"float_value", 150},
		{"string_value", 200},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("key_%s", tc.key), func(t *testing.T) {
			cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanupKlog()

			// Execute
			value, err := getJsonIntValue(testData, tc.key)

			// Verify
			if err != nil {
				t.Errorf("Expected no error for key %s, got: %v", tc.key, err)
			}
			if value != tc.expected {
				t.Errorf("Expected %d for key %s, got: %d", tc.expected, tc.key, value)
			}
		})
	}
}

// TestgetJsonIntValue_InvalidIntString_ReturnsError tests invalid integer string parsing
func TestGetJsonIntValue_InvalidIntString_ReturnsError(t *testing.T) {
	cleanupKlog := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanupKlog()

	// Setup
	testData := map[string]interface{}{
		"invalid_int": "not_an_int",
	}

	// Execute
	_, err := getJsonIntValue(testData, "invalid_int")

	// Verify
	if err == nil {
		t.Error("Expected error for invalid int string, got nil")
	}
}
*/
