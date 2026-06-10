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

	"exporter_module/internal/server/implementation"
	"exporter_module/internal/server/interfaces"
	"exporter_module/internal/server/test_utils"

	"k8s.io/klog/v2"
)

// TestCreateDatabaseInstance_ValidLogger_ReturnsValidInstance tests database instance creation
func TestCreateDatabaseInstance_ValidLogger_ReturnsValidInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute
	instance := CreateDatabaseInstance(logger, nil)

	// Verify
	if instance == nil {
		t.Fatal("Expected database instance, got nil")
	}

	// Verify it's the correct type
	dbImpl, ok := instance.(*implementation.DatabaseImplement)
	if !ok {
		t.Errorf("Expected *implementation.DatabaseImplement, got %T", instance)
	}

	// Verify logger is set
	if dbImpl.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	// Verify it implements the interface
	var _ interfaces.Database = instance
}

// TestCreateServerInstance_ValidParameters_ReturnsValidInstance tests server instance creation
func TestCreateServerInstance_ValidParameters_ReturnsValidInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := CreateAnsibleInstance(logger)
	api := CreateAPIInstance(logger)
	metrics := CreateMetricsInstance(logger)

	// Execute
	instance := CreateServerInstance(logger, ansible, api, metrics, nil)

	// Verify
	if instance == nil {
		t.Fatal("Expected server instance, got nil")
	}

	// Verify it's the correct type
	serverImpl, ok := instance.(*implementation.ServerImplement)
	if !ok {
		t.Errorf("Expected *implementation.ServerImplement, got %T", instance)
	}

	// Verify all dependencies are set
	if serverImpl.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}
	if serverImpl.Ansible != ansible {
		t.Error("Expected ansible to be set correctly")
	}
	if serverImpl.API != api {
		t.Error("Expected api to be set correctly")
	}
	if serverImpl.Metrics != metrics {
		t.Error("Expected metrics to be set correctly")
	}

	// Verify it implements the interface
	var _ interfaces.Server = instance
}

// TestCreateServerInstance_NilDependencies_ReturnsInstanceWithNilFields tests server creation with nil dependencies
func TestCreateServerInstance_NilDependencies_ReturnsInstanceWithNilFields(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute with nil dependencies
	instance := CreateServerInstance(logger, nil, nil, nil, nil)

	// Verify
	if instance == nil {
		t.Fatal("Expected server instance, got nil")
	}

	serverImpl, ok := instance.(*implementation.ServerImplement)
	if !ok {
		t.Errorf("Expected *implementation.ServerImplement, got %T", instance)
	}

	// Verify logger is set but dependencies are nil
	if serverImpl.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}
	if serverImpl.Ansible != nil {
		t.Error("Expected ansible to be nil")
	}
	if serverImpl.API != nil {
		t.Error("Expected api to be nil")
	}
	if serverImpl.Metrics != nil {
		t.Error("Expected metrics to be nil")
	}
}

// TestCreateNetworkInstance_ValidParameters_ReturnsValidInstance tests network instance creation
func TestCreateNetworkInstance_ValidParameters_ReturnsValidInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := CreateAnsibleInstance(logger)
	metrics := CreateMetricsInstance(logger)

	// Execute
	instance := CreateNetworkInstance(logger, ansible, metrics, nil)

	// Verify
	if instance == nil {
		t.Fatal("Expected network instance, got nil")
	}

	// Verify it's the correct type
	networkImpl, ok := instance.(*implementation.NetworkImplement)
	if !ok {
		t.Errorf("Expected *implementation.NetworkImplement, got %T", instance)
	}

	// Verify all dependencies are set
	if networkImpl.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}
	if networkImpl.Ansible != ansible {
		t.Error("Expected ansible to be set correctly")
	}
	if networkImpl.Metrics != metrics {
		t.Error("Expected metrics to be set correctly")
	}

	// Verify it implements the interface
	var _ interfaces.Network = instance
}

// TestCreateNetworkInstance_NilDependencies_ReturnsInstanceWithNilFields tests network creation with nil dependencies
func TestCreateNetworkInstance_NilDependencies_ReturnsInstanceWithNilFields(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute with nil dependencies
	instance := CreateNetworkInstance(logger, nil, nil, nil)

	// Verify
	if instance == nil {
		t.Fatal("Expected network instance, got nil")
	}

	networkImpl, ok := instance.(*implementation.NetworkImplement)
	if !ok {
		t.Errorf("Expected *implementation.NetworkImplement, got %T", instance)
	}

	// Verify logger is set but dependencies are nil
	if networkImpl.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}
	if networkImpl.Ansible != nil {
		t.Error("Expected ansible to be nil")
	}
	if networkImpl.Metrics != nil {
		t.Error("Expected metrics to be nil")
	}
}

