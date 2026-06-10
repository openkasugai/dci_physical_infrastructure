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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/klog/v2"

	"log_module/internal/server/interfaces/mocks"
	"log_module/internal/server/test_utils"
)

// TestDatabaseImplement_SelectCDITable_APIExecuteJWTAuthError_ReturnsError tests API call failure
func TestDatabaseImplement_SelectCDITable_APIExecuteJWTAuthError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns error
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi", gomock.Any(), "").
		Return(nil, errors.New("API connection failed"))

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectCDITable
	targetList, err := dbImpl.SelectCDITable()

	// Verify - should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API connection failed")
	assert.Nil(t, targetList)
}

// TestDatabaseImplement_SelectCDITable_ParseError_ReturnsError tests Parse failure
func TestDatabaseImplement_SelectCDITable_ParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns invalid data format
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi", gomock.Any(), "").
		Return([]interface{}{
			map[string]interface{}{
				"cdi_id": 12345, // Invalid type - should be string
				// missing required fields
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectCDITable
	targetList, err := dbImpl.SelectCDITable()

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
	dbImpl := DatabaseImplement{
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
				"server_type":           float64(1), // COTS
				"status":                float64(1),
				"os_id":                 float64(1),
				"host_ip_address":       "192.168.1.10",
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
	dbImpl := DatabaseImplement{
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
				"cots_server_id":        nil, // nil for CDI server
				"cdi_compute_server_id": cdiServerID,
				"server_type":           float64(0), // CDI
				"status":                float64(1),
				"os_id":                 float64(1),
				"host_ip_address":       "192.168.1.10",
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
	dbImpl := DatabaseImplement{
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

// TestDatabaseImplement_SelectServerTable_CDIComposedServer_Success tests CDI Composed Server selection
func TestDatabaseImplement_SelectServerTable_CDIComposedServer_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cdiServerID := "cdi-compute-001"

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)

	// Mock call 1: t_logical_server table query
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), "status=eq.1").
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        nil,
				"cdi_compute_server_id": cdiServerID,
				"server_type":           float64(0), // CDI Composed Server
				"status":                float64(1),
				"os_id":                 float64(1),
				"host_ip_address":       "192.168.3.50",
				"p2p_enabled":           false,
				"cdi_machine_name":      "cdi-machine-test",
			},
		}, nil)

	// Mock call 2: t_cdi_compute_pool table query
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi_compute_pool", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"server_id":       cdiServerID,
				"cdi_id":          "cdi-mgr-001",
				"ipmi_address":    "192.168.3.50",
				"ipmi_user":       "testuser",
				"ipmi_password":   "testpass",
				"product_info":    "PG-CDI 1.1",
				"extra_parameters": "{}",
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := DatabaseImplement{
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
	assert.Equal(t, "cdi-compute-001", cdiServer.ServerID)
	assert.Equal(t, "192.168.3.50", cdiServer.IPMIAddress)
	assert.Equal(t, "testuser", cdiServer.IPMIUser)
	assert.Equal(t, "testpass", cdiServer.IPMIPassword)
	assert.Equal(t, "PG-CDI 1.1", cdiServer.ProductInfo)
}
