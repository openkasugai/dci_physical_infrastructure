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

package utils

import (
	"os"
	"strings"
	"sync"
	"testing"

	"maas_module/internal/server/test_utils"
)

// Test helper to set environment variables and clean up after test
func setTestEnv(t *testing.T, env map[string]string) {
	// Set test environment variables
	for key, value := range env {
		os.Setenv(key, value)
	}

	// Clean up after test
	t.Cleanup(func() {
		for key := range env {
			os.Unsetenv(key)
		}
		// Reset global config for next test
		globalConfig = nil
		configOnce = sync.Once{}
		configError = nil
	})
}

// TestInitializeConfig_ValidEnvironment_ReturnsSuccess tests successful config initialization
func TestInitializeConfig_ValidEnvironment_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"LOG_LEVEL":        "2",
		"MAAS_SERVER_PORT": "8080",
		"VM_HOST_DISK":     "50",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "test-ssh-key",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}
	setTestEnv(t, testEnv)

	// Act
	err := InitializeConfig()

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	config := GetConfig()
	if config == nil {
		t.Error("Expected config to be initialized")
	}

	// Verify configuration values
	if config.LogLevel != "2" {
		t.Errorf("Expected LogLevel to be '2', got: %s", config.LogLevel)
	}
	if config.ServerPort != 8080 {
		t.Errorf("Expected ServerPort to be 8080, got: %d", config.ServerPort)
	}
	if config.VmHostDisk != 50 {
		t.Errorf("Expected VmHostDisk to be 50, got: %d", config.VmHostDisk)
	}
}

// TestInitializeConfig_MissingRequiredEnv_ReturnsError tests config initialization with missing environment variables
func TestInitializeConfig_MissingRequiredEnv_ReturnsError(t *testing.T) {

	// Arrange - Missing LOG_LEVEL
	testEnv := map[string]string{
		"MAAS_SERVER_PORT": "8080",
		"VM_HOST_DISK":     "50",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "test-ssh-key",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}
	setTestEnv(t, testEnv)

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for missing LOG_LEVEL, got nil")
	}
	if !strings.Contains(err.Error(), "LOG_LEVEL is required") {
		t.Errorf("Expected error to mention LOG_LEVEL is required, got: %v", err)
	}
}

// TestInitializeConfig_InvalidLogLevel_ReturnsError tests config initialization with invalid log level
func TestInitializeConfig_InvalidLogLevel_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"LOG_LEVEL":        "invalid",
		"MAAS_SERVER_PORT": "8080",
		"VM_HOST_DISK":     "50",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "test-ssh-key",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}
	setTestEnv(t, testEnv)

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for invalid LOG_LEVEL, got nil")
	}
	if !strings.Contains(err.Error(), "invalid logLevel of configuration") {
		t.Errorf("Expected error to mention invalid logLevel, got: %v", err)
	}
}

// TestInitializeConfig_InvalidPort_ReturnsError tests config initialization with invalid port
func TestInitializeConfig_InvalidPort_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"LOG_LEVEL":        "2",
		"MAAS_SERVER_PORT": "invalid",
		"VM_HOST_DISK":     "50",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "test-ssh-key",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}
	setTestEnv(t, testEnv)

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for invalid MAAS_SERVER_PORT, got nil")
	}
	if !strings.Contains(err.Error(), "invalid serverPort of configuration") {
		t.Errorf("Expected error to mention invalid serverPort, got: %v", err)
	}
}

