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
	"testing"

	proto "maas_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf

	"google.golang.org/grpc/codes"
)

// TestDetailCode_Constants_HaveCorrectValues tests that DetailCode constants have expected values
func TestDetailCode_Constants_HaveCorrectValues(t *testing.T) {
	// Arrange & Assert
	testCases := []struct {
		name     string
		value    proto.DetailCode
		expected int32
	}{
		{"proto.DetailCode_IF_PARAMETER_INVALID", proto.DetailCode_IF_PARAMETER_INVALID, 41},
		{"proto.DetailCode_IF_SEQUENCE_ERROR", proto.DetailCode_IF_SEQUENCE_ERROR, 52},
		{"proto.DetailCode_IF_CANCEL_UNAVAILABLE", proto.DetailCode_IF_CANCEL_UNAVAILABLE, 53},
		{"proto.DetailCode_MAAS_ENVIRONMENT_ERROR", proto.DetailCode_MAAS_ENVIRONMENT_ERROR, 61},
		{"proto.DetailCode_MAAS_RESPONSE_INVALID", proto.DetailCode_MAAS_RESPONSE_INVALID, 81},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if int32(tc.value) != tc.expected {
				t.Errorf("Expected %s to be %d, got %d", tc.name, tc.expected, int32(tc.value))
			}
		})
	}
}

// TestErrorMessageToJSON_ValidErrorMessage_ReturnsJSONString tests ErrorMessageToJSON with valid ErrorMessage
func TestErrorMessageToJSON_ValidErrorMessage_ReturnsJSONString(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
		Message:    "test error message",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled common.ErrorMessage
	err := json.Unmarshal([]byte(result), &unmarshaled)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Verify content
	if unmarshaled.ErrorCode != errorMessage.ErrorCode {
		t.Errorf("Expected ErrorCode %d, got %d", errorMessage.ErrorCode, unmarshaled.ErrorCode)
	}
	if unmarshaled.DetailCode != errorMessage.DetailCode {
		t.Errorf("Expected DetailCode %d, got %d", errorMessage.DetailCode, unmarshaled.DetailCode)
	}
	if unmarshaled.Message != errorMessage.Message {
		t.Errorf("Expected Message %s, got %s", errorMessage.Message, unmarshaled.Message)
	}
}

// TestErrorMessageToJSON_NilErrorMessage_ReturnsNullJSON tests ErrorMessageToJSON with nil ErrorMessage
func TestErrorMessageToJSON_NilErrorMessage_ReturnsNullJSON(t *testing.T) {
	// Arrange
	var errorMessage *common.ErrorMessage = nil

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result != "null" {
		t.Errorf("Expected 'null', got %s", result)
	}
}

// TestErrorMessageToJSON_EmptyErrorMessage_ReturnsJSONString tests ErrorMessageToJSON with empty ErrorMessage
func TestErrorMessageToJSON_EmptyErrorMessage_ReturnsJSONString(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled common.ErrorMessage
	err := json.Unmarshal([]byte(result), &unmarshaled)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Verify default values
	if unmarshaled.ErrorCode != 0 {
		t.Errorf("Expected ErrorCode 0, got %d", unmarshaled.ErrorCode)
	}
	if unmarshaled.DetailCode != 0 {
		t.Errorf("Expected DetailCode 0, got %d", unmarshaled.DetailCode)
	}
	if unmarshaled.Message != "" {
		t.Errorf("Expected empty Message, got %s", unmarshaled.Message)
	}
}

// TestErrorMessageToJSON_ErrorMessageWithSpecialCharacters_ReturnsJSONString tests ErrorMessageToJSON with special characters
func TestErrorMessageToJSON_ErrorMessageWithSpecialCharacters_ReturnsJSONString(t *testing.T) {
	// Arrange
	specialMessage := "Error with \"quotes\" and \n newlines \t tabs \\ backslashes"
	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.InvalidArgument),
		DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
		Message:    specialMessage,
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled common.ErrorMessage
	err := json.Unmarshal([]byte(result), &unmarshaled)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Verify the special characters are properly escaped and preserved
	if unmarshaled.Message != specialMessage {
		t.Errorf("Expected Message %s, got %s", specialMessage, unmarshaled.Message)
	}
}

// TestErrorMessageToJSON_ErrorMessageWithUnicodeCharacters_ReturnsJSONString tests ErrorMessageToJSON with Unicode characters
func TestErrorMessageToJSON_ErrorMessageWithUnicodeCharacters_ReturnsJSONString(t *testing.T) {
	// Arrange
	unicodeMessage := "エラーメッセージ 错误信息 сообщение об ошибке 🚫"
	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: int32(proto.DetailCode_MAAS_RESPONSE_INVALID),
		Message:    unicodeMessage,
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled common.ErrorMessage
	err := json.Unmarshal([]byte(result), &unmarshaled)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Verify Unicode characters are properly preserved
	if unmarshaled.Message != unicodeMessage {
		t.Errorf("Expected Message %s, got %s", unicodeMessage, unmarshaled.Message)
	}
}

