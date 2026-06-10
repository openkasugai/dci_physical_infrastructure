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
	"testing"

	proto "maas_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf
	"maas_module/internal/server/test_utils"

	"google.golang.org/grpc/codes"
)

// TestSeqError_Error_ReturnsFormattedMessage tests SeqError.Error method
func TestSeqError_Error_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	message := "operation in progress"
	err := &SeqError{Message: message}

	// Act
	result := err.Error()

	// Assert
	expected := "maas controller is busy: " + message
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestSeqError_Error_EmptyMessage_ReturnsFormattedMessage tests SeqError.Error with empty message
func TestSeqError_Error_EmptyMessage_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := &SeqError{Message: ""}

	// Act
	result := err.Error()

	// Assert
	expected := "maas controller is busy: "
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestSeqError_ErrorDetail_ReturnsCorrectErrorMessage tests SeqError.ErrorDetail method
func TestSeqError_ErrorDetail_ReturnsCorrectErrorMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := &SeqError{Message: "test message"}

	// Act
	result := err.ErrorDetail()

	// Assert
	if result == nil {
		t.Error("Expected ErrorMessage, got nil")
	}
	if result.ErrorCode != int32(codes.Unavailable) {
		t.Errorf("Expected ErrorCode %d, got %d", codes.Unavailable, result.ErrorCode)
	}
	if result.DetailCode != int32(proto.DetailCode_IF_SEQUENCE_ERROR) {
		t.Errorf("Expected DetailCode %d, got %d", proto.DetailCode_IF_SEQUENCE_ERROR, result.DetailCode)
	}
	if result.Message != "maas controller is busy." {
		t.Errorf("Expected message 'maas controller is busy.', got %s", result.Message)
	}
}

// TestCancelError_Error_ReturnsCorrectMessage tests CancelError.Error method
func TestCancelError_Error_ReturnsCorrectMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := &CancelError{}

	// Act
	result := err.Error()

	// Assert
	expected := "This order cannot be cancelled due to its current status."
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestCancelError_ErrorDetail_ReturnsCorrectErrorMessage tests CancelError.ErrorDetail method
func TestCancelError_ErrorDetail_ReturnsCorrectErrorMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := &CancelError{}

	// Act
	result := err.ErrorDetail()

	// Assert
	if result == nil {
		t.Error("Expected ErrorMessage, got nil")
	}
	if result.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected ErrorCode %d, got %d", codes.Internal, result.ErrorCode)
	}
	if result.DetailCode != int32(proto.DetailCode_IF_CANCEL_UNAVAILABLE) {
		t.Errorf("Expected DetailCode %d, got %d", proto.DetailCode_IF_CANCEL_UNAVAILABLE, result.DetailCode)
	}
	expectedMessage := "This order cannot be cancelled due to its current status."
	if result.Message != expectedMessage {
		t.Errorf("Expected message %s, got %s", expectedMessage, result.Message)
	}
}

// TestEnvError_Error_ReturnsMessage tests EnvError.Error method
func TestEnvError_Error_ReturnsMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	message := "environment variable not set"
	err := &EnvError{Message: message}

	// Act
	result := err.Error()

	// Assert
	if result != message {
		t.Errorf("Expected %s, got %s", message, result)
	}
}

// TestEnvError_Error_EmptyMessage_ReturnsEmptyString tests EnvError.Error with empty message
func TestEnvError_Error_EmptyMessage_ReturnsEmptyString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := &EnvError{Message: ""}

	// Act
	result := err.Error()

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

// TestEnvError_ErrorDetail_ReturnsCorrectErrorMessage tests EnvError.ErrorDetail method
func TestEnvError_ErrorDetail_ReturnsCorrectErrorMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	message := "invalid configuration"
	err := &EnvError{Message: message}

	// Act
	result := err.ErrorDetail()

	// Assert
	if result == nil {
		t.Error("Expected ErrorMessage, got nil")
	}
	if result.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected ErrorCode %d, got %d", codes.Internal, result.ErrorCode)
	}
	if result.DetailCode != int32(proto.DetailCode_MAAS_ENVIRONMENT_ERROR) {
		t.Errorf("Expected DetailCode %d, got %d", proto.DetailCode_MAAS_ENVIRONMENT_ERROR, result.DetailCode)
	}
	if result.Message != message {
		t.Errorf("Expected message %s, got %s", message, result.Message)
	}
}

// TestHttpError_Error_ReturnsFormattedMessage tests HttpError.Error method
func TestHttpError_Error_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	statusCode := 404
	message := "Not Found"
	err := &HttpError{StatusCode: statusCode, Message: message}

	// Act
	result := err.Error()

	// Assert
	expected := "<404> Not Found"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestHttpError_Error_ZeroStatusCode_ReturnsFormattedMessage tests HttpError.Error with zero status code
func TestHttpError_Error_ZeroStatusCode_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := &HttpError{StatusCode: 0, Message: "test message"}

	// Act
	result := err.Error()

	// Assert
	expected := "<0> test message"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestHttpError_Error_EmptyMessage_ReturnsFormattedMessage tests HttpError.Error with empty message
func TestHttpError_Error_EmptyMessage_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := &HttpError{StatusCode: 500, Message: ""}

	// Act
	result := err.Error()

	// Assert
	expected := "<500> "
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestHttpError_ErrorDetail_ReturnsCorrectErrorMessage tests HttpError.ErrorDetail method
func TestHttpError_ErrorDetail_ReturnsCorrectErrorMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	statusCode := 503
	message := "Service Unavailable"
	err := &HttpError{StatusCode: statusCode, Message: message}

	// Act
	result := err.ErrorDetail()

	// Assert
	if result == nil {
		t.Error("Expected ErrorMessage, got nil")
	}
	if result.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected ErrorCode %d, got %d", codes.Internal, result.ErrorCode)
	}
	if result.DetailCode != int32(statusCode) {
		t.Errorf("Expected DetailCode %d, got %d", statusCode, result.DetailCode)
	}
	if result.Message != message {
		t.Errorf("Expected message %s, got %s", message, result.Message)
	}
}

