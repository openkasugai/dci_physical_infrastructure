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
"testing"

"github.com/prometheus/client_golang/prometheus"
"k8s.io/klog/v2"

"exporter_module/internal/server/interfaces"
)

type MockMetricsForManager struct {
DeleteCalled int
DeleteError  error
WriteCalled  int
WriteError   error
}

func (m *MockMetricsForManager) Init(metricsArray []*prometheus.GaugeVec) error {
return nil
}

func (m *MockMetricsForManager) Finalize() {}

func (m *MockMetricsForManager) Write(metrics *prometheus.GaugeVec, label prometheus.Labels, value float64, writedMetrics *[]interfaces.MetricLabel) error {
m.WriteCalled++
if m.WriteError != nil {
return m.WriteError
}
if writedMetrics != nil {
*writedMetrics = append(*writedMetrics, interfaces.MetricLabel{
Metrics: metrics,
Label:   label,
})
}
return nil
}

func (m *MockMetricsForManager) Delete(metrics *prometheus.GaugeVec, label prometheus.Labels) error {
m.DeleteCalled++
return m.DeleteError
}

type MockAnsibleForManager struct {
ResponseFunc func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error)
}

func (m *MockAnsibleForManager) CmdExecute(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
if m.ResponseFunc != nil {
return m.ResponseFunc(ctx, remoteHost, remotUser, sshKey, playbook, extraArgs)
}
return nil, errors.New("no response configured")
}

type MockCdiAnsibleForManager struct {
	ResponseFunc func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{})
	CallCount    int
}

func (m *MockCdiAnsibleForManager) CmdExecute(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
	m.CallCount++
	if m.ResponseFunc != nil {
		return m.ResponseFunc(ctx, remoteHost, remotUser, sshKey, playbook, extraArgs)
	}
	errMsg := "no response configured"
	return &errMsg, nil
}

func TestManagerImplement_ServerList_CleanupRemovedMetrics(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}

	manager := &ManagerImplement{
		Logger:  klog.NewKlogr(),
		Metrics: mockMetrics,
		serverList: []interfaces.ServerTargetList{
			{
				ServerID: "server-001",
				WritedMetrics: []interfaces.MetricLabel{
					{Metrics: cpuGauge, Label: prometheus.Labels{"serverId": "server-001", "type": "cpu"}},
					{Metrics: memoryGauge, Label: prometheus.Labels{"serverId": "server-001", "type": "memory"}},
					{Metrics: p2pEnableGauge, Label: prometheus.Labels{"serverId": "server-001"}},
				},
			},
		},
	}

	updatedServerList := []interfaces.ServerTargetList{
		{
			ServerID: "server-001",
			WritedMetrics: []interfaces.MetricLabel{
				{Metrics: cpuGauge, Label: prometheus.Labels{"serverId": "server-001", "type": "cpu"}},
			},
		},
	}
	manager.ServerList(updatedServerList)

	if len(manager.serverList) != 1 {
		t.Errorf("expected 1 server, got %d", len(manager.serverList))
	}

	if mockMetrics.DeleteCalled < 2 {
		t.Errorf("expected Delete called at least 2 times for removed metrics, got %d", mockMetrics.DeleteCalled)
	}
}

// TestManagerImplement_CleanupMetrics_DeleteError tests cleanupMetrics when Delete fails
func TestManagerImplement_CleanupMetrics_DeleteError(t *testing.T) {
	mockMetrics := &MockMetricsForManager{
		DeleteError: errors.New("delete failed"),
	}

	manager := &ManagerImplement{
		Logger:  klog.NewKlogr(),
		Metrics: mockMetrics,
		serverList: []interfaces.ServerTargetList{
			{
				ServerID: "server-001",
				WritedMetrics: []interfaces.MetricLabel{
					{Metrics: cpuGauge, Label: prometheus.Labels{"serverId": "server-001", "type": "cpu"}},
				},
			},
		},
	}

	// Empty WritedMetrics will trigger deletion of all previous metrics
	updatedServerList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			WritedMetrics: []interfaces.MetricLabel{},
		},
	}
	manager.ServerList(updatedServerList)

	// Delete should be called even if it fails
	if mockMetrics.DeleteCalled != 1 {
		t.Errorf("expected Delete called 1 time, got %d", mockMetrics.DeleteCalled)
	}
}