// TestCreateMetricsInstance_ValidLogger_ReturnsValidInstance tests metrics instance creation
func TestCreateMetricsInstance_ValidLogger_ReturnsValidInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute
	instance := CreateMetricsInstance(logger)

	// Verify
	if instance == nil {
		t.Fatal("Expected metrics instance, got nil")
	}

	// Verify it's the correct type
	metricsImpl, ok := instance.(*implementation.MetricsImplement)
	if !ok {
		t.Errorf("Expected *implementation.MetricsImplement, got %T", instance)
	}

	// Verify logger is set
	if metricsImpl.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	// Verify it implements the interface
	var _ interfaces.Metrics = instance
}

// TestCreateAnsibleInstance_ValidLogger_ReturnsValidInstance tests ansible instance creation
func TestCreateAnsibleInstance_ValidLogger_ReturnsValidInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute
	instance := CreateAnsibleInstance(logger)

	// Verify
	if instance == nil {
		t.Fatal("Expected ansible instance, got nil")
	}

	// Verify it's the correct type
	ansibleImpl, ok := instance.(*implementation.AnsibleImplement)
	if !ok {
		t.Errorf("Expected *implementation.AnsibleImplement, got %T", instance)
	}

	// Verify logger is set
	if ansibleImpl.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	// Verify it implements the interface
	var _ interfaces.Ansible = instance
}

// TestCreateAPIInstance_ValidLogger_ReturnsValidInstance tests API instance creation
func TestCreateAPIInstance_ValidLogger_ReturnsValidInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute
	instance := CreateAPIInstance(logger)

	// Verify
	if instance == nil {
		t.Fatal("Expected api instance, got nil")
	}

	// Verify it's the correct type
	apiImpl, ok := instance.(*implementation.APIImplement)
	if !ok {
		t.Errorf("Expected *implementation.APIImplement, got %T", instance)
	}

	// Verify logger is set
	if apiImpl.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	// Verify it implements the interface
	var _ interfaces.API = instance
}

// TestFactoryVariableModification_CreateDatabaseInstance_UsesModifiedFunction tests factory variable modification
func TestFactoryVariableModification_CreateDatabaseInstance_UsesModifiedFunction(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - backup original function
	originalCreator := CreateDatabaseInstance
	defer func() { CreateDatabaseInstance = originalCreator }()

	// Modify factory function
	customCreatorCalled := false
	CreateDatabaseInstance = func(logger klog.Logger, api interfaces.API) interfaces.Database {
		customCreatorCalled = true
		return &implementation.DatabaseImplement{Logger: logger, API: api}
	}

	// Execute
	logger := klog.NewKlogr()
	instance := CreateDatabaseInstance(logger, nil)

	// Verify
	if !customCreatorCalled {
		t.Error("Expected custom creator to be called")
	}
	if instance == nil {
		t.Fatal("Expected database instance, got nil")
	}
}

// TestFactoryVariableModification_CreateServerInstance_UsesModifiedFunction tests server factory modification
func TestFactoryVariableModification_CreateServerInstance_UsesModifiedFunction(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - backup original function
	originalCreator := CreateServerInstance
	defer func() { CreateServerInstance = originalCreator }()

	// Modify factory function
	customCreatorCalled := false
	CreateServerInstance = func(logger klog.Logger, ansible interfaces.Ansible, api interfaces.API, metrics interfaces.Metrics, manager interfaces.Manager) interfaces.Server {
		customCreatorCalled = true
		return &implementation.ServerImplement{Logger: logger, Ansible: ansible, API: api, Metrics: metrics, Manager: manager}
	}

	// Execute
	logger := klog.NewKlogr()
	instance := CreateServerInstance(logger, nil, nil, nil, nil)

	// Verify
	if !customCreatorCalled {
		t.Error("Expected custom creator to be called")
	}
	if instance == nil {
		t.Fatal("Expected server instance, got nil")
	}
}

// TestFactoryVariableModification_CreateNetworkInstance_UsesModifiedFunction tests network factory modification
func TestFactoryVariableModification_CreateNetworkInstance_UsesModifiedFunction(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - backup original function
	originalCreator := CreateNetworkInstance
	defer func() { CreateNetworkInstance = originalCreator }()

	// Modify factory function
	customCreatorCalled := false
	CreateNetworkInstance = func(logger klog.Logger, ansible interfaces.Ansible, metrics interfaces.Metrics, manager interfaces.Manager) interfaces.Network {
		customCreatorCalled = true
		return &implementation.NetworkImplement{Logger: logger, Ansible: ansible, Metrics: metrics, Manager: manager}
	}

	// Execute
	logger := klog.NewKlogr()
	instance := CreateNetworkInstance(logger, nil, nil, nil)

	// Verify
	if !customCreatorCalled {
		t.Error("Expected custom creator to be called")
	}
	if instance == nil {
		t.Fatal("Expected network instance, got nil")
	}
}

// TestFactoryVariableModification_CreateMetricsInstance_UsesModifiedFunction tests metrics factory modification
func TestFactoryVariableModification_CreateMetricsInstance_UsesModifiedFunction(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - backup original function
	originalCreator := CreateMetricsInstance
	defer func() { CreateMetricsInstance = originalCreator }()

	// Modify factory function
	customCreatorCalled := false
	CreateMetricsInstance = func(logger klog.Logger) interfaces.Metrics {
		customCreatorCalled = true
		return &implementation.MetricsImplement{Logger: logger}
	}

	// Execute
	logger := klog.NewKlogr()
	instance := CreateMetricsInstance(logger)

	// Verify
	if !customCreatorCalled {
		t.Error("Expected custom creator to be called")
	}
	if instance == nil {
		t.Fatal("Expected metrics instance, got nil")
	}
}

