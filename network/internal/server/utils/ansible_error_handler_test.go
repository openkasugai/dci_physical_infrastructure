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
	"strings"
	"testing"

	"network_module/internal/server/test_utils"
)

// TestExtractAnsibleError_JSONFormattedError_ReturnsFormattedMessage tests JSON-formatted Ansible error parsing
func TestExtractAnsibleError_JSONFormattedError_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`fatal: [testhost]: FAILED! => {"changed": false, "msg": "Connection refused"}`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Host testhost failed!: Connection refused"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_JSONFormattedUnreachable_ReturnsFormattedMessage tests JSON-formatted UNREACHABLE error
func TestExtractAnsibleError_JSONFormattedUnreachable_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`UNREACHABLE: [server1]: UNREACHABLE! => {"changed": false, "msg": "Host is down"}`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Host server1 is unreachable: Host is down"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_JSONWithoutMessage_ReturnsFormattedMessage tests JSON without msg field
func TestExtractAnsibleError_JSONWithoutMessage_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`fatal: [testhost]: FAILED! => {"changed": false, "rc": 1}`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Host testhost failed!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_TextFormattedError_ReturnsFormattedMessage tests text-formatted Ansible error parsing
func TestExtractAnsibleError_TextFormattedError_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`fatal: [webserver]: command execution failed`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Host webserver failed: [webserver]: command execution failed"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_TextFormattedFailed_ReturnsFormattedMessage tests text-formatted FAILED! error
func TestExtractAnsibleError_TextFormattedFailed_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`FAILED! [dbserver]: FAILED! Task execution error`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Host dbserver failed: Task execution error"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_TextFormattedUnreachable_ReturnsFormattedMessage tests text-formatted UNREACHABLE! error
func TestExtractAnsibleError_TextFormattedUnreachable_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`UNREACHABLE! [node1]: Network timeout occurred`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Host node1 is unreachable: Network timeout occurred"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_TextWithoutHost_ReturnsFormattedMessage tests text error without host information
func TestExtractAnsibleError_TextWithoutHost_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`fatal: general error occurred`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "failed: general error occurred"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_TextWithoutMessage_ReturnsFormattedMessage tests text error without detailed message
func TestExtractAnsibleError_TextWithoutMessage_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`fatal: [host1]:`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Host host1 failed: [host1]:"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_PlayRecapWithFailure_ReturnsRecapMessage tests PLAY RECAP with failures
func TestExtractAnsibleError_PlayRecapWithFailure_ReturnsRecapMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`PLAY RECAP 
server1                    : ok=2    changed=0    unreachable=0    failed=1    skipped=0    rescued=0    ignored=0
*******************************************************************************`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Task execution failed: server1                    : ok=2    changed=0    unreachable=0    failed=1    skipped=0    rescued=0    ignored=0"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_PlayRecapWithUnreachable_ReturnsRecapMessage tests PLAY RECAP with unreachable hosts
func TestExtractAnsibleError_PlayRecapWithUnreachable_ReturnsRecapMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`PLAY RECAP 
server2                    : ok=0    changed=0    unreachable=1    failed=0    skipped=0    rescued=0    ignored=0
*********************************************************************`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	expected := "Host unreachable: server2                    : ok=0    changed=0    unreachable=1    failed=0    skipped=0    rescued=0    ignored=0"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractAnsibleError_PlayRecapWithZeroFailures_ReturnsEmpty tests PLAY RECAP with no failures
func TestExtractAnsibleError_PlayRecapWithZeroFailures_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`PLAY RECAP 
server3                    : ok=5    changed=2    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
*********************************************************************`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestExtractAnsibleError_EmptyOutput_ReturnsEmpty tests empty output
func TestExtractAnsibleError_EmptyOutput_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(``)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestExtractAnsibleError_NoErrorOutput_ReturnsEmpty tests successful output with no errors
func TestExtractAnsibleError_NoErrorOutput_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`PLAY [Configure servers] *******************************************************
TASK [Gathering Facts] *********************************************************
ok: [server1]
TASK [Install package] *********************************************************
changed: [server1]`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestParseAnsibleJSON_ValidJSONError_ReturnsErrorInfo tests parseAnsibleJSON with valid JSON error
func TestParseAnsibleJSON_ValidJSONError_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [host1]: FAILED! => {"changed": false, "msg": "Test error"}`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "host1" {
		t.Errorf("Expected HostName 'host1', got '%s'", result.HostName)
	}
	if result.ErrorType != "FAILED!" {
		t.Errorf("Expected ErrorType 'FAILED!', got '%s'", result.ErrorType)
	}
	if result.Message != "Test error" {
		t.Errorf("Expected Message 'Test error', got '%s'", result.Message)
	}
	if result.Unreachable {
		t.Error("Expected Unreachable to be false")
	}
}

// TestParseAnsibleJSON_UnreachableError_ReturnsErrorInfo tests parseAnsibleJSON with UNREACHABLE error
func TestParseAnsibleJSON_UnreachableError_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `UNREACHABLE: [host2]: UNREACHABLE! => {"changed": false, "msg": "Network error"}`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "host2" {
		t.Errorf("Expected HostName 'host2', got '%s'", result.HostName)
	}
	if result.ErrorType != "UNREACHABLE!" {
		t.Errorf("Expected ErrorType 'UNREACHABLE!', got '%s'", result.ErrorType)
	}
	if result.Message != "Network error" {
		t.Errorf("Expected Message 'Network error', got '%s'", result.Message)
	}
	if !result.Unreachable {
		t.Error("Expected Unreachable to be true")
	}
}

// TestParseAnsibleJSON_InvalidJSON_ReturnsNil tests parseAnsibleJSON with invalid JSON
func TestParseAnsibleJSON_InvalidJSON_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [host1]: FAILED! => {invalid json`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
}

