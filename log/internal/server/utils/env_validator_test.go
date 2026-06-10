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

	"github.com/stretchr/testify/assert"
)

// Helper function to set environment variable and return cleanup function
func setEnvVar(key, value string) func() {
	oldValue := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		if oldValue != "" {
			os.Setenv(key, oldValue)
		} else {
			os.Unsetenv(key)
		}
	}
}

// Helper function to clear all required environment variables
func clearAllEnvVars() func() {
	envVars := []string{
		"LOG_LEVEL", "INTERVAL",
		"IPMI_LOGFILE", "IPMI_LOGPATH", "IPMI_MAXSIZE", "IPMI_MAXBACKUPS", "IPMI_MAXAGE",
		"CDI_LOGFILE", "CDI_LOGPATH", "CDI_MAXSIZE", "CDI_MAXBACKUPS", "CDI_MAXAGE",
		"DB_HOST", "DB_PORT", "DB_NAME", "DB_USERNAME", "SECRET_NAME", "SECRET_NAMESPACE",
	}

	oldValues := make(map[string]string)
	for _, key := range envVars {
		oldValues[key] = os.Getenv(key)
		os.Unsetenv(key)
	}

	return func() {
		for key, value := range oldValues {
			if value != "" {
				os.Setenv(key, value)
			}
		}
	}
}

// Helper function to set all valid environment variables
func setValidEnvVars() func() {
	envVars := map[string]string{
		"LOG_LEVEL":       "2",
		"INTERVAL":        "60",
		"IPMI_LOGFILE":    "ipmi.log",
		"IPMI_LOGPATH":    "/var/log/ipmi",
		"IPMI_MAXSIZE":    "100",
		"IPMI_MAXBACKUPS": "5",
		"IPMI_MAXAGE":     "7",
		"CDI_LOGFILE":     "cdi.log",
		"CDI_LOGPATH":     "/var/log/cdi",
		"CDI_MAXSIZE":     "200",
		"CDI_MAXBACKUPS":  "10",
		"CDI_MAXAGE":      "14",
		"DB_URL":          "https://localhost:3000",
	}

	oldValues := make(map[string]string)
	for key := range envVars {
		oldValues[key] = os.Getenv(key)
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}

	return func() {
		for key, value := range oldValues {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}

// TestInitializeConfig_ValidEnvironment_Success tests successful configuration initialization
func TestInitializeConfig_ValidEnvironment_Success(t *testing.T) {
	// Setup
	ResetConfigForTesting()
	cleanup := setValidEnvVars()
	defer cleanup()

	// Execute
	err := InitializeConfig()

	// Verify
	assert.NoError(t, err)
	config := GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "2", config.LogLevel)
	assert.Equal(t, 60, config.Interval)
	assert.Equal(t, "ipmi.log", config.IpmiLogFile)
	assert.Equal(t, "/var/log/ipmi", config.IpmiLogPath)
	assert.Equal(t, 100, config.IpmiMaxSize)
	assert.Equal(t, 5, config.IpmiMaxBackups)
	assert.Equal(t, 7, config.IpmiMaxAge)
	assert.Equal(t, "cdi.log", config.CdiLogFile)
	assert.Equal(t, "/var/log/cdi", config.CdiLogPath)
	assert.Equal(t, 200, config.CdiMaxSize)
	assert.Equal(t, 10, config.CdiMaxBackups)
	assert.Equal(t, 14, config.CdiMaxAge)
	assert.Equal(t, "https://localhost:3000", config.DbAccessURL)
}

// TestInitializeConfig_MissingRequiredEnvVar_ReturnsError tests missing environment variables
func TestInitializeConfig_MissingRequiredEnvVar_ReturnsError(t *testing.T) {
	// Setup
	ResetConfigForTesting()
	cleanup := clearAllEnvVars()
	defer cleanup()

	// Execute
	err := InitializeConfig()

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is required")
	config := GetConfig()
	assert.Nil(t, config)
}

// TestInitializeConfig_InvalidLogLevel_ReturnsError tests invalid log level values
func TestInitializeConfig_InvalidLogLevel_ReturnsError(t *testing.T) {
	testCases := []struct {
		name     string
		logLevel string
	}{
		{"Negative", "-1"},
		{"TooHigh", "10"},
		{"NonNumeric", "invalid"},
		{"Empty", ""},
	}

	for _, tc := range testCases {
		t.Run("LogLevel_"+tc.name, func(t *testing.T) {
			// Setup
			ResetConfigForTesting()
			cleanup := setValidEnvVars()
			defer cleanup()
			defer setEnvVar("LOG_LEVEL", tc.logLevel)()

			// Execute
			err := InitializeConfig()

			// Verify
			if tc.logLevel == "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "LOG_LEVEL is required")
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid logLevel")
			}
		})
	}
}