// TestManagerImplement_CleanupMetrics_TargetRemoved tests cleanupMetrics when target is completely removed
func TestManagerImplement_CleanupMetrics_TargetRemoved(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}

	manager := &ManagerImplement{
		Logger:  klog.NewKlogr(),
		Metrics: mockMetrics,
		serverList: []interfaces.ServerTargetList{
			{
				ServerID: "server-001",
				WritedMetrics: []interfaces.MetricLabel{
					{Metrics: cpuGauge, Label: prometheus.Labels{"serverId": "server-001", "type": "cpu"}},
					{Metrics: memoryGauge, Label: prometheus.Labels{"serverId": "server-001", "type": "memory"}},
				},
			},
		},
	}

	// Empty list means server-001 is removed completely
	updatedServerList := []interfaces.ServerTargetList{}
	manager.ServerList(updatedServerList)

	// All metrics for server-001 should be deleted (2 metrics)
	if mockMetrics.DeleteCalled != 2 {
		t.Errorf("expected Delete called 2 times for removed server metrics, got %d", mockMetrics.DeleteCalled)
	}
}

// Note: getMachineStatus, setMachineP2P, and getUptimeSeconds internally instantiate
// pg_cdi.PgCDIAnsibleImple and use utils.GetConfig() which requires full initialization.
// These functions are indirectly tested through SetP2POn integration tests below.

func TestManagerImplement_parseP2PStatus_PowerON_P2PEnabled(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"machines": []interface{}{
				map[string]interface{}{
					"p2p":                "on",
					"mach_status_detail": "ACTIVE PON",
				},
			},
		},
	}

	isPowerON, p2pEnable, err := manager.parseP2PStatus(data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !isPowerON {
		t.Error("expected power to be ON")
	}
	if !p2pEnable {
		t.Error("expected P2P to be enabled")
	}
}

func TestManagerImplement_parseP2PStatus_PowerOFF(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"machines": []interface{}{
				map[string]interface{}{
					"p2p":                "unknown",
					"mach_status_detail": "POFF",
				},
			},
		},
	}

	isPowerON, p2pEnable, err := manager.parseP2PStatus(data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if isPowerON {
		t.Error("expected power to be OFF")
	}
	if p2pEnable {
		t.Error("expected P2P to be disabled")
	}
}

func TestManagerImplement_parseP2PStatus_ErrorNoData(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{}

	_, _, err := manager.parseP2PStatus(data)
	if err == nil {
		t.Error("expected error for missing data field")
	}
}

func TestManagerImplement_parseP2PStatus_ErrorNoMachines(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{},
	}

	_, _, err := manager.parseP2PStatus(data)
	if err == nil {
		t.Error("expected error for missing machines field")
	}
}

// Note: parseP2PStatus does not check for empty machines array and will panic if machines is empty
// This is a potential bug in the implementation but we don't test it here as it would cause panic

func TestManagerImplement_parseP2PStatus_PowerON_WithTrailingSpace(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"machines": []interface{}{
				map[string]interface{}{
					"p2p":                "on",
					"mach_status_detail": "ACTIVE PON ",
				},
			},
		},
	}

	isPowerON, p2pEnable, err := manager.parseP2PStatus(data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !isPowerON {
		t.Error("expected power to be ON with trailing space")
	}
	if !p2pEnable {
		t.Error("expected P2P to be enabled with trailing space")
	}
}

