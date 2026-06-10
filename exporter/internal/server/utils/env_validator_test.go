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
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Helper function to set up environment variables for a test
func setEnvForEnvValidatorTest(t *testing.T, envVars map[string]string) func() {
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
		// Reset global config for next test
		globalConfig = nil
		configOnce = sync.Once{}
		configError = nil
	}
}

// Helper function to set up complete valid environment variables
func setupValidEnvVars() map[string]string {
	return map[string]string{
		"LOG_LEVEL":        "2",
		"INTERVAL":         "300",
		"P2P_ENABLE":       "true",
		"P2P_INTERVAL":     "600",
		"SSH_KEY":          "/path/to/ssh/key",
		"METRICS_PORT":     "8080",
		"METRICS_ENDPOINT": "/metrics",
		"DB_URL":           "postgresql://exporter_user:password@localhost:5432/exporter_db",
	}
}

// TestInitializeConfig_ValidEnvironment_ReturnsSuccess tests successful configuration initialization
func TestInitializeConfig_ValidEnvironment_ReturnsSuccess(t *testing.T) {
	// Setup
	cleanup := setEnvForEnvValidatorTest(t, setupValidEnvVars())
	defer cleanup()

	// Execute
	err := InitializeConfig()

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	config := GetConfig()
	if config == nil {
		t.Fatal("Expected config to be set, got nil")
	}

	// Verify parsed values
	if config.LogLevel != "2" {
		t.Errorf("Expected LogLevel '2', got: '%s'", config.LogLevel)
	}
	if config.Interval != 300 {
		t.Errorf("Expected Interval 300, got: %d", config.Interval)
	}
	if config.P2PInterval != 600 {
		t.Errorf("Expected P2PInterval 600, got: %d", config.P2PInterval)
	}
	if config.MetricsPort != 8080 {
		t.Errorf("Expected MetricsPort 8080, got: %d", config.MetricsPort)
	}
}

// TestInitializeConfig_MissingLogLevel_ReturnsError tests missing LOG_LEVEL environment variable
func TestInitializeConfig_MissingLogLevel_ReturnsError(t *testing.T) {
	// Setup - complete env vars except LOG_LEVEL
	envVars := setupValidEnvVars()
	delete(envVars, "LOG_LEVEL")
	cleanup := setEnvForEnvValidatorTest(t, envVars)
	defer cleanup()

	// Execute
	err := InitializeConfig()

	// Verify
	if err == nil {
		t.Error("Expected error for missing LOG_LEVEL, got nil")
	}
	if !strings.Contains(err.Error(), "LOG_LEVEL is required") {
		t.Errorf("Expected error to contain 'LOG_LEVEL is required', got: %s", err.Error())
	}
}

// TestInitializeConfig_MissingInterval_ReturnsError tests missing INTERVAL environment variable
func TestInitializeConfig_MissingInterval_ReturnsError(t *testing.T) {
	// Setup - complete env vars except INTERVAL
	envVars := setupValidEnvVars()
	delete(envVars, "INTERVAL")
	cleanup := setEnvForEnvValidatorTest(t, envVars)
	defer cleanup()

	// Execute
	err := InitializeConfig()

	// Verify
	if err == nil {
		t.Error("Expected error for missing INTERVAL, got nil")
	}
	if !strings.Contains(err.Error(), "INTERVAL is required") {
		t.Errorf("Expected error to contain 'INTERVAL is required', got: %s", err.Error())
	}
}

// TestInitializeConfig_MultipleMissingVars_ReturnsAllErrors tests multiple missing environment variables
func TestInitializeConfig_MultipleMissingVars_ReturnsAllErrors(t *testing.T) {
	// Setup - only set a few env vars, leaving many missing
	partialEnvVars := map[string]string{
		"LOG_LEVEL": "2",
		"INTERVAL":  "300",
	}
	cleanup := setEnvForEnvValidatorTest(t, partialEnvVars)
	defer cleanup()

	// Execute
	err := InitializeConfig()

	// Verify
	if err == nil {
		t.Error("Expected error for missing environment variables, got nil")
	}

	errorMessage := err.Error()
	requiredVars := []string{"SSH_KEY", "METRICS_PORT", "METRICS_ENDPOINT",
		"DB_URL", "P2P_INTERVAL"}

	for _, requiredVar := range requiredVars {
		expectedMessage := requiredVar + " is required"
		if !strings.Contains(errorMessage, expectedMessage) {
			t.Errorf("Expected error to contain '%s', got: %s", expectedMessage, errorMessage)
		}
	}
}