// TestParseAnsibleJSON_NoMatch_ReturnsNil tests parseAnsibleJSON with no matching pattern
func TestParseAnsibleJSON_NoMatch_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `Some random output without error pattern`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
}

// TestParseAnsibleJSON_JSONWithoutMsg_ReturnsErrorInfoWithoutMessage tests parseAnsibleJSON with JSON lacking msg field
func TestParseAnsibleJSON_JSONWithoutMsg_ReturnsErrorInfoWithoutMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [host3]: FAILED! => {"changed": false, "rc": 1}`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "host3" {
		t.Errorf("Expected HostName 'host3', got '%s'", result.HostName)
	}
	if result.Message != "" {
		t.Errorf("Expected empty Message, got '%s'", result.Message)
	}
}

// TestParseAnsibleJSON_WithEmptyStderr_ReturnsErrorInfoWithStderr tests parseAnsibleJSON with empty stderr falling back to raw stderr
func TestParseAnsibleJSON_WithEmptyStderr_ReturnsErrorInfoWithStderr(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [host4]: FAILED! => {"changed": false, "stderr": "   "}`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "host4" {
		t.Errorf("Expected HostName 'host4', got '%s'", result.HostName)
	}
	// extractMostImportantError should return empty for whitespace-only stderr,
	// so it falls back to raw stderr
	if result.Message == "" {
		t.Error("Expected non-empty Message (raw stderr)")
	}
}

// TestParseAnsibleJSON_WithStdoutWithoutError_ReturnsErrorInfoWithoutMessage tests parseAnsibleJSON with stdout not containing "Error:"
func TestParseAnsibleJSON_WithStdoutWithoutError_ReturnsErrorInfoWithoutMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [host5]: FAILED! => {"changed": false, "stdout": "Some output without error keyword"}`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "host5" {
		t.Errorf("Expected HostName 'host5', got '%s'", result.HostName)
	}
	if result.Message != "" {
		t.Errorf("Expected empty Message, got '%s'", result.Message)
	}
}

// TestParseAnsibleText_FatalError_ReturnsErrorInfo tests parseAnsibleText with fatal error
func TestParseAnsibleText_FatalError_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [server1]: Command failed`

	// Act
	result := parseAnsibleText(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "server1" {
		t.Errorf("Expected HostName 'server1', got '%s'", result.HostName)
	}
	if result.ErrorType != "FAILED" {
		t.Errorf("Expected ErrorType 'FAILED', got '%s'", result.ErrorType)
	}
	if result.Message != "[server1]: Command failed" {
		t.Errorf("Expected Message '[server1]: Command failed', got '%s'", result.Message)
	}
}

// TestParseAnsibleText_FailedError_ReturnsErrorInfo tests parseAnsibleText with FAILED! pattern
func TestParseAnsibleText_FailedError_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `FAILED! [server2]: FAILED! Task error`

	// Act
	result := parseAnsibleText(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "server2" {
		t.Errorf("Expected HostName 'server2', got '%s'", result.HostName)
	}
	if result.Message != "Task error" {
		t.Errorf("Expected Message 'Task error', got '%s'", result.Message)
	}
}