// TestInitializeConfig_InvalidInterval_ReturnsError tests invalid interval values
func TestInitializeConfig_InvalidInterval_ReturnsError(t *testing.T) {
	testCases := []struct {
		name     string
		interval string
	}{
		{"Negative", "-1"},
		{"TooLow", "0"},
		{"TooHigh", "3601"},
		{"NonNumeric", "invalid"},
	}

	for _, tc := range testCases {
		t.Run("Interval_"+tc.name, func(t *testing.T) {
			// Setup
			ResetConfigForTesting()
			cleanup := setValidEnvVars()
			defer cleanup()
			defer setEnvVar("INTERVAL", tc.interval)()

			// Execute
			err := InitializeConfig()

			// Verify
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid interval")
		})
	}
}

// TestInitializeConfig_InvalidMaxSize_ReturnsError tests invalid max size values
func TestInitializeConfig_InvalidMaxSize_ReturnsError(t *testing.T) {
	testCases := []struct {
		name   string
		envVar string
		value  string
	}{
		{"IPMI_Zero", "IPMI_MAXSIZE", "0"},
		{"IPMI_TooHigh", "IPMI_MAXSIZE", "10241"},
		{"IPMI_NonNumeric", "IPMI_MAXSIZE", "invalid"},
		{"CDI_Zero", "CDI_MAXSIZE", "0"},
		{"CDI_TooHigh", "CDI_MAXSIZE", "10241"},
		{"CDI_NonNumeric", "CDI_MAXSIZE", "invalid"},
	}

	for _, tc := range testCases {
		t.Run("MaxSize_"+tc.name, func(t *testing.T) {
			// Setup
			ResetConfigForTesting()
			cleanup := setValidEnvVars()
			defer cleanup()
			defer setEnvVar(tc.envVar, tc.value)()

			// Execute
			err := InitializeConfig()

			// Verify
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "maxSize")
		})
	}
}

// TestInitializeConfig_InvalidMaxBackups_ReturnsError tests invalid max backups values
func TestInitializeConfig_InvalidMaxBackups_ReturnsError(t *testing.T) {
	testCases := []struct {
		name   string
		envVar string
		value  string
	}{
		{"IPMI_Zero", "IPMI_MAXBACKUPS", "0"},
		{"IPMI_TooHigh", "IPMI_MAXBACKUPS", "32"},
		{"IPMI_NotInt", "IPMI_MAXBACKUPS", "invalid"},
		{"CDI_Zero", "CDI_MAXBACKUPS", "0"},
		{"CDI_TooHigh", "CDI_MAXBACKUPS", "32"},
		{"CDI_NotInt", "CDI_MAXBACKUPS", "invalid"},
	}

	for _, tc := range testCases {
		t.Run("MaxBackups_"+tc.name, func(t *testing.T) {
			// Setup
			ResetConfigForTesting()
			cleanup := setValidEnvVars()
			defer cleanup()
			defer setEnvVar(tc.envVar, tc.value)()

			// Execute
			err := InitializeConfig()

			// Verify
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "maxBackups")
		})
	}
}

// TestInitializeConfig_InvalidMaxAge_ReturnsError tests invalid max age values
func TestInitializeConfig_InvalidMaxAge_ReturnsError(t *testing.T) {
	testCases := []struct {
		name   string
		envVar string
		value  string
	}{
		{"IPMI_Zero", "IPMI_MAXAGE", "0"},
		{"IPMI_TooHigh", "IPMI_MAXAGE", "32"},
		{"IPMI_NotInt", "IPMI_MAXAGE", "invalid"},
		{"CDI_Zero", "CDI_MAXAGE", "0"},
		{"CDI_TooHigh", "CDI_MAXAGE", "32"},
		{"CDI_NotInt", "CDI_MAXAGE", "invalid"},
	}

	for _, tc := range testCases {
		t.Run("MaxAge_"+tc.name, func(t *testing.T) {
			// Setup
			ResetConfigForTesting()
			cleanup := setValidEnvVars()
			defer cleanup()
			defer setEnvVar(tc.envVar, tc.value)()

			// Execute
			err := InitializeConfig()

			// Verify
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "maxAge")
		})
	}
}

// TestInitializeConfig_CalledMultipleTimes_ReturnsSameResult tests singleton behavior
func TestInitializeConfig_CalledMultipleTimes_ReturnsSameResult(t *testing.T) {
	// Setup
	ResetConfigForTesting()
	cleanup := setValidEnvVars()
	defer cleanup()

	// Execute multiple times
	err1 := InitializeConfig()
	err2 := InitializeConfig()
	err3 := InitializeConfig()

	config1 := GetConfig()
	config2 := GetConfig()

	// Verify
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Same(t, config1, config2) // Should be the same instance
}

// TestGetConfig_BeforeInitialize_ReturnsNil tests GetConfig before initialization
func TestGetConfig_BeforeInitialize_ReturnsNil(t *testing.T) {
	// Setup
	ResetConfigForTesting()

	// Execute
	config := GetConfig()

	// Verify
	assert.Nil(t, config)
}