// TestInitializeConfig_InvalidLogLevel_ReturnsError tests invalid log level values
func TestInitializeConfig_InvalidLogLevel_ReturnsError(t *testing.T) {
	testCases := []string{
		"invalid",
		"-1",
		"10",
		"abc",
	}

	for _, invalidLevel := range testCases {
		t.Run("logLevel_"+invalidLevel, func(t *testing.T) {
			// Setup
			envVars := setupValidEnvVars()
			envVars["LOG_LEVEL"] = invalidLevel
			cleanup := setEnvForEnvValidatorTest(t, envVars)
			defer cleanup()

			// Execute
			err := InitializeConfig()

			// Verify
			if err == nil {
				t.Errorf("Expected error for invalid log level '%s', got nil", invalidLevel)
			}
			if !strings.Contains(err.Error(), "invalid logLevel") {
				t.Errorf("Expected error to contain 'invalid logLevel', got: %s", err.Error())
			}
		})
	}

	// Test empty log level separately (should be required field error)
	t.Run("logLevel_empty", func(t *testing.T) {
		// Setup
		envVars := setupValidEnvVars()
		envVars["LOG_LEVEL"] = ""
		cleanup := setEnvForEnvValidatorTest(t, envVars)
		defer cleanup()

		// Execute
		err := InitializeConfig()

		// Verify
		if err == nil {
			t.Error("Expected error for empty log level, got nil")
		}
		if !strings.Contains(err.Error(), "LOG_LEVEL is required") {
			t.Errorf("Expected error to contain 'LOG_LEVEL is required', got: %s", err.Error())
		}
	})
}

// TestInitializeConfig_ValidLogLevels_ReturnsSuccess tests all valid log level values
func TestInitializeConfig_ValidLogLevels_ReturnsSuccess(t *testing.T) {
	for logLevel := 0; logLevel <= 9; logLevel++ {
		t.Run("logLevel_"+strconv.Itoa(logLevel), func(t *testing.T) {
			// Setup
			envVars := setupValidEnvVars()
			envVars["LOG_LEVEL"] = strconv.Itoa(logLevel)
			cleanup := setEnvForEnvValidatorTest(t, envVars)
			defer cleanup()

			// Execute
			err := InitializeConfig()

			// Verify
			if err != nil {
				t.Errorf("Expected no error for valid log level %d, got: %v", logLevel, err)
			}

			config := GetConfig()
			if config.LogLevel != strconv.Itoa(logLevel) {
				t.Errorf("Expected LogLevel '%d', got: '%s'", logLevel, config.LogLevel)
			}
		})
	}
}

// TestInitializeConfig_InvalidInterval_ReturnsError tests invalid interval values
func TestInitializeConfig_InvalidInterval_ReturnsError(t *testing.T) {
	testCases := []string{
		"invalid",
		"-1",
		"0",
		"3601",
		"abc",
	}

	for _, invalidInterval := range testCases {
		t.Run("interval_"+invalidInterval, func(t *testing.T) {
			// Setup
			envVars := setupValidEnvVars()
			envVars["INTERVAL"] = invalidInterval
			cleanup := setEnvForEnvValidatorTest(t, envVars)
			defer cleanup()

			// Execute
			err := InitializeConfig()

			// Verify
			if err == nil {
				t.Errorf("Expected error for invalid interval '%s', got nil", invalidInterval)
			}
			// Note: Error message may vary based on validation logic
		})
	}

	// Test empty interval separately (should be required field error)
	t.Run("interval_empty", func(t *testing.T) {
		// Setup
		envVars := setupValidEnvVars()
		envVars["INTERVAL"] = ""
		cleanup := setEnvForEnvValidatorTest(t, envVars)
		defer cleanup()

		// Execute
		err := InitializeConfig()

		// Verify
		if err == nil {
			t.Error("Expected error for empty interval, got nil")
		}
		if !strings.Contains(err.Error(), "INTERVAL is required") {
			t.Errorf("Expected error to contain 'INTERVAL is required', got: %s", err.Error())
		}
	})
}