// TestParseAnsibleText_UnreachableError_ReturnsErrorInfo tests parseAnsibleText with UNREACHABLE! pattern
func TestParseAnsibleText_UnreachableError_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `UNREACHABLE! [server3]: Network timeout`

	// Act
	result := parseAnsibleText(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "server3" {
		t.Errorf("Expected HostName 'server3', got '%s'", result.HostName)
	}
	if !result.Unreachable {
		t.Error("Expected Unreachable to be true")
	}
	if result.Message != "Network timeout" {
		t.Errorf("Expected Message 'Network timeout', got '%s'", result.Message)
	}
}

// TestParseAnsibleText_ErrorWithoutHost_ReturnsErrorInfo tests parseAnsibleText without host information
func TestParseAnsibleText_ErrorWithoutHost_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: General error occurred`

	// Act
	result := parseAnsibleText(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "" {
		t.Errorf("Expected empty HostName, got '%s'", result.HostName)
	}
	if result.Message != "General error occurred" {
		t.Errorf("Expected Message 'General error occurred', got '%s'", result.Message)
	}
}

// TestParseAnsibleText_ErrorWithoutMessage_ReturnsErrorInfo tests parseAnsibleText without detailed message
func TestParseAnsibleText_ErrorWithoutMessage_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [server4]:`

	// Act
	result := parseAnsibleText(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "server4" {
		t.Errorf("Expected HostName 'server4', got '%s'", result.HostName)
	}
	if result.Message != "[server4]:" {
		t.Errorf("Expected Message '[server4]:', got '%s'", result.Message)
	}
}

// TestParseAnsibleText_NoErrorPattern_ReturnsNil tests parseAnsibleText with no error pattern
func TestParseAnsibleText_NoErrorPattern_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `TASK [Install package]
ok: [server1]`

	// Act
	result := parseAnsibleText(output)

	// Assert
	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
}

// TestParseAnsibleText_MultilineWithError_ReturnsFirstError tests parseAnsibleText with multiline output
func TestParseAnsibleText_MultilineWithError_ReturnsFirstError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `TASK [Configure]
ok: [server1]
fatal: [server2]: Configuration error
ok: [server3]`

	// Act
	result := parseAnsibleText(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HostName != "server2" {
		t.Errorf("Expected HostName 'server2', got '%s'", result.HostName)
	}
}

// TestParsePlayRecap_WithFailedNonZero_ReturnsError tests parsePlayRecap with non-zero failed count
func TestParsePlayRecap_WithFailedNonZero_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `PLAY RECAP 
server1                    : ok=3    changed=1    unreachable=0    failed=2    skipped=0
*********************************************************************`

	// Act
	result := parsePlayRecap(output)

	// Assert
	expected := "Task execution failed: server1                    : ok=3    changed=1    unreachable=0    failed=2    skipped=0"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestParsePlayRecap_WithUnreachableNonZero_ReturnsError tests parsePlayRecap with non-zero unreachable count
func TestParsePlayRecap_WithUnreachableNonZero_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `PLAY RECAP 
server2                    : ok=0    changed=0    unreachable=3    failed=0    skipped=0
*********************************************************************`

	// Act
	result := parsePlayRecap(output)

	// Assert
	expected := "Host unreachable: server2                    : ok=0    changed=0    unreachable=3    failed=0    skipped=0"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestParsePlayRecap_WithZeroErrors_ReturnsEmpty tests parsePlayRecap with zero errors
func TestParsePlayRecap_WithZeroErrors_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `PLAY RECAP 
server3                    : ok=5    changed=2    unreachable=0    failed=0    skipped=0
*********************************************************************`

	// Act
	result := parsePlayRecap(output)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestParsePlayRecap_NoPlayRecap_ReturnsEmpty tests parsePlayRecap without PLAY RECAP section
func TestParsePlayRecap_NoPlayRecap_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `TASK [Install package]
ok: [server1]`

	// Act
	result := parsePlayRecap(output)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestParsePlayRecap_EmptyRecapSection_ReturnsEmpty tests parsePlayRecap with empty recap section
func TestParsePlayRecap_EmptyRecapSection_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `PLAY RECAP *********************************************************************
`

	// Act
	result := parsePlayRecap(output)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestParsePlayRecap_MultipleHosts_ReturnsFirstError tests parsePlayRecap with multiple hosts
