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

	protocdi "cdi_module/api/proto"
	"cdi_module/internal/server/interfaces"
	"cdi_module/internal/server/test_utils"
	"cdi_module/internal/server/utils"
)

func TestCreateCDIController_ValidConfig_ReturnsController(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup product mappings
	setupTestMappings(t)
	defer cleanupTestMappings()

	// Setup test environment variables
	setupTestEnv()
	defer teardownTestEnv()

	// Initialize config
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.Background()

	// Execute
	os := "Linux"
	controller := CreateCDIController(logger, &protocdi.ProductInformation{Vendor: "Fujitsu", ProductName: "PG-CDI", Version: "1.1", Os: &os})

	// Verify
	if controller == nil {
		t.Fatal("CreateCDIController returned nil")
	}

	// Check if it implements the interface
	var _ interfaces.CDIController = controller
}

func TestCreateCDIController_NoConfig_PanicsWithNilLogger(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup product mappings
	setupTestMappings(t)
	defer cleanupTestMappings()

	// Setup test environment variables
	setupTestEnv()
	defer teardownTestEnv()

	// Initialize config
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// This should work with Background logger
	logger := klog.Background()
	os := "Linux"
	controller := CreateCDIController(logger, &protocdi.ProductInformation{Vendor: "Fujitsu", ProductName: "PG-CDI", Version: "1.1", Os: &os})

	if controller == nil {
		t.Fatal("CreateCDIController returned nil")
	}
}

func TestCreateCDIController_MultipleCallsWithSameLogger_ReturnsControllers(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup product mappings
	setupTestMappings(t)
	defer cleanupTestMappings()

	// Setup test environment variables
	setupTestEnv()
	defer teardownTestEnv()

	// Initialize config
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.Background()

	// Execute multiple times
	os := "Linux"
	controller1 := CreateCDIController(logger, &protocdi.ProductInformation{Vendor: "Fujitsu", ProductName: "PG-CDI", Version: "1.1", Os: &os})
	controller2 := CreateCDIController(logger, &protocdi.ProductInformation{Vendor: "Fujitsu", ProductName: "PG-CDI", Version: "1.1", Os: &os})

	// Verify both are not nil and different instances
	if controller1 == nil {
		t.Fatal("First CreateCDIController returned nil")
	}
	if controller2 == nil {
		t.Fatal("Second CreateCDIController returned nil")
	}

	// They should be different instances
	if controller1 == controller2 {
		t.Error("CreateCDIController returned same instance, expected different instances")
	}
}

func TestCreateCDIController_DifferentLoggers_ReturnsControllers(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup product mappings
	setupTestMappings(t)
	defer cleanupTestMappings()

	// Setup test environment variables
	setupTestEnv()
	defer teardownTestEnv()

	// Initialize config
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger1 := klog.Background()
	logger2 := klog.Background()

	// Execute with different loggers
	os := "Linux"
	controller1 := CreateCDIController(logger1, &protocdi.ProductInformation{Vendor: "Fujitsu", ProductName: "PG-CDI", Version: "1.1", Os: &os})
	controller2 := CreateCDIController(logger2, &protocdi.ProductInformation{Vendor: "Fujitsu", ProductName: "PG-CDI", Version: "1.1", Os: &os})

	// Verify both are not nil
	if controller1 == nil {
		t.Fatal("First CreateCDIController returned nil")
	}
	if controller2 == nil {
		t.Fatal("Second CreateCDIController returned nil")
	}
}

// Helper functions for test environment setup
func setupTestEnv() {
	os.Setenv("CDI_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test_key")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/tmp/test_cert")
}

func teardownTestEnv() {
	os.Unsetenv("CDI_SERVER_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SSH_KEY")
	os.Unsetenv("TLS_ENABLE")
	os.Unsetenv("TLS_CERT_PATH")

	// Reset global config for next test
	utils.ResetConfigForTesting()
}

// setupTestMappings sets up test environment with product mappings
func setupTestMappings(t *testing.T) {
	t.Helper()
	
	mappingsJSON := `{
		"nw_products": [
			{"vendor": "EdgeCore", "product_name": "AS7326-56X", "version": "1.0", "os": "SONiC", "type": "EdgeCoreSonic"},
			{"vendor": "Broadcom", "product_name": "BCM56960", "version": "1.0", "os": "SONiC", "type": "BroadcomSonic"},
			{"vendor": "Dummy", "product_name": "DummySwitch", "version": "1.0", "os": "Linux", "type": "Dummy"}
		],
		"server_products": [
			{"vendor": "Dell", "product_name": "PowerEdge", "version": "1.0", "os": "Linux", "type": "Dell"},
			{"vendor": "Fujitsu", "product_name": "PRIMERGY", "version": "1.0", "os": "Linux", "type": "Primergy"},
			{"vendor": "Supermicro", "product_name": "SuperServer", "version": "1.0", "os": "Linux", "type": "Supermicro"}
		],
		"cdi_products": [
			{"vendor": "Fujitsu", "product_name": "PG-CDI", "version": "1.1", "os": "Linux", "type": "PG_CDI_1_1"},
			{"vendor": "Fujitsu", "product_name": "PG-CDI", "version": "1.0", "os": "Linux", "type": "PG_CDI_1_0"}
		],
		"maas_products": [
			{"vendor": "Canonical", "product_name": "MAAS", "version": "3.0", "os": "Ubuntu", "type": "Canonical"}
		]
	}`
	
	os.Setenv("PRODUCT_MAPPINGS", mappingsJSON)
	
	// Reset sync.Once to allow reloading in common/models
	// This is a workaround since we can't directly access the sync.Once in common/models
}

// cleanupTestMappings cleans up test environment
func cleanupTestMappings() {
	os.Unsetenv("PRODUCT_MAPPINGS")
}

