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

package main

import (
	"errors"
	"os"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"k8s.io/klog/v2"

	"log_module/factory"                          // import of factory
	"log_module/internal/server/interfaces"       // import of interface
	"log_module/internal/server/interfaces/mocks" // Mock for test code
	"log_module/internal/server/utils"
)

// Helper function to set up environment variables for main tests
func setEnvForMainTest(t *testing.T, envVars map[string]string) func() {
	originalValues := make(map[string]string)

	// Save original values
	for key := range envVars {
		originalValues[key] = os.Getenv(key)
	}

	// Set test values
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	return func() {
		// Restore original values
		for key, originalValue := range originalValues {
			if originalValue != "" {
				os.Setenv(key, originalValue)
			} else {
				os.Unsetenv(key)
			}
		}
		// Reset global states for next test
		running = true
		utils.ResetConfigForTesting()
	}
}

// Helper function to set up complete valid environment variables for main tests
func setupValidMainEnvVars() map[string]string {
	return map[string]string{
		"LOG_LEVEL":       "2",
		"INTERVAL":        "5", // Short interval for testing
		"IPMI_LOGFILE":    "server",
		"IPMI_LOGPATH":    "/var/log/dci_physical_infrastructure/log",
		"IPMI_MAXSIZE":    "2048",
		"IPMI_MAXBACKUPS": "10",
		"IPMI_MAXAGE":     "30",
		"CDI_LOGFILE":     "cdi",
		"CDI_LOGPATH":     "/var/log/cdi_hhchk",
		"CDI_MAXSIZE":     "2048",
		"CDI_MAXBACKUPS":  "10",
		"CDI_MAXAGE":      "30",
		"DB_URL":          "https://localhost:3000",
	}
}

func Test_Main(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock instanses
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockAPI := mocks.NewMockAPI(ctrl)
	mockDatabase := mocks.NewMockDatabase(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)
	mockIPMI := mocks.NewMockIPMI(ctrl)
	mockCDI := mocks.NewMockCDI(ctrl)

	// Setup
	cleanup := setEnvForMainTest(t, setupValidMainEnvVars())
	defer cleanup()

	// Replace factory functions with mock instances
	orgCreateAnsibleInstance := factory.CreateAnsibleInstance
	factory.CreateAnsibleInstance = func(klog.Logger) interfaces.Ansible {
		return mockAnsible
	}
	orgCreateAPIInstance := factory.CreateAPIInstance
	factory.CreateAPIInstance = func(klog.Logger) interfaces.API {
		return mockAPI
	}
	orgCreateDatabaseInstance := factory.CreateDatabaseInstance
	factory.CreateDatabaseInstance = func(klog.Logger, interfaces.API) interfaces.Database {
		return mockDatabase
	}
	orgCreateLoggingInstance := factory.CreateLoggingInstance
	factory.CreateLoggingInstance = func(klog.Logger) interfaces.Logging {
		return mockLogging
	}
	orgCreateIPMIInstance := factory.CreateIPMIInstance
	factory.CreateIPMIInstance = func(klog.Logger, interfaces.API, interfaces.Logging) interfaces.IPMI {
		return mockIPMI
	}
	orgCreateCDIInstance := factory.CreateCDIInstance
	factory.CreateCDIInstance = func(klog.Logger, interfaces.Ansible, interfaces.Logging) interfaces.CDI {
		return mockCDI
	}
	defer func() {
		factory.CreateAnsibleInstance = orgCreateAnsibleInstance
		factory.CreateAPIInstance = orgCreateAPIInstance
		factory.CreateDatabaseInstance = orgCreateDatabaseInstance
		factory.CreateLoggingInstance = orgCreateLoggingInstance
		factory.CreateIPMIInstance = orgCreateIPMIInstance
		factory.CreateCDIInstance = orgCreateCDIInstance
	}()

	// test data
	testCDITargetList := []interfaces.CDITargetList{
		{CDIHost: "192.168.1.1", ProductInfo: `{"product_type":"pg-cdi-1.1"}`, ExtraParameters: `{"cdi_guest":"192.168.10.1"}`},
		{CDIHost: "192.168.1.2", ProductInfo: `{"product_type":"pg-cdi-1.1"}`, ExtraParameters: `{"cdi_guest":"192.168.10.2"}`},
		{CDIHost: "192.168.1.3", ProductInfo: `{"product_type":"pg-cdi-1.1"}`, ExtraParameters: `{"cdi_guest":"192.168.10.3"}`},
	}
	testServerTargetList := []interfaces.IPMITargetList{
		{ServerID: "Server-1", IPMIAddress: "10.10.10.1", IPMIUser: "IPMIUser1", IPMIPassword: "pass1", ProductInfo: `{"product_type":"cots"}`, ExtraParameters: "{}"},
		{ServerID: "Server-2", IPMIAddress: "10.10.10.2", IPMIUser: "IPMIUser2", IPMIPassword: "pass2", ProductInfo: `{"product_type":"cots"}`, ExtraParameters: "{}"},
		{ServerID: "Server-3", IPMIAddress: "10.10.10.3", IPMIUser: "IPMIUser3", IPMIPassword: "pass3", ProductInfo: `{"product_type":"cots"}`, ExtraParameters: "{}"},
		{ServerID: "Server-4", IPMIAddress: "10.10.10.4", IPMIUser: "IPMIUser4", IPMIPassword: "pass4", ProductInfo: `{"product_type":"cots"}`, ExtraParameters: "{}"},
		{ServerID: "Server-5", IPMIAddress: "10.10.10.5", IPMIUser: "IPMIUser5", IPMIPassword: "pass5", ProductInfo: `{"product_type":"cots"}`, ExtraParameters: "{}"},
	}

	// Expectations for mock calls
	mockDatabase.EXPECT().Init().Return(nil).AnyTimes()
	mockCDI.EXPECT().Init().Return(nil).AnyTimes()
	mockIPMI.EXPECT().Init().Return(nil).AnyTimes()
	mockLogging.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
	mockDatabase.EXPECT().SelectCDITable().Return(testCDITargetList, nil).AnyTimes()
	mockDatabase.EXPECT().SelectServerTable().Return(testServerTargetList, nil).AnyTimes()
	mockDatabase.EXPECT().SelectMaasTable().Return([]interfaces.MaasServerTargetList{}, nil).AnyTimes()
	mockCDI.EXPECT().Collection(testCDITargetList).AnyTimes()
	mockIPMI.EXPECT().Collection(testServerTargetList).AnyTimes()

	// Run the main function in a goroutine
	running = true
	go main()

	// Give the server some time to start
	time.Sleep(10 * time.Second)
}