// TestInitializeConfig_VmHostDiskBoundary_ReturnsSuccess tests boundary values for VM host disk
func TestInitializeConfig_VmHostDiskBoundary_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name       string
		vmHostDisk string
		expected   int
	}{
		{"Minimum VM host disk", "10", 10},
		{"Valid VM host disk", "50", 50},
		{"Maximum VM host disk", "90", 90},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			testEnv := map[string]string{
				"LOG_LEVEL":        "2",
				"MAAS_SERVER_PORT": "8080",
				"VM_HOST_DISK":     tc.vmHostDisk,
				"LXD_PORT":         "8443",
				"SSH_KEY":          "test-ssh-key",
				"TLS_ENABLE":       "false",
				"TLS_CERT_PATH":    "/certs",
			}
			setTestEnv(t, testEnv)

			// Act
			err := InitializeConfig()

			// Assert
			if err != nil {
				t.Errorf("Expected no error for VM host disk %s, got: %v", tc.vmHostDisk, err)
			}

			config := GetConfig()
			if config.VmHostDisk != tc.expected {
				t.Errorf("Expected VmHostDisk to be %d, got: %d", tc.expected, config.VmHostDisk)
			}
		})
	}
}

// TestInitializeConfig_VmHostDiskOutOfBounds_ReturnsError tests VM host disk values outside valid range
func TestInitializeConfig_VmHostDiskOutOfBounds_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name       string
		vmHostDisk string
	}{
		{"Below minimum", "9"},
		{"Below minimum", "0"},
		{"Negative value", "-1"},
		{"Above maximum", "91"},
		{"Above maximum", "100"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			testEnv := map[string]string{
				"LOG_LEVEL":        "2",
				"MAAS_SERVER_PORT": "8080",
				"VM_HOST_DISK":     tc.vmHostDisk,
				"LXD_PORT":         "8443",
				"SSH_KEY":          "test-ssh-key",
				"TLS_ENABLE":       "false",
				"TLS_CERT_PATH":    "/certs",
			}
			setTestEnv(t, testEnv)


		// Act
		err := InitializeConfig()

		// Assert
		if err == nil {
			t.Errorf("Expected error for VM host disk %s, got nil", tc.vmHostDisk)
		}
		if !strings.Contains(err.Error(), "invalid vmHostDisk of configuration: value must be integer and 5 (GiB) or greater") {
			t.Errorf("Expected error to mention invalid vmHostDisk, got: %v", err)
		}
	})
	}
}
// TestParseAndValidateVmHostDisk_ValidValues_ReturnsSuccess tests parseAndValidateVmHostDisk with valid values
func TestParseAndValidateVmHostDisk_ValidValues_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	validator := newEnvValidator()
	testCases := []struct {
		value    string
		expected int
	}{
		{"10", 10},
		{"50", 50},
		{"90", 90},
		{"75", 75},
	}

	for _, tc := range testCases {
		t.Run("VmHostDisk_"+tc.value, func(t *testing.T) {
			// Act
			result, err := validator.parseAndValidateVmHostDisk(tc.value)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for VM host disk %s, got: %v", tc.value, err)
			}
			if result != tc.expected {
				t.Errorf("Expected result to be %d, got: %d", tc.expected, result)
			}
		})
	}
}

// TestParseAndValidateVmHostDisk_InvalidValues_ReturnsError tests parseAndValidateVmHostDisk with invalid values
func TestParseAndValidateVmHostDisk_InvalidValues_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	validator := newEnvValidator()
	testCases := []string{
		"invalid",
		"9",
		"0",
		"-1",
		"91",
		"100",
		"",
		"abc",
		"5.5",
	}

	for _, tc := range testCases {
		t.Run("VmHostDisk_"+tc, func(t *testing.T) {
		// Act
		result, err := validator.parseAndValidateVmHostDisk(tc)

		// Assert
		if err == nil {
			t.Errorf("Expected error for VM host disk %s, got nil", tc)
		}
		if result != 0 {
			t.Errorf("Expected result to be 0 on error, got: %d", result)
		}
		if !strings.Contains(err.Error(), "invalid vmHostDisk of configuration: value must be integer and 5 (GiB) or greater") {
			t.Errorf("Expected error message to contain vmHostDisk validation message, got: %v", err)
		}
	})
	}
}
// TestInitializeConfig_InvalidVmHostDisk_ReturnsError tests config initialization with invalid VM host disk
func TestInitializeConfig_InvalidVmHostDisk_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"LOG_LEVEL":        "2",
		"MAAS_SERVER_PORT": "8080",
		"VM_HOST_DISK":     "invalid",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "test-ssh-key",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}
	setTestEnv(t, testEnv)

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for invalid VM_HOST_DISK, got nil")
	}
	if !strings.Contains(err.Error(), "invalid vmHostDisk of configuration: value must be integer and 5 (GiB) or greater") {
		t.Errorf("Expected error to mention invalid vmHostDisk of configuration, got: %v", err)
	}
}

