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

package factory

import (
	"testing"

	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	"log_module/internal/server/interfaces"
	"log_module/internal/server/interfaces/mocks"
	"log_module/internal/server/test_utils"
)

// TestCreateDatabaseInstance_ValidLogger_ReturnsInstance tests database instance creation
func TestCreateDatabaseInstance_ValidLogger_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	logger := klog.NewKlogr()
	mockAPI := mocks.NewMockAPI(ctrl)

	// Execute
	instance := CreateDatabaseInstance(logger, mockAPI)

	// Verify
	assert.NotNil(t, instance)
	assert.Implements(t, (*interfaces.Database)(nil), instance)
}

// TestCreateDatabaseInstance_FunctionType_MatchesSignature tests function type
func TestCreateDatabaseInstance_FunctionType_MatchesSignature(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Verify that CreateDatabaseInstance has correct type
	var creator DatabaseInstanceCreator = CreateDatabaseInstance
	assert.NotNil(t, creator)
}

// TestCreateIPMIInstance_ValidDependencies_ReturnsInstance tests IPMI instance creation
func TestCreateIPMIInstance_ValidDependencies_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Execute
	instance := CreateIPMIInstance(logger, mockAPI, mockLogging)

	// Verify
	assert.NotNil(t, instance)
	assert.Implements(t, (*interfaces.IPMI)(nil), instance)
}

// TestCreateIPMIInstance_FunctionType_MatchesSignature tests function type
func TestCreateIPMIInstance_FunctionType_MatchesSignature(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Verify that CreateIPMIInstance has correct type
	var creator IPMIInstanceCreator = CreateIPMIInstance
	assert.NotNil(t, creator)
}

// TestCreateCDIInstance_ValidDependencies_ReturnsInstance tests CDI instance creation
func TestCreateCDIInstance_ValidDependencies_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Execute
	instance := CreateCDIInstance(logger, mockAnsible, mockLogging)

	// Verify
	assert.NotNil(t, instance)
	assert.Implements(t, (*interfaces.CDI)(nil), instance)
}

// TestCreateCDIInstance_FunctionType_MatchesSignature tests function type
func TestCreateCDIInstance_FunctionType_MatchesSignature(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Verify that CreateCDIInstance has correct type
	var creator CDIInstanceCreator = CreateCDIInstance
	assert.NotNil(t, creator)
}

// TestCreateCDISoftInstance_ValidDependencies_ReturnsInstance tests CDISoft instance creation
func TestCreateCDISoftInstance_ValidDependencies_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Execute
	instance := CreateCDISoftInstance(logger, mockAnsible, mockLogging)

	// Verify
	assert.NotNil(t, instance)
	assert.Implements(t, (*interfaces.CDI)(nil), instance)
}

// TestCreateMaasServerInstance_ValidDependencies_ReturnsInstance tests MaasServer instance creation
func TestCreateMaasServerInstance_ValidDependencies_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Execute
	instance := CreateMaasServerInstance(logger, mockAPI, mockLogging)

	// Verify
	assert.NotNil(t, instance)
	assert.Implements(t, (*interfaces.MaasServer)(nil), instance)
}

// TestCreateAnsibleInstance_ValidLogger_ReturnsInstance tests Ansible instance creation
func TestCreateAnsibleInstance_ValidLogger_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute
	instance := CreateAnsibleInstance(logger)

	// Verify
	assert.NotNil(t, instance)
	assert.Implements(t, (*interfaces.Ansible)(nil), instance)
}

// TestCreateAnsibleInstance_FunctionType_MatchesSignature tests function type
func TestCreateAnsibleInstance_FunctionType_MatchesSignature(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Verify that CreateAnsibleInstance has correct type
	var creator AnsibleInstanceCreator = CreateAnsibleInstance
	assert.NotNil(t, creator)
}

// TestCreateAPIInstance_ValidLogger_ReturnsInstance tests API instance creation
func TestCreateAPIInstance_ValidLogger_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute
	instance := CreateAPIInstance(logger)

	// Verify
	assert.NotNil(t, instance)
	assert.Implements(t, (*interfaces.API)(nil), instance)
}

// TestCreateAPIInstance_FunctionType_MatchesSignature tests function type
func TestCreateAPIInstance_FunctionType_MatchesSignature(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Verify that CreateAPIInstance has correct type
	var creator APIInstanceCreator = CreateAPIInstance
	assert.NotNil(t, creator)
}

// TestCreateLoggingInstance_ValidLogger_ReturnsInstance tests Logging instance creation
func TestCreateLoggingInstance_ValidLogger_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute
	instance := CreateLoggingInstance(logger)

	// Verify
	assert.NotNil(t, instance)
	assert.Implements(t, (*interfaces.Logging)(nil), instance)
}

