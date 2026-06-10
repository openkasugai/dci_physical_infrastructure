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

// TestCDISoftImplement_Init_Success tests successful initialization
func TestCDISoftImplement_Init_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup environment variables
	defer setEnvCDISoft("CDISOFT_LOGFILE", "cdisoft.log")()
	defer setEnvCDISoft("CDISOFT_LOGPATH", "/var/log/cdisoft")()
	defer setEnvCDISoft("CDISOFT_MAXSIZE", "100")()
	defer setEnvCDISoft("CDISOFT_MAXBACKUPS", "5")()
	defer setEnvCDISoft("CDISOFT_MAXAGE", "7")()

	// Create mock Logging
	mockLogging := mocks.NewMockLogging(ctrl)
	mockLogging.EXPECT().
		Init(interfaces.LoggingConfig{
			LogFile:    "cdisoft.log",
			LogPath:    "/var/log/cdisoft",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     7,
		}).
		Return(nil)

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Execute Init
	err := cdiSoft.Init()

	// Assertions
	assert.NoError(t, err)
}

// TestCDISoftImplement_Init_LoggingInitError_ReturnsError tests Init with logging initialization failure
func TestCDISoftImplement_Init_LoggingInitError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup environment variables
	defer setEnvCDISoft("CDISOFT_LOGFILE", "cdisoft.log")()
	defer setEnvCDISoft("CDISOFT_LOGPATH", "/invalid/path")()
	defer setEnvCDISoft("CDISOFT_MAXSIZE", "100")()
	defer setEnvCDISoft("CDISOFT_MAXBACKUPS", "5")()
	defer setEnvCDISoft("CDISOFT_MAXAGE", "7")()

	// Create mock Logging that returns error
	mockLogging := mocks.NewMockLogging(ctrl)
	mockLogging.EXPECT().
		Init(gomock.Any()).
		Return(errors.New("failed to initialize logging"))

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Execute Init
	err := cdiSoft.Init()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize logging")
}

// TestCDISoftImplement_Finalize_Success tests successful finalization
func TestCDISoftImplement_Finalize_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock Logging
	mockLogging := mocks.NewMockLogging(ctrl)
	mockLogging.EXPECT().Finalize()

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Execute Finalize - should not panic
	assert.NotPanics(t, func() {
		cdiSoft.Finalize()
	})
}

// TestCDISoftImplement_Collection_PGCDI10_Success tests collection for PG-CDI v1.0
func TestCDISoftImplement_Collection_PGCDI10_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Setup expectations
	mockAnsible.EXPECT().
		CmdExecute(
			gomock.Any(),
			"192.168.10.1",
			"guest_user1",
			"cdi_spec_list.yaml",
			"cdi_user=guest_user1 cdi_password=guest_pass1 cdi_guest=192.168.10.1",
		).
		Return(nil, nil) // Success

	mockLogging.EXPECT().
		Write("192.168.10.1", "{\"Health\":\"OK\"}").
		Return(nil)

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Create target list
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0"}`,
			ExtraParameters: `{"cdi_user":"user1","cdi_password":"pass1","cdi_guest":"192.168.10.1","cdimgr_guest_user":"guest_user1","cdimgr_guest_password":"guest_pass1"}`,
		},
	}

	// Execute Collection
	assert.NotPanics(t, func() {
		cdiSoft.Collection(targetList)
	})
}

// TestCDISoftImplement_Collection_PGCDI11_Success tests collection for PG-CDI v1.1
func TestCDISoftImplement_Collection_PGCDI11_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Setup expectations
	mockAnsible.EXPECT().
		CmdExecute(
			gomock.Any(),
			"192.168.10.2",
			"guest_user2",
			"cdi_spec_list.yaml",
			"cdi_user=guest_user2 cdi_password=guest_pass2 cdi_guest=192.168.10.2",
		).
		Return(nil, nil)

	mockLogging.EXPECT().
		Write("192.168.10.2", "{\"Health\":\"OK\"}").
		Return(nil)

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Create target list
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.2",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
			ExtraParameters: `{"cdi_user":"user2","cdi_password":"pass2","cdi_guest":"192.168.10.2","cdimgr_guest_user":"guest_user2","cdimgr_guest_password":"guest_pass2"}`,
		},
	}

	// Execute Collection
	assert.NotPanics(t, func() {
		cdiSoft.Collection(targetList)
	})
}

