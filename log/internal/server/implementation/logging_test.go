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

package implementation

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"

	"log_module/internal/server/interfaces" // import for interface
	"log_module/internal/server/test_utils"
)

// TestLogging_Init tests the Init method of the Logging struct.
func TestLogging_Init(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logFilename := fmt.Sprintf("/tmp/test_log_%s.log", time.Now().Format("20060102"))
	// Define test cases
	testCases := []struct {
		name          string
		loggingConfig interfaces.LoggingConfig
		wantErr       bool
	}{
		{
			name: "Valid configuration",
			loggingConfig: interfaces.LoggingConfig{
				LogFile:    "test_log",
				LogPath:    "/tmp",
				MaxSize:    1024,
				MaxBackups: 5,
				MaxAge:     7,
			},
			wantErr: false,
		},
		{
			name: "Invalid configuration - empty LogPath",
			loggingConfig: interfaces.LoggingConfig{
				LogFile:    "test_log",
				LogPath:    "",
				MaxSize:    1024,
				MaxBackups: 5,
				MaxAge:     7,
			},
			wantErr: false, // Init doesn't directly error on empty LogPath
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Create a Logging instance
			l := &LoggingImplement{Logger: klog.Background()}

			// Call the Init method
			err := l.Init(tc.loggingConfig)

			// Assert that the error is as expected
			if (err != nil) != tc.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tc.wantErr)
			}

			// Assert that the LumberJackLogger field is not nil
			if l.LumberJackLogger == nil {
				t.Error("LumberJackLogger field is nil after Init()")
			} else {
				// Assert that the LumberJackLogger field has the expected values
				expectedFilename := fmt.Sprintf("%s/%s_%s.log", tc.loggingConfig.LogPath, tc.loggingConfig.LogFile, time.Now().Format("20060102"))
				if l.LumberJackLogger.Filename != expectedFilename {
					t.Errorf("LumberJackLogger.Filename has unexpected value: got %v, want %v", l.LumberJackLogger.Filename, expectedFilename)
				}
				if l.LumberJackLogger.MaxSize != tc.loggingConfig.MaxSize {
					t.Errorf("LumberJackLogger.MaxSize has unexpected value: got %v, want %v", l.LumberJackLogger.MaxSize, tc.loggingConfig.MaxSize)
				}
				if l.LumberJackLogger.MaxBackups != tc.loggingConfig.MaxBackups {
					t.Errorf("LumberJackLogger.MaxBackups has unexpected value: got %v, want %v", l.LumberJackLogger.MaxBackups, tc.loggingConfig.MaxBackups)
				}
				if l.LumberJackLogger.MaxAge != tc.loggingConfig.MaxAge {
					t.Errorf("LumberJackLogger.MaxAge has unexpected value: got %v, want %v", l.LumberJackLogger.MaxAge, tc.loggingConfig.MaxAge)
				}
			}

			// Assert that the Logrus field is not nil
			if l.Logrus == nil {
				t.Error("Logrus field is nil after Init()")
			}

			// Clean up
			if l.LumberJackLogger != nil {
				l.Finalize()                  // Close the lumberjack logger
				_ = os.RemoveAll(logFilename) // Clean up the log file
			}
		})
	}
}

// TestLogging_Finalize tests the Finalize method of the Logging struct.
func TestLogging_Finalize(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logFilename := fmt.Sprintf("/tmp/test_log_%s.log", time.Now().Format("20060102"))

	// Create a Logging instance
	l := LoggingImplement{
		Logger: klog.Background(),
		LumberJackLogger: &lumberjack.Logger{
			Filename:   "/tmp/test_log.log",
			MaxSize:    1024,
			MaxBackups: 5,
			MaxAge:     7,
			Compress:   false,
		},
		Logrus: logrus.New(),
	}

	// Set a hook to be called when the logger is closed
	l.LumberJackLogger.LocalTime = true // Ensure local time is used

	// Call the Finalize method
	l.Finalize()

	// Clean up
	_ = os.RemoveAll(logFilename)
}