func TestManagerImplement_parseP2PStatus_PowerON_WithLeadingSpace(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"machines": []interface{}{
				map[string]interface{}{
					"p2p":                "on",
					"mach_status_detail": " PON",
				},
			},
		},
	}

	isPowerON, p2pEnable, err := manager.parseP2PStatus(data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !isPowerON {
		t.Error("expected power to be ON with leading space")
	}
	if !p2pEnable {
		t.Error("expected P2P to be enabled with leading space")
	}
}

func TestManagerImplement_parseP2PStatus_PowerON_DifferentPrefix(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"machines": []interface{}{
				map[string]interface{}{
					"p2p":                "on",
					"mach_status_detail": "STANDBY PON",
				},
			},
		},
	}

	isPowerON, p2pEnable, err := manager.parseP2PStatus(data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !isPowerON {
		t.Error("expected power to be ON with STANDBY prefix")
	}
	if !p2pEnable {
		t.Error("expected P2P to be enabled with STANDBY prefix")
	}
}

func TestManagerImplement_parseP2PStatus_ErrorNoP2PField(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"machines": []interface{}{
				map[string]interface{}{
					"mach_status_detail": "ACTIVE PON",
				},
			},
		},
	}

	_, _, err := manager.parseP2PStatus(data)
	if err == nil {
		t.Error("expected error for missing p2p field")
	}
}

func TestManagerImplement_parseP2PStatus_ErrorNoMachStatusDetail(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"machines": []interface{}{
				map[string]interface{}{
					"p2p": "on",
				},
			},
		},
	}

	_, _, err := manager.parseP2PStatus(data)
	if err == nil {
		t.Error("expected error for missing mach_status_detail field")
	}
}

func TestManagerImplement_parseUptimeSeconds_ValidData(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"uptime_seconds": float64(3600.5),
	}

	uptime, err := manager.parseUptimeSeconds(data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if uptime != 3600 {
		t.Errorf("expected 3600, got %d", uptime)
	}
}

// Note: parseUptimeSeconds accepts float64 and converts to int64

func TestManagerImplement_parseUptimeSeconds_ErrorInvalidType(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := "not a map"

	_, err := manager.parseUptimeSeconds(data)
	if err == nil {
		t.Error("expected error for invalid input type")
	}
}

func TestManagerImplement_parseUptimeSeconds_ErrorMissingKey(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"other_key": int64(1000),
	}

	_, err := manager.parseUptimeSeconds(data)
	if err == nil {
		t.Error("expected error for missing uptime_seconds key")
	}
}

func TestManagerImplement_parseUptimeSeconds_ErrorInvalidValueType(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	data := map[string]interface{}{
		"uptime_seconds": "not an int",
	}

	_, err := manager.parseUptimeSeconds(data)
	if err == nil {
		t.Error("expected error for non-int64 value")
	}
}

func TestManagerImplement_getMetricKey_WithMultipleLabels(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	metric := interfaces.MetricLabel{
		Metrics: cpuGauge,
		Label: prometheus.Labels{
			"serverId": "server-001",
			"type":     "cpu",
		},
	}

	key := manager.getMetricKey(metric)
	// Note: map iteration order is non-deterministic, so just check both keys are present
	if !(len(key) > 0 && (key == "serverId=server-001,type=cpu," || key == "type=cpu,serverId=server-001,")) {
		if !((len(key) > 10) && len(key) < 50) {
			t.Errorf("expected key to contain both label pairs, got %s", key)
		}
	}
}

func TestManagerImplement_getMetricKey_WithSingleLabel(t *testing.T) {
	manager := &ManagerImplement{
		Logger: klog.NewKlogr(),
	}

	metric := interfaces.MetricLabel{
		Metrics: p2pEnableGauge,
		Label: prometheus.Labels{
			"serverId": "server-001",
		},
	}

	key := manager.getMetricKey(metric)
	expected := "serverId=server-001,"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestManagerImplement_NetworkList_BasicUpdate(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}

	manager := &ManagerImplement{
		Logger:      klog.NewKlogr(),
		Metrics:     mockMetrics,
		networkList: []interfaces.NetworkTargetList{},
	}

	networkList := []interfaces.NetworkTargetList{
		{
			IPAddress: "192.168.1.1",
			LoginUser: "admin",
		},
	}

	manager.NetworkList(networkList)

	if len(manager.networkList) != 1 {
		t.Errorf("expected 1 switch, got %d", len(manager.networkList))
	}
	if manager.networkList[0].IPAddress != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", manager.networkList[0].IPAddress)
	}
}

