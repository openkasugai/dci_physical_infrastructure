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
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/klog/v2"

	"exporter_module/internal/server/interfaces"
	"exporter_module/internal/server/interfaces/mocks"
	"exporter_module/internal/server/test_utils"
	"exporter_module/internal/server/utils"
)

// TestDatabaseImplement_Init_GetSecretDataError_ReturnsError tests JWT retrieval failure
func TestDatabaseImplement_Init_GetSecretDataError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - set all required environment variables
	defer setEnvDB("INTERVAL", "60")()
	defer setEnvDB("P2P_ENABLE", "true")()
	defer setEnvDB("P2P_INTERVAL", "300")()
	defer setEnvDB("SSH_KEY", "/path/to/ssh/key")()
	defer setEnvDB("METRICS_PORT", "9090")()
	defer setEnvDB("METRICS_ENDPOINT", "/metrics")()
	defer setEnvDB("DB_URL", "https://postgrest:3000")()

	// Reset and initialize config
	utils.ResetConfigForTesting()
	err := utils.InitializeConfig()
	assert.NoError(t, err)

	logger := klog.NewKlogr()
	db := &DatabaseImplement{Logger: logger}

	// Execute - In test environment (not in a Kubernetes cluster),
	// GetSecretData will fail because rest.InClusterConfig() fails
	err = db.Init()

	// Verify - should fail with JWT retrieval error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve JWT")
}

// TestDatabaseImplement_SelectNwSwitchTable_APIExecuteJWTAuthError_ReturnsError tests API call failure
func TestDatabaseImplement_SelectNwSwitchTable_APIExecuteJWTAuthError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns error
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_nw_switch", gomock.Any(), "").
		Return(nil, errors.New("API connection failed"))

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectNwSwitchTable
	targetList, err := dbImpl.SelectNwSwitchTable()

	// Verify - should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API connection failed")
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectNwSwitchTable_ParseError_ReturnsError tests Parse failure
func TestDatabaseImplement_SelectNwSwitchTable_ParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns invalid data format
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_nw_switch", gomock.Any(), "").
		Return([]interface{}{
			map[string]interface{}{
				"nw_id": 12345, // Invalid type - should be string
				// missing required fields
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectNwSwitchTable
	targetList, err := dbImpl.SelectNwSwitchTable()

	// Verify - should return parse error
	assert.Error(t, err)
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectServerTable_APIExecuteJWTAuthError_ReturnsError tests API call failure
func TestDatabaseImplement_SelectServerTable_APIExecuteJWTAuthError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns error
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return(nil, errors.New("API connection failed"))

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API connection failed")
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectServerTable_LogicalServerParseError_ReturnsError tests Parse failure
func TestDatabaseImplement_SelectServerTable_LogicalServerParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns invalid data format for logical server
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"server_id": 12345, // Invalid type - should be string
				// missing required fields
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should return parse error
	assert.Error(t, err)
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectServerTable_COTSServerParseError_ReturnsError tests COTS server parse failure
func TestDatabaseImplement_SelectServerTable_COTSServerParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cotsServerID := "cots-001"

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// First call: logical server query succeeds with COTS server
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        cotsServerID,
				"cdi_compute_server_id": "",
				"server_type":           float64(1),
				"status":                float64(1),
				"os_id":                 float64(1),
				"mgr_ip_address":       "192.168.1.10",
				"p2p_enabled":           false,
				"cdi_machine_name":      "",
			},
		}, nil)

	// Second call: COTS server query returns invalid data
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cots_server_pool", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"server_id": 12345, // Invalid type - should be string
				// missing required fields
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should return parse error
	assert.Error(t, err)
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectServerTable_CDIComputeParseError_ReturnsError tests CDI compute server parse failure
func TestDatabaseImplement_SelectServerTable_CDIComputeParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cdiServerID := "cdi-001"

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// First call: logical server query succeeds with CDI server
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        nil,
				"cdi_compute_server_id": cdiServerID,
				"server_type":           float64(0),
				"status":                float64(1),
				"os_id":                 float64(1),
				"mgr_ip_address":       "192.168.1.10",
				"p2p_enabled":           false,
				"cdi_machine_name":      "cdi-machine-1",
			},
		}, nil)

	// Second call: CDI server query returns invalid data
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi_compute_pool", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"server_id": 12345, // Invalid type - should be string
				// missing required fields
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should return parse error
	assert.Error(t, err)
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectServerTable_CDITableParseError_ReturnsError tests CDI table parse failure
func TestDatabaseImplement_SelectServerTable_CDITableParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cdiServerID := "cdi-001"
	cdiID := "cdi-host-001"

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// First call: logical server query succeeds with CDI server
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        nil,
				"cdi_compute_server_id": cdiServerID,
				"server_type":           float64(0),
				"status":                float64(1),
				"os_id":                 float64(1),
				"mgr_ip_address":       "192.168.1.10",
				"p2p_enabled":           false,
				"cdi_machine_name":      "cdi-machine-1",
			},
		}, nil)

	// Second call: CDI compute pool query succeeds
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi_compute_pool", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"server_id":        cdiServerID,
				"cdi_id":           cdiID,
				"ipmi_address":     "10.10.10.1",
				"ipmi_user":        "IPMIUser1",
				"ipmi_password":    "pass1",
				"product_info":     "{}",
				"extra_parameters": "{}",
			},
		}, nil)

	// Third call: CDI table query returns invalid data
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"cdi_id": 12345, // Invalid type - should be string
				// missing required fields
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should return parse error
	assert.Error(t, err)
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectServerTable_OSInfoParseError_ReturnsError tests OS info parse failure
func TestDatabaseImplement_SelectServerTable_OSInfoParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cotsServerID := "cots-001"

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// First call: logical server query succeeds
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        cotsServerID,
				"cdi_compute_server_id": "",
				"server_type":           float64(1),
				"status":                float64(1),
				"os_id":                 float64(1),
				"mgr_ip_address":       "192.168.1.10",
				"p2p_enabled":           false,
				"cdi_machine_name":      "",
			},
		}, nil)

	// Second call: COTS server query succeeds
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cots_server_pool", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"server_id":        cotsServerID,
				"ipmi_address":     "10.10.10.1",
				"ipmi_user":        "IPMIUser1",
				"ipmi_password":    "pass1",
				"product_info":     "{}",
				"extra_parameters": "{}",
			},
		}, nil)

	// Third call: OS info query returns invalid data
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_os_info", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"id": "invalid", // Invalid type - should be float64
				// missing required fields
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should return parse error
	assert.Error(t, err)
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectServerTable_VMServer_ReturnsSuccess tests VM server processing
func TestDatabaseImplement_SelectServerTable_VMServer_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vmServerID := "vm-001"

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// First call: logical server query succeeds with VM server (ServerType=2)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        vmServerID,
				"cdi_compute_server_id": "",
				"server_type":           float64(2), // VM
				"status":                float64(1),
				"os_id":                 float64(1),
				"mgr_ip_address":       "192.168.1.50",
				"p2p_enabled":           true,
				"cdi_machine_name":      "",
			},
		}, nil)

	// Second call: OS info query succeeds (VM also requires OS info)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_os_info", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"id":         float64(1),
				"login_user": "ubuntu",
			},
		}, nil)

	// Note: For VM (ServerType=2), no COTS server pool API call should be made

	// Create DatabaseImplement instance
	dbImpl := DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should succeed with VM target
	assert.NoError(t, err)
	assert.NotNil(t, targetList)
	assert.Equal(t, 1, len(targetList))
	
	// Verify VM target has correct fields
	vmTarget := targetList[0]
	assert.Equal(t, vmServerID, vmTarget.ServerID)
	assert.True(t, vmTarget.P2PEnable)
	// VM should not have IPMI or other COTS-specific fields populated
	assert.Equal(t, "", vmTarget.IpmiAddress)
	assert.Equal(t, "", vmTarget.IpmiUser)
	assert.Equal(t, "", vmTarget.IpmiPassword)
}