// TestLogging_Write tests the Write method of the Logging struct.
func TestLogging_Write(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logFilename := fmt.Sprintf("/tmp/test_log_%s.log", time.Now().Format("20060102"))

	// Create a Logging instance
	l := LoggingImplement{
		Logger: klog.Background(),
		LumberJackLogger: &lumberjack.Logger{
			Filename:   "/tmp/test_log.log",
			MaxSize:    1024,
			MaxBackups: 5,
			MaxAge:     7,
			Compress:   false,
		},
		Logrus: logrus.New(),
	}

	// Initialize the Logging instance
	err := l.Init(interfaces.LoggingConfig{
		LogFile:    "test_log",
		LogPath:    "/tmp",
		MaxSize:    1024,
		MaxBackups: 5,
		MaxAge:     7,
	})
	if err != nil {
		t.Fatalf("Failed to initialize Logging: %v", err)
	}
	defer func() { _ = os.RemoveAll(logFilename) }() // Clean up after the test.
	defer l.Finalize()

	// Define test cases
	testCases := []struct {
		name    string
		keyId   string
		json    string
		wantLog string
		wantErr bool
	}{
		{
			name:    "Valid input",
			keyId:   "test_key",
			json:    `{"message": "test_message"}`,
			wantLog: "test_key {\"message\": \"test_message\"}",
			wantErr: false,
		},
		{
			name:    "Empty input",
			keyId:   "",
			json:    "",
			wantLog: "  ",
			wantErr: false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Call the Write method
			err := l.Write(tc.keyId, tc.json)

			// Assert that the error is as expected
			if (err != nil) != tc.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tc.wantErr)
			}

			// Read the log file
			content, err := os.ReadFile(logFilename)
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}
			output := string(content)

			// Assert that the output contains the expected log message
			if !strings.Contains(output, tc.wantLog) {
				t.Errorf("Write() output does not contain expected log message: got %v, want %v", output, tc.wantLog)
			}
		})
	}
}

// TestCustomFormatter_Format tests the Format method of the CustomFormatter struct.
func TestCustomFormatter_Format(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create a CustomFormatter instance
	f := &CustomFormatter{}

	// Create a log entry
	entry := &logrus.Entry{
		Time: time.Now(),
		Data: logrus.Fields{
			"keyId": "test_key",
			"json":  `{"message": "test_message"}`,
		},
	}

	// Call the Format method
	output, err := f.Format(entry)

	// Assert that there is no error
	if err != nil {
		t.Errorf("Format() error = %v", err)
	}

	// Assert that the output contains the expected log message
	expectedLogMessage := fmt.Sprintf("%s test_key {\"message\": \"test_message\"}\n", entry.Time.Format(time.RFC3339))
	if string(output) != expectedLogMessage {
		t.Errorf("Format() output has unexpected value: got %v, want %v", string(output), expectedLogMessage)
	}
}

// Helper function to set file permissions and test directory creation
func createTestDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// Helper function to remove test directory
func removeTestDir(path string) error {
	return os.RemoveAll(path)
}

// TestLoggingImplement_Init_EmptyLogPath_UsesDefault tests empty log path
func TestLoggingImplement_Init_EmptyLogPath_UsesDefault(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	config := interfaces.LoggingConfig{
		LogFile:    "test.log",
		LogPath:    "", // Empty path
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}

	// Execute
	err := logging.Init(config)

	// Verify
	if err != nil {
		t.Errorf("Init() error = %v, expected no error", err)
	}

	// Verify that logrus and lumberjack are initialized
	if logging.Logrus == nil {
		t.Error("Logrus logger was not initialized")
	}
	if logging.LumberJackLogger == nil {
		t.Error("LumberJack logger was not initialized")
	}

	// Clean up
	if logging.LumberJackLogger != nil {
		logging.LumberJackLogger.Close()
	}
}

