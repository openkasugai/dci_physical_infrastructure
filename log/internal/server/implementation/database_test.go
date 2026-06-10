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

	"log_module/internal/server/interfaces/mocks"
	"log_module/internal/server/test_utils"
	"log_module/internal/server/utils"
)

// TestDatabaseImplement_Init_GormOpenError_ReturnsError tests gorm.Open failure
func TestDatabaseImplement_Init_GormOpenError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Note: This test is designed to test the error path but may be limited
	// by the actual implementation's ability to mock gorm.Open

	// Setup - set all required environment variables
	defer setEnvDB("LOG_LEVEL", "2")()
	defer setEnvDB("INTERVAL", "60")()
	defer setEnvDB("IPMI_LOGFILE", "ipmi.log")()
	defer setEnvDB("IPMI_LOGPATH", "/var/log/ipmi")()
	defer setEnvDB("IPMI_MAXSIZE", "100")()
	defer setEnvDB("IPMI_MAXBACKUPS", "5")()
	defer setEnvDB("IPMI_MAXAGE", "7")()
	defer setEnvDB("CDI_LOGFILE", "cdi.log")()
	defer setEnvDB("CDI_LOGPATH", "/var/log/cdi")()
	defer setEnvDB("CDI_MAXSIZE", "200")()
	defer setEnvDB("CDI_MAXBACKUPS", "10")()
	defer setEnvDB("CDI_MAXAGE", "14")()
	defer setEnvDB("DB_URL", "https://invalid-host:3000")()
	
	// Reset and initialize config
	utils.ResetConfigForTesting()
	err := utils.InitializeConfig()
	assert.NoError(t, err)

	logger := klog.NewKlogr()
	db := &DatabaseImplement{Logger: logger}

	// Execute - will fail due to Kubernetes secret access
	err = db.Init()

	// Verify - should return error for invalid host
	assert.Error(t, err)
}

func Test_Database_Finalize(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test cases
	testCases := []struct {
		name string
	}{
		{
			name: "Successful finalize",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Create DatabaseImplement instance
		dbImpl := &DatabaseImplement{Logger: klog.Background()}

			// Execute Finalize (no error returned, just calls logger)
			dbImpl.Finalize()
		})
	}
}

func Test_SelectCDITable(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API with valid PG-CDI products and valid ExtraParameters
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi", gomock.Any(), "").
		Return([]interface{}{
			map[string]interface{}{
				"cdi_id":           "1",
				"remote_host":      "192.168.1.1",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
				"extra_parameters": `{"cdi_user":"user1","cdi_password":"pass1","cdi_guest":"192.168.10.1"}`,
			},
			map[string]interface{}{
				"cdi_id":           "2",
				"remote_host":      "192.168.1.2",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0"}`,
				"extra_parameters": `{"cdi_user":"user2","cdi_password":"pass2","cdi_guest":"192.168.10.2"}`,
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

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 2, len(targetList))
	assert.Equal(t, "192.168.1.1", targetList[0].CDIHost)
	assert.Equal(t, "192.168.1.2", targetList[1].CDIHost)
}

func Test_SelectServerTable(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// Mock logical server query
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        "Server-1-cots",
				"cdi_compute_server_id": "",
				"server_type":           float64(1), // COTS server
				"status":                float64(1),
				"os_id":                 float64(1),
				"host_ip_address":       "192.168.1.10",
				"p2p_enabled":           false,
				"cdi_machine_name":      "",
			},
		}, nil)
	
	// Mock COTS server query
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cots_server_pool", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"server_id":        "Server-1-cots",
				"ipmi_address":     "10.10.10.1",
				"ipmi_user":        "IPMIUser1",
				"ipmi_password":    "pass1",
				"product_info":     "{\"vendor\":\"fujitsu\",\"product_name\":\"PRIMERGY RX2530\",\"version\":\"M7\"}",
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

	// Assertions
	assert.NoError(t, err)
	assert.Greater(t, len(targetList), 0)
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