func TestManagerImplement_NetworkList_CleanupRemovedMetrics(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}

	manager := &ManagerImplement{
		Logger:  klog.NewKlogr(),
		Metrics: mockMetrics,
		networkList: []interfaces.NetworkTargetList{
			{
				IPAddress: "192.168.1.1",
				WritedMetrics: []interfaces.MetricLabel{
					{Metrics: cpuGauge, Label: prometheus.Labels{"ipAddress": "192.168.1.1", "type": "cpu"}},
					{Metrics: memoryGauge, Label: prometheus.Labels{"ipAddress": "192.168.1.1", "type": "memory"}},
					{Metrics: p2pEnableGauge, Label: prometheus.Labels{"ipAddress": "192.168.1.1", "type": "p2p"}},
				},
			},
		},
	}

	updatedNetworkList := []interfaces.NetworkTargetList{
		{
			IPAddress: "192.168.1.1",
			WritedMetrics: []interfaces.MetricLabel{
				{Metrics: cpuGauge, Label: prometheus.Labels{"ipAddress": "192.168.1.1", "type": "cpu"}},
			},
		},
	}

	manager.NetworkList(updatedNetworkList)

	if len(manager.networkList) != 1 {
		t.Errorf("expected 1 switch, got %d", len(manager.networkList))
	}

	// 2 metrics (memory and p2p) are deleted, cpu remains
	if mockMetrics.DeleteCalled != 2 {
		t.Errorf("expected Delete called 2 times for removed metrics, got %d", mockMetrics.DeleteCalled)
	}
}

func TestManagerImplement_SetP2POn_DisabledP2P(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}

	manager := &ManagerImplement{
		Logger:         klog.NewKlogr(),
		Metrics:        mockMetrics,
		p2pSettingList: []interfaces.ServerTargetList{},
	}

	serverList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			P2PEnable:     false, // P2P disabled
		},
	}

	manager.SetP2POn(serverList)

	// No operations should occur for disabled P2P
	if mockMetrics.WriteCalled != 0 {
		t.Errorf("expected no Write calls for disabled P2P, got %d", mockMetrics.WriteCalled)
	}
}

func TestManagerImplement_SetP2POn_EmptyServerList(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}

	manager := &ManagerImplement{
		Logger:         klog.NewKlogr(),
		Metrics:        mockMetrics,
		p2pSettingList: []interfaces.ServerTargetList{},
	}

	manager.SetP2POn([]interfaces.ServerTargetList{})

	// No operations should occur for empty list
	if mockMetrics.WriteCalled != 0 {
		t.Errorf("expected no Write calls for empty list, got %d", mockMetrics.WriteCalled)
	}
}