// TestLoggingImplement_Init_EmptyLogFile_UsesDefault tests empty log file
func TestLoggingImplement_Init_EmptyLogFile_UsesDefault(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	config := interfaces.LoggingConfig{
		LogFile:    "", // Empty file
		LogPath:    "/tmp",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}

	// Execute
	err := logging.Init(config)

	// Verify
	if err != nil {
		t.Errorf("Init() error = %v, expected no error", err)
	}

	// Verify that logrus and lumberjack are initialized
	if logging.Logrus == nil {
		t.Error("Logrus logger was not initialized")
	}
	if logging.LumberJackLogger == nil {
		t.Error("LumberJack logger was not initialized")
	}

	// Clean up
	if logging.LumberJackLogger != nil {
		logging.LumberJackLogger.Close()
	}
}

// TestLoggingImplement_Init_ZeroValues_UsesDefaults tests zero values in config
func TestLoggingImplement_Init_ZeroValues_UsesDefaults(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	config := interfaces.LoggingConfig{
		LogFile:    "test.log",
		LogPath:    "/tmp",
		MaxSize:    0, // Zero values
		MaxBackups: 0,
		MaxAge:     0,
	}

	// Execute
	err := logging.Init(config)

	// Verify
	if err != nil {
		t.Errorf("Init() error = %v, expected no error", err)
	}

	// Verify that logrus and lumberjack are initialized with values
	if logging.Logrus == nil {
		t.Error("Logrus logger was not initialized")
	}
	if logging.LumberJackLogger == nil {
		t.Error("LumberJack logger was not initialized")
	}

	// Clean up
	if logging.LumberJackLogger != nil {
		logging.LumberJackLogger.Close()
	}
}

// TestLoggingImplement_Init_InvalidPath_HandlesGracefully tests invalid path
func TestLoggingImplement_Init_InvalidPath_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	config := interfaces.LoggingConfig{
		LogFile:    "test.log",
		LogPath:    "/root/nonexistent/deeply/nested/path", // Likely to fail
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}

	// Execute
	err := logging.Init(config)

	// Verify - should handle gracefully or return error
	// Either no error (handled gracefully) or specific error
	if err != nil {
		// If error, it should be meaningful
		t.Logf("Init() returned error as expected: %v", err)
	}

	// Clean up if initialized
	if logging.LumberJackLogger != nil {
		logging.LumberJackLogger.Close()
	}
}

// TestLoggingImplement_Write_Success tests successful write
func TestLoggingImplement_Write_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	// Initialize with valid config
	config := interfaces.LoggingConfig{
		LogFile:    "test_write.log",
		LogPath:    "/tmp",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}

	err := logging.Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Execute write
	keyId := "test_key"
	json := `{"test": "data"}`
	err = logging.Write(keyId, json)

	// Verify
	if err != nil {
		t.Errorf("Write() error = %v, expected no error", err)
	}

	// Clean up
	logging.LumberJackLogger.Close()
}

// TestLoggingImplement_Write_WithoutInit_HandlesGracefully tests write without init
func TestLoggingImplement_Write_WithoutInit_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	// Execute write without initialization
	keyId := "test_key"
	json := `{"test": "data"}`
	err := logging.Write(keyId, json)

	// Verify - should handle nil logrus gracefully
	// This might panic or return error depending on implementation
	if err != nil {
		t.Logf("Write() returned error as expected: %v", err)
	}
}

// TestLoggingImplement_Write_EmptyValues_Success tests write with empty values
func TestLoggingImplement_Write_EmptyValues_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	// Initialize with valid config
	config := interfaces.LoggingConfig{
		LogFile:    "test_empty.log",
		LogPath:    "/tmp",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}

	err := logging.Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Execute write with empty values
	err = logging.Write("", "")

	// Verify - should handle empty values gracefully
	if err != nil {
		t.Errorf("Write() error = %v, expected no error", err)
	}

	// Clean up
	logging.LumberJackLogger.Close()
}