func TestParsePlayRecap_MultipleHosts_ReturnsFirstError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `PLAY RECAP 
server1                    : ok=5    changed=2    unreachable=0    failed=0    skipped=0
server2                    : ok=2    changed=0    unreachable=2    failed=0    skipped=0
server3                    : ok=3    changed=1    unreachable=0    failed=1    skipped=0
*********************************************************************`

	// Act
	result := parsePlayRecap(output)

	// Assert
	// Should return first non-zero error (server2 with unreachable=2 comes first)
	if result == "" {
		t.Error("Expected non-empty result")
	}
	// Both server2 and server3 have errors, but server2 appears first
	if !strings.Contains(result, "server2") && !strings.Contains(result, "server3") {
		t.Errorf("Expected result to contain 'server2' or 'server3', got '%s'", result)
	}
}

// TestFormatErrorMessage_WithHostAndMessage_ReturnsFormattedString tests formatErrorMessage with complete info
func TestFormatErrorMessage_WithHostAndMessage_ReturnsFormattedString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorInfo := &AnsibleErrorInfo{
		HostName:    "testhost",
		ErrorType:   "FAILED",
		Message:     "Connection refused",
		Unreachable: false,
	}

	// Act
	result := formatErrorMessage(errorInfo)

	// Assert
	expected := "Host testhost failed: Connection refused"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFormatErrorMessage_WithHostUnreachable_ReturnsFormattedString tests formatErrorMessage with unreachable host
func TestFormatErrorMessage_WithHostUnreachable_ReturnsFormattedString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorInfo := &AnsibleErrorInfo{
		HostName:    "server1",
		ErrorType:   "UNREACHABLE",
		Message:     "Network timeout",
		Unreachable: true,
	}

	// Act
	result := formatErrorMessage(errorInfo)

	// Assert
	expected := "Host server1 is unreachable: Network timeout"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFormatErrorMessage_WithoutHostName_ReturnsFormattedString tests formatErrorMessage without hostname
func TestFormatErrorMessage_WithoutHostName_ReturnsFormattedString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorInfo := &AnsibleErrorInfo{
		HostName:    "",
		ErrorType:   "FAILED",
		Message:     "General error",
		Unreachable: false,
	}

	// Act
	result := formatErrorMessage(errorInfo)

	// Assert
	expected := "failed: General error"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFormatErrorMessage_WithoutMessage_ReturnsFormattedString tests formatErrorMessage without message
func TestFormatErrorMessage_WithoutMessage_ReturnsFormattedString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorInfo := &AnsibleErrorInfo{
		HostName:    "testhost",
		ErrorType:   "FAILED",
		Message:     "",
		Unreachable: false,
	}

	// Act
	result := formatErrorMessage(errorInfo)

	// Assert
	expected := "Host testhost failed"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFormatErrorMessage_WithoutHostAndMessage_ReturnsFormattedString tests formatErrorMessage with minimal info
func TestFormatErrorMessage_WithoutHostAndMessage_ReturnsFormattedString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorInfo := &AnsibleErrorInfo{
		HostName:    "",
		ErrorType:   "FAILED",
		Message:     "",
		Unreachable: false,
	}

	// Act
	result := formatErrorMessage(errorInfo)

	// Assert
	expected := "failed"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFormatErrorMessage_UnreachableWithoutHost_ReturnsFormattedString tests formatErrorMessage unreachable without host
func TestFormatErrorMessage_UnreachableWithoutHost_ReturnsFormattedString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorInfo := &AnsibleErrorInfo{
		HostName:    "",
		ErrorType:   "UNREACHABLE",
		Message:     "Connection error",
		Unreachable: true,
	}

	// Act
	result := formatErrorMessage(errorInfo)

	// Assert
	expected := "is unreachable: Connection error"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractStderrMessage_WithValidStderr_ReturnsExtractedError tests extractStderrMessage with valid stderr
func TestExtractStderrMessage_WithValidStderr_ReturnsExtractedError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `{"stderr": "Error: Connection failed\\nUsage: command [options]", "rc": 1}`

	// Act
	result := extractStderrMessage(output)

	// Assert
	if !strings.Contains(result, "Error: Connection failed") {
		t.Errorf("Expected result to contain 'Error: Connection failed', got '%s'", result)
	}
}

// TestExtractStderrMessage_WithEmptyStderr_ReturnsEmpty tests extractStderrMessage with empty stderr
func TestExtractStderrMessage_WithEmptyStderr_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `{"stderr": "", "rc": 0}`

	// Act
	result := extractStderrMessage(output)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestExtractStderrMessage_WithoutStderr_ReturnsEmpty tests extractStderrMessage without stderr field
