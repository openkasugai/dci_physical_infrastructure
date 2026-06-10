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

package maas_api

import (
	"context"
	"testing"

	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/utils"

	"k8s.io/klog/v2"
)

// Mock CanonicalMaasApi for testing
type mockCanonicalMaasApi struct {
	statusCode int
	data       []byte
	err        error
}

func (m *mockCanonicalMaasApi) APIExecute(ctx context.Context, method, apiname, reqBody string) (int, []byte, error) {
	return m.statusCode, m.data, m.err
}

// Helper function to create AbstractMaas instance for testing
func createTestAbstractMaas(statusCode int, data []byte, err error) *AbstractMaas {
	return &AbstractMaas{
		API:    &mockCanonicalMaasApi{statusCode: statusCode, data: data, err: err},
		Logger: klog.NewKlogr(),
	}
}

// TestAbstractMaas_Success_2xxStatusCode_ReturnsTrue tests Success method with 2xx status codes
func TestAbstractMaas_Success_2xxStatusCode_ReturnsTrue(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	testCases := []int{200, 201, 202, 204, 299}

	for _, statusCode := range testCases {
		t.Run("StatusCode_"+string(rune(statusCode+48)), func(t *testing.T) {
			// Act
			result := abstract.Success(statusCode)

			// Assert
			if !result {
				t.Errorf("Expected true for status code %d, got false", statusCode)
			}
		})
	}
}

// TestAbstractMaas_Success_Non2xxStatusCode_ReturnsFalse tests Success method with non-2xx status codes
func TestAbstractMaas_Success_Non2xxStatusCode_ReturnsFalse(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(400, nil, nil)
	testCases := []int{100, 199, 300, 301, 400, 404, 500, 503}

	for _, statusCode := range testCases {
		t.Run("StatusCode_"+string(rune(statusCode+48)), func(t *testing.T) {
			// Act
			result := abstract.Success(statusCode)

			// Assert
			if result {
				t.Errorf("Expected false for status code %d, got true", statusCode)
			}
		})
	}
}

// TestAbstractMaas_HTTPError_SuccessStatusCode_ReturnsNil tests HTTPError method with success status codes
func TestAbstractMaas_HTTPError_SuccessStatusCode_ReturnsNil(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, []byte("success"), nil)
	testCases := []int{200, 201, 202, 299}

	for _, statusCode := range testCases {
		t.Run("StatusCode_"+string(rune(statusCode+48)), func(t *testing.T) {
			// Act
			result := abstract.HTTPError(statusCode, []byte("test data"))

			// Assert
			if result != nil {
				t.Errorf("Expected nil for success status code %d, got error: %v", statusCode, result)
			}
		})
	}
}

// TestAbstractMaas_HTTPError_ErrorStatusCode_ReturnsError tests HTTPError method with error status codes
func TestAbstractMaas_HTTPError_ErrorStatusCode_ReturnsError(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(400, []byte("error"), nil)
	testData := []byte("Bad Request")
	testCases := []int{400, 404, 500, 503}

	for _, statusCode := range testCases {
		t.Run("StatusCode_"+string(rune(statusCode+48)), func(t *testing.T) {
			// Act
			result := abstract.HTTPError(statusCode, testData)

			// Assert
			if result == nil {
				t.Errorf("Expected error for status code %d, got nil", statusCode)
			}

			// Verify it's HttpError type
			if httpErr, ok := result.(*utils.HttpError); ok {
				if httpErr.StatusCode != statusCode {
					t.Errorf("Expected status code %d, got %d", statusCode, httpErr.StatusCode)
				}
				if httpErr.Message != string(testData) {
					t.Errorf("Expected message %s, got %s", string(testData), httpErr.Message)
				}
			} else {
				t.Error("Expected HttpError type")
			}
		})
	}
}

// TestAbstractMaas_NewResponseCommon_SuccessStatus_ReturnsDataInRawJSON tests NewResponseCommon with success status
func TestAbstractMaas_NewResponseCommon_SuccessStatus_ReturnsDataInRawJSON(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	statusCode := 200
	respData := []byte(`{"success": true}`)

	// Act
	result := abstract.NewResponseCommon(statusCode, respData)

	// Assert
	if result.HTTPStatus != statusCode {
		t.Errorf("Expected HTTPStatus %d, got %d", statusCode, result.HTTPStatus)
	}
	if result.ErrorMessage != "" {
		t.Errorf("Expected empty ErrorMessage, got %s", result.ErrorMessage)
	}
	if result.RawJSONData != string(respData) {
		t.Errorf("Expected RawJSONData %s, got %s", string(respData), result.RawJSONData)
	}
}

