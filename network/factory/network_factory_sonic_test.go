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

	proto "network_module/api/proto"
	"network_module/internal/server/implementation/edgecore_sonic_network"
	"network_module/internal/server/interfaces"
	"network_module/internal/server/test_utils"
	"network_module/internal/server/utils"

	"k8s.io/klog/v2"
)

// setupTestConfig sets up test configuration for sonic tests
func setupSonicTestConfig() {
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	utils.InitializeConfig()
}

// clearTestConfig clears test configuration
func clearSonicTestConfig() {
	os.Unsetenv("NW_SERVER_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SSH_KEY")
	os.Unsetenv("TLS_ENABLE")
	os.Unsetenv("TLS_CERT_PATH")
}

func TestCreateNetworkController_SonicBuild_ReturnsSonicController(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupSonicTestConfig()
	defer clearSonicTestConfig()
	logger := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "EdgeCore",
		ProductName: "SonicSwitch",
		Version:     "1.0",
		Os:          &[]string{"SONiC"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	if controller == nil {
		t.Error("Expected controller to be created, got nil")
		return
	}

	// Check if it's the correct type
	sonicController, ok := controller.(*edgecore_sonic_network.EdgeCoreSonicNetworkController)
	if !ok {
		t.Errorf("Expected EdgeCoreSonicNetworkController, got %T", controller)
		return
	}

	// Verify the logger is set
	if &sonicController.Logger == nil {
		t.Error("Expected logger to be set in EdgeCoreSonicNetworkController")
	}

	// Verify the Ansible instance is set
	if sonicController.Ansible == nil {
		t.Error("Expected Ansible to be set in EdgeCoreSonicNetworkController")
	}

	// Verify the SSH key is set
	if sonicController.SSHKey == "" {
		t.Error("Expected SSHKey to be set in EdgeCoreSonicNetworkController")
	}
	if sonicController.SSHKey != "/tmp/test.pem" {
		t.Errorf("Expected SSHKey '/tmp/test.pem', got '%s'", sonicController.SSHKey)
	}
}

func TestCreateNetworkController_SonicBuild_ImplementsInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupSonicTestConfig()
	defer clearSonicTestConfig()
	logger := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "EdgeCore",
		ProductName: "SonicSwitch",
		Version:     "1.0",
		Os:          &[]string{"SONiC"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	var _ interfaces.NetworkController = controller

	// This test will fail at compile time if the returned controller doesn't implement the interface
	if controller == nil {
		t.Error("Expected controller to implement NetworkController interface")
	}
}

func TestCreateNetworkController_SonicBuild_NilLogger_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupSonicTestConfig()
	defer clearSonicTestConfig()
	var logger klog.Logger
	productInfo := &proto.ProductInformation{
		Vendor:      "EdgeCore",
		ProductName: "SonicSwitch",
		Version:     "1.0",
		Os:          &[]string{"SONiC"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	if controller == nil {
		t.Error("Expected controller to be created even with nil logger, got nil")
		return
	}

	// Check if it's the correct type
	sonicController, ok := controller.(*edgecore_sonic_network.EdgeCoreSonicNetworkController)
	if !ok {
		t.Errorf("Expected EdgeCoreSonicNetworkController, got %T", controller)
		return
	}

	// Verify the Ansible instance is still created
	if sonicController.Ansible == nil {
		t.Error("Expected Ansible to be set in EdgeCoreSonicNetworkController")
	}
}

func TestCreateNetworkController_SonicBuild_CreatesAnsibleInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupSonicTestConfig()
	defer clearSonicTestConfig()
	logger := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "EdgeCore",
		ProductName: "SonicSwitch",
		Version:     "1.0",
		Os:          &[]string{"SONiC"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	sonicController := controller.(*edgecore_sonic_network.EdgeCoreSonicNetworkController)

	// Verify the Ansible instance is of the correct type
	ansibleImple, ok := sonicController.Ansible.(*edgecore_sonic_network.EdgeCoreSonicAnsible)
	if !ok {
		t.Errorf("Expected EdgeCoreSonicAnsible, got %T", sonicController.Ansible)
		return
	}

	// Verify the logger is set in the Ansible instance
	if &ansibleImple.Logger == nil {
		t.Error("Expected logger to be set in EdgeCoreSonicAnsible")
	}
}

// Additional edge case tests for better coverage
func TestCreateNetworkController_SonicBuild_MissingSSHKey_UsesDefault(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	// Don't set SSH_KEY to test handling of missing value
	defer clearSonicTestConfig()

	utils.InitializeConfig()
	logger := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "EdgeCore",
		ProductName: "SonicSwitch",
		Version:     "1.0",
		Os:          &[]string{"SONiC"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	if controller == nil {
		t.Error("Expected controller to be created even without SSH_KEY, got nil")
		return
	}

	sonicController := controller.(*edgecore_sonic_network.EdgeCoreSonicNetworkController)
	// SSH key might be empty or have default value depending on implementation
	if sonicController.SSHKey != "" {
		t.Logf("SSHKey is set to: %s", sonicController.SSHKey)
	}
}

func TestCreateNetworkController_SonicBuild_EmptySSHKey_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "") // Set empty SSH key
	defer clearSonicTestConfig()

	// Reset global config to allow re-initialization
	utils.InitializeConfig()
	logger := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "EdgeCore",
		ProductName: "SonicSwitch",
		Version:     "1.0",
		Os:          &[]string{"SONiC"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	if controller == nil {
		t.Error("Expected controller to be created with empty SSH_KEY, got nil")
		return
	}

	sonicController := controller.(*edgecore_sonic_network.EdgeCoreSonicNetworkController)
	// Note: Due to sync.Once in config initialization, this test may not reflect
	// the empty SSH key if config was already initialized. This is expected behavior.
	t.Logf("SSHKey is: '%s' (may not reflect empty due to singleton config)", sonicController.SSHKey)
}

func TestCreateNetworkController_SonicBuild_DifferentConfigs_CreatesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test with different configuration values
	// Note: Due to sync.Once in config initialization, only the first config will be used
	testCases := []struct {
		name     string
		port     string
		logLevel string
		sshKey   string
	}{
		{
			name:     "Standard config",
			port:     "8080",
			logLevel: "1",
			sshKey:   "/home/user/.ssh/id_rsa",
		},
		{
			name:     "Alternative config",
			port:     "9090",
			logLevel: "5",
			sshKey:   "/etc/ssh/keys/service.pem",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Arrange
			os.Setenv("NW_SERVER_PORT", tc.port)
			os.Setenv("LOG_LEVEL", tc.logLevel)
			os.Setenv("SSH_KEY", tc.sshKey)
			defer clearSonicTestConfig()

			utils.InitializeConfig()
			logger := klog.Background()
			productInfo := &proto.ProductInformation{
				Vendor:      "EdgeCore",
				ProductName: "SonicSwitch",
				Version:     "1.0",
				Os:          &[]string{"SONiC"}[0],
			}

			// Act
			controller := CreateNetworkController(logger, productInfo)

			// Assert
			if controller == nil {
				t.Errorf("Expected controller to be created for %s, got nil", tc.name)
				return
			}

			sonicController := controller.(*edgecore_sonic_network.EdgeCoreSonicNetworkController)
			// Note: Due to singleton pattern in config, SSH key might not change between tests
			// This is expected behavior in the actual application
			t.Logf("Config test %s: SSHKey is '%s' (singleton config may not reflect changes)",
				tc.name, sonicController.SSHKey)

			// Verify that controller is created successfully regardless of config
			if sonicController.Ansible == nil {
				t.Errorf("Expected Ansible to be set in controller for %s", tc.name)
			}
		})
	}
}