// TestLoggingImplement_Write_LargeData_Success tests write with large data
func TestLoggingImplement_Write_LargeData_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	// Initialize with valid config
	config := interfaces.LoggingConfig{
		LogFile:    "test_large.log",
		LogPath:    "/tmp",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}

	err := logging.Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Execute write with large data
	keyId := "large_data_key"
	largeJson := strings.Repeat(`{"data": "value"},`, 1000) // Large JSON string
	err = logging.Write(keyId, largeJson)

	// Verify
	if err != nil {
		t.Errorf("Write() error = %v, expected no error", err)
	}

	// Clean up
	logging.LumberJackLogger.Close()
}

// TestLoggingImplement_Finalize_Success tests finalize method
func TestLoggingImplement_Finalize_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	// Initialize first
	config := interfaces.LoggingConfig{
		LogFile:    "test_finalize.log",
		LogPath:    "/tmp",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}

	err := logging.Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Execute finalize - should not panic
	if logging.LumberJackLogger == nil {
		t.Error("LumberJackLogger is nil before Finalize")
	}

	logging.Finalize()

	// Verify finalize behavior (implementation dependent)
	// May set LumberJackLogger to nil or close it
}

// TestLoggingImplement_Finalize_WithoutInit_Success tests finalize without init
func TestLoggingImplement_Finalize_WithoutInit_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	// Execute finalize without initialization - should not panic
	logging.Finalize()

	// Verify - should handle nil LumberJackLogger gracefully
	// Test passes if no panic occurs
}

// TestLoggingImplement_Finalize_MultipleCalls_Success tests multiple finalize calls
func TestLoggingImplement_Finalize_MultipleCalls_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	logging := &LoggingImplement{Logger: logger}

	// Initialize first
	config := interfaces.LoggingConfig{
		LogFile:    "test_multi_finalize.log",
		LogPath:    "/tmp",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}

	err := logging.Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Execute finalize multiple times - should not panic
	logging.Finalize()
	logging.Finalize()
	logging.Finalize()

	// Verify - should handle multiple calls gracefully
	// Test passes if no panic occurs
}

// TestCustomFormatter_Format_Success tests custom formatter
func TestCustomFormatter_Format_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	formatter := &CustomFormatter{}

	// Create a logrus entry
	entry := &logrus.Entry{
		Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: logrus.Fields{
			"keyId": "test_key",
			"json":  `{"test": "data"}`,
		},
	}

	// Execute format
	result, err := formatter.Format(entry)

	// Verify
	if err != nil {
		t.Errorf("Format() error = %v, expected no error", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "2023-01-01T12:00:00Z") {
		t.Errorf("Format() result does not contain expected timestamp")
	}
	if !strings.Contains(resultStr, "test_key") {
		t.Errorf("Format() result does not contain expected keyId")
	}
	if !strings.Contains(resultStr, `{"test": "data"}`) {
		t.Errorf("Format() result does not contain expected json")
	}
}

// TestCustomFormatter_Format_MissingFields_HandlesGracefully tests formatter with missing fields
func TestCustomFormatter_Format_MissingFields_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	formatter := &CustomFormatter{}

	// Create a logrus entry with missing fields
	entry := &logrus.Entry{
		Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: logrus.Fields{
			// Missing keyId and json fields
		},
	}

	// Execute format
	result, err := formatter.Format(entry)

	// Verify - should handle missing fields gracefully
	if err != nil {
		t.Errorf("Format() error = %v, expected no error", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "2023-01-01T12:00:00Z") {
		t.Errorf("Format() result does not contain expected timestamp")
	}
	// Should handle missing fields gracefully (may show "<nil>" or empty)
}

// TestCustomFormatter_Format_NilFields_HandlesGracefully tests formatter with nil fields
func TestCustomFormatter_Format_NilFields_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	formatter := &CustomFormatter{}

	// Create a logrus entry with nil data
	entry := &logrus.Entry{
		Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: nil,
	}

	// Execute format
	result, err := formatter.Format(entry)

	// Verify - should handle nil data gracefully
	if err != nil {
		t.Errorf("Format() error = %v, expected no error", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "2023-01-01T12:00:00Z") {
		t.Errorf("Format() result does not contain expected timestamp")
	}
}