// TestAbstractMaas_NewResponseCommon_ErrorStatus_ReturnsErrorMessage tests NewResponseCommon with error status
func TestAbstractMaas_NewResponseCommon_ErrorStatus_ReturnsErrorMessage(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(400, nil, nil)
	statusCode := 400
	respData := []byte(`{"error": "Bad Request"}`)

	// Act
	result := abstract.NewResponseCommon(statusCode, respData)

	// Assert
	if result.HTTPStatus != statusCode {
		t.Errorf("Expected HTTPStatus %d, got %d", statusCode, result.HTTPStatus)
	}
	if result.ErrorMessage != string(respData) {
		t.Errorf("Expected ErrorMessage %s, got %s", string(respData), result.ErrorMessage)
	}
	if result.RawJSONData != "" {
		t.Errorf("Expected empty RawJSONData, got %s", result.RawJSONData)
	}
}

// TestAbstractMaas_NewResponseCommon_BoundaryStatusCodes tests NewResponseCommon with boundary status codes
func TestAbstractMaas_NewResponseCommon_BoundaryStatusCodes(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	testCases := []struct {
		name       string
		statusCode int
		isSuccess  bool
	}{
		{"StatusCode_199", 199, false},
		{"StatusCode_200", 200, true},
		{"StatusCode_299", 299, true},
		{"StatusCode_300", 300, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			respData := []byte("test data")

			// Act
			result := abstract.NewResponseCommon(tc.statusCode, respData)

			// Assert
			if tc.isSuccess {
				if result.RawJSONData != string(respData) {
					t.Errorf("Expected RawJSONData for success status, got empty")
				}
				if result.ErrorMessage != "" {
					t.Errorf("Expected empty ErrorMessage for success status, got %s", result.ErrorMessage)
				}
			} else {
				if result.ErrorMessage != string(respData) {
					t.Errorf("Expected ErrorMessage for error status, got empty")
				}
				if result.RawJSONData != "" {
					t.Errorf("Expected empty RawJSONData for error status, got %s", result.RawJSONData)
				}
			}
		})
	}
}

// TestAbstractMaas_ExtractValue_ValidJSON_ReturnsValue tests ExtractValue with valid JSON
func TestAbstractMaas_ExtractValue_ValidJSON_ReturnsValue(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	jsonData := []byte(`{"name": "test", "id": 123, "active": true}`)
	testCases := []struct {
		key      string
		expected interface{}
	}{
		{"name", "test"},
		{"id", float64(123)}, // JSON numbers are parsed as float64
		{"active", true},
	}

	for _, tc := range testCases {
		t.Run("Key_"+tc.key, func(t *testing.T) {
			// Act
			value, result := abstract.ExtractValue(jsonData, tc.key)

			// Assert
			if !result {
				t.Errorf("Expected true result for key %s", tc.key)
			}
			if value != tc.expected {
				t.Errorf("Expected value %v for key %s, got %v", tc.expected, tc.key, value)
			}
		})
	}
}

// TestAbstractMaas_ExtractValue_NestedJSON_ReturnsValue tests ExtractValue with nested JSON
func TestAbstractMaas_ExtractValue_NestedJSON_ReturnsValue(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	jsonData := []byte(`{
		"user": {"username": "john", "details": {"age": 30}},
		"items": [{"item_id": 1}, {"item_id": 2, "item_desc": "second_item"}]
	}`)

	testCases := []struct {
		key        string
		expected   interface{}
		shouldFind bool
	}{
		{"username", "john", true},
		{"age", float64(30), true},
		{"item_id", float64(1), true}, // Should find first occurrence
		{"item_desc", "second_item", true},
		{"nonexistent", nil, false},
	}

	for _, tc := range testCases {
		t.Run("Key_"+tc.key, func(t *testing.T) {
			// Act
			value, result := abstract.ExtractValue(jsonData, tc.key)

			// Assert
			if result != tc.shouldFind {
				t.Errorf("Expected result %t for key %s, got %t", tc.shouldFind, tc.key, result)
			}
			if tc.shouldFind && value != tc.expected {
				t.Errorf("Expected value %v for key %s, got %v", tc.expected, tc.key, value)
			}
		})
	}
}