// TestInitializeConfig_LogLevelBoundary_ReturnsSuccess tests boundary values for log level
func TestInitializeConfig_LogLevelBoundary_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	tests := []struct {
		name     string
		logLevel string
	}{
		{"Minimum log level", "0"},
		{"Maximum log level", "9"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			testEnv := map[string]string{
				"LOG_LEVEL":        tc.logLevel,
				"MAAS_SERVER_PORT": "8080",
				"VM_HOST_DISK":     "50",
				"LXD_PORT":         "8443",
				"SSH_KEY":          "test-ssh-key",
				"TLS_ENABLE":       "false",
				"TLS_CERT_PATH":    "/certs",
			}
			setTestEnv(t, testEnv)

			// Act
			err := InitializeConfig()

			// Assert
			if err != nil {
				t.Errorf("Expected no error for log level %s, got: %v", tc.logLevel, err)
			}
		})
	}
}

// TestInitializeConfig_LogLevelOutOfBounds_ReturnsError tests log level values outside valid range
func TestInitializeConfig_LogLevelOutOfBounds_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name     string
		logLevel string
	}{
		{"Below minimum", "-1"},
		{"Above maximum", "10"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			testEnv := map[string]string{
				"LOG_LEVEL":        tc.logLevel,
				"MAAS_SERVER_PORT": "8080",
				"VM_HOST_DISK":     "50",
				"LXD_PORT":         "8443",
				"SSH_KEY":          "test-ssh-key",
				"TLS_ENABLE":       "false",
				"TLS_CERT_PATH":    "/certs",
			}
			setTestEnv(t, testEnv)

			// Act
			err := InitializeConfig()

			// Assert
			if err == nil {
				t.Errorf("Expected error for log level %s, got nil", tc.logLevel)
			}
		})
	}
}

// TestInitializeConfig_PortBoundary_ReturnsSuccess tests boundary values for ports
func TestInitializeConfig_PortBoundary_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name string
		port string
	}{
		{"Minimum port", "0"},
		{"Maximum port", "65535"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			testEnv := map[string]string{
				"LOG_LEVEL":        "2",
				"MAAS_SERVER_PORT": tc.port,
				"VM_HOST_DISK":     "50",
				"LXD_PORT":         tc.port,
				"SSH_KEY":          "test-ssh-key",
				"TLS_ENABLE":       "false",
				"TLS_CERT_PATH":    "/certs",
			}
			setTestEnv(t, testEnv)

			// Act
			err := InitializeConfig()

			// Assert
			if err != nil {
				t.Errorf("Expected no error for port %s, got: %v", tc.port, err)
			}
		})
	}
}

// TestInitializeConfig_PortOutOfBounds_ReturnsError tests port values outside valid range
func TestInitializeConfig_PortOutOfBounds_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name string
		port string
	}{
		{"Below minimum", "-1"},
		{"Above maximum", "65536"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			testEnv := map[string]string{
				"LOG_LEVEL":        "2",
				"MAAS_SERVER_PORT": "8080",
				"VM_HOST_DISK":     "50",
				"LXD_PORT":         tc.port,

				"SSH_KEY":          "test-ssh-key",
				"TLS_ENABLE":       "false",
				"TLS_CERT_PATH":    "/certs",
			}
			setTestEnv(t, testEnv)

			// Act
			err := InitializeConfig()

			// Assert
			if err == nil {
				t.Errorf("Expected error for port %s, got nil", tc.port)
			}
		})
	}
}

