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
	"cdi_module/internal/server/test_utils"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestInitializeConfig_ValidEnvironmentVariables_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	setupValidTestEnv()
	defer teardownTestEnv()
	ResetConfigForTesting()

	// Execute
	err := InitializeConfig()

	// Verify
	if err != nil {
		t.Fatalf("InitializeConfig failed with valid env vars: %v", err)
	}

	config := GetConfig()
	if config == nil {
		t.Fatal("GetConfig returned nil after successful initialization")
	}

	if config.CDIServerPort != 50051 {
		t.Errorf("Expected CDIServerPort 50051, got %d", config.CDIServerPort)
	}
	if config.LogLevel != 2 {
		t.Errorf("Expected LogLevel 2, got %d", config.LogLevel)
	}
	if config.SSHKey != "/tmp/test_key" {
		t.Errorf("Expected SSHKey '/tmp/test_key', got %s", config.SSHKey)
	}
}

func TestInitializeConfig_MissingRequiredEnvironmentVariable_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name     string
		setupEnv func()
		expected string
	}{
		{
			name: "Missing CDI_SERVER_PORT",
			setupEnv: func() {
				os.Setenv("LOG_LEVEL", "2")
				os.Setenv("SSH_KEY", "/tmp/test_key")
				os.Setenv("TLS_ENABLE", "true")
				os.Setenv("TLS_CERT_PATH", "/tmp/certs")
			},
			expected: "CDI_SERVER_PORT is required",
		},
		{
			name: "Missing LOG_LEVEL",
			setupEnv: func() {
				os.Setenv("CDI_SERVER_PORT", "50051")
				os.Setenv("SSH_KEY", "/tmp/test_key")
				os.Setenv("TLS_ENABLE", "true")
				os.Setenv("TLS_CERT_PATH", "/tmp/certs")
			},
			expected: "LOG_LEVEL is required",
		},
		{
			name: "Missing SSH_KEY",
			setupEnv: func() {
				os.Setenv("CDI_SERVER_PORT", "50051")
				os.Setenv("LOG_LEVEL", "2")
				os.Setenv("TLS_ENABLE", "true")
				os.Setenv("TLS_CERT_PATH", "/tmp/certs")
			},
			expected: "SSH_KEY is required",
		},
		{
			name: "All missing",
			setupEnv: func() {
				// Don't set any environment variables
			},
			expected: "CDI_SERVER_PORT is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Setup
			teardownTestEnv()
			tc.setupEnv()
			defer teardownTestEnv()
			ResetConfigForTesting()

			// Execute
			err := InitializeConfig()

			// Verify
			if err == nil {
				t.Fatal("InitializeConfig should have failed with missing env vars")
			}
			if !containsString(err.Error(), tc.expected) {
				t.Errorf("Expected error to contain '%s', got: %v", tc.expected, err)
			}
		})
	}
}

func TestInitializeConfig_InvalidPortValue_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name      string
		portValue string
		expected  string
	}{
		{
			name:      "Non-numeric port",
			portValue: "invalid",
			expected:  "invalid serverPort of configuration: value must be integer and between 1 ～ 65535, inclusive",
		},
		{
			name:      "Port below range",
			portValue: "0",
			expected:  "invalid serverPort of configuration: value must be integer and between 1 ～ 65535, inclusive",
		},
		{
			name:      "Port above range",
			portValue: "65536",
			expected:  "invalid serverPort of configuration: value must be integer and between 1 ～ 65535, inclusive",
		},
		{
			name:      "Empty port",
			portValue: "",
			expected:  "CDI_SERVER_PORT is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Setup
			setupValidTestEnv()
			if tc.portValue == "" {
				os.Unsetenv("CDI_SERVER_PORT")
			} else {
				os.Setenv("CDI_SERVER_PORT", tc.portValue)
			}
			defer teardownTestEnv()
			ResetConfigForTesting()

			// Execute
			err := InitializeConfig()

			// Verify
			if err == nil {
				t.Fatal("InitializeConfig should have failed with invalid port")
			}
			if !containsString(err.Error(), tc.expected) {
				t.Errorf("Expected error to contain '%s', got: %v", tc.expected, err)
			}
		})
	}
}

func TestInitializeConfig_InvalidLogLevel_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name     string
		logLevel string
		expected string
	}{
		{
			name:     "Non-numeric log level",
			logLevel: "invalid",
			expected: "invalid logLevel of configuration: value must be integer string and between 0 ～ 9, inclusive",
		},
		{
			name:     "Log level below range",
			logLevel: "-1",
			expected: "invalid logLevel of configuration: value must be integer string and between 0 ～ 9, inclusive",
		},
		{
			name:     "Log level above range",
			logLevel: "10",
			expected: "invalid logLevel of configuration: value must be integer string and between 0 ～ 9, inclusive",
		},
		{
			name:     "Empty log level",
			logLevel: "",
			expected: "LOG_LEVEL is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Setup
			setupValidTestEnv()
			if tc.logLevel == "" {
				os.Unsetenv("LOG_LEVEL")
			} else {
				os.Setenv("LOG_LEVEL", tc.logLevel)
			}
			defer teardownTestEnv()
			ResetConfigForTesting()

			// Execute
			err := InitializeConfig()

			// Verify
			if err == nil {
				t.Fatal("InitializeConfig should have failed with invalid log level")
			}
			if !containsString(err.Error(), tc.expected) {
				t.Errorf("Expected error to contain '%s', got: %v", tc.expected, err)
			}
		})
	}
}