// TestDatabaseImplement_SelectServerTable_MixedServerTypes_ReturnsSuccess tests mixed COTS and VM servers
func TestDatabaseImplement_SelectServerTable_MixedServerTypes_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cotsServerID := "cots-001"
	vmServerID := "vm-001"

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// First call: logical server query succeeds with both COTS and VM servers
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        cotsServerID,
				"cdi_compute_server_id": "",
				"server_type":           float64(1), // COTS
				"status":                float64(1),
				"os_id":                 float64(1),
				"mgr_ip_address":       "192.168.1.10",
				"p2p_enabled":           false,
				"cdi_machine_name":      "",
			},
			map[string]interface{}{
				"cots_server_id":        vmServerID,
				"cdi_compute_server_id": "",
				"server_type":           float64(2), // VM
				"status":                float64(1),
				"os_id":                 float64(1),
				"mgr_ip_address":       "192.168.1.50",
				"p2p_enabled":           true,
				"cdi_machine_name":      "",
			},
		}, nil)

	// Second call: COTS server query succeeds (only for COTS server, not VM)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cots_server_pool", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"server_id":        cotsServerID,
				"ipmi_address":     "10.10.10.1",
				"ipmi_user":        "IPMIUser1",
				"ipmi_password":    "pass1",
				"product_info":     "{}",
				"extra_parameters": "{}",
			},
		}, nil)

	// Third call: OS info query succeeds (for both servers)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_os_info", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"id":         float64(1),
				"login_user": "ubuntu",
			},
		}, nil).Times(2) // Called twice, once for COTS and once for VM

	// Create DatabaseImplement instance
	dbImpl := DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should succeed with both COTS and VM targets
	assert.NoError(t, err)
	assert.NotNil(t, targetList)
	assert.Equal(t, 2, len(targetList))
	
	// Find COTS and VM targets
	var cotsTarget, vmTarget *interfaces.ServerTargetList
	for i := range targetList {
		if targetList[i].ServerID == cotsServerID {
			cotsTarget = &targetList[i]
		} else if targetList[i].ServerID == vmServerID {
			vmTarget = &targetList[i]
		}
	}
	
	// Verify COTS target has IPMI fields
	assert.NotNil(t, cotsTarget)
	assert.Equal(t, "10.10.10.1", cotsTarget.IpmiAddress)
	assert.Equal(t, "IPMIUser1", cotsTarget.IpmiUser)
	assert.Equal(t, "pass1", cotsTarget.IpmiPassword)
	
	// Verify VM target does not have IPMI fields
	assert.NotNil(t, vmTarget)
	assert.Equal(t, "", vmTarget.IpmiAddress)
	assert.Equal(t, "", vmTarget.IpmiUser)
	assert.Equal(t, "", vmTarget.IpmiPassword)
	assert.True(t, vmTarget.P2PEnable)
}

