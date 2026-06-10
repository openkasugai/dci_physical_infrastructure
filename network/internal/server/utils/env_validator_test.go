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
	"sync"
	"testing"
)

// resetGlobalConfig resets the global configuration for testing
func resetGlobalConfig() {
	globalConfig = nil
	configOnce = sync.Once{}
	configError = nil
}

// setTestEnvVars sets up test environment variables
func setTestEnvVars() {
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
}

// clearTestEnvVars clears test environment variables
func clearTestEnvVars() {
	os.Unsetenv("NW_SERVER_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SSH_KEY")
	os.Unsetenv("TLS_ENABLE")
	os.Unsetenv("TLS_CERT_PATH")
}

func TestInitializeConfig_ValidEnvironment_ReturnsSuccess(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	setTestEnvVars()
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	config := GetConfig()
	if config == nil {
		t.Error("Expected config to be set, got nil")
		return
	}

	if config.NWServerPort != 50051 {
		t.Errorf("Expected NWServerPort 50051, got %d", config.NWServerPort)
	}
	if config.LogLevel != "2" {
		t.Errorf("Expected LogLevel '2', got %s", config.LogLevel)
	}
	if config.SSHKey != "/tmp/test.pem" {
		t.Errorf("Expected SSHKey '/tmp/test.pem', got %s", config.SSHKey)
	}
}

func TestInitializeConfig_MissingNWServerPort_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for missing NW_SERVER_PORT, got nil")
	}
}

func TestInitializeConfig_MissingLogLevel_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for missing LOG_LEVEL, got nil")
	}
}

func TestInitializeConfig_MissingSshKey_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for missing SSH_KEY, got nil")
	}
}

func TestInitializeConfig_InvalidPortNumber_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("NW_SERVER_PORT", "invalid")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for invalid port number, got nil")
	}
}

func TestInitializeConfig_PortNumberTooLow_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("NW_SERVER_PORT", "0")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for port number too low, got nil")
	}
}

func TestInitializeConfig_PortNumberTooHigh_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("NW_SERVER_PORT", "65536")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for port number too high, got nil")
	}
}

func TestInitializeConfig_InvalidLogLevel_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "invalid")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for invalid log level, got nil")
	}
}

func TestInitializeConfig_LogLevelTooLow_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "-1")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for log level too low, got nil")
	}
}

func TestInitializeConfig_LogLevelTooHigh_ReturnsError(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "10")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := InitializeConfig()

	// Assert
	if err == nil {
		t.Error("Expected error for log level too high, got nil")
	}
}

func TestGetConfig_BeforeInitialization_ReturnsNil(t *testing.T) {
	// Arrange
	resetGlobalConfig()

	// Act
	config := GetConfig()

	// Assert
	if config != nil {
		t.Error("Expected nil config before initialization, got non-nil")
	}
}

func TestNewEnvValidator_CreatesValidInstance(t *testing.T) {
	// Act
	validator := newEnvValidator()

	// Assert
	if validator == nil {
		t.Error("Expected non-nil validator, got nil")
	}
	if validator.validator == nil {
		t.Error("Expected non-nil internal validator, got nil")
	}
}

func TestParseAndValidateNWServerPort_ValidPort_ReturnsPort(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act
	port, err := validator.parseAndValidateNWServerPort("8080")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected port 8080, got %d", port)
	}
}

func TestParseAndValidateNWServerPort_MinValidPort_ReturnsPort(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act
	port, err := validator.parseAndValidateNWServerPort("1")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if port != 1 {
		t.Errorf("Expected port 1, got %d", port)
	}
}

func TestParseAndValidateNWServerPort_MaxValidPort_ReturnsPort(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act
	port, err := validator.parseAndValidateNWServerPort("65535")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if port != 65535 {
		t.Errorf("Expected port 65535, got %d", port)
	}
}

func TestParseAndValidateLogLevel_ValidLevel_ReturnsLevel(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act
	level, err := validator.parseAndValidateLogLevel("5")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if level != "5" {
		t.Errorf("Expected level '5', got '%s'", level)
	}
}

func TestParseAndValidateLogLevel_ZeroValue_ReturnsValid(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act
	result, err := validator.parseAndValidateLogLevel("0")

	// Assert
	if err != nil {
		t.Errorf("Expected no error for valid log level '0', got %v", err)
	}
	if result != "0" {
		t.Errorf("Expected '0', got %s", result)
	}
}

// Additional coverage tests
func TestEnvValidator_FormatValidationError_SingleRequiredError_ReturnsFormattedMessage(t *testing.T) {
	// Arrange
	validator := newEnvValidator()
	config := &EnvConfig{}

	// Act
	err := validator.validator.Struct(config)
	formattedErr := validator.formatValidationError(err)

	// Assert
	if formattedErr == nil {
		t.Error("Expected formatted error, got nil")
	}
	errorMsg := formattedErr.Error()
	expectedVars := []string{"NW_SERVER_PORT", "LOG_LEVEL", "SSH_KEY"}
	for _, envVar := range expectedVars {
		if !contains(errorMsg, envVar) {
			t.Errorf("Expected error message to contain %s, got: %s", envVar, errorMsg)
		}
	}
}

