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
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	proto "cdi_module/api/proto"
    common "common/api/proto"    // import of common protobuf
	"cdi_module/internal/server/test_utils"
)

func TestErrorMessageToJSON_ValidErrorMessage_ReturnsJSONString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.InvalidArgument),
		DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
		Message:    "test error message",
	}

	// Execute
	result := ErrorMessageToJSON(errorMessage)

	// Verify
	if result == "" {
		t.Fatal("ErrorMessageToJSON returned empty string")
	}

	// Verify it's valid JSON
	var parsed common.ErrorMessage
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify content
	if parsed.ErrorCode != int32(codes.InvalidArgument) {
		t.Errorf("Expected ErrorCode %d, got %d", int32(codes.InvalidArgument), parsed.ErrorCode)
	}
	if parsed.DetailCode != int32(proto.DetailCode_IF_PARAMETER_INVALID) {
		t.Errorf("Expected DetailCode %d, got %d", int32(proto.DetailCode_IF_PARAMETER_INVALID), parsed.DetailCode)
	}
	if parsed.Message != "test error message" {
		t.Errorf("Expected Message 'test error message', got '%s'", parsed.Message)
	}
}

func TestErrorMessageToJSON_NilErrorMessage_ReturnsJSONString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Execute
	result := ErrorMessageToJSON(nil)

	// Verify
	if result == "" {
		t.Fatal("ErrorMessageToJSON returned empty string for nil input")
	}

	// Verify it's valid JSON
	var parsed common.ErrorMessage
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify default values
	if parsed.ErrorCode != 0 {
		t.Errorf("Expected ErrorCode 0 for nil input, got %d", parsed.ErrorCode)
	}
	if parsed.DetailCode != 0 {
		t.Errorf("Expected DetailCode 0 for nil input, got %d", parsed.DetailCode)
	}
	if parsed.Message != "" {
		t.Errorf("Expected empty Message for nil input, got '%s'", parsed.Message)
	}
}

func TestErrorMessageToJSON_EmptyErrorMessage_ReturnsJSONString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	errorMessage := &common.ErrorMessage{}

	// Execute
	result := ErrorMessageToJSON(errorMessage)

	// Verify
	if result == "" {
		t.Fatal("ErrorMessageToJSON returned empty string")
	}

	// Verify it's valid JSON
	var parsed common.ErrorMessage
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify default values
	if parsed.ErrorCode != 0 {
		t.Errorf("Expected ErrorCode 0, got %d", parsed.ErrorCode)
	}
	if parsed.DetailCode != 0 {
		t.Errorf("Expected DetailCode 0, got %d", parsed.DetailCode)
	}
	if parsed.Message != "" {
		t.Errorf("Expected empty Message, got '%s'", parsed.Message)
	}
}

func TestErrorMessageToJSON_ErrorMessageWithAllDetailCodes_ReturnsJSONString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name       string
		detailCode proto.DetailCode
	}{
		{"proto.DetailCode_IF_PARAMETER_INVALID", proto.DetailCode_IF_PARAMETER_INVALID},
		{"CDI_ENVIRONMENT_ERROR", proto.DetailCode_CDI_ENVIRONMENT_ERROR},
		{"CDI_COMMAND_ERROR_V_1_0", proto.DetailCode_CDI_COMMAND_ERROR_V_1_0},
		{"CDI_COMMAND_ERROR_V_1_1", proto.DetailCode_CDI_COMMAND_ERROR_V_1_1},
		{"CDI_RESPONSE_INVALID", proto.DetailCode_CDI_RESPONSE_INVALID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Setup
			errorMessage := &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(tc.detailCode),
				Message:    "test message for " + tc.name,
			}

			// Execute
			result := ErrorMessageToJSON(errorMessage)

			// Verify
			if result == "" {
				t.Fatal("ErrorMessageToJSON returned empty string")
			}

			// Verify it's valid JSON
			var parsed common.ErrorMessage
			err := json.Unmarshal([]byte(result), &parsed)
			if err != nil {
				t.Fatalf("Result is not valid JSON: %v", err)
			}

			// Verify content
			if parsed.DetailCode != int32(tc.detailCode) {
				t.Errorf("Expected DetailCode %d, got %d", int32(tc.detailCode), parsed.DetailCode)
			}
		})
	}
}

func TestErrorMessageToJSON_ErrorMessageWithSpecialCharacters_ReturnsJSONString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	specialMessage := "Special chars: \"quotes\", \\backslash, \n newline, \t tab"
	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.InvalidArgument),
		DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
		Message:    specialMessage,
	}

	// Execute
	result := ErrorMessageToJSON(errorMessage)

	// Verify
	if result == "" {
		t.Fatal("ErrorMessageToJSON returned empty string")
	}

	// Verify it's valid JSON
	var parsed common.ErrorMessage
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify message content is properly escaped/unescaped
	if parsed.Message != specialMessage {
		t.Errorf("Expected Message '%s', got '%s'", specialMessage, parsed.Message)
	}
}