// TestDatabaseImplement_SelectNwSwitchTable_Success_ReturnsTargetList tests successful network switch selection
func TestDatabaseImplement_SelectNwSwitchTable_Success_ReturnsTargetList(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns valid network switch data
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_nw_switch", gomock.Any(), "").
		Return([]interface{}{
			map[string]interface{}{
				"nw_ip_address":   "192.168.1.10",
				"nw_user":         "admin",
				"product_info":    "Cisco Nexus 9000",
				"extra_parameters": "{}",
			},
			map[string]interface{}{
				"nw_ip_address":   "192.168.1.11",
				"nw_user":         "admin",
				"product_info":    "Arista 7050",
				"extra_parameters": "{}",
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectNwSwitchTable
	targetList, err := dbImpl.SelectNwSwitchTable()

	// Verify - should return success with 2 network switches
	assert.NoError(t, err)
	assert.NotNil(t, targetList)
	assert.Len(t, targetList, 2)

	// Verify first network switch
	assert.Equal(t, "192.168.1.10", targetList[0].IPAddress)
	assert.Equal(t, "admin", targetList[0].LoginUser)
	assert.Equal(t, "Cisco Nexus 9000", targetList[0].ProductInfo)

	// Verify second network switch
	assert.Equal(t, "192.168.1.11", targetList[1].IPAddress)
	assert.Equal(t, "admin", targetList[1].LoginUser)
	assert.Equal(t, "Arista 7050", targetList[1].ProductInfo)
}

// TestDatabaseImplement_SelectServerTable_CDIComposedServer_Success tests CDI Composed Server selection
func TestDatabaseImplement_SelectServerTable_CDIComposedServer_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)

	// Mock call 1: t_logical_server table query
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        nil,
				"cdi_compute_server_id": "cdi001",
				"server_type":           float64(0), // CDI Composed Server
				"status":                float64(1),
				"os_id":                 float64(1),
				"p2p_enabled":           false,
				"cdi_machine_name":      "CDI-Server-001",
			},
		}, nil)

	// Mock call 2: t_cdi_compute_pool table query
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi_compute_pool", gomock.Any(), "server_id=eq.cdi001").
		Return([]interface{}{
			map[string]interface{}{
				"server_id":       "cdi001",
				"ipmi_address":    "192.168.2.100",
				"ipmi_user":       "root",
				"ipmi_password":   "pass123",
				"cdi_id":          "cdi-mgr-001",
				"product_info":    "PG-CDI 1.1",
				"extra_parameters": "{}",
			},
		}, nil)

	// Mock call 3: t_cdi table query
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi", gomock.Any(), "cdi_id=eq.cdi-mgr-001").
		Return([]interface{}{
			map[string]interface{}{
				"cdi_id":          "cdi-mgr-001",
				"remote_host":     "192.168.100.10",
				"remote_user":     "cdiuser",
				"group_name":      "compute-group-1",
				"product_info":    "PG-CDI 1.1",
				"extra_parameters": "{}",
			},
		}, nil)

	// Mock call 4: t_os_info table query
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_os_info", gomock.Any(), "id=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"id":           float64(1),
				"login_user":   "ubuntu",
				"product_info": "Ubuntu 22.04",
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectServerTable
	targetList, err := dbImpl.SelectServerTable()

	// Verify - should return success with CDI Composed Server
	assert.NoError(t, err)
	assert.NotNil(t, targetList)
	assert.Len(t, targetList, 1)

	// Verify CDI Composed Server details
	cdiServer := targetList[0]
	assert.Equal(t, "cdi001", cdiServer.ServerID)
	assert.Equal(t, "192.168.2.100", cdiServer.IpmiAddress)
	assert.Equal(t, "root", cdiServer.IpmiUser)
	assert.Equal(t, "pass123", cdiServer.IpmiPassword)
	assert.Equal(t, "PG-CDI 1.1", cdiServer.ProductInfo)

	// Verify CDI Info (should be set from t_cdi table)
	assert.Equal(t, "192.168.100.10", cdiServer.CdiInfo.RemoteHost)
	assert.Equal(t, "cdiuser", cdiServer.CdiInfo.RemoteUser)
	assert.Equal(t, "CDI-Server-001", cdiServer.CdiInfo.MachineName)
	assert.Equal(t, "PG-CDI 1.1", cdiServer.CdiInfo.ProductInfo)
}

// Helper function to set environment variable and return cleanup function
func setEnvDB(key, value string) func() {
	original := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		if original != "" {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	}
}