// TestDatabaseImplement_Init_MissingEnvVars_ReturnsError tests missing environment variables
func TestDatabaseImplement_Init_MissingEnvVars_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - clear all DB-related environment variables
	envVars := []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USERNAME", "SECRET_NAME", "SECRET_NAMESPACE"}
	cleanupFuncs := make([]func(), 0, len(envVars))

	for _, env := range envVars {
		cleanupFuncs = append(cleanupFuncs, setEnvDB(env, ""))
	}

	defer func() {
		for _, cleanup := range cleanupFuncs {
			cleanup()
		}
	}()

	logger := klog.NewKlogr()
	db := &DatabaseImplement{Logger: logger}

	// Execute
	err := db.Init()

	// Verify
	assert.Error(t, err)
}

// TestDatabaseImplement_SelectServerTable_DatabaseError_ReturnsError tests database query error
func TestDatabaseImplement_SelectServerTable_DatabaseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns error
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), gomock.Any()).
		Return(nil, errors.New("database error"))

	logger := klog.NewKlogr()
	db := &DatabaseImplement{Logger: logger, API: mockAPI, AccessURL: "https://test:3000", JWT: "test-jwt"}

	// Execute
	_, err := db.SelectServerTable()

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

// TestDatabaseImplement_SelectServerTable_NoServers_ReturnsEmpty tests no servers found
func TestDatabaseImplement_SelectServerTable_NoServers_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns empty list
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), gomock.Any()).
		Return([]interface{}{}, nil)

	logger := klog.NewKlogr()
	db := &DatabaseImplement{Logger: logger, API: mockAPI, AccessURL: "https://test:3000", JWT: "test-jwt"}

	// Execute
	targetList, err := db.SelectServerTable()

	// Verify
	assert.NoError(t, err)
	assert.Empty(t, targetList)
}

// TestDatabaseImplement_SelectServerTable_CotsServerQueryError_ReturnsError tests COTS server query error
func TestDatabaseImplement_SelectServerTable_CotsServerQueryError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// Mock logical server query to return COTS server
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        "Server-1-cots",
				"cdi_compute_server_id": nil,
				"server_type":           float64(1), // COTS
				"status":                float64(1),
				"os_id":                 float64(1),
				"host_ip_address":       "192.168.1.10",
				"p2p_enabled":           false,
				"cdi_machine_name":      nil,
			},
		}, nil)
	
	// Mock COTS server query to fail
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cots_server_pool", gomock.Any(), gomock.Any()).
		Return(nil, errors.New("COTS server query failed"))

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{Logger: klog.Background(), API: mockAPI, AccessURL: "https://test:3000", JWT: "test-jwt"}

	// Execute SelectServerTable
	_, err := dbImpl.SelectServerTable()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "COTS server query failed")
}

// TestDatabaseImplement_SelectServerTable_CDIServerQueryError_ReturnsError tests CDI server query error
func TestDatabaseImplement_SelectServerTable_CDIServerQueryError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	
	// Mock logical server query to return CDI server
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_logical_server", gomock.Any(), gomock.Any()).
		Return([]interface{}{
			map[string]interface{}{
				"cots_server_id":        nil,
				"cdi_compute_server_id": "Server-1-cdi",
				"server_type":           float64(0), // CDI
				"status":                float64(1),
				"os_id":                 float64(1),
				"host_ip_address":       "192.168.1.10",
				"p2p_enabled":           false,
				"cdi_machine_name":      "cdi-machine-1",
			},
		}, nil)
	
	// Mock CDI server query to fail
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi_compute_pool", gomock.Any(), gomock.Any()).
		Return(nil, errors.New("CDI server query failed"))

	// Create DatabaseImplement instance
	dbImpl := &DatabaseImplement{Logger: klog.Background(), API: mockAPI, AccessURL: "https://test:3000", JWT: "test-jwt"}

	// Execute SelectServerTable
	_, err := dbImpl.SelectServerTable()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CDI server query failed")
}

// TestDatabaseImplement_Finalize_Success tests finalize method
func TestDatabaseImplement_Finalize_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	db := &DatabaseImplement{Logger: logger}

	// Execute - should not panic
	assert.NotPanics(t, func() {
		db.Finalize()
	})
}