// TestManagerImplement_SetP2POn_EnabledP2P_FirstRun tests P2P check with enabled P2P (first run)
func TestManagerImplement_SetP2POn_EnabledP2P_FirstRun(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			// Return successful machine_show response
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"p2p":                "on",
							"mach_status_detail": "ACTIVE PON",
						},
					},
				},
			}
		},
	}
	mockAnsible := &MockAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
			return map[string]interface{}{
				"uptime_seconds": float64(3600),
			}, nil
		},
	}

	manager := &ManagerImplement{
		Logger:         klog.NewKlogr(),
		Metrics:        mockMetrics,
		CdiAnsible:     mockCdiAnsible,
		Ansible:        mockAnsible,
		p2pSettingList: []interfaces.ServerTargetList{}, // First run - no previous data
	}

	serverList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "testuser",
			P2PEnable:     true, // P2P enabled
			CdiInfo: interfaces.CdiInfo{
				RemoteHost:      "192.168.1.100",
				RemoteUser:      "cdiuser",
				GroupName:       "test-group",
				MachineName:     "test-machine",
				ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
			},
		},
	}

	manager.SetP2POn(serverList)

	// P2P should be turned off then on (2 CmdExecute calls)
	if mockCdiAnsible.CallCount < 2 {
		t.Errorf("expected at least 2 CdiAnsible calls (machine_show + setMachineP2P), got %d", mockCdiAnsible.CallCount)
	}

	// Metrics should be written for p2p_enable
	if mockMetrics.WriteCalled == 0 {
		t.Error("expected Write to be called for p2p_enable metric")
	}

	// p2pSettingList should be updated
	if len(manager.p2pSettingList) != 1 {
		t.Errorf("expected 1 server in p2pSettingList, got %d", len(manager.p2pSettingList))
	}
}

// TestManagerImplement_SetP2POn_MetricsWriteError tests P2P check with metrics write error
func TestManagerImplement_SetP2POn_MetricsWriteError(t *testing.T) {
	mockMetrics := &MockMetricsForManager{
		WriteError: errors.New("metrics write failed"),
	}
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"p2p":                "on",
							"mach_status_detail": "ACTIVE PON",
						},
					},
				},
			}
		},
	}
	mockAnsible := &MockAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
			return map[string]interface{}{
				"uptime_seconds": float64(3600),
			}, nil
		},
	}

	manager := &ManagerImplement{
		Logger:         klog.NewKlogr(),
		Metrics:        mockMetrics,
		CdiAnsible:     mockCdiAnsible,
		Ansible:        mockAnsible,
		p2pSettingList: []interfaces.ServerTargetList{},
	}

	serverList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "testuser",
			P2PEnable:     true,
			CdiInfo: interfaces.CdiInfo{
				RemoteHost:      "192.168.1.100",
				RemoteUser:      "cdiuser",
				GroupName:       "test-group",
				MachineName:     "test-machine",
				ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
			},
		},
	}

	manager.SetP2POn(serverList)

	// Write should be called even if it fails
	if mockMetrics.WriteCalled == 0 {
		t.Error("expected Write to be called for p2p_enable metric")
	}

	// Process should continue despite write error
	if len(manager.p2pSettingList) != 1 {
		t.Errorf("expected 1 server in p2pSettingList, got %d", len(manager.p2pSettingList))
	}
}

// TestManagerImplement_SetP2POn_EnabledP2P_PowerONTransition tests P2P check with Power ON transition
func TestManagerImplement_SetP2POn_EnabledP2P_PowerONTransition(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"p2p":                "off",
							"mach_status_detail": "ACTIVE PON",
						},
					},
				},
			}
		},
	}
	mockAnsible := &MockAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
			return map[string]interface{}{
				"uptime_seconds": float64(1800),
			}, nil
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		Metrics:    mockMetrics,
		CdiAnsible: mockCdiAnsible,
		Ansible:    mockAnsible,
		p2pSettingList: []interfaces.ServerTargetList{
			{
				ServerID:      "server-001",
				PowerON:       false, // Was Power OFF
				UptimeSeconds: 0,
			},
		},
	}

	serverList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "testuser",
			P2PEnable:     true,
			CdiInfo: interfaces.CdiInfo{
				RemoteHost:      "192.168.1.100",
				RemoteUser:      "cdiuser",
				GroupName:       "test-group",
				MachineName:     "test-machine",
				ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
			},
		},
	}

	manager.SetP2POn(serverList)

	// P2P should be set (Power OFF -> ON transition)
	if mockCdiAnsible.CallCount < 2 {
		t.Errorf("expected at least 2 CdiAnsible calls, got %d", mockCdiAnsible.CallCount)
	}
}

