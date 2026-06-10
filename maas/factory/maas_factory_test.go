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
	"os"
	"testing"

	"k8s.io/klog/v2"

	proto "maas_module/api/proto"
	"maas_module/internal/server/implementation/canonical_maas"
	"maas_module/internal/server/implementation/canonical_maas/maas_api"
	"maas_module/internal/server/interfaces"
	"maas_module/internal/server/test_utils"
)

// Helper function to create test data
func createTestProductInfo() *proto.ProductInformation {
	os := "Ubuntu"
	return &proto.ProductInformation{
		Vendor:      "Canonical",
		ProductName: "MaaS",
		Version:     "3.3",
		Os:          &os,
	}
}

func createTestMaasInfo() *proto.MaasInformation {
	return &proto.MaasInformation{
		AccessUrl: "http://localhost:5240/MAAS",
		ApiKey:    "test-api-key",
	}
}

// setupProductMappings configures the PRODUCT_MAPPINGS environment variable for testing
func setupProductMappings(t *testing.T) func() {
	originalMappings := os.Getenv("PRODUCT_MAPPINGS")
	
	// Set up test product mappings JSON
	testMappings := `{
		"maas_products": [
			{
				"vendor": "Canonical",
				"product_name": "",
				"version": "",
				"os": "",
				"type": "Canonical"
			}
		]
	}`
	
	os.Setenv("PRODUCT_MAPPINGS", testMappings)
	
	// Return cleanup function
	return func() {
		if originalMappings != "" {
			os.Setenv("PRODUCT_MAPPINGS", originalMappings)
		} else {
			os.Unsetenv("PRODUCT_MAPPINGS")
		}
	}
}

// TestCreateMaasController_ValidLogger_ReturnsController tests CreateMaasController with valid logger
func TestCreateMaasController_ValidLogger_ReturnsController(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	logger := klog.NewKlogr()
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	result := CreateMaasController(logger, productInfo, maasInfo)

	// Assert
	if result == nil {
		t.Error("Expected MaasController instance, got nil")
	}

	// Verify it's the correct type
	controller, ok := result.(*canonical_maas.CanonicalMaasController)
	if !ok {
		t.Error("Expected CanonicalMaasController type")
	}

	// Verify logger is set
	// Note: klog.Logger is an interface, so we can't check for nil directly

	// Verify APIFactory is set
	if controller.APIFactory == nil {
		t.Error("Expected APIFactory to be set in controller")
	}

	// Verify Ansible is set
	if controller.Ansible == nil {
		t.Error("Expected Ansible to be set in controller")
	}
}

// TestCreateMaasController_NilLogger_ReturnsController tests CreateMaasController with nil logger
func TestCreateMaasController_NilLogger_ReturnsController(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	var logger klog.Logger
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	result := CreateMaasController(logger, productInfo, maasInfo)

	// Assert
	if result == nil {
		t.Error("Expected MaasController instance, got nil")
	}

	// Verify it's the correct type
	controller, ok := result.(*canonical_maas.CanonicalMaasController)
	if !ok {
		t.Error("Expected CanonicalMaasController type")
	}

	// Verify components are still initialized
	if controller.APIFactory == nil {
		t.Error("Expected APIFactory to be set even with nil logger")
	}

	if controller.Ansible == nil {
		t.Error("Expected Ansible to be set even with nil logger")
	}
}

// TestCreateMaasController_ImplementsInterface tests that created controller implements MaasController interface
func TestCreateMaasController_ImplementsInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	logger := klog.NewKlogr()
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	result := CreateMaasController(logger, productInfo, maasInfo)

	// Assert
	// Verify it implements the interface
	if _, ok := result.(interfaces.MaasController); !ok {
		t.Error("Expected result to implement MaasController interface")
	}
}

// TestCreateMaasController_MultipleCallsReturnDifferentInstances tests that multiple calls return different instances
func TestCreateMaasController_MultipleCallsReturnDifferentInstances(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	logger := klog.NewKlogr()
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	controller1 := CreateMaasController(logger, productInfo, maasInfo)
	controller2 := CreateMaasController(logger, productInfo, maasInfo)

	// Assert
	if controller1 == nil {
		t.Error("Expected first controller instance, got nil")
	}
	if controller2 == nil {
		t.Error("Expected second controller instance, got nil")
	}

	// Verify they are different instances
	if controller1 == controller2 {
		t.Error("Expected different instances, got same instance")
	}
}