// TestParseExtraParameter_ValidJSON_ReturnsSuccess tests ParseExtraParameter with valid JSON
func TestParseExtraParameter_ValidJSON_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Valid JSON with all required fields
	validJSON := `{
		"cdi_user": "cdi_admin",
		"cdi_password": "cdi_pass123",
		"cdi_guest": "192.168.100.1",
		"cdimgr_guest_user": "guest_user",
		"cdimgr_guest_password": "guest_pass456"
	}`

	result, err := ParseExtraParameter(validJSON)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "cdi_admin", result.CDIUser)
	assert.Equal(t, "cdi_pass123", result.CDIPassword)
	assert.Equal(t, "192.168.100.1", result.CDIGuest)
	assert.Equal(t, "guest_user", result.CDIMgrGuestUser)
	assert.Equal(t, "guest_pass456", result.CDIMgrGuestPassword)
}

// TestParseExtraParameter_InvalidJSON_ReturnsError tests ParseExtraParameter with invalid JSON
func TestParseExtraParameter_InvalidJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	invalidJSON := `{"cdi_user": "admin", "invalid_json"`

	result, err := ParseExtraParameter(invalidJSON)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestParseExtraParameter_MissingRequiredFields_ReturnsError tests validation failure
func TestParseExtraParameter_MissingRequiredFields_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Missing required fields
	incompleteJSON := `{
		"cdi_user": "admin"
	}`

	result, err := ParseExtraParameter(incompleteJSON)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestDatabaseImplement_SelectCDITable_ProductTypeFiltering_ReturnsFilteredList tests new product type filtering