// TestErrorMessageToJSON_ErrorMessageWithMaxValues_ReturnsJSONString tests ErrorMessageToJSON with maximum int32 values
func TestErrorMessageToJSON_ErrorMessageWithMaxValues_ReturnsJSONString(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  2147483647, // max int32
		DetailCode: 2147483647, // max int32
		Message:    "Maximum values test",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled common.ErrorMessage
	err := json.Unmarshal([]byte(result), &unmarshaled)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Verify values
	if unmarshaled.ErrorCode != 2147483647 {
		t.Errorf("Expected ErrorCode 2147483647, got %d", unmarshaled.ErrorCode)
	}
	if unmarshaled.DetailCode != 2147483647 {
		t.Errorf("Expected DetailCode 2147483647, got %d", unmarshaled.DetailCode)
	}
}

// TestErrorMessageToJSON_ErrorMessageWithMinValues_ReturnsJSONString tests ErrorMessageToJSON with minimum int32 values
func TestErrorMessageToJSON_ErrorMessageWithMinValues_ReturnsJSONString(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  -2147483648, // min int32
		DetailCode: -2147483648, // min int32
		Message:    "Minimum values test",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled common.ErrorMessage
	err := json.Unmarshal([]byte(result), &unmarshaled)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Verify values
	if unmarshaled.ErrorCode != -2147483648 {
		t.Errorf("Expected ErrorCode -2147483648, got %d", unmarshaled.ErrorCode)
	}
	if unmarshaled.DetailCode != -2147483648 {
		t.Errorf("Expected DetailCode -2147483648, got %d", unmarshaled.DetailCode)
	}
}

// TestErrorMessageToJSON_AllDetailCodes_ReturnsJSONString tests ErrorMessageToJSON with all defined DetailCode constants
func TestErrorMessageToJSON_AllDetailCodes_ReturnsJSONString(t *testing.T) {
	// Arrange
	testCases := []struct {
		name       string
		detailCode proto.DetailCode
	}{
		{"proto.DetailCode_IF_PARAMETER_INVALID", proto.DetailCode_IF_PARAMETER_INVALID},
		{"proto.DetailCode_IF_SEQUENCE_ERROR", proto.DetailCode_IF_SEQUENCE_ERROR},
		{"proto.DetailCode_IF_CANCEL_UNAVAILABLE", proto.DetailCode_IF_CANCEL_UNAVAILABLE},
		{"proto.DetailCode_MAAS_ENVIRONMENT_ERROR", proto.DetailCode_MAAS_ENVIRONMENT_ERROR},
		{"proto.DetailCode_MAAS_RESPONSE_INVALID", proto.DetailCode_MAAS_RESPONSE_INVALID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			errorMessage := &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(tc.detailCode),
				Message:    "Test message for " + tc.name,
			}

			// Act
			result := ErrorMessageToJSON(errorMessage)

			// Assert
			if result == "" {
				t.Error("Expected non-empty JSON string")
			}

			// Verify it's valid JSON by unmarshaling
			var unmarshaled common.ErrorMessage
			err := json.Unmarshal([]byte(result), &unmarshaled)
			if err != nil {
				t.Errorf("Expected valid JSON, got error: %v", err)
			}

			// Verify DetailCode
			if unmarshaled.DetailCode != int32(tc.detailCode) {
				t.Errorf("Expected DetailCode %d, got %d", int32(tc.detailCode), unmarshaled.DetailCode)
			}
		})
	}
}

// TestErrorMessageToJSON_LongMessage_ReturnsJSONString tests ErrorMessageToJSON with very long message
func TestErrorMessageToJSON_LongMessage_ReturnsJSONString(t *testing.T) {
	// Arrange
	longMessage := ""
	for i := 0; i < 1000; i++ {
		longMessage += "This is a very long error message. "
	}

	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: int32(proto.DetailCode_MAAS_RESPONSE_INVALID),
		Message:    longMessage,
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled common.ErrorMessage
	err := json.Unmarshal([]byte(result), &unmarshaled)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Verify the long message is preserved
	if unmarshaled.Message != longMessage {
		t.Error("Expected long message to be preserved")
	}
}

// TestErrorMessageToJSON_JSONStructure_MatchesExpectedFormat tests that generated JSON has expected structure
func TestErrorMessageToJSON_JSONStructure_MatchesExpectedFormat(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(codes.InvalidArgument),
		DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
		Message:    "validation failed",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	// Check that JSON contains expected fields
	expectedFields := []string{
		`"errorCode":3`,   // codes.InvalidArgument = 3
		`"detailCode":41`, // proto.DetailCode_IF_PARAMETER_INVALID = 41
		`"message":"validation failed"`,
	}

	for _, field := range expectedFields {
		if !contains(result, field) {
			t.Errorf("Expected JSON to contain %s, got: %s", field, result)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOfSubstring(s, substr) != -1)))
}

// Helper function to find index of substring
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