// TestInitializeConfig_CalledMultipleTimes_ReturnsSameResult tests that InitializeConfig can be called multiple times safely
func TestInitializeConfig_CalledMultipleTimes_ReturnsSameResult(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"LOG_LEVEL":        "2",
		"MAAS_SERVER_PORT": "8080",
		"VM_HOST_DISK":     "50",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "test-ssh-key",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}
	setTestEnv(t, testEnv)

	// Act
	err1 := InitializeConfig()
	err2 := InitializeConfig()
	config1 := GetConfig()
	config2 := GetConfig()

	// Assert
	if err1 != nil {
		t.Errorf("Expected no error on first call, got: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Expected no error on second call, got: %v", err2)
	}
	if config1 != config2 {
		t.Error("Expected same config instance on multiple calls")
	}
}

// TestGetConfig_BeforeInitialize_ReturnsNil tests that GetConfig returns nil before initialization
func TestGetConfig_BeforeInitialize_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - Reset global state
	globalConfig = nil
	configOnce = sync.Once{}
	configError = nil

	// Act
	config := GetConfig()

	// Assert
	if config != nil {
		t.Error("Expected nil config before initialization")
	}
}

// TestNewEnvValidator_Creates_ValidValidator tests that newEnvValidator creates a valid validator
func TestNewEnvValidator_Creates_ValidValidator(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Act
	validator := newEnvValidator()

	// Assert
	if validator == nil {
		t.Error("Expected validator to be created")
	}
	if validator.validator == nil {
		t.Error("Expected validator.validator to be initialized")
	}
}

// TestParseAndValidateLogLevel_ValidValues_ReturnsSuccess tests parseAndValidateLogLevel with valid values
func TestParseAndValidateLogLevel_ValidValues_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	validator := newEnvValidator()
	testCases := []string{"0", "1", "5", "9"}

	for _, testCase := range testCases {
		t.Run("LogLevel_"+testCase, func(t *testing.T) {
			// Act
			result, err := validator.parseAndValidateLogLevel(testCase)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for log level %s, got: %v", testCase, err)
			}
			if result != testCase {
				t.Errorf("Expected result to be %s, got: %s", testCase, result)
			}
		})
	}
}

// TestParseAndValidateLogLevel_InvalidValues_ReturnsError tests parseAndValidateLogLevel with invalid values
func TestParseAndValidateLogLevel_InvalidValues_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	validator := newEnvValidator()
	testCases := []string{"invalid", "-1", "10", "abc", ""}

	for _, testCase := range testCases {
		t.Run("LogLevel_"+testCase, func(t *testing.T) {
			// Act
			result, err := validator.parseAndValidateLogLevel(testCase)

			// Assert
			if err == nil {
				t.Errorf("Expected error for log level %s, got nil", testCase)
			}
			if result != "0" {
				t.Errorf("Expected result to be '0' on error, got: %s", result)
			}
		})
	}
}

// TestParseAndValidatePort_ValidValues_ReturnsSuccess tests parseAndValidatePort with valid values
func TestParseAndValidatePort_ValidValues_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	validator := newEnvValidator()
	testCases := []struct {
		key      string
		value    string
		expected int
	}{
		{"serverPort", "8080", 8080},
		{"lxdPort", "0", 0},
		{"testPort", "65535", 65535},
	}

	for _, tc := range testCases {
		t.Run(tc.key+"_"+tc.value, func(t *testing.T) {
			// Act
			result, err := validator.parseAndValidatePort(tc.key, tc.value)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for port %s, got: %v", tc.value, err)
			}
			if result != tc.expected {
				t.Errorf("Expected result to be %d, got: %d", tc.expected, result)
			}
		})
	}
}