// TestInitializeConfig_ValidIntervals_ReturnsSuccess tests valid interval values
func TestInitializeConfig_ValidIntervals_ReturnsSuccess(t *testing.T) {
	testCases := []int{1, 2, 300, 1800, 3600}

	for _, validInterval := range testCases {
		t.Run("interval_"+strconv.Itoa(validInterval), func(t *testing.T) {
			// Setup
			envVars := setupValidEnvVars()
			envVars["INTERVAL"] = strconv.Itoa(validInterval)
			cleanup := setEnvForEnvValidatorTest(t, envVars)
			defer cleanup()

			// Execute
			err := InitializeConfig()

			// Verify
			if err != nil {
				t.Errorf("Expected no error for valid interval %d, got: %v", validInterval, err)
			}

			config := GetConfig()
			if config.Interval != validInterval {
				t.Errorf("Expected Interval %d, got: %d", validInterval, config.Interval)
			}
		})
	}
}

// TestInitializeConfig_InvalidMetricsPort_ReturnsError tests invalid metrics port values
func TestInitializeConfig_InvalidMetricsPort_ReturnsError(t *testing.T) {
	testCases := []string{
		"invalid",
		"-1",
		"65536",
		"abc",
	}

	for _, invalidPort := range testCases {
		t.Run("metricsPort_"+invalidPort, func(t *testing.T) {
			// Setup
			envVars := setupValidEnvVars()
			envVars["METRICS_PORT"] = invalidPort
			cleanup := setEnvForEnvValidatorTest(t, envVars)
			defer cleanup()

			// Execute
			err := InitializeConfig()

			// Verify
			if err == nil {
				t.Errorf("Expected error for invalid metrics port '%s', got nil", invalidPort)
			}
			// Note: Error message may vary based on validation logic
		})
	}

	// Test empty metrics port separately (should be required field error)
	t.Run("metricsPort_empty", func(t *testing.T) {
		// Setup
		envVars := setupValidEnvVars()
		envVars["METRICS_PORT"] = ""
		cleanup := setEnvForEnvValidatorTest(t, envVars)
		defer cleanup()

		// Execute
		err := InitializeConfig()

		// Verify
		if err == nil {
			t.Error("Expected error for empty metrics port, got nil")
		}
		if !strings.Contains(err.Error(), "METRICS_PORT is required") {
			t.Errorf("Expected error to contain 'METRICS_PORT is required', got: %s", err.Error())
		}
	})
}



// TestInitializeConfig_ValidPorts_ReturnsSuccess tests valid port values
func TestInitializeConfig_ValidPorts_ReturnsSuccess(t *testing.T) {
	testCases := []int{0, 1, 80, 443, 8080, 65535}

	for _, validPort := range testCases {
		t.Run("port_"+strconv.Itoa(validPort), func(t *testing.T) {
			// Setup
			envVars := setupValidEnvVars()
			envVars["METRICS_PORT"] = strconv.Itoa(validPort)
			envVars["DB_PORT"] = strconv.Itoa(validPort)
			cleanup := setEnvForEnvValidatorTest(t, envVars)
			defer cleanup()

			// Execute
			err := InitializeConfig()

			// Verify
			if err != nil {
				t.Errorf("Expected no error for valid port %d, got: %v", validPort, err)
			}

			config := GetConfig()
			if config.MetricsPort != validPort {
				t.Errorf("Expected MetricsPort %d, got: %d", validPort, config.MetricsPort)
			}
		})
	}
}