// TestCreateLoggingInstance_FunctionType_MatchesSignature tests function type
func TestCreateLoggingInstance_FunctionType_MatchesSignature(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Verify that CreateLoggingInstance has correct type
	var creator LoggingInstanceCreator = CreateLoggingInstance
	assert.NotNil(t, creator)
}

// TestFactoryFunctions_CanBeReplacedForTesting tests factory function replacement
func TestFactoryFunctions_CanBeReplacedForTesting(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - save original functions
	orgCreateDatabaseInstance := CreateDatabaseInstance
	orgCreateAPIInstance := CreateAPIInstance
	orgCreateLoggingInstance := CreateLoggingInstance
	orgCreateAnsibleInstance := CreateAnsibleInstance
	orgCreateIPMIInstance := CreateIPMIInstance
	orgCreateCDIInstance := CreateCDIInstance

	defer func() {
		// Restore original functions
		CreateDatabaseInstance = orgCreateDatabaseInstance
		CreateAPIInstance = orgCreateAPIInstance
		CreateLoggingInstance = orgCreateLoggingInstance
		CreateAnsibleInstance = orgCreateAnsibleInstance
		CreateIPMIInstance = orgCreateIPMIInstance
		CreateCDIInstance = orgCreateCDIInstance
	}()

	// Setup mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDatabase := mocks.NewMockDatabase(ctrl)
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)
	mockAnsible := mocks.NewMockAnsible(ctrl)
	mockIPMI := mocks.NewMockIPMI(ctrl)
	mockCDI := mocks.NewMockCDI(ctrl)

	// Replace factory functions
	CreateDatabaseInstance = func(klog.Logger, interfaces.API) interfaces.Database { return mockDatabase }
	CreateAPIInstance = func(klog.Logger) interfaces.API { return mockAPI }
	CreateLoggingInstance = func(klog.Logger) interfaces.Logging { return mockLogging }
	CreateAnsibleInstance = func(klog.Logger) interfaces.Ansible { return mockAnsible }
	CreateIPMIInstance = func(klog.Logger, interfaces.API, interfaces.Logging) interfaces.IPMI { return mockIPMI }
	CreateCDIInstance = func(klog.Logger, interfaces.Ansible, interfaces.Logging) interfaces.CDI { return mockCDI }

	logger := klog.NewKlogr()

	// Execute
	dbInstance := CreateDatabaseInstance(logger, mockAPI)
	apiInstance := CreateAPIInstance(logger)
	loggingInstance := CreateLoggingInstance(logger)
	ansibleInstance := CreateAnsibleInstance(logger)
	ipmiInstance := CreateIPMIInstance(logger, mockAPI, mockLogging)
	cdiInstance := CreateCDIInstance(logger, mockAnsible, mockLogging)

	// Verify - all instances should be mocks
	assert.Equal(t, mockDatabase, dbInstance)
	assert.Equal(t, mockAPI, apiInstance)
	assert.Equal(t, mockLogging, loggingInstance)
	assert.Equal(t, mockAnsible, ansibleInstance)
	assert.Equal(t, mockIPMI, ipmiInstance)
	assert.Equal(t, mockCDI, cdiInstance)
}

// TestFactoryFunctions_MultipleCallsSameLogger_ReturnsDifferentInstances tests independence
func TestFactoryFunctions_MultipleCallsSameLogger_ReturnsDifferentInstances(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	logger := klog.NewKlogr()
	mockAPI := mocks.NewMockAPI(ctrl)

	// Execute - create multiple instances
	db1 := CreateDatabaseInstance(logger, mockAPI)
	db2 := CreateDatabaseInstance(logger, mockAPI)
	api1 := CreateAPIInstance(logger)
	api2 := CreateAPIInstance(logger)
	logging1 := CreateLoggingInstance(logger)
	logging2 := CreateLoggingInstance(logger)
	ansible1 := CreateAnsibleInstance(logger)
	ansible2 := CreateAnsibleInstance(logger)

	// Verify - each call should return a new instance
	assert.NotSame(t, db1, db2)
	assert.NotSame(t, api1, api2)
	assert.NotSame(t, logging1, logging2)
	assert.NotSame(t, ansible1, ansible2)

	// But they should all implement the correct interfaces
	assert.Implements(t, (*interfaces.Database)(nil), db1)
	assert.Implements(t, (*interfaces.Database)(nil), db2)
	assert.Implements(t, (*interfaces.API)(nil), api1)
	assert.Implements(t, (*interfaces.API)(nil), api2)
	assert.Implements(t, (*interfaces.Logging)(nil), logging1)
	assert.Implements(t, (*interfaces.Logging)(nil), logging2)
	assert.Implements(t, (*interfaces.Ansible)(nil), ansible1)
	assert.Implements(t, (*interfaces.Ansible)(nil), ansible2)
}