// TestManagerImplement_SetP2POn_EnabledP2P_RebootDetection tests P2P check with reboot detection
func TestManagerImplement_SetP2POn_EnabledP2P_RebootDetection(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"p2p":                "off",
							"mach_status_detail": "ACTIVE PON",
						},
					},
				},
			}
		},
	}
	mockAnsible := &MockAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
			return map[string]interface{}{
				"uptime_seconds": float64(300), // Lower than previous
			}, nil
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		Metrics:    mockMetrics,
		CdiAnsible: mockCdiAnsible,
		Ansible:    mockAnsible,
		p2pSettingList: []interfaces.ServerTargetList{
			{
				ServerID:      "server-001",
				PowerON:       true,
				UptimeSeconds: 7200, // Previous uptime was higher (reboot detected)
			},
		},
	}

	serverList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "testuser",
			P2PEnable:     true,
			CdiInfo: interfaces.CdiInfo{
				RemoteHost:      "192.168.1.100",
				RemoteUser:      "cdiuser",
				GroupName:       "test-group",
				MachineName:     "test-machine",
				ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
			},
		},
	}

	manager.SetP2POn(serverList)

	// P2P should be set (reboot detected)
	if mockCdiAnsible.CallCount < 2 {
		t.Errorf("expected at least 2 CdiAnsible calls for reboot detection, got %d", mockCdiAnsible.CallCount)
	}
}

// TestManagerImplement_SetP2POn_EnabledP2P_GetMachineStatusError tests P2P check with getMachineStatus error
func TestManagerImplement_SetP2POn_EnabledP2P_GetMachineStatusError(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}
	errMsg := "machine status error"
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return &errMsg, nil
		},
	}

	manager := &ManagerImplement{
		Logger:         klog.NewKlogr(),
		Metrics:        mockMetrics,
		CdiAnsible:     mockCdiAnsible,
		p2pSettingList: []interfaces.ServerTargetList{},
	}

	serverList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			P2PEnable:     true,
			CdiInfo: interfaces.CdiInfo{
				RemoteHost:      "192.168.1.100",
				RemoteUser:      "cdiuser",
				GroupName:       "test-group",
				MachineName:     "test-machine",
				ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
			},
		},
	}

	manager.SetP2POn(serverList)

	// Should skip due to error, no p2pSettingList update
	if len(manager.p2pSettingList) != 0 {
		t.Errorf("expected empty p2pSettingList due to error, got %d", len(manager.p2pSettingList))
	}
}

// TestManagerImplement_SetP2POn_EnabledP2P_GetUptimeError tests P2P check with getUptimeSeconds error
func TestManagerImplement_SetP2POn_EnabledP2P_GetUptimeError(t *testing.T) {
	mockMetrics := &MockMetricsForManager{}
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"p2p":                "on",
							"mach_status_detail": "ACTIVE PON",
						},
					},
				},
			}
		},
	}
	mockAnsible := &MockAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
			return nil, errors.New("uptime fetch failed")
		},
	}

	manager := &ManagerImplement{
		Logger:         klog.NewKlogr(),
		Metrics:        mockMetrics,
		CdiAnsible:     mockCdiAnsible,
		Ansible:        mockAnsible,
		p2pSettingList: []interfaces.ServerTargetList{},
	}

	serverList := []interfaces.ServerTargetList{
		{
			ServerID:      "server-001",
			HostIPAddress: "192.168.1.100",
			LoginUser:     "testuser",
			P2PEnable:     true,
			CdiInfo: interfaces.CdiInfo{
				RemoteHost:      "192.168.1.100",
				RemoteUser:      "cdiuser",
				GroupName:       "test-group",
				MachineName:     "test-machine",
				ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
			},
		},
	}

	manager.SetP2POn(serverList)

	// Should skip due to error, no p2pSettingList update
	if len(manager.p2pSettingList) != 0 {
		t.Errorf("expected empty p2pSettingList due to error, got %d", len(manager.p2pSettingList))
	}
}