// TestRespError_Error_ReturnsFormattedMessage tests RespError.Error method
func TestRespError_Error_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	message := "malformed JSON"
	err := &RespError{Message: message}

	// Act
	result := err.Error()

	// Assert
	expected := "invalid maas response: " + message
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestRespError_Error_EmptyMessage_ReturnsFormattedMessage tests RespError.Error with empty message
func TestRespError_Error_EmptyMessage_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := &RespError{Message: ""}

	// Act
	result := err.Error()

	// Assert
	expected := "invalid maas response: "
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestRespError_ErrorDetail_ReturnsCorrectErrorMessage tests RespError.ErrorDetail method
func TestRespError_ErrorDetail_ReturnsCorrectErrorMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	message := "unexpected format"
	err := &RespError{Message: message}

	// Act
	result := err.ErrorDetail()

	// Assert
	if result == nil {
		t.Error("Expected ErrorMessage, got nil")
	}
	if result.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected ErrorCode %d, got %d", codes.Internal, result.ErrorCode)
	}
	if result.DetailCode != int32(proto.DetailCode_MAAS_RESPONSE_INVALID) {
		t.Errorf("Expected DetailCode %d, got %d", proto.DetailCode_MAAS_RESPONSE_INVALID, result.DetailCode)
	}
	if result.Message != "invalid maas response." {
		t.Errorf("Expected message 'invalid maas response.', got %s", result.Message)
	}
}

// TestAllErrorTypes_ImplementErrorInterface tests that all custom error types implement the error interface
func TestAllErrorTypes_ImplementErrorInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errors := []interface{}{
		&SeqError{Message: "test"},
		&CancelError{},
		&EnvError{Message: "test"},
		&HttpError{StatusCode: 500, Message: "test"},
		&RespError{Message: "test"},
	}

	// Act & Assert
	for i, err := range errors {
		if _, ok := err.(error); !ok {
			t.Errorf("Error type %d does not implement error interface", i)
		}
	}
}

// TestErrorDetailTypes_ConsistentStructure tests that all ErrorDetail methods return consistent structure
func TestErrorDetailTypes_ConsistentStructure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testCases := []struct {
		name               string
		err                interface{ ErrorDetail() *common.ErrorMessage }
		expectedErrorCode  int32
		expectedDetailCode int32
	}{
		{
			name:               "SeqError",
			err:                &SeqError{Message: "test"},
			expectedErrorCode:  int32(codes.Unavailable),
			expectedDetailCode: int32(proto.DetailCode_IF_SEQUENCE_ERROR),
		},
		{
			name:               "CancelError",
			err:                &CancelError{},
			expectedErrorCode:  int32(codes.Internal),
			expectedDetailCode: int32(proto.DetailCode_IF_CANCEL_UNAVAILABLE),
		},
		{
			name:               "EnvError",
			err:                &EnvError{Message: "test"},
			expectedErrorCode:  int32(codes.Internal),
			expectedDetailCode: int32(proto.DetailCode_MAAS_ENVIRONMENT_ERROR),
		},
		{
			name:               "RespError",
			err:                &RespError{Message: "test"},
			expectedErrorCode:  int32(codes.Internal),
			expectedDetailCode: int32(proto.DetailCode_MAAS_RESPONSE_INVALID),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Act
			result := tc.err.ErrorDetail()

			// Assert
			if result == nil {
				t.Error("Expected ErrorMessage, got nil")
			}
			if result.ErrorCode != tc.expectedErrorCode {
				t.Errorf("Expected ErrorCode %d, got %d", tc.expectedErrorCode, result.ErrorCode)
			}
			if result.DetailCode != tc.expectedDetailCode {
				t.Errorf("Expected DetailCode %d, got %d", tc.expectedDetailCode, result.DetailCode)
			}
			if result.Message == "" {
				t.Error("Expected non-empty message")
			}
		})
	}
}

// TestHttpError_ErrorDetail_UsesStatusCodeAsDetailCode tests that HttpError uses StatusCode as DetailCode
func TestHttpError_ErrorDetail_UsesStatusCodeAsDetailCode(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testCases := []int{400, 401, 404, 500, 503}

	for _, statusCode := range testCases {
		t.Run("StatusCode_"+string(rune(statusCode)), func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Arrange
			err := &HttpError{StatusCode: statusCode, Message: "test"}

			// Act
			result := err.ErrorDetail()

			// Assert
			if result.DetailCode != int32(statusCode) {
				t.Errorf("Expected DetailCode %d, got %d", statusCode, result.DetailCode)
			}
		})
	}
}

// TestErrorMessages_NonEmpty tests that all error messages are non-empty
func TestErrorMessages_NonEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testCases := []struct {
		name string
		err  error
	}{
		{"SeqError", &SeqError{Message: "test"}},
		{"CancelError", &CancelError{}},
		{"EnvError", &EnvError{Message: "test"}},
		{"HttpError", &HttpError{StatusCode: 500, Message: "test"}},
		{"RespError", &RespError{Message: "test"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Act
			result := tc.err.Error()

			// Assert
			if result == "" {
				t.Error("Expected non-empty error message")
			}
		})
	}
}