// TestInitializeConfig_CalledMultipleTimes_ReturnsConsistentResult tests singleton behavior
func TestInitializeConfig_CalledMultipleTimes_ReturnsConsistentResult(t *testing.T) {
	// Setup
	cleanup := setEnvForEnvValidatorTest(t, setupValidEnvVars())
	defer cleanup()

	// Execute - call multiple times
	err1 := InitializeConfig()
	err2 := InitializeConfig()
	err3 := InitializeConfig()

	// Verify - should all return the same result
	if err1 != nil {
		t.Errorf("Expected no error on first call, got: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Expected no error on second call, got: %v", err2)
	}
	if err3 != nil {
		t.Errorf("Expected no error on third call, got: %v", err3)
	}

	// Verify configs are the same instance
	config1 := GetConfig()
	config2 := GetConfig()
	if config1 != config2 {
		t.Error("Expected same config instance on multiple calls")
	}
}

// TestGetConfig_BeforeInitialize_ReturnsNil tests GetConfig before initialization
func TestGetConfig_BeforeInitialize_ReturnsNil(t *testing.T) {
	// Setup - reset global state
	globalConfig = nil
	configOnce = sync.Once{}
	configError = nil

	// Execute
	config := GetConfig()

	// Verify
	if config != nil {
		t.Errorf("Expected nil config before initialization, got: %v", config)
	}
}

// TestnewEnvValidator_CreateValidator_ReturnsValidInstance tests EnvValidator creation
func TestNewEnvValidator_CreateValidator_ReturnsValidInstance(t *testing.T) {
	// Execute
	validator := newEnvValidator()

	// Verify
	if validator == nil {
		t.Fatal("Expected validator instance, got nil")
	}
	if validator.validator == nil {
		t.Error("Expected validator.validator to be set")
	}
}

// TestEnvValidator_parseAndValidateLogLevel_ValidValues_ReturnsSuccess tests log level parsing
func TestEnvValidator_parseAndValidateLogLevel_ValidValues_ReturnsSuccess(t *testing.T) {
	// Setup
	validator := newEnvValidator()

	for level := 0; level <= 9; level++ {
		t.Run("level_"+strconv.Itoa(level), func(t *testing.T) {
			levelStr := strconv.Itoa(level)

			// Execute
			result, err := validator.parseAndValidateLogLevel(levelStr)

			// Verify
			if err != nil {
				t.Errorf("Expected no error for valid level %s, got: %v", levelStr, err)
			}
			if result != levelStr {
				t.Errorf("Expected result '%s', got: '%s'", levelStr, result)
			}
		})
	}
}

// TestEnvValidator_parseAndValidateLogLevel_InvalidValues_ReturnsError tests invalid log level values
func TestEnvValidator_parseAndValidateLogLevel_InvalidValues_ReturnsError(t *testing.T) {
	// Setup
	validator := newEnvValidator()
	testCases := []string{"invalid", "-1", "10", "abc", ""}

	for _, invalidLevel := range testCases {
		t.Run("invalid_"+invalidLevel, func(t *testing.T) {
			// Execute
			result, err := validator.parseAndValidateLogLevel(invalidLevel)

			// Verify
			if err == nil {
				t.Errorf("Expected error for invalid level '%s', got nil", invalidLevel)
			}
			if result != "0" {
				t.Errorf("Expected default result '0', got: '%s'", result)
			}
		})
	}
}

// TestEnvValidator_parseAndValidateInterval_ValidValues_ReturnsSuccess tests interval parsing
func TestEnvValidator_parseAndValidateInterval_ValidValues_ReturnsSuccess(t *testing.T) {
	// Setup
	validator := newEnvValidator()
	testCases := []int{1, 2, 300, 1800, 3600}

	for _, validInterval := range testCases {
		t.Run("interval_"+strconv.Itoa(validInterval), func(t *testing.T) {
			intervalStr := strconv.Itoa(validInterval)

			// Execute
			result, err := validator.parseAndValidateInterval("interval", intervalStr)

			// Verify
			if err != nil {
				t.Errorf("Expected no error for valid interval %s, got: %v", intervalStr, err)
			}
			if result != validInterval {
				t.Errorf("Expected result %d, got: %d", validInterval, result)
			}
		})
	}
}