// TestAbstractMaas_ExtractValue_InvalidJSON_ReturnsFalse tests ExtractValue with invalid JSON
func TestAbstractMaas_ExtractValue_InvalidJSON_ReturnsFalse(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	invalidJSON := []byte(`{"invalid": json}`)

	// Act
	value, result := abstract.ExtractValue(invalidJSON, "any")

	// Assert
	if result {
		t.Error("Expected false result for invalid JSON")
	}
	if value != nil {
		t.Error("Expected nil value for invalid JSON")
	}
}

// TestAbstractMaas_ExtractValue_EmptyJSON_ReturnsFalse tests ExtractValue with empty JSON
func TestAbstractMaas_ExtractValue_EmptyJSON_ReturnsFalse(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	emptyJSON := []byte(`{}`)

	// Act
	value, result := abstract.ExtractValue(emptyJSON, "nonexistent")

	// Assert
	if result {
		t.Error("Expected false result for nonexistent key in empty JSON")
	}
	if value != nil {
		t.Error("Expected nil value for nonexistent key")
	}
}

// TestAbstractMaas_FindValue_SimpleMap_ReturnsValue tests FindValue with simple map
func TestAbstractMaas_FindValue_SimpleMap_ReturnsValue(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	testCases := []struct {
		key        string
		expected   interface{}
		shouldFind bool
	}{
		{"key1", "value1", true},
		{"key2", 42, true},
		{"key3", true, true},
		{"nonexistent", nil, false},
	}

	for _, tc := range testCases {
		t.Run("Key_"+tc.key, func(t *testing.T) {
			// Act
			value, result := abstract.FindValue(data, tc.key)

			// Assert
			if result != tc.shouldFind {
				t.Errorf("Expected result %t for key %s, got %t", tc.shouldFind, tc.key, result)
			}
			if tc.shouldFind && value != tc.expected {
				t.Errorf("Expected value %v for key %s, got %v", tc.expected, tc.key, value)
			}
		})
	}
}

// TestAbstractMaas_FindValue_NestedMap_ReturnsValue tests FindValue with nested map
func TestAbstractMaas_FindValue_NestedMap_ReturnsValue(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"targetKey": "foundValue",
			},
		},
		"array": []interface{}{
			map[string]interface{}{
				"itemKey": "itemValue",
			},
		},
	}

	testCases := []struct {
		key        string
		expected   interface{}
		shouldFind bool
	}{
		{"targetKey", "foundValue", true},
		{"itemKey", "itemValue", true},
		{"level1", data["level1"], true},
		{"nonexistent", nil, false},
	}

	for _, tc := range testCases {
		t.Run("Key_"+tc.key, func(t *testing.T) {
			// Act
			value, result := abstract.FindValue(data, tc.key)

			// Assert
			if result != tc.shouldFind {
				t.Errorf("Expected result %t for key %s, got %t", tc.shouldFind, tc.key, result)
			}
			if tc.shouldFind && tc.key != "level1" { // Skip deep comparison for complex objects
				if value != tc.expected {
					t.Errorf("Expected value %v for key %s, got %v", tc.expected, tc.key, value)
				}
			}
		})
	}
}

// TestAbstractMaas_FindValue_Array_ReturnsValue tests FindValue with array data
func TestAbstractMaas_FindValue_Array_ReturnsValue(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	data := []interface{}{
		map[string]interface{}{"id": 1, "name": "first"},
		map[string]interface{}{"id": 2, "name": "second"},
		"string item",
		42,
	}

	// Act & Assert
	value, result := abstract.FindValue(data, "name")
	if !result {
		t.Error("Expected to find 'name' key in array")
	}
	if value != "first" { // Should find first occurrence
		t.Errorf("Expected 'first', got %v", value)
	}

	value, result = abstract.FindValue(data, "id")
	if !result {
		t.Error("Expected to find 'id' key in array")
	}
	if value != 1 { // Should find first occurrence
		t.Errorf("Expected 1, got %v", value)
	}

	value, result = abstract.FindValue(data, "nonexistent")
	if result {
		t.Error("Expected not to find 'nonexistent' key in array")
	}
}

// TestAbstractMaas_FindValue_NonContainerTypes_ReturnsFalse tests FindValue with non-container types
func TestAbstractMaas_FindValue_NonContainerTypes_ReturnsFalse(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	testCases := []interface{}{
		"string",
		42,
		true,
		42.5,
		nil,
	}

	for _, data := range testCases {
		t.Run("DataType", func(t *testing.T) {
			// Act
			value, result := abstract.FindValue(data, "anykey")

			// Assert
			if result {
				t.Error("Expected false result for non-container type")
			}
			if value != nil {
				t.Error("Expected nil value for non-container type")
			}
		})
	}
}