// Test_Main_SelectCDITableError tests SelectCDITable failure handling
func Test_Main_SelectCDITableError(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock instances
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockAPI := mocks.NewMockAPI(ctrl)
	mockDatabase := mocks.NewMockDatabase(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)
	mockIPMI := mocks.NewMockIPMI(ctrl)
	mockCDI := mocks.NewMockCDI(ctrl)

	// Setup
	cleanup := setEnvForMainTest(t, setupValidMainEnvVars())
	defer cleanup()

	// Replace factory functions with mock instances
	orgCreateAnsibleInstance := factory.CreateAnsibleInstance
	factory.CreateAnsibleInstance = func(klog.Logger) interfaces.Ansible {
		return mockAnsible
	}
	orgCreateAPIInstance := factory.CreateAPIInstance
	factory.CreateAPIInstance = func(klog.Logger) interfaces.API {
		return mockAPI
	}
	orgCreateDatabaseInstance := factory.CreateDatabaseInstance
	factory.CreateDatabaseInstance = func(klog.Logger, interfaces.API) interfaces.Database {
		return mockDatabase
	}
	orgCreateLoggingInstance := factory.CreateLoggingInstance
	factory.CreateLoggingInstance = func(klog.Logger) interfaces.Logging {
		return mockLogging
	}
	orgCreateIPMIInstance := factory.CreateIPMIInstance
	factory.CreateIPMIInstance = func(klog.Logger, interfaces.API, interfaces.Logging) interfaces.IPMI {
		return mockIPMI
	}
	orgCreateCDIInstance := factory.CreateCDIInstance
	factory.CreateCDIInstance = func(klog.Logger, interfaces.Ansible, interfaces.Logging) interfaces.CDI {
		return mockCDI
	}
	defer func() {
		factory.CreateAnsibleInstance = orgCreateAnsibleInstance
		factory.CreateAPIInstance = orgCreateAPIInstance
		factory.CreateDatabaseInstance = orgCreateDatabaseInstance
		factory.CreateLoggingInstance = orgCreateLoggingInstance
		factory.CreateIPMIInstance = orgCreateIPMIInstance
		factory.CreateCDIInstance = orgCreateCDIInstance
	}()

	// test data
	testServerTargetList := []interfaces.IPMITargetList{
		{ServerID: "Server-1", IPMIAddress: "10.10.10.1", IPMIUser: "IPMIUser1", IPMIPassword: "pass1"},
	}

	// Expectations for mock calls - SelectCDITable fails
	mockDatabase.EXPECT().Init().Return(nil).AnyTimes()
	mockCDI.EXPECT().Init().Return(nil).AnyTimes()
	mockIPMI.EXPECT().Init().Return(nil).AnyTimes()
	mockLogging.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
	mockDatabase.EXPECT().SelectCDITable().Return(nil, errors.New("CDI table selection failed")).AnyTimes()
	mockDatabase.EXPECT().SelectServerTable().Return(testServerTargetList, nil).AnyTimes()
	mockDatabase.EXPECT().SelectMaasTable().Return([]interfaces.MaasServerTargetList{}, nil).AnyTimes()
	mockIPMI.EXPECT().Collection(testServerTargetList).AnyTimes()
	mockCDI.EXPECT().Collection(gomock.Any()).AnyTimes()

	// Run the main function in a goroutine
	running = true
	go main()

	// Give the server some time to start
	time.Sleep(10 * time.Second)

	// Stop the monitoring goroutine
	running = false
}