// TestParseAndValidatePort_InvalidValues_ReturnsError tests parseAndValidatePort with invalid values
func TestParseAndValidatePort_InvalidValues_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	validator := newEnvValidator()
	testCases := []struct {
		key   string
		value string
	}{
		{"serverPort", "invalid"},
		{"serverPort", "-1"},
		{"serverPort", "65536"},
		{"serverPort", ""},
		{"lxdPort", "abc"},
	}

	for _, tc := range testCases {
		t.Run(tc.key+"_"+tc.value, func(t *testing.T) {
			// Act
			result, err := validator.parseAndValidatePort(tc.key, tc.value)

			// Assert
			if err == nil {
				t.Errorf("Expected error for port %s, got nil", tc.value)
			}
			if result != 0 {
				t.Errorf("Expected result to be 0 on error, got: %d", result)
			}
		})
	}
}

// TestGetEnvVarName_AllFields_ReturnsCorrectNames tests getEnvVarName for all struct fields
func TestGetEnvVarName_AllFields_ReturnsCorrectNames(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	validator := newEnvValidator()
	testCases := []struct {
		fieldName string
		expected  string
	}{
		{"LogLevel", "LOG_LEVEL"},
		{"ServerPort", "MAAS_SERVER_PORT"},
		{"VmHostDisk", "VM_HOST_DISK"},
		{"LxdListenPort", "LXD_PORT"},
		{"SshKey", "SSH_KEY"},
		{"TlsEnable", "TLS_ENABLE"},
		{"TlsCertPath", "TLS_CERT_PATH"},
		{"UnknownField", "UnknownField"},
	}

	for _, tc := range testCases {
		t.Run(tc.fieldName, func(t *testing.T) {
			// Act
			result := validator.getEnvVarName(tc.fieldName)

			// Assert
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestFormatValidationError_ValidatorErrors_ReturnsFormattedMessage tests formatValidationError with validator errors
func TestFormatValidationError_ValidatorErrors_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	validator := newEnvValidator()
	config := &EnvConfig{
		// Missing required fields to trigger validation errors
	}
	validationErr := validator.validator.Struct(config)

	// Act
	result := validator.formatValidationError(validationErr)

	// Assert
	if result == nil {
		t.Error("Expected formatted error, got nil")
	}

	errorMsg := result.Error()
	expectedFields := []string{"LOG_LEVEL is required", "MAAS_SERVER_PORT is required"}
	for _, expected := range expectedFields {
		if !strings.Contains(errorMsg, expected) {
			t.Errorf("Expected error message to contain '%s', got: %s", expected, errorMsg)
		}
	}
}

// TestLoadAndValidateConfig_CompleteFlow_ReturnsConfig tests the complete loadAndValidateConfig flow
func TestLoadAndValidateConfig_CompleteFlow_ReturnsConfig(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"LOG_LEVEL":        "5",
		"MAAS_SERVER_PORT": "9090",
		"VM_HOST_DISK":     "75",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "ssh-rsa AAAAB3...",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}
	setTestEnv(t, testEnv)

	validator := newEnvValidator()

	// Act
	config, err := validator.loadAndValidateConfig()

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if config == nil {
		t.Error("Expected config to be created")
	}

	// Verify all fields
	if config.LogLevel != "5" {
		t.Errorf("Expected LogLevel '5', got: %s", config.LogLevel)
	}
	if config.ServerPort != 9090 {
		t.Errorf("Expected ServerPort 9090, got: %d", config.ServerPort)
	}
	if config.VmHostDisk != 75 {
		t.Errorf("Expected VmHostDisk 75, got: %d", config.VmHostDisk)
	}
	if config.LxdListenPort != 8443 {
		t.Errorf("Expected LxdListenPort 8443, got: %d", config.LxdListenPort)
	}
	if config.SshKey != "ssh-rsa AAAAB3..." {
		t.Errorf("Expected SshKey 'ssh-rsa AAAAB3...', got: %s", config.SshKey)
	}
}