// TestAbstractMaas_GET_ReturnsNotImplemented tests default GET method
func TestAbstractMaas_GET_ReturnsNotImplemented(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	ctx := context.Background()

	// Act
	result, err := abstract.GET(ctx)

	// Assert
	if result != nil {
		t.Error("Expected nil result for unimplemented GET")
	}
	if err == nil {
		t.Error("Expected error for unimplemented GET")
	}
	if err.Error() != "not implementation" {
		t.Errorf("Expected 'not implementation' error, got: %v", err)
	}
}

// TestAbstractMaas_POST_ReturnsNotImplemented tests default POST method
func TestAbstractMaas_POST_ReturnsNotImplemented(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	ctx := context.Background()
	reqBody := &request_body.ReqbodyCommon{}

	// Act
	result, err := abstract.POST(ctx, reqBody)

	// Assert
	if result != nil {
		t.Error("Expected nil result for unimplemented POST")
	}
	if err == nil {
		t.Error("Expected error for unimplemented POST")
	}
	if err.Error() != "not implementation" {
		t.Errorf("Expected 'not implementation' error, got: %v", err)
	}
}

// TestAbstractMaas_PUT_ReturnsNotImplemented tests default PUT method
func TestAbstractMaas_PUT_ReturnsNotImplemented(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	ctx := context.Background()
	reqBody := &request_body.ReqbodyCommon{}

	// Act
	result, err := abstract.PUT(ctx, reqBody)

	// Assert
	if result != nil {
		t.Error("Expected nil result for unimplemented PUT")
	}
	if err == nil {
		t.Error("Expected error for unimplemented PUT")
	}
	if err.Error() != "not implementation" {
		t.Errorf("Expected 'not implementation' error, got: %v", err)
	}
}

// TestAbstractMaas_DELETE_ReturnsNotImplemented tests default DELETE method
func TestAbstractMaas_DELETE_ReturnsNotImplemented(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	ctx := context.Background()

	// Act
	result, err := abstract.DELETE(ctx)

	// Assert
	if result != nil {
		t.Error("Expected nil result for unimplemented DELETE")
	}
	if err == nil {
		t.Error("Expected error for unimplemented DELETE")
	}
	if err.Error() != "not implementation" {
		t.Errorf("Expected 'not implementation' error, got: %v", err)
	}
}

// TestAbstractMaas_HTTPError_EmptyData_HandlesCorrectly tests HTTPError with empty data
func TestAbstractMaas_HTTPError_EmptyData_HandlesCorrectly(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(400, nil, nil)
	emptyData := []byte("")

	// Act
	result := abstract.HTTPError(400, emptyData)

	// Assert
	if result == nil {
		t.Error("Expected error for status code 400")
	}

	if httpErr, ok := result.(*utils.HttpError); ok {
		if httpErr.Message != "" {
			t.Errorf("Expected empty message, got %s", httpErr.Message)
		}
	}
}

// TestAbstractMaas_NewResponseCommon_EmptyData_HandlesCorrectly tests NewResponseCommon with empty data
func TestAbstractMaas_NewResponseCommon_EmptyData_HandlesCorrectly(t *testing.T) {
	// Arrange
	abstract := createTestAbstractMaas(200, nil, nil)
	emptyData := []byte("")

	testCases := []struct {
		name       string
		statusCode int
		isSuccess  bool
	}{
		{"SuccessWithEmptyData", 200, true},
		{"ErrorWithEmptyData", 400, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result := abstract.NewResponseCommon(tc.statusCode, emptyData)

			// Assert
			if result.HTTPStatus != tc.statusCode {
				t.Errorf("Expected HTTPStatus %d, got %d", tc.statusCode, result.HTTPStatus)
			}

			if tc.isSuccess {
				if result.RawJSONData != "" {
					t.Errorf("Expected empty RawJSONData for success with empty data, got %s", result.RawJSONData)
				}
				if result.ErrorMessage != "" {
					t.Errorf("Expected empty ErrorMessage for success, got %s", result.ErrorMessage)
				}
			} else {
				if result.ErrorMessage != "" {
					t.Errorf("Expected empty ErrorMessage for error with empty data, got %s", result.ErrorMessage)
				}
				if result.RawJSONData != "" {
					t.Errorf("Expected empty RawJSONData for error, got %s", result.RawJSONData)
				}
			}
		})
	}
}