// Test_Main_SelectServerTableError tests SelectServerTable failure handling
func Test_Main_SelectServerTableError(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock instances
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockAPI := mocks.NewMockAPI(ctrl)
	mockDatabase := mocks.NewMockDatabase(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)
	mockIPMI := mocks.NewMockIPMI(ctrl)
	mockCDI := mocks.NewMockCDI(ctrl)

	// Setup
	cleanup := setEnvForMainTest(t, setupValidMainEnvVars())
	defer cleanup()

	// Replace factory functions with mock instances
	orgCreateAnsibleInstance := factory.CreateAnsibleInstance
	factory.CreateAnsibleInstance = func(klog.Logger) interfaces.Ansible {
		return mockAnsible
	}
	orgCreateAPIInstance := factory.CreateAPIInstance
	factory.CreateAPIInstance = func(klog.Logger) interfaces.API {
		return mockAPI
	}
	orgCreateDatabaseInstance := factory.CreateDatabaseInstance
	factory.CreateDatabaseInstance = func(klog.Logger, interfaces.API) interfaces.Database {
		return mockDatabase
	}
	orgCreateLoggingInstance := factory.CreateLoggingInstance
	factory.CreateLoggingInstance = func(klog.Logger) interfaces.Logging {
		return mockLogging
	}
	orgCreateIPMIInstance := factory.CreateIPMIInstance
	factory.CreateIPMIInstance = func(klog.Logger, interfaces.API, interfaces.Logging) interfaces.IPMI {
		return mockIPMI
	}
	orgCreateCDIInstance := factory.CreateCDIInstance
	factory.CreateCDIInstance = func(klog.Logger, interfaces.Ansible, interfaces.Logging) interfaces.CDI {
		return mockCDI
	}
	defer func() {
		factory.CreateAnsibleInstance = orgCreateAnsibleInstance
		factory.CreateAPIInstance = orgCreateAPIInstance
		factory.CreateDatabaseInstance = orgCreateDatabaseInstance
		factory.CreateLoggingInstance = orgCreateLoggingInstance
		factory.CreateIPMIInstance = orgCreateIPMIInstance
		factory.CreateCDIInstance = orgCreateCDIInstance
	}()

	// test data
	testCDITargetList := []interfaces.CDITargetList{
		{CDIHost: "192.168.1.1", ProductInfo: `{"product_type":"pg-cdi-1.1"}`, ExtraParameters: `{"cdi_guest":"192.168.10.1"}`},
	}

	// Expectations for mock calls - SelectServerTable fails
	mockDatabase.EXPECT().Init().Return(nil).AnyTimes()
	mockCDI.EXPECT().Init().Return(nil).AnyTimes()
	mockIPMI.EXPECT().Init().Return(nil).AnyTimes()
	mockLogging.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
	mockDatabase.EXPECT().SelectCDITable().Return(testCDITargetList, nil).AnyTimes()
	mockDatabase.EXPECT().SelectServerTable().Return(nil, errors.New("server table selection failed")).AnyTimes()
	mockDatabase.EXPECT().SelectMaasTable().Return([]interfaces.MaasServerTargetList{}, nil).AnyTimes()
	mockCDI.EXPECT().Collection(testCDITargetList).AnyTimes()
	mockIPMI.EXPECT().Collection(gomock.Any()).AnyTimes()

	// Run the main function in a goroutine
	running = true
	go main()

	// Give the server some time to start
	time.Sleep(10 * time.Second)

	// Stop the monitoring goroutine
	running = false
}