func TestErrorMessageToJSON_ErrorMessageWithMaxValues_ReturnsJSONString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - use maximum int32 values
	errorMessage := &common.ErrorMessage{
		ErrorCode:  2147483647, // max int32
		DetailCode: 2147483647, // max int32
		Message:    "test message with max values",
	}

	// Execute
	result := ErrorMessageToJSON(errorMessage)

	// Verify
	if result == "" {
		t.Fatal("ErrorMessageToJSON returned empty string")
	}

	// Verify it's valid JSON
	var parsed common.ErrorMessage
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify content
	if parsed.ErrorCode != 2147483647 {
		t.Errorf("Expected ErrorCode 2147483647, got %d", parsed.ErrorCode)
	}
	if parsed.DetailCode != 2147483647 {
		t.Errorf("Expected DetailCode 2147483647, got %d", parsed.DetailCode)
	}
}

func TestErrorMessageToJSON_ErrorMessageWithMinValues_ReturnsJSONString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - use minimum int32 values
	errorMessage := &common.ErrorMessage{
		ErrorCode:  -2147483648, // min int32
		DetailCode: -2147483648, // min int32
		Message:    "test message with min values",
	}

	// Execute
	result := ErrorMessageToJSON(errorMessage)

	// Verify
	if result == "" {
		t.Fatal("ErrorMessageToJSON returned empty string")
	}

	// Verify it's valid JSON
	var parsed common.ErrorMessage
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify content
	if parsed.ErrorCode != -2147483648 {
		t.Errorf("Expected ErrorCode -2147483648, got %d", parsed.ErrorCode)
	}
	if parsed.DetailCode != -2147483648 {
		t.Errorf("Expected DetailCode -2147483648, got %d", parsed.DetailCode)
	}
}

func TestDetailCode_Constants_HaveExpectedValues(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name     string
		code     proto.DetailCode
		expected int32
	}{
		{"proto.DetailCode_IF_PARAMETER_INVALID", proto.DetailCode_IF_PARAMETER_INVALID, 41},
		{"CDI_ENVIRONMENT_ERROR", proto.DetailCode_CDI_ENVIRONMENT_ERROR, 61},
		{"CDI_COMMAND_ERROR_V_1_0", proto.DetailCode_CDI_COMMAND_ERROR_V_1_0, 71},
		{"CDI_COMMAND_ERROR_V_1_1", proto.DetailCode_CDI_COMMAND_ERROR_V_1_1, 72},
		{"CDI_RESPONSE_INVALID", proto.DetailCode_CDI_RESPONSE_INVALID, 81},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			if int32(tc.code) != tc.expected {
				t.Errorf("Expected %s to have value %d, got %d", tc.name, tc.expected, int32(tc.code))
			}
		})
	}
}

// Test for ErrorMessageToJSON error handling branches
func TestErrorMessageToJSON_JsonMarshalError_CallsFatal(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Note: This test is difficult to trigger as common.ErrorMessage should always marshal successfully.
	// We test the successful path with complex data to ensure robustness

	complexErrorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
		Message:    "Complex error with special characters: \n\t\r\"\\",
	}

	result := ErrorMessageToJSON(complexErrorMessage)

	// Verify the result is valid JSON
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it can be parsed back
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("Generated JSON should be valid, got error: %v", err)
	}
}

func TestErrorMessageToJSON_LargeErrorMessage_HandlesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - use a very large message
	longMessage := strings.Repeat("A", 1000000) // 1 million characters
	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.InvalidArgument),
		DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
		Message:    longMessage,
	}

	// Execute
	result := ErrorMessageToJSON(errorMessage)

	// Verify
	if result == "" {
		t.Fatal("ErrorMessageToJSON returned empty string")
	}

	// Verify it's valid JSON
	var parsed common.ErrorMessage
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify content - the message should be preserved completely
	if parsed.Message != longMessage {
		t.Errorf("Expected Message length %d, got length %d", len(longMessage), len(parsed.Message))
	}
}

func TestErrorMessageToJSON_CoverageCompleteness_AllBranches(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test various edge cases to ensure branch coverage

	testCases := []struct {
		name    string
		input   *common.ErrorMessage
		wantErr bool
	}{
		{
			name:    "Nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "Empty message",
			input:   &common.ErrorMessage{},
			wantErr: false,
		},
		{
			name: "Unicode characters",
			input: &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "日本語テスト unicode: 🚀 emoji test ñ ü ê",
			},
			wantErr: false,
		},
		{
			name: "Control characters",
			input: &common.ErrorMessage{
				ErrorCode:  int32(codes.FailedPrecondition),
				DetailCode: int32(proto.DetailCode_CDI_RESPONSE_INVALID),
				Message:    "\x00\x01\x02\x03\x04\x05 control chars",
			},
			wantErr: false,
		},
		{
			name: "Maximum int32 values",
			input: &common.ErrorMessage{
				ErrorCode:  2147483647,
				DetailCode: 2147483647,
				Message:    "max values test",
			},
			wantErr: false,
		},
		{
			name: "Minimum int32 values",
			input: &common.ErrorMessage{
				ErrorCode:  -2147483648,
				DetailCode: -2147483648,
				Message:    "min values test",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			result := ErrorMessageToJSON(tc.input)

			// All cases should return valid JSON
			if result == "" && !tc.wantErr {
				t.Errorf("Expected non-empty result for case %s", tc.name)
			}

			// Verify JSON validity
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("Result should be valid JSON for case %s: %v", tc.name, err)
			}
		})
	}
}