// TestResetConfigForTesting_ResetsGlobalState tests reset functionality
func TestResetConfigForTesting_ResetsGlobalState(t *testing.T) {
	// Setup - initialize config first
	cleanup := setValidEnvVars()
	defer cleanup()
	err := InitializeConfig()
	assert.NoError(t, err)
	assert.NotNil(t, GetConfig())

	// Execute
	ResetConfigForTesting()

	// Verify
	assert.Nil(t, GetConfig())

	// Should be able to initialize again
	err = InitializeConfig()
	assert.NoError(t, err)
	assert.NotNil(t, GetConfig())
}

// TestEnvValidator_NewEnvValidator_ReturnsInstance tests validator creation
func TestEnvValidator_NewEnvValidator_ReturnsInstance(t *testing.T) {
	// Execute
	validator := newEnvValidator()

	// Verify
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.validator)
}

// TestEnvValidator_ParseAndValidateLogLevel_ValidValues_Success tests log level parsing
func TestEnvValidator_ParseAndValidateLogLevel_ValidValues_Success(t *testing.T) {
	// Setup
	validator := newEnvValidator()

	testCases := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

	for _, tc := range testCases {
		t.Run("LogLevel_"+tc, func(t *testing.T) {
			// Execute
			result, err := validator.parseAndValidateLogLevel(tc)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, tc, result)
		})
	}
}

// TestEnvValidator_ParseAndValidateInterval_BoundaryValues_Success tests interval boundary values
func TestEnvValidator_ParseAndValidateInterval_BoundaryValues_Success(t *testing.T) {
	// Setup
	validator := newEnvValidator()

	testCases := []struct {
		input    string
		expected int
	}{
		{"1", 1},
		{"2", 2},
		{"3599", 3599},
		{"3600", 3600},
	}

	for _, tc := range testCases {
		t.Run("Interval_"+tc.input, func(t *testing.T) {
			// Execute
			result, err := validator.parseAndValidateInterval(tc.input)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestEnvValidator_ParseAndValidateMaxSize_BoundaryValues_Success tests max size boundary values
func TestEnvValidator_ParseAndValidateMaxSize_BoundaryValues_Success(t *testing.T) {
	// Setup
	validator := newEnvValidator()

	testCases := []struct {
		input    string
		expected int
	}{
		{"1", 1},
		{"100", 100},
		{"10240", 10240},
	}

	for _, tc := range testCases {
		t.Run("MaxSize_"+tc.input, func(t *testing.T) {
			// Execute
			result, err := validator.parseAndValidateMaxSize("test.maxSize", tc.input)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestEnvValidator_GetEnvVarName_AllFields_ReturnsCorrectNames tests field name mapping
func TestEnvValidator_GetEnvVarName_AllFields_ReturnsCorrectNames(t *testing.T) {
	// Setup
	validator := newEnvValidator()

	testCases := map[string]string{
		"LogLevel":       "LOG_LEVEL",
		"Interval":       "INTERVAL",
		"IpmiLogFile":    "IPMI_LOGFILE",
		"IpmiLogPath":    "IPMI_LOGPATH",
		"IpmiMaxSize":    "IPMI_MAXSIZE",
		"IpmiMaxBackups": "IPMI_MAXBACKUPS",
		"IpmiMaxAge":     "IPMI_MAXAGE",
		"CdiLogFile":     "CDI_LOGFILE",
		"CdiLogPath":     "CDI_LOGPATH",
		"CdiMaxSize":     "CDI_MAXSIZE",
		"CdiMaxBackups":  "CDI_MAXBACKUPS",
		"CdiMaxAge":      "CDI_MAXAGE",
		"DbAccessURL":    "DB_URL",
		"UnknownField":   "UnknownField",
	}

	for fieldName, expectedEnvVar := range testCases {
		t.Run("Field_"+fieldName, func(t *testing.T) {
			// Execute
			result := validator.getEnvVarName(fieldName)

			// Verify
			assert.Equal(t, expectedEnvVar, result)
		})
	}
}

// TestConfig_ConcurrentAccess_ThreadSafe tests thread safety
func TestConfig_ConcurrentAccess_ThreadSafe(t *testing.T) {
	// Setup
	ResetConfigForTesting()
	cleanup := setValidEnvVars()
	defer cleanup()

	// Execute - simulate concurrent access
	var wg sync.WaitGroup
	errors := make(chan error, 10)
	configs := make(chan *Config, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := InitializeConfig()
			errors <- err
			configs <- GetConfig()
		}()
	}

	wg.Wait()
	close(errors)
	close(configs)

	// Verify - all calls should succeed and return the same config
	var firstConfig *Config
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
		}
	}
	assert.Equal(t, 0, errorCount)

	configCount := 0
	for config := range configs {
		if firstConfig == nil {
			firstConfig = config
		} else {
			assert.Same(t, firstConfig, config)
		}
		configCount++
	}
	assert.Equal(t, 10, configCount)
}