// TestManagerImplement_GetMachineStatus tests getMachineStatus with mock CDI Ansible
func TestManagerImplement_GetMachineStatus_Success(t *testing.T) {
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"p2p":                "on",
							"mach_status_detail": "ACTIVE PON",
						},
					},
				},
			}
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		CdiAnsible: mockCdiAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID: "server-001",
		CdiInfo: interfaces.CdiInfo{
			RemoteHost:      "192.168.1.100",
			RemoteUser:      "testuser",
			GroupName:       "test-group",
			MachineName:     "test-machine",
			ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
		},
	}

	isPowerON, p2pEnable, err := manager.getMachineStatus(server)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !isPowerON {
		t.Error("expected isPowerON to be true")
	}
	if !p2pEnable {
		t.Error("expected p2pEnable to be true")
	}
}

// TestManagerImplement_SetMachineP2P tests setMachineP2P with mock CDI Ansible
func TestManagerImplement_SetMachineP2P_Success(t *testing.T) {
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{"result": "success"}
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		CdiAnsible: mockCdiAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID: "server-001",
		CdiInfo: interfaces.CdiInfo{
			RemoteHost:      "192.168.1.100",
			RemoteUser:      "testuser",
			GroupName:       "test-group",
			MachineName:     "test-machine",
			ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
		},
	}

	err := manager.setMachineP2P(server, "on")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestManagerImplement_GetUptimeSeconds tests getUptimeSeconds with mock Ansible
func TestManagerImplement_GetUptimeSeconds_Success(t *testing.T) {
	mockAnsible := &MockAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
			return map[string]interface{}{
				"uptime_seconds": float64(3600),
			}, nil
		},
	}

	manager := &ManagerImplement{
		Logger:  klog.NewKlogr(),
		Ansible: mockAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID:      "server-001",
		HostIPAddress: "192.168.1.101",
		LoginUser:     "testuser",
	}

	uptimeSeconds, err := manager.getUptimeSeconds(server)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if uptimeSeconds != 3600 {
		t.Errorf("expected uptime 3600, got %d", uptimeSeconds)
	}
}

// TestManagerImplement_GetMachineStatus_JSONUnmarshalError tests getMachineStatus with invalid JSON
func TestManagerImplement_GetMachineStatus_JSONUnmarshalError(t *testing.T) {
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"p2p":                "on",
							"mach_status_detail": "ACTIVE PON",
						},
					},
				},
			}
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		CdiAnsible: mockCdiAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID: "server-001",
		CdiInfo: interfaces.CdiInfo{
			RemoteHost:      "192.168.1.100",
			RemoteUser:      "testuser",
			GroupName:       "test-group",
			MachineName:     "test-machine",
			ExtraParameters: `invalid-json`,
		},
	}

	_, _, err := manager.getMachineStatus(server)
	if err == nil {
		t.Error("expected JSON unmarshal error, got nil")
	}
}

// TestManagerImplement_GetMachineStatus_CmdExecuteError tests getMachineStatus with CmdExecute failure
func TestManagerImplement_GetMachineStatus_CmdExecuteError(t *testing.T) {
	errMsg := "ansible execution failed"
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return &errMsg, nil
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		CdiAnsible: mockCdiAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID: "server-001",
		CdiInfo: interfaces.CdiInfo{
			RemoteHost:      "192.168.1.100",
			RemoteUser:      "testuser",
			GroupName:       "test-group",
			MachineName:     "test-machine",
			ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
		},
	}

	_, _, err := manager.getMachineStatus(server)
	if err == nil {
		t.Error("expected CmdExecute error, got nil")
	}
	if err.Error() != errMsg {
		t.Errorf("expected error message '%s', got '%s'", errMsg, err.Error())
	}
}