// TestFactoryVariableModification_CreateAnsibleInstance_UsesModifiedFunction tests ansible factory modification
func TestFactoryVariableModification_CreateAnsibleInstance_UsesModifiedFunction(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - backup original function
	originalCreator := CreateAnsibleInstance
	defer func() { CreateAnsibleInstance = originalCreator }()

	// Modify factory function
	customCreatorCalled := false
	CreateAnsibleInstance = func(logger klog.Logger) interfaces.Ansible {
		customCreatorCalled = true
		return &implementation.AnsibleImplement{Logger: logger}
	}

	// Execute
	logger := klog.NewKlogr()
	instance := CreateAnsibleInstance(logger)

	// Verify
	if !customCreatorCalled {
		t.Error("Expected custom creator to be called")
	}
	if instance == nil {
		t.Fatal("Expected ansible instance, got nil")
	}
}

// TestFactoryVariableModification_CreateAPIInstance_UsesModifiedFunction tests API factory modification
func TestFactoryVariableModification_CreateAPIInstance_UsesModifiedFunction(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - backup original function
	originalCreator := CreateAPIInstance
	defer func() { CreateAPIInstance = originalCreator }()

	// Modify factory function
	customCreatorCalled := false
	CreateAPIInstance = func(logger klog.Logger) interfaces.API {
		customCreatorCalled = true
		return &implementation.APIImplement{Logger: logger}
	}

	// Execute
	logger := klog.NewKlogr()
	instance := CreateAPIInstance(logger)

	// Verify
	if !customCreatorCalled {
		t.Error("Expected custom creator to be called")
	}
	if instance == nil {
		t.Fatal("Expected api instance, got nil")
	}
}

// TestAllFactories_IntegrationTest_CreateCompleteObjectGraph tests creating a complete object graph
func TestAllFactories_IntegrationTest_CreateCompleteObjectGraph(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute - create all instances
	metrics := CreateMetricsInstance(logger)
	ansible := CreateAnsibleInstance(logger)
	api := CreateAPIInstance(logger)
	database := CreateDatabaseInstance(logger, api)
	manager := CreateManagerInstance(logger, ansible, metrics)
	server := CreateServerInstance(logger, ansible, api, metrics, manager)
	network := CreateNetworkInstance(logger, ansible, metrics, manager)

	// Verify all instances are created
	if database == nil {
		t.Error("Expected database instance")
	}
	if metrics == nil {
		t.Error("Expected metrics instance")
	}
	if ansible == nil {
		t.Error("Expected ansible instance")
	}
	if api == nil {
		t.Error("Expected api instance")
	}
	if server == nil {
		t.Error("Expected server instance")
	}
	if network == nil {
		t.Error("Expected network instance")
	}

	// Verify dependency injection worked correctly
	serverImpl := server.(*implementation.ServerImplement)
	networkImpl := network.(*implementation.NetworkImplement)

	if serverImpl.Ansible != ansible {
		t.Error("Expected server to have correct ansible dependency")
	}
	if serverImpl.API != api {
		t.Error("Expected server to have correct api dependency")
	}
	if serverImpl.Metrics != metrics {
		t.Error("Expected server to have correct metrics dependency")
	}

	if networkImpl.Ansible != ansible {
		t.Error("Expected network to have correct ansible dependency")
	}
	if networkImpl.Metrics != metrics {
		t.Error("Expected network to have correct metrics dependency")
	}
}

// TestFactoryTypedefs_CorrectSignatures_MatchImplementations tests that type definitions match implementations
func TestFactoryTypedefs_CorrectSignatures_MatchImplementations(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Test that the function variables match their type definitions
	var dbCreator DatabaseInstanceCreator = CreateDatabaseInstance
	var serverCreator ServerInstanceCreator = CreateServerInstance
	var networkCreator NetworkInstanceCreator = CreateNetworkInstance
	var metricsCreator MetricsInstanceCreator = CreateMetricsInstance
	var ansibleCreator AnsibleInstanceCreator = CreateAnsibleInstance
	var apiCreator APIInstanceCreator = CreateAPIInstance
	var managerCreator ManagerInstanceCreator = CreateManagerInstance

	// Execute to verify they work
	metrics := metricsCreator(logger)
	ansible := ansibleCreator(logger)
	api := apiCreator(logger)
	database := dbCreator(logger, api)
	manager := managerCreator(logger, ansible, metrics)
	server := serverCreator(logger, ansible, api, metrics, manager)
	network := networkCreator(logger, ansible, metrics, manager)

	// Verify
	if database == nil || metrics == nil || ansible == nil || api == nil || server == nil || network == nil {
		t.Error("One or more factory creators failed to create instances")
	}
}