// TestEnvValidator_parseAndValidateInterval_InvalidValues_ReturnsError tests invalid interval values
func TestEnvValidator_parseAndValidateInterval_InvalidValues_ReturnsError(t *testing.T) {
	// Setup
	validator := newEnvValidator()
	testCases := []string{"invalid", "-1", "0", "3601", "abc", ""}

	for _, invalidInterval := range testCases {
		t.Run("invalid_"+invalidInterval, func(t *testing.T) {
			// Execute
			result, err := validator.parseAndValidateInterval("interval", invalidInterval)

			// Verify
			if err == nil {
				t.Errorf("Expected error for invalid interval '%s', got nil", invalidInterval)
			}
			if result != 0 {
				t.Errorf("Expected default result 0, got: %d", result)
			}
		})
	}
}

// TestEnvValidator_parseAndValidatePort_ValidValues_ReturnsSuccess tests port parsing
func TestEnvValidator_parseAndValidatePort_ValidValues_ReturnsSuccess(t *testing.T) {
	// Setup
	validator := newEnvValidator()
	testCases := []int{0, 1, 80, 443, 8080, 65535}

	for _, validPort := range testCases {
		t.Run("port_"+strconv.Itoa(validPort), func(t *testing.T) {
			portStr := strconv.Itoa(validPort)

			// Execute
			result, err := validator.parseAndValidatePort("test.port", portStr)

			// Verify
			if err != nil {
				t.Errorf("Expected no error for valid port %s, got: %v", portStr, err)
			}
			if result != validPort {
				t.Errorf("Expected result %d, got: %d", validPort, result)
			}
		})
	}
}

// TestEnvValidator_parseAndValidatePort_InvalidValues_ReturnsError tests invalid port values
func TestEnvValidator_parseAndValidatePort_InvalidValues_ReturnsError(t *testing.T) {
	// Setup
	validator := newEnvValidator()
	testCases := []string{"invalid", "-1", "65536", "abc", ""}

	for _, invalidPort := range testCases {
		t.Run("invalid_"+invalidPort, func(t *testing.T) {
			// Execute
			result, err := validator.parseAndValidatePort("test.port", invalidPort)

			// Verify
			if err == nil {
				t.Errorf("Expected error for invalid port '%s', got nil", invalidPort)
			}
			if result != 0 {
				t.Errorf("Expected default result 0, got: %d", result)
			}
			if !strings.Contains(err.Error(), "test.port") {
				t.Errorf("Expected error to contain 'test.port', got: %s", err.Error())
			}
		})
	}
}

// TestEnvValidator_getEnvVarName_AllFields_ReturnsCorrectNames tests environment variable name mapping
func TestEnvValidator_getEnvVarName_AllFields_ReturnsCorrectNames(t *testing.T) {
	// Setup
	validator := newEnvValidator()

	testCases := map[string]string{
		"Interval":        "INTERVAL",
		"P2PEnable":       "P2P_ENABLE",
		"P2PInterval":     "P2P_INTERVAL",
		"MetricsPort":     "METRICS_PORT",
		"MetricsEndpoint": "METRICS_ENDPOINT",
		"DbAccessURL":     "DB_URL",
		"SshKey":          "SSH_KEY",
		"LogLevel":        "LOG_LEVEL",
		"UnknownField":    "UnknownField", // Default case
	}

	for fieldName, expectedEnvVar := range testCases {
		t.Run("field_"+fieldName, func(t *testing.T) {
			// Execute
			result := validator.getEnvVarName(fieldName)

			// Verify
			if result != expectedEnvVar {
				t.Errorf("Expected env var '%s' for field '%s', got: '%s'", expectedEnvVar, fieldName, result)
			}
		})
	}
}