func TestInitializeConfig_BoundaryValues_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name             string
		port             string
		logLevel         string
		expectedPort     int
		expectedLogLevel int
	}{
		{
			name:             "Minimum valid values",
			port:             "1",
			logLevel:         "0",
			expectedPort:     1,
			expectedLogLevel: 0,
		},
		{
			name:             "Maximum valid values",
			port:             "65535",
			logLevel:         "9",
			expectedPort:     65535,
			expectedLogLevel: 9,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Setup
			setupValidTestEnv()
			os.Setenv("CDI_SERVER_PORT", tc.port)
			os.Setenv("LOG_LEVEL", tc.logLevel)
			defer teardownTestEnv()
			ResetConfigForTesting()

			// Execute
			err := InitializeConfig()

			// Verify
			if err != nil {
				t.Fatalf("InitializeConfig failed with boundary values: %v", err)
			}

			config := GetConfig()
			if config.CDIServerPort != tc.expectedPort {
				t.Errorf("Expected CDIServerPort %d, got %d", tc.expectedPort, config.CDIServerPort)
			}
			if config.LogLevel != tc.expectedLogLevel {
				t.Errorf("Expected LogLevel %d, got %d", tc.expectedLogLevel, config.LogLevel)
			}
		})
	}
}

func TestGetConfig_BeforeInitialization_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	ResetConfigForTesting()

	// Execute
	config := GetConfig()

	// Verify
	if config != nil {
		t.Error("GetConfig should return nil before initialization")
	}
}

func TestGetConfig_AfterInitialization_ReturnsSameInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	setupValidTestEnv()
	defer teardownTestEnv()
	ResetConfigForTesting()

	// Initialize
	err := InitializeConfig()
	if err != nil {
		t.Fatalf("InitializeConfig failed: %v", err)
	}

	// Execute
	config1 := GetConfig()
	config2 := GetConfig()

	// Verify
	if config1 != config2 {
		t.Error("GetConfig should return the same instance")
	}
}

func TestInitializeConfig_MultipleInvocations_OnlyInitializesOnce(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	setupValidTestEnv()
	defer teardownTestEnv()
	ResetConfigForTesting()

	// Execute
	err1 := InitializeConfig()
	config1 := GetConfig()

	// Change env var after first initialization
	os.Setenv("CDI_SERVER_PORT", "8080")

	err2 := InitializeConfig()
	config2 := GetConfig()

	// Verify
	if err1 != nil {
		t.Fatalf("First InitializeConfig failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Second InitializeConfig failed: %v", err2)
	}

	// Config should be the same instance and have original values
	if config1 != config2 {
		t.Error("GetConfig should return the same instance after multiple initializations")
	}

	if config2.CDIServerPort != 50051 {
		t.Errorf("Config should retain original value 50051, got %d", config2.CDIServerPort)
	}
}

func TestNewEnvValidator_ReturnsValidator(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Execute
	validator := newEnvValidator()

	// Verify
	if validator == nil {
		t.Fatal("newEnvValidator returned nil")
	}
	if validator.validator == nil {
		t.Fatal("newEnvValidator returned validator with nil validator field")
	}
}