func TestEnvValidator_GetEnvVarName_AllFields_ReturnsCorrectNames(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act & Assert
	testCases := []struct {
		fieldName string
		expected  string
	}{
		{"NWServerPort", "NW_SERVER_PORT"},
		{"LogLevel", "LOG_LEVEL"},
		{"SSHKey", "SSH_KEY"},
		{"UnknownField", "UnknownField"},
	}

	for _, tc := range testCases {
		result := validator.getEnvVarName(tc.fieldName)
		if result != tc.expected {
			t.Errorf("Expected %s for field %s, got %s", tc.expected, tc.fieldName, result)
		}
	}
}

func TestEnvValidator_ValidateEnvironment_LegacyMethod_AllValidEnvs_ReturnsSuccess(t *testing.T) {
	// Arrange
	validator := newEnvValidator()
	setTestEnvVars()
	defer clearTestEnvVars()

	// Act
	err := validator.validateEnvironment()

	// Assert
	if err != nil {
		t.Errorf("Expected no error for valid environment, got %v", err)
	}
}

func TestEnvValidator_ValidateEnvironment_LegacyMethod_InvalidPort_ReturnsError(t *testing.T) {
	// Arrange
	validator := newEnvValidator()
	os.Setenv("NW_SERVER_PORT", "invalid")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := validator.validateEnvironment()

	// Assert
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}

func TestEnvValidator_ValidateEnvironment_LegacyMethod_InvalidLogLevel_ReturnsError(t *testing.T) {
	// Arrange
	validator := newEnvValidator()
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "invalid")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	err := validator.validateEnvironment()

	// Assert
	if err == nil {
		t.Error("Expected error for invalid log level, got nil")
	}
}

func TestInitializeConfig_CalledMultipleTimes_OnlyInitializesOnce(t *testing.T) {
	// Arrange
	resetGlobalConfig()
	setTestEnvVars()
	defer clearTestEnvVars()

	// Act
	err1 := InitializeConfig()
	config1 := GetConfig()

	// Change environment and call again
	os.Setenv("NW_SERVER_PORT", "60000")
	err2 := InitializeConfig()
	config2 := GetConfig()

	// Assert
	if err1 != nil {
		t.Errorf("Expected no error on first call, got %v", err1)
	}
	if err2 != nil {
		t.Errorf("Expected no error on second call, got %v", err2)
	}

	// Config should be the same (only initialized once)
	if config1 != config2 {
		t.Error("Expected same config instance on multiple calls")
	}

	// Port should still be 50051 from first initialization
	if config2.NWServerPort != 50051 {
		t.Errorf("Expected port 50051 (from first init), got %d", config2.NWServerPort)
	}
}

func TestLoadAndValidateConfig_EdgeCases_HandlesGracefully(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Test with boundary values (minimum port and log level)
	os.Setenv("NW_SERVER_PORT", "1")
	os.Setenv("LOG_LEVEL", "0")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	defer clearTestEnvVars()

	// Act
	config, err := validator.loadAndValidateConfig()

	// Assert - All fields are valid (boundary values), should succeed
	if err != nil {
		t.Errorf("Expected no error for boundary values, got: %v", err)
	}
	if config == nil {
		t.Error("Expected config to be returned, got nil")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseAndValidateLogLevel_MaxValidLevel_ReturnsLevel(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act
	level, err := validator.parseAndValidateLogLevel("9")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if level != "9" {
		t.Errorf("Expected level '9', got '%s'", level)
	}
}

func TestValidateNWServerPort_ValidPort_ReturnsNoError(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act
	err := validator.validateNWServerPort("8080")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestValidateLogLevel_ValidLevel_ReturnsNoError(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act
	err := validator.validateLogLevel("5")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestValidateEnvironment_ValidEnvironment_ReturnsNoError(t *testing.T) {
	// Arrange
	validator := newEnvValidator()
	setTestEnvVars()
	defer clearTestEnvVars()

	// Act
	err := validator.validateEnvironment()

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestGetEnvVarName_KnownFields_ReturnsCorrectNames(t *testing.T) {
	// Arrange
	validator := newEnvValidator()

	// Act & Assert
	testCases := []struct {
		field    string
		expected string
	}{
		{"NWServerPort", "NW_SERVER_PORT"},
		{"LogLevel", "LOG_LEVEL"},
		{"SSHKey", "SSH_KEY"},
		{"UnknownField", "UnknownField"},
	}

	for _, tc := range testCases {
		result := validator.getEnvVarName(tc.field)
		if result != tc.expected {
			t.Errorf("For field %s, expected %s, got %s", tc.field, tc.expected, result)
		}
	}
}