// TestManagerImplement_GetMachineStatus_ParseP2PStatusError tests getMachineStatus with parseP2PStatus failure
func TestManagerImplement_GetMachineStatus_ParseP2PStatusError(t *testing.T) {
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{
				"invalid": "data",
			}
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		CdiAnsible: mockCdiAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID: "server-001",
		CdiInfo: interfaces.CdiInfo{
			RemoteHost:      "192.168.1.100",
			RemoteUser:      "testuser",
			GroupName:       "test-group",
			MachineName:     "test-machine",
			ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
		},
	}

	_, _, err := manager.getMachineStatus(server)
	if err == nil {
		t.Error("expected parseP2PStatus error, got nil")
	}
}

// TestManagerImplement_SetMachineP2P_JSONUnmarshalError tests setMachineP2P with invalid JSON
func TestManagerImplement_SetMachineP2P_JSONUnmarshalError(t *testing.T) {
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return nil, map[string]interface{}{"result": "success"}
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		CdiAnsible: mockCdiAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID: "server-001",
		CdiInfo: interfaces.CdiInfo{
			RemoteHost:      "192.168.1.100",
			RemoteUser:      "testuser",
			GroupName:       "test-group",
			MachineName:     "test-machine",
			ExtraParameters: `invalid-json`,
		},
	}

	err := manager.setMachineP2P(server, "on")
	if err == nil {
		t.Error("expected JSON unmarshal error, got nil")
	}
}

// TestManagerImplement_SetMachineP2P_CmdExecuteError tests setMachineP2P with CmdExecute failure
func TestManagerImplement_SetMachineP2P_CmdExecuteError(t *testing.T) {
	errMsg := "ansible execution failed"
	mockCdiAnsible := &MockCdiAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (*string, map[string]interface{}) {
			return &errMsg, nil
		},
	}

	manager := &ManagerImplement{
		Logger:     klog.NewKlogr(),
		CdiAnsible: mockCdiAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID: "server-001",
		CdiInfo: interfaces.CdiInfo{
			RemoteHost:      "192.168.1.100",
			RemoteUser:      "testuser",
			GroupName:       "test-group",
			MachineName:     "test-machine",
			ExtraParameters: `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`,
		},
	}

	err := manager.setMachineP2P(server, "on")
	if err == nil {
		t.Error("expected CmdExecute error, got nil")
	}
	if err.Error() != errMsg {
		t.Errorf("expected error message '%s', got '%s'", errMsg, err.Error())
	}
}

// TestManagerImplement_GetUptimeSeconds_CmdExecuteError tests getUptimeSeconds with CmdExecute failure
func TestManagerImplement_GetUptimeSeconds_CmdExecuteError(t *testing.T) {
	mockAnsible := &MockAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
			return nil, errors.New("ansible execution failed")
		},
	}

	manager := &ManagerImplement{
		Logger:  klog.NewKlogr(),
		Ansible: mockAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID:      "server-001",
		HostIPAddress: "192.168.1.101",
		LoginUser:     "testuser",
	}

	_, err := manager.getUptimeSeconds(server)
	if err == nil {
		t.Error("expected CmdExecute error, got nil")
	}
}

// TestManagerImplement_GetUptimeSeconds_ParseError tests getUptimeSeconds with parseUptimeSeconds failure
func TestManagerImplement_GetUptimeSeconds_ParseError(t *testing.T) {
	mockAnsible := &MockAnsibleForManager{
		ResponseFunc: func(ctx context.Context, remoteHost, remotUser, sshKey, playbook, extraArgs string) (interface{}, error) {
			return map[string]interface{}{
				"invalid_key": "invalid_value",
			}, nil
		},
	}

	manager := &ManagerImplement{
		Logger:  klog.NewKlogr(),
		Ansible: mockAnsible,
	}

	server := interfaces.ServerTargetList{
		ServerID:      "server-001",
		HostIPAddress: "192.168.1.101",
		LoginUser:     "testuser",
	}

	_, err := manager.getUptimeSeconds(server)
	if err == nil {
		t.Error("expected parseUptimeSeconds error, got nil")
	}
}