func TestDatabaseImplement_SelectCDITable_ProductTypeFiltering_ReturnsFilteredList(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API with mixed product types
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi", gomock.Any(), "").
		Return([]interface{}{
			map[string]interface{}{
				"cdi_id":           "1",
				"remote_host":      "192.168.1.1",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0"}`,
				"extra_parameters": `{"cdi_user":"user1","cdi_password":"pass1","cdi_guest":"192.168.10.1","cdimgr_guest_user":"guest1","cdimgr_guest_password":"gpass1"}`,
			},
			map[string]interface{}{
				"cdi_id":           "2",
				"remote_host":      "192.168.1.2",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
				"extra_parameters": `{"cdi_user":"user2","cdi_password":"pass2","cdi_guest":"192.168.10.2","cdimgr_guest_user":"guest2","cdimgr_guest_password":"gpass2"}`,
			},
			map[string]interface{}{
				"cdi_id":           "3",
				"remote_host":      "192.168.1.3",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{"vendor":"unknown","product_name":"Unsupported","version":"1.0"}`,
				"extra_parameters": `{"cdi_user":"user3","cdi_password":"pass3","cdi_guest":"192.168.10.3","cdimgr_guest_user":"guest3","cdimgr_guest_password":"gpass3"}`,
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

	// Assertions - should only include PG-CDI v1.0 and v1.1, exclude unsupported product
	assert.NoError(t, err)
	assert.Equal(t, 2, len(targetList))
	assert.Equal(t, "192.168.1.1", targetList[0].CDIHost)
	assert.Equal(t, "192.168.1.2", targetList[1].CDIHost)
	
	// Verify new fields are populated
	assert.Equal(t, "admin", targetList[0].CDIHostUser)
	assert.Equal(t, "user1", targetList[0].CDISoftUser)
	assert.Equal(t, "pass1", targetList[0].CDISoftPassword)
}

// TestDatabaseImplement_SelectCDITable_InvalidExtraParameters_SkipsEntry tests invalid ExtraParameters handling
func TestDatabaseImplement_SelectCDITable_InvalidExtraParameters_SkipsEntry(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API with one valid and one invalid ExtraParameters
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_cdi", gomock.Any(), "").
		Return([]interface{}{
			map[string]interface{}{
				"cdi_id":           "1",
				"remote_host":      "192.168.1.1",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0"}`,
				"extra_parameters": `{"invalid_json}`, // Invalid JSON
			},
			map[string]interface{}{
				"cdi_id":           "2",
				"remote_host":      "192.168.1.2",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
				"extra_parameters": `{"cdi_user":"user2","cdi_password":"pass2","cdi_guest":"192.168.10.2","cdimgr_guest_user":"guest2","cdimgr_guest_password":"gpass2"}`,
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

	// Assertions - should skip invalid entry, only return the valid one
	assert.NoError(t, err)
	assert.Equal(t, 1, len(targetList))
	assert.Equal(t, "192.168.1.2", targetList[0].CDIHost)
}

// TestDatabaseImplement_SelectMaasTable_Success tests successful MaaS table selection
func TestDatabaseImplement_SelectMaasTable_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_maas", gomock.Any(), "").
		Return([]interface{}{
			map[string]interface{}{
				"id":                  float64(1),
				"physical_infra_id":   float64(100),
				"access_url":          "http://maas1.example.com",
				"api_key":             "api-key-123",
				"status":              float64(1),
				"product_info":        "{\"vendor\":\"canonical\",\"product_name\":\"MAAS\",\"version\":\"3.6.2\"}",
				"extra_parameters":    "{}",
			},
			map[string]interface{}{
				"id":                  float64(2),
				"physical_infra_id":   float64(200),
				"access_url":          "http://maas2.example.com",
				"api_key":             "api-key-456",
				"status":              float64(1),
				"product_info":        "{\"vendor\":\"canonical\",\"product_name\":\"MAAS\",\"version\":\"3.6.2\"}",
				"extra_parameters":    "{}",
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectMaasTable
	targetList, err := dbImpl.SelectMaasTable()

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 2, len(targetList))
	assert.Equal(t, "http://maas1.example.com", targetList[0].MaasAccessUrl)
	assert.Equal(t, "api-key-123", targetList[0].MaasApiKey)
	assert.Equal(t, "{\"vendor\":\"canonical\",\"product_name\":\"MAAS\",\"version\":\"3.6.2\"}", targetList[0].ProductInfo)
	assert.Equal(t, "http://maas2.example.com", targetList[1].MaasAccessUrl)
	assert.Equal(t, "api-key-456", targetList[1].MaasApiKey)
}

// TestDatabaseImplement_SelectMaasTable_APIError_ReturnsError tests API failure handling
func TestDatabaseImplement_SelectMaasTable_APIError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API that returns error
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_maas", gomock.Any(), "").
		Return(nil, errors.New("API connection failed"))

	// Create DatabaseImplement instance
	dbImpl := DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectMaasTable
	targetList, err := dbImpl.SelectMaasTable()

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, 0, len(targetList))
	assert.Contains(t, err.Error(), "API connection failed")
}

// TestDatabaseImplement_SelectMaasTable_ParseError_ReturnsError tests parse error handling
func TestDatabaseImplement_SelectMaasTable_ParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API with invalid data format (missing required field)
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_maas", gomock.Any(), "").
		Return([]interface{}{
			map[string]interface{}{
				// Missing required fields
				"invalid_field": "value",
			},
		}, nil)

	// Create DatabaseImplement instance
	dbImpl := DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectMaasTable
	targetList, err := dbImpl.SelectMaasTable()

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, 0, len(targetList))
}

// TestDatabaseImplement_SelectMaasTable_EmptyResponse_ReturnsEmptyList tests empty MaaS list handling
func TestDatabaseImplement_SelectMaasTable_EmptyResponse_ReturnsEmptyList(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock API with empty response
	mockAPI := mocks.NewMockAPI(ctrl)
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), "GET", gomock.Any(), "t_maas", gomock.Any(), "").
		Return([]interface{}{}, nil)

	// Create DatabaseImplement instance
	dbImpl := DatabaseImplement{
		Logger:    klog.Background(),
		API:       mockAPI,
		AccessURL: "https://test:3000",
		JWT:       "test-jwt",
	}

	// Execute SelectMaasTable
	targetList, err := dbImpl.SelectMaasTable()

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 0, len(targetList))
}