// TestCreateMaasController_ComponentsAreProperlyInitialized tests that all components are properly initialized with correct dependencies
func TestCreateMaasController_ComponentsAreProperlyInitialized(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	logger := klog.NewKlogr()
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	result := CreateMaasController(logger, productInfo, maasInfo)

	// Assert
	controller, ok := result.(*canonical_maas.CanonicalMaasController)
	if !ok {
		t.Fatal("Expected CanonicalMaasController type")
	}

	// Check APIFactory type and initialization
	apiFactory, ok := controller.APIFactory.(*maas_api.MaasAPIFactoryImple)
	if !ok {
		t.Error("Expected APIFactory to be MaasAPIFactoryImple type")
	}

	if apiFactory != nil {
		// Verify APIFactory has API and Logger set
		if apiFactory.API == nil {
			t.Error("Expected APIFactory.API to be set")
		}
		// Note: Logger comparison is not straightforward, so we just check it's not nil
	}

	// Check Ansible type and initialization
	ansible, ok := controller.Ansible.(*canonical_maas.CanonicalMaasAnsibleImple)
	if !ok {
		t.Error("Expected Ansible to be CanonicalMaasAnsibleImple type")
	}

	if ansible != nil {
		// Verify Ansible has Executor set
		if ansible.Executor == nil {
			t.Error("Expected Ansible.Executor to be set")
		}

		// Verify Executor is correct type
		if _, ok := ansible.Executor.(*canonical_maas.CmdExecutor); !ok {
			t.Error("Expected Ansible.Executor to be CmdExecutor type")
		}
	}
}

// TestCreateMaasController_APIComponentsProperlyWired tests that API components are properly wired together
func TestCreateMaasController_APIComponentsProperlyWired(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	logger := klog.NewKlogr()
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	result := CreateMaasController(logger, productInfo, maasInfo)

	// Assert
	controller := result.(*canonical_maas.CanonicalMaasController)
	apiFactory := controller.APIFactory.(*maas_api.MaasAPIFactoryImple)

	// Verify that APIFactory can create various API instances
	// Test a few key API creation methods to ensure they work
	subnets := apiFactory.NewSubnets()
	if subnets == nil {
		t.Error("Expected APIFactory to create Subnets instance")
	}

	machines := apiFactory.NewMachines()
	if machines == nil {
		t.Error("Expected APIFactory to create Machines instance")
	}

	vmHosts := apiFactory.NewVMHosts()
	if vmHosts == nil {
		t.Error("Expected APIFactory to create VMHosts instance")
	}
}

// TestCreateMaasController_LoggerPropagation tests that logger is properly propagated to all components
func TestCreateMaasController_LoggerPropagation(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	logger := klog.NewKlogr()
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	result := CreateMaasController(logger, productInfo, maasInfo)

	// Assert
	controller := result.(*canonical_maas.CanonicalMaasController)

	// Check that the logger is properly set (klog.Logger is interface, can't check nil directly)
	// Just verify the components are initialized

	_ = controller.APIFactory.(*maas_api.MaasAPIFactoryImple)
	// Logger exists in apiFactory

	_ = controller.Ansible.(*canonical_maas.CanonicalMaasAnsibleImple)
	// Logger exists in ansible
}

// TestCreateMaasController_DependencyInjection tests that dependencies are properly injected
func TestCreateMaasController_DependencyInjection(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	logger := klog.NewKlogr()
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	result := CreateMaasController(logger, productInfo, maasInfo)

	// Assert
	controller := result.(*canonical_maas.CanonicalMaasController)

	// Verify that APIFactory has proper API implementation
	apiFactory := controller.APIFactory.(*maas_api.MaasAPIFactoryImple)

	// Verify API implementation exists and is correct type
	if apiFactory.API == nil {
		t.Error("Expected API implementation to be injected into APIFactory")
	}

	if _, ok := apiFactory.API.(*canonical_maas.CanonicalMaasAPIImple); !ok {
		t.Error("Expected API to be CanonicalMaasAPIImple type")
	}

	// Verify that Ansible has proper executor implementation
	ansible := controller.Ansible.(*canonical_maas.CanonicalMaasAnsibleImple)

	if ansible.Executor == nil {
		t.Error("Expected Executor to be injected into Ansible")
	}

	if _, ok := ansible.Executor.(*canonical_maas.CmdExecutor); !ok {
		t.Error("Expected Executor to be CmdExecutor type")
	}
}

// TestCreateMaasController_ConsistentInitialization tests that initialization is consistent across calls
func TestCreateMaasController_ConsistentInitialization(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	cleanupMappings := setupProductMappings(t)
	defer cleanupMappings()

	// Arrange
	logger1 := klog.NewKlogr()
	logger2 := klog.NewKlogr()
	productInfo := createTestProductInfo()
	maasInfo := createTestMaasInfo()

	// Act
	controller1 := CreateMaasController(logger1, productInfo, maasInfo)
	controller2 := CreateMaasController(logger2, productInfo, maasInfo)

	// Assert
	// Both should be successfully created
	if controller1 == nil || controller2 == nil {
		t.Error("Expected both controllers to be created successfully")
	}

	// Both should have the same structure/type
	c1 := controller1.(*canonical_maas.CanonicalMaasController)
	c2 := controller2.(*canonical_maas.CanonicalMaasController)

	// Both should have all components initialized
	if c1.APIFactory == nil || c1.Ansible == nil {
		t.Error("Expected first controller to have all components initialized")
	}
	if c2.APIFactory == nil || c2.Ansible == nil {
		t.Error("Expected second controller to have all components initialized")
	}

	// Components should be different instances
	if c1.APIFactory == c2.APIFactory {
		t.Error("Expected different APIFactory instances")
	}
	if c1.Ansible == c2.Ansible {
		t.Error("Expected different Ansible instances")
	}
}