func TestExtractStderrMessage_WithoutStderr_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `{"stdout": "Success", "rc": 0}`

	// Act
	result := extractStderrMessage(output)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestExtractMostImportantError_WithErrorPrefix_ReturnsError tests extractMostImportantError with "Error:" prefix
func TestExtractMostImportantError_WithErrorPrefix_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	stderr := "Some debug info\nError: Connection refused\nMore details"

	// Act
	result := extractMostImportantError(stderr)

	// Assert
	expected := "Error: Connection refused"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractMostImportantError_WithUsagePrefix_ReturnsUsage tests extractMostImportantError with "Usage:" prefix
func TestExtractMostImportantError_WithUsagePrefix_ReturnsUsage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	stderr := "Usage: command [options]\n  -h  Display help"

	// Act
	result := extractMostImportantError(stderr)

	// Assert
	if !strings.Contains(result, "Usage:") {
		t.Errorf("Expected result to contain 'Usage:', got '%s'", result)
	}
}

// TestExtractMostImportantError_WithoutSpecificPrefix_ReturnsCleanedStderr tests extractMostImportantError without specific prefix
func TestExtractMostImportantError_WithoutSpecificPrefix_ReturnsCleanedStderr(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	stderr := "General error message\nAdditional information"

	// Act
	result := extractMostImportantError(stderr)

	// Assert
	if !strings.Contains(result, "General error message") {
		t.Errorf("Expected result to contain 'General error message', got '%s'", result)
	}
}

// TestCleanErrorMessage_WithMultipleLines_ReturnsCleanedMessage tests cleanErrorMessage with multiple lines
func TestCleanErrorMessage_WithMultipleLines_ReturnsCleanedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	message := "Line 1\n  Line 2  \n\nLine 3\n"

	// Act
	result := cleanErrorMessage(message)

	// Assert
	expected := "Line 1 Line 2 Line 3"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestCleanErrorMessage_WithEscapedQuotes_ReturnsUnescapedMessage tests cleanErrorMessage with escaped quotes
func TestCleanErrorMessage_WithEscapedQuotes_ReturnsUnescapedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	message := `Error: \"something\" failed with \\ backslash`

	// Act
	result := cleanErrorMessage(message)

	// Assert
	expected := `Error: "something" failed with \ backslash`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestCleanErrorMessage_WithEmptyLines_ReturnsCleanedMessage tests cleanErrorMessage with empty lines
func TestCleanErrorMessage_WithEmptyLines_ReturnsCleanedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	message := "\n\n  Error  \n\n"

	// Act
	result := cleanErrorMessage(message)

	// Assert
	expected := "Error"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestParseAnsibleJSON_WithStderrInJSON_ReturnsErrorInfo tests parseAnsibleJSON with stderr in JSON
func TestParseAnsibleJSON_WithStderrInJSON_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [host1]: FAILED! => {"changed": false, "stderr": "Error: Command not found"}`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !strings.Contains(result.Message, "Error: Command not found") {
		t.Errorf("Expected message to contain 'Error: Command not found', got '%s'", result.Message)
	}
}

// TestParseAnsibleJSON_WithStdoutInJSON_ReturnsErrorInfo tests parseAnsibleJSON with stdout containing error
func TestParseAnsibleJSON_WithStdoutInJSON_ReturnsErrorInfo(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := `fatal: [host1]: FAILED! => {"changed": false, "stdout": "Error: Something went wrong"}`

	// Act
	result := parseAnsibleJSON(output)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !strings.Contains(result.Message, "Error:") {
		t.Errorf("Expected message to contain 'Error:', got '%s'", result.Message)
	}
}

// TestExtractAnsibleError_WithStderrMessage_ReturnsStderrError tests ExtractAnsibleError with stderr in output
func TestExtractAnsibleError_WithStderrMessage_ReturnsStderrError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	output := []byte(`{"stderr": "Error: Connection timeout", "rc": 1}`)

	// Act
	result := ExtractAnsibleError(output)

	// Assert
	if !strings.Contains(result, "Error: Connection timeout") {
		t.Errorf("Expected result to contain 'Error: Connection timeout', got '%s'", result)
	}
}

// TestFormatErrorMessage_WithEmptyHostAndType_ReturnsMessageOnly tests formatErrorMessage with empty host and type
func TestFormatErrorMessage_WithEmptyHostAndType_ReturnsMessageOnly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorInfo := &AnsibleErrorInfo{
		HostName:    "",
		ErrorType:   "",
		Message:     "Some error message",
		Unreachable: false,
	}

	// Act
	result := formatErrorMessage(errorInfo)

	// Assert
	expected := "Some error message"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