// TestInitializeConfig_MissingP2PEnable_ReturnsError tests missing P2P_ENABLE environment variable
func TestInitializeConfig_MissingP2PEnable_ReturnsError(t *testing.T) {
	envVars := setupValidEnvVars()
	delete(envVars, "P2P_ENABLE")
	cleanup := setEnvForEnvValidatorTest(t, envVars)
	defer cleanup()

	err := InitializeConfig()

	if err == nil {
		t.Error("Expected error for missing P2P_ENABLE, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "P2P_ENABLE") {
		t.Errorf("Expected error message to contain 'P2P_ENABLE', got: %v", err)
	}
}

// TestInitializeConfig_InvalidP2PEnable_ReturnsError tests invalid P2P_ENABLE values
func TestInitializeConfig_InvalidP2PEnable_ReturnsError(t *testing.T) {
	invalidValues := []string{"yes", "no", "1", "0", "enabled", "disabled", "TRUE1", ""}

	for _, value := range invalidValues {
		t.Run("value_"+value, func(t *testing.T) {
			envVars := setupValidEnvVars()
			envVars["P2P_ENABLE"] = value
			cleanup := setEnvForEnvValidatorTest(t, envVars)
			defer cleanup()

			err := InitializeConfig()

			if err == nil {
				t.Errorf("Expected error for invalid P2P_ENABLE value '%s', got nil", value)
			}
			if err != nil && !strings.Contains(err.Error(), "P2P_ENABLE") {
				t.Errorf("Expected error message to contain 'P2P_ENABLE', got: %v", err)
			}
		})
	}
}

// TestInitializeConfig_ValidP2PEnable_ReturnsSuccess tests valid P2P_ENABLE values
func TestInitializeConfig_ValidP2PEnable_ReturnsSuccess(t *testing.T) {
	validValues := map[string]bool{
		"true":  true,
		"True":  true,
		"TRUE":  true,
		"false": false,
		"False": false,
		"FALSE": false,
	}

	for value, expected := range validValues {
		t.Run("value_"+value, func(t *testing.T) {
			envVars := setupValidEnvVars()
			envVars["P2P_ENABLE"] = value
			cleanup := setEnvForEnvValidatorTest(t, envVars)
			defer cleanup()

			err := InitializeConfig()

			if err != nil {
				t.Errorf("Expected no error for valid P2P_ENABLE value '%s', got: %v", value, err)
			}

			config := GetConfig()
			if config.P2PEnable != expected {
				t.Errorf("Expected P2PEnable to be %v for value '%s', got: %v", expected, value, config.P2PEnable)
			}
		})
	}
}

// TestEnvValidator_parseAndValidateP2PEnable_ValidValues tests parseAndValidateP2PEnable with valid values
func TestEnvValidator_parseAndValidateP2PEnable_ValidValues(t *testing.T) {
	validator := newEnvValidator()

	testCases := map[string]bool{
		"true":  true,
		"True":  true,
		"TRUE":  true,
		"TrUe":  true,
		"false": false,
		"False": false,
		"FALSE": false,
		"FaLsE": false,
	}

	for input, expected := range testCases {
		t.Run("value_"+input, func(t *testing.T) {
			result, err := validator.parseAndValidateP2PEnable(input)

			if err != nil {
				t.Errorf("Expected no error for valid value '%s', got: %v", input, err)
			}
			if result != expected {
				t.Errorf("Expected %v for input '%s', got: %v", expected, input, result)
			}
		})
	}
}

// TestEnvValidator_parseAndValidateP2PEnable_InvalidValues tests parseAndValidateP2PEnable with invalid values
func TestEnvValidator_parseAndValidateP2PEnable_InvalidValues(t *testing.T) {
	validator := newEnvValidator()

	invalidValues := []string{
		"yes", "no", "1", "0", "on", "off", "enabled", "disabled",
		"t", "f", "Y", "N", "", " ", "true1", "false0", "truee", "falsee",
	}

	for _, input := range invalidValues {
		t.Run("value_"+input, func(t *testing.T) {
			result, err := validator.parseAndValidateP2PEnable(input)

			if err == nil {
				t.Errorf("Expected error for invalid value '%s', got nil", input)
			}
			if err != nil && !strings.Contains(err.Error(), "P2P_ENABLE") {
				t.Errorf("Expected error message to contain 'P2P_ENABLE', got: %v", err)
			}
			if err != nil && !strings.Contains(err.Error(), "boolean") {
				t.Errorf("Expected error message to contain 'boolean', got: %v", err)
			}
			if result != false {
				t.Errorf("Expected false return value on error for input '%s', got: %v", input, result)
			}
		})
	}
}