// TestCDISoftImplement_Collection_AnsibleError_WritesNG tests Ansible execution failure
func TestCDISoftImplement_Collection_AnsibleError_WritesNG(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Ansible fails
	mockAnsible.EXPECT().
		CmdExecute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("ansible execution failed"))

	// Should write NG status
	mockLogging.EXPECT().
		Write("192.168.10.1", "{\"Health\":\"NG\"}").
		Return(nil)

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Create target list
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0"}`,
			ExtraParameters: `{"cdi_user":"user1","cdi_password":"pass1","cdi_guest":"192.168.10.1","cdimgr_guest_user":"guest_user1","cdimgr_guest_password":"guest_pass1"}`,
		},
	}

	// Execute Collection
	assert.NotPanics(t, func() {
		cdiSoft.Collection(targetList)
	})
}

// TestCDISoftImplement_Collection_InvalidExtraParameters_SkipsTarget tests invalid ExtraParameters handling
func TestCDISoftImplement_Collection_InvalidExtraParameters_SkipsTarget(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks - should not be called due to parse error
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Create target list with invalid JSON
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0"}`,
			ExtraParameters: `{"invalid_json}`, // Invalid JSON
		},
	}

	// Execute Collection - should not panic, just skip
	assert.NotPanics(t, func() {
		cdiSoft.Collection(targetList)
	})
}

// TestCDISoftImplement_Collection_LoggingWriteError_ContinuesProcessing tests logging write failure
func TestCDISoftImplement_Collection_LoggingWriteError_ContinuesProcessing(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Ansible succeeds
	mockAnsible.EXPECT().
		CmdExecute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)

	// Logging write fails
	mockLogging.EXPECT().
		Write(gomock.Any(), gomock.Any()).
		Return(errors.New("logging write failed"))

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Create target list
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0"}`,
			ExtraParameters: `{"cdi_user":"user1","cdi_password":"pass1","cdi_guest":"192.168.10.1","cdimgr_guest_user":"guest_user1","cdimgr_guest_password":"guest_pass1"}`,
		},
	}

	// Execute Collection - should not panic, continues processing
	assert.NotPanics(t, func() {
		cdiSoft.Collection(targetList)
	})
}

// TestCDISoftImplement_Collection_UnsupportedProduct_SkipsTarget tests unsupported product type
func TestCDISoftImplement_Collection_UnsupportedProduct_SkipsTarget(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks - should not be called for unsupported product
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Create target list with unsupported product
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"unknown","product_name":"Unsupported","version":"1.0"}`,
			ExtraParameters: `{"cdi_user":"user1","cdi_password":"pass1","cdi_guest":"192.168.10.1"}`,
		},
	}

	// Execute Collection - should not panic, just skip
	assert.NotPanics(t, func() {
		cdiSoft.Collection(targetList)
	})
}

// TestCDISoftImplement_Collection_EmptyTargetList_ReturnsImmediately tests empty target list
func TestCDISoftImplement_Collection_EmptyTargetList_ReturnsImmediately(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks - should not be called
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Execute Collection with empty list
	assert.NotPanics(t, func() {
		cdiSoft.Collection([]interfaces.CDITargetList{})
	})
}

// TestCDISoftImplement_Collection_MultipleTargets_ProcessesAll tests multiple targets processing
func TestCDISoftImplement_Collection_MultipleTargets_ProcessesAll(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := test_utils.SetupProductMappings()
	defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// First target - v1.0
	mockAnsible.EXPECT().
		CmdExecute(gomock.Any(), "192.168.10.1", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)
	mockLogging.EXPECT().
		Write("192.168.10.1", "{\"Health\":\"OK\"}").
		Return(nil)

	// Second target - v1.1
	mockAnsible.EXPECT().
		CmdExecute(gomock.Any(), "192.168.10.2", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)
	mockLogging.EXPECT().
		Write("192.168.10.2", "{\"Health\":\"OK\"}").
		Return(nil)

	// Create CDISoftImplement instance
	cdiSoft := CDISoftImplement{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Create target list with multiple targets
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0"}`,
			ExtraParameters: `{"cdi_user":"user1","cdi_password":"pass1","cdi_guest":"192.168.10.1","cdimgr_guest_user":"guest_user1","cdimgr_guest_password":"guest_pass1"}`,
		},
		{
			CDIHost:         "192.168.1.2",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
			ExtraParameters: `{"cdi_user":"user2","cdi_password":"pass2","cdi_guest":"192.168.10.2","cdimgr_guest_user":"guest_user2","cdimgr_guest_password":"guest_pass2"}`,
		},
	}

	// Execute Collection
	assert.NotPanics(t, func() {
		cdiSoft.Collection(targetList)
	})
}

// Helper function to set environment variable and return cleanup function
func setEnvCDISoft(key, value string) func() {
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