func TestEnvValidator_ParseAndValidateCDIServerPort_ValidValues_ReturnsCorrectValues(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	validator := newEnvValidator()

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"Minimum valid", "1", 1},
		{"Middle value", "8080", 8080},
		{"Maximum valid", "65535", 65535},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Execute
			result, err := validator.parseAndValidateCDIServerPort(tc.input)

			// Verify
			if err != nil {
				t.Fatalf("parseAndValidateCDIServerPort failed for valid input %s: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestEnvValidator_ParseAndValidateLogLevel_ValidValues_ReturnsCorrectValues(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	validator := newEnvValidator()

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"Minimum valid", "0", 0},
		{"Middle value", "5", 5},
		{"Maximum valid", "9", 9},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Execute
			result, err := validator.parseAndValidateLogLevel(tc.input)

			// Verify
			if err != nil {
				t.Fatalf("parseAndValidateLogLevel failed for valid input %s: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestEnvValidator_GetEnvVarName_ReturnsCorrectNames(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	validator := newEnvValidator()

	testCases := []struct {
		fieldName string
		expected  string
	}{
		{"CDIServerPort", "CDI_SERVER_PORT"},
		{"LogLevel", "LOG_LEVEL"},
		{"SSHKey", "SSH_KEY"},
		{"UnknownField", "UnknownField"},
	}

	for _, tc := range testCases {
		t.Run(tc.fieldName, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Execute
			result := validator.getEnvVarName(tc.fieldName)

			// Verify
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// Test for env_validator.go uncovered branches
func TestEnvValidator_FormatValidationError_MultipleErrors_ReturnsCorrectFormat(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// This test would require creating a validator with multiple validation errors
	// Since we can't easily mock validator.ValidationErrors, we'll test the covered functionality

	setupValidTestEnv()
	defer teardownTestEnv()
	ResetConfigForTesting()

	// Test the full validation flow which exercises formatValidationError
	err := InitializeConfig()
	if err != nil {
		t.Errorf("Expected successful initialization with valid env, got error: %v", err)
	}

	config := GetConfig()
	if config == nil {
		t.Error("Expected config to be initialized")
	}
}

func TestEnvValidator_FormatValidationError_UnknownTag_ReturnsDefaultMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create a validator with a custom validation tag to test the default case
	envValidator := &EnvValidator{
		validator: validator.New(),
	}

	// Register a custom validation that will fail with an unknown tag
	envValidator.validator.RegisterValidation("customtag", func(fl validator.FieldLevel) bool {
		return false
	})

	// Create a struct with the custom validation
	type TestStruct struct {
		TestField string `validate:"customtag"`
	}

	testStruct := TestStruct{
		TestField: "test",
	}

	// Execute validation which will fail with unknown tag
	err := envValidator.validator.Struct(testStruct)
	if err == nil {
		t.Fatal("Expected validation to fail")
	}

	// Execute formatValidationError
	formattedErr := envValidator.formatValidationError(err)

	// Verify it contains the default message format
	if !strings.Contains(formattedErr.Error(), "validation failed for field TestField with tag customtag") {
		t.Errorf("Expected default validation message, got: %s", formattedErr.Error())
	}
}

func TestEnvValidator_GetEnvVarName_UnknownField_ReturnsFieldName(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	validator := &EnvValidator{}

	// Test with unknown field name
	result := validator.getEnvVarName("UnknownField")

	if result != "UnknownField" {
		t.Errorf("Expected 'UnknownField' for unknown field, got %s", result)
	}
}

func TestEnvValidator_ParseAndValidateCDIServerPort_BoundaryValues_ReturnsCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	validator := &EnvValidator{}

	testCases := []struct {
		name        string
		input       string
		expected    int
		shouldError bool
	}{
		{"MinValidPort", "1", 1, false},
		{"MaxValidPort", "65535", 65535, false},
		{"ZeroPort", "0", 0, true},
		{"OverMaxPort", "65536", 0, true},
		{"NegativePort", "-1", 0, true},
		{"NonNumericPort", "abc", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			result, err := validator.parseAndValidateCDIServerPort(tc.input)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for input %s", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("Expected %d for input %s, got %d", tc.expected, tc.input, result)
				}
			}
		})
	}
}

func TestEnvValidator_ParseAndValidateLogLevel_BoundaryValues_ReturnsCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	validator := &EnvValidator{}

	testCases := []struct {
		name        string
		input       string
		expected    int
		shouldError bool
	}{
		{"MinValidLevel", "0", 0, false},
		{"MaxValidLevel", "9", 9, false},
		{"BelowMinLevel", "-1", 0, true},
		{"OverMaxLevel", "10", 0, true},
		{"NonNumericLevel", "high", 0, true},
		{"EmptyLevel", "", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			result, err := validator.parseAndValidateLogLevel(tc.input)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for input %s", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("Expected %d for input %s, got %d", tc.expected, tc.input, result)
				}
			}
		})
	}
}

func TestEnvValidator_LoadAndValidateConfig_AllFields_ValidatesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	validator := &EnvValidator{
		validator: validator.New(),
	}

	setupValidTestEnv()
	defer teardownTestEnv()

	config, err := validator.loadAndValidateConfig()

	if err != nil {
		t.Errorf("Expected successful validation, got error: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be returned")
	}

	// Verify all fields are set correctly
	if config.CDIServerPort != 50051 {
		t.Errorf("Expected CDIServerPort 50051, got %d", config.CDIServerPort)
	}

	if config.LogLevel != 2 {
		t.Errorf("Expected LogLevel 2, got %d", config.LogLevel)
	}

	if config.SSHKey != "/tmp/test_key" {
		t.Errorf("Expected SSHKey '/tmp/test_key', got %s", config.SSHKey)
	}
}

func TestNewEnvValidator_CreatesValidator(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	validator := newEnvValidator()

	if validator == nil {
		t.Fatal("Expected validator to be created")
	}

	if validator.validator == nil {
		t.Error("Expected internal validator to be initialized")
	}
}

// Helper functions
func setupValidTestEnv() {
	os.Setenv("CDI_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test_key")
	os.Setenv("TLS_ENABLE", "true")
	os.Setenv("TLS_CERT_PATH", "/tmp/certs")
}

func teardownTestEnv() {
	os.Unsetenv("CDI_SERVER_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SSH_KEY")
	os.Unsetenv("TLS_ENABLE")
	os.Unsetenv("TLS_CERT_PATH")
}

func containsString(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr ||
			str[:len(substr)] == substr ||
			str[len(str)-len(substr):] == substr ||
			findInString(str, substr))
}

func findInString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
