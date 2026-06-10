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

	"log_module/internal/server/interfaces"
	"log_module/internal/server/interfaces/mocks"
	"log_module/internal/server/test_utils"
)

// TestMaaSImplement_Init_Success tests successful initialization
func TestMaaSImplement_Init_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup environment variables
	defer setEnvMaaS("MAAS_LOGFILE", "maas.log")()
	defer setEnvMaaS("MAAS_LOGPATH", "/var/log/maas")()
	defer setEnvMaaS("MAAS_MAXSIZE", "100")()
	defer setEnvMaaS("MAAS_MAXBACKUPS", "5")()
	defer setEnvMaaS("MAAS_MAXAGE", "7")()

	// Create mock Logging
	mockLogging := mocks.NewMockLogging(ctrl)
	mockLogging.EXPECT().
		Init(interfaces.LoggingConfig{
			LogFile:    "maas.log",
			LogPath:    "/var/log/maas",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     7,
		}).
		Return(nil)

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Execute Init
	err := maas.Init()

	// Assertions
	assert.NoError(t, err)
}

// TestMaaSImplement_Init_LoggingInitError_ReturnsError tests Init with logging initialization failure
func TestMaaSImplement_Init_LoggingInitError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup environment variables
	defer setEnvMaaS("MAAS_LOGFILE", "maas.log")()
	defer setEnvMaaS("MAAS_LOGPATH", "/invalid/path")()
	defer setEnvMaaS("MAAS_MAXSIZE", "100")()
	defer setEnvMaaS("MAAS_MAXBACKUPS", "5")()
	defer setEnvMaaS("MAAS_MAXAGE", "7")()

	// Create mock Logging that returns error
	mockLogging := mocks.NewMockLogging(ctrl)
	mockLogging.EXPECT().
		Init(gomock.Any()).
		Return(errors.New("failed to initialize logging"))

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Execute Init
	err := maas.Init()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize logging")
}

// TestMaaSImplement_Finalize_Success tests successful finalization
func TestMaaSImplement_Finalize_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock Logging
	mockLogging := mocks.NewMockLogging(ctrl)
	mockLogging.EXPECT().Finalize()

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Execute Finalize - should not panic
	assert.NotPanics(t, func() {
		maas.Finalize()
	})
}

// TestMaaSImplement_Collection_CanonicalMaaS_Success tests collection for Canonical MaaS
func TestMaaSImplement_Collection_CanonicalMaaS_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Setup expectations
	mockAPI.EXPECT().
		APIExecuteJWTAUth(
			gomock.Any(),
			"GET",
			"http://maas1.example.com",
			"maas/op-get_config",
			"api-key-123",
			"name=maas_name",
		).
		Return(nil, nil) // Success

	mockLogging.EXPECT().
		Write("http://maas1.example.com", "{\"Health\":\"OK\"}").
		Return(nil)

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Create target list
	targetList := []interfaces.MaasServerTargetList{
		{
			MaasAccessUrl: "http://maas1.example.com",
			MaasApiKey:    "api-key-123",
			ProductInfo:   `{"vendor":"canonical","product_name":"MAAS","version":"3.6.2"}`,
		},
	}

	// Execute Collection
	assert.NotPanics(t, func() {
		maas.Collection(targetList)
	})
}

// TestMaaSImplement_Collection_APIError_WritesNG tests API execution failure
func TestMaaSImplement_Collection_APIError_WritesNG(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// API fails
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("API execution failed"))

	// Should write NG status
	mockLogging.EXPECT().
		Write("http://maas1.example.com", "{\"Health\":\"NG\"}").
		Return(nil)

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Create target list
	targetList := []interfaces.MaasServerTargetList{
		{
			MaasAccessUrl: "http://maas1.example.com",
			MaasApiKey:    "api-key-123",
			ProductInfo:   `{"vendor":"canonical","product_name":"MAAS","version":"3.6.2"}`,
		},
	}

	// Execute Collection
	assert.NotPanics(t, func() {
		maas.Collection(targetList)
	})
}

// TestMaaSImplement_Collection_LoggingWriteError_ContinuesProcessing tests logging write failure
func TestMaaSImplement_Collection_LoggingWriteError_ContinuesProcessing(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// API succeeds
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)

	// Logging write fails
	mockLogging.EXPECT().
		Write(gomock.Any(), gomock.Any()).
		Return(errors.New("logging write failed"))

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Create target list
	targetList := []interfaces.MaasServerTargetList{
		{
			MaasAccessUrl: "http://maas1.example.com",
			MaasApiKey:    "api-key-123",
			ProductInfo:   `{"vendor":"canonical","product_name":"MAAS","version":"3.6.2"}`,
		},
	}

	// Execute Collection - should not panic, continues processing
	assert.NotPanics(t, func() {
		maas.Collection(targetList)
	})
}

// TestMaaSImplement_Collection_UnsupportedProduct_SkipsTarget tests unsupported product type
func TestMaaSImplement_Collection_UnsupportedProduct_SkipsTarget(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks - should not be called for unsupported product
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Create target list with unsupported product
	targetList := []interfaces.MaasServerTargetList{
		{
			MaasAccessUrl: "http://maas1.example.com",
			MaasApiKey:    "api-key-123",
			ProductInfo:   `{"vendor":"unknown","product_name":"Unsupported","version":"1.0"}`,
		},
	}

	// Execute Collection - should not panic, just skip
	assert.NotPanics(t, func() {
		maas.Collection(targetList)
	})
}

// TestMaaSImplement_Collection_EmptyTargetList_ReturnsImmediately tests empty target list
func TestMaaSImplement_Collection_EmptyTargetList_ReturnsImmediately(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks - should not be called
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Execute Collection with empty list
	assert.NotPanics(t, func() {
		maas.Collection([]interfaces.MaasServerTargetList{})
	})
}

// TestMaaSImplement_Collection_MultipleTargets_ProcessesAll tests multiple targets processing
func TestMaaSImplement_Collection_MultipleTargets_ProcessesAll(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// First target
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), gomock.Any(), "http://maas1.example.com", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)
	mockLogging.EXPECT().
		Write("http://maas1.example.com", "{\"Health\":\"OK\"}").
		Return(nil)

	// Second target
	mockAPI.EXPECT().
		APIExecuteJWTAUth(gomock.Any(), gomock.Any(), "http://maas2.example.com", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)
	mockLogging.EXPECT().
		Write("http://maas2.example.com", "{\"Health\":\"OK\"}").
		Return(nil)

	// Create MaaSImplement instance
	maas := MaaSImplement{
		Logger:  klog.Background(),
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Create target list with multiple targets
	targetList := []interfaces.MaasServerTargetList{
		{
			MaasAccessUrl: "http://maas1.example.com",
			MaasApiKey:    "api-key-123",
			ProductInfo:   `{"vendor":"canonical","product_name":"MAAS","version":"3.6.2"}`,
		},
		{
			MaasAccessUrl: "http://maas2.example.com",
			MaasApiKey:    "api-key-456",
			ProductInfo:   `{"vendor":"canonical","product_name":"MAAS","version":"3.6.2"}`,
		},
	}

	// Execute Collection
	assert.NotPanics(t, func() {
		maas.Collection(targetList)
	})
}

// Helper function to set environment variable and return cleanup function
func setEnvMaaS(key, value string) func() {
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
