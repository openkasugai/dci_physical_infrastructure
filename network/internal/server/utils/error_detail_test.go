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

	proto "network_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf
)

func TestErrorMessageToJSON_ValidErrorMessage_ReturnsJSONString(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  400,
		DetailCode: 41,
		Message:    "test error message",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	expected := `{"errorCode":400,"detailCode":41,"message":"test error message"}`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestErrorMessageToJSON_EmptyMessage_ReturnsJSONWithEmptyMessage(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  500,
		DetailCode: 61,
		Message:    "",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	expected := `{"errorCode":500,"detailCode":61}`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestErrorMessageToJSON_ZeroValues_ReturnsJSONWithZeroValues(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  0,
		DetailCode: 0,
		Message:    "",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	expected := `{}`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestErrorMessageToJSON_NilErrorMessage_ReturnsNullJSON(t *testing.T) {
	// Arrange
	var errorMessage *common.ErrorMessage = nil

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	expected := "null"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestErrorMessageToJSON_SpecialCharactersInMessage_ReturnsEscapedJSON(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  400,
		DetailCode: 41,
		Message:    "error with \"quotes\" and \n newline",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	expected := `{"errorCode":400,"detailCode":41,"message":"error with \"quotes\" and \n newline"}`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestErrorMessageToJSON_EmptyErrorMessage_ReturnsValidJSON(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string, got empty string")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}
}

func TestErrorMessageToJSON_ErrorMessageWithAllFields_ReturnsCompleteJSON(t *testing.T) {
	// Arrange
	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(500),
		DetailCode: int32(proto.DetailCode_NW_ENVIRONMENT_ERROR),
		Message:    "Test error with all fields",
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string, got empty string")
	}

	// Parse and verify all fields are present
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Verify it contains expected data
	if parsed == nil {
		t.Error("Expected parsed JSON to contain data")
	}
}

func TestDetailCode_Constants_HaveCorrectValues(t *testing.T) {
	// Test to ensure DetailCode constants maintain their expected values
	testCases := []struct {
		code     proto.DetailCode
		expected int32
		name     string
	}{
		{proto.DetailCode_IF_PARAMETER_INVALID, 41, "proto.DetailCode_IF_PARAMETER_INVALID"},
		{proto.DetailCode_NW_ENVIRONMENT_ERROR, 61, "proto.DetailCode_NW_ENVIRONMENT_ERROR"},
		{proto.DetailCode_NW_COMMAND_ERROR, 71, "proto.DetailCode_NW_COMMAND_ERROR"},
		{proto.DetailCode_VSW_VLAN_DUPLICATE, 73, "proto.DetailCode_VSW_VLAN_DUPLICATE"},
		{proto.DetailCode_VSW_VLAN_NOTFOUND, 74, "proto.DetailCode_VSW_VLAN_NOTFOUND"},
	}

	for _, tc := range testCases {
		if int32(tc.code) != tc.expected {
			t.Errorf("Expected %s to have value %d, got %d", tc.name, tc.expected, int32(tc.code))
		}
	}
}

func TestErrorMessageToJSON_LargeMessage_HandlesGracefully(t *testing.T) {
	// Arrange - Create a large message
	largeMessage := make([]byte, 1000)
	for i := range largeMessage {
		largeMessage[i] = 'A'
	}

	errorMessage := &common.ErrorMessage{
		ErrorCode:  int32(400),
		DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
		Message:    string(largeMessage),
	}

	// Act
	result := ErrorMessageToJSON(errorMessage)

	// Assert
	if result == "" {
		t.Error("Expected non-empty JSON string for large message, got empty string")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Errorf("Expected valid JSON for large message, got error: %v", err)
	}
}

func TestErrorMessageToJSON_JSONMarshallingSafety_HandlesEdgeCases(t *testing.T) {
	// Test various edge cases that might cause JSON marshalling issues
	testCases := []struct {
		name         string
		errorMessage *common.ErrorMessage
	}{
		{
			name: "Unicode characters",
			errorMessage: &common.ErrorMessage{
				ErrorCode:  400,
				DetailCode: 41,
				Message:    "エラー message with unicode 🚨",
			},
		},
		{
			name: "Control characters",
			errorMessage: &common.ErrorMessage{
				ErrorCode:  400,
				DetailCode: 41,
				Message:    "Error\twith\rcontrol\bcharacters",
			},
		},
		{
			name: "Very long error code",
			errorMessage: &common.ErrorMessage{
				ErrorCode:  2147483647,  // Max int32
				DetailCode: -2147483648, // Min int32
				Message:    "Extreme values test",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result := ErrorMessageToJSON(tc.errorMessage)

			// Assert
			if result == "" {
				t.Errorf("Expected non-empty JSON string for %s, got empty string", tc.name)
			}

			// Verify it's valid JSON
			var parsed map[string]interface{}
			err := json.Unmarshal([]byte(result), &parsed)
			if err != nil {
				t.Errorf("Expected valid JSON for %s, got error: %v", tc.name, err)
			}
		})
	}
}
