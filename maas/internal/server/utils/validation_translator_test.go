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
	"errors"
	"strings"
	"testing"
)

// TestTranslateValidationError_ValidRegexError_ReturnsCustomMessage tests translation of MAC address validation errors
func TestTranslateValidationError_ValidRegexError_ReturnsCustomMessage(t *testing.T) {
	// Arrange
	testCases := []struct {
		name     string
		error    error
		expected string
	}{
		{
			name:     "MacAddress field error",
			error:    errors.New("validation failed on field MacAddress with regex pattern"),
			expected: "MAC address must be in format XX:XX:XX:XX:XX:XX (e.g., 00:11:22:33:44:55)",
		},
		{
			name:     "NetworkInformation.MacAddress field error",
			error:    errors.New("validation failed on field NetworkInformation.MacAddress with regex pattern"),
			expected: "MAC address must be in format XX:XX:XX:XX:XX:XX (e.g., 00:11:22:33:44:55)",
		},
		{
			name:     "IpmiAddress field error",
			error:    errors.New("validation failed on field IpmiAddress with regex pattern"),
			expected: "IPMI address must be a valid IPv4 address (e.g., 192.168.1.100)",
		},
		{
			name:     "Cidr field error",
			error:    errors.New("validation failed on field Cidr with regex pattern"),
			expected: "CIDR must be in format x.x.x.x/y (e.g., 192.168.1.0/24)",
		},
		{
			name:     "AddressStart field error",
			error:    errors.New("validation failed on field AddressStart with regex pattern"),
			expected: "IP address must be a valid IPv4 address (e.g., 192.168.1.10)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result := TranslateValidationError(tc.error)

			// Assert
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestTranslateValidationError_ArrayFieldError_ReturnsCustomMessage tests translation of array field validation errors
func TestTranslateValidationError_ArrayFieldError_ReturnsCustomMessage(t *testing.T) {
	// Arrange
	testCases := []struct {
		name     string
		error    error
		expected string
	}{
		{
			name:     "NetworkInformation array MacAddress field error",
			error:    errors.New("validation failed on field NetworkInformation[0].MacAddress with regex pattern"),
			expected: "MAC address must be in format XX:XX:XX:XX:XX:XX (e.g., 00:11:22:33:44:55)",
		},
		{
			name:     "NetworkInformation array Cidr field error",
			error:    errors.New("validation failed on field NetworkInformation[1].Cidr with regex pattern"),
			expected: "CIDR must be in format x.x.x.x/y (e.g., 192.168.1.0/24)",
		},
		{
			name:     "NetworkInformationCni array Cidr field error",
			error:    errors.New("validation failed on field NetworkInformationCni[0].Cidr with regex pattern"),
			expected: "CIDR must be in format x.x.x.x/y (e.g., 192.168.1.0/24)",
		},
		{
			name:     "NetworkInformationCni array AddressStart field error",
			error:    errors.New("validation failed on field NetworkInformationCni[2].AddressStart with regex pattern"),
			expected: "IP address must be a valid IPv4 address (e.g., 192.168.1.10)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result := TranslateValidationError(tc.error)

			// Assert
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestTranslateValidationError_NilError_ReturnsEmpty tests handling of nil error
func TestTranslateValidationError_NilError_ReturnsEmpty(t *testing.T) {
	// Arrange
	var err error = nil

	// Act
	result := TranslateValidationError(err)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

// TestTranslateValidationError_NonRegexError_ReturnsOriginal tests handling of non-regex validation errors
func TestTranslateValidationError_NonRegexError_ReturnsOriginal(t *testing.T) {
	// Arrange
	originalError := "field required validation failed"
	err := errors.New(originalError)

	// Act
	result := TranslateValidationError(err)

	// Assert
	if result != originalError {
		t.Errorf("Expected %s, got %s", originalError, result)
	}
}

// TestTranslateValidationError_UnknownField_ReturnsOriginal tests handling of unknown field validation errors
func TestTranslateValidationError_UnknownField_ReturnsOriginal(t *testing.T) {
	// Arrange
	originalError := "validation failed on field UnknownField with regex pattern"
	err := errors.New(originalError)

	// Act
	result := TranslateValidationError(err)

	// Assert
	if result != originalError {
		t.Errorf("Expected %s, got %s", originalError, result)
	}
}

// TestTranslateValidationError_MultipleMatches_ReturnsFirst tests priority when multiple patterns might match
func TestTranslateValidationError_MultipleMatches_ReturnsFirst(t *testing.T) {
	// Arrange
	err := errors.New("validation failed on field NetworkInformation.MacAddress with regex pattern")

	// Act
	result := TranslateValidationError(err)

	// Assert
	expected := "MAC address must be in format XX:XX:XX:XX:XX:XX (e.g., 00:11:22:33:44:55)"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// Mock validator for testing ValidateAndTranslateError
type mockValidator struct {
	shouldFail bool
	errorMsg   string
}

func (m *mockValidator) Validate() error {
	if m.shouldFail {
		return errors.New(m.errorMsg)
	}
	return nil
}

// TestValidateAndTranslateError_ValidatorReturnsError_ReturnsTranslatedMessage tests ValidateAndTranslateError with validation error
func TestValidateAndTranslateError_ValidatorReturnsError_ReturnsTranslatedMessage(t *testing.T) {
	// Arrange
	validator := &mockValidator{
		shouldFail: true,
		errorMsg:   "validation failed on field MacAddress with regex pattern",
	}

	// Act
	result := ValidateAndTranslateError(validator)

	// Assert
	expected := "MAC address must be in format XX:XX:XX:XX:XX:XX (e.g., 00:11:22:33:44:55)"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestValidateAndTranslateError_ValidatorReturnsNil_ReturnsEmpty tests ValidateAndTranslateError with no validation error
func TestValidateAndTranslateError_ValidatorReturnsNil_ReturnsEmpty(t *testing.T) {
	// Arrange
	validator := &mockValidator{
		shouldFail: false,
	}

	// Act
	result := ValidateAndTranslateError(validator)

	// Assert
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

// TestValidateAndTranslateError_ValidatorReturnsNonRegexError_ReturnsOriginal tests ValidateAndTranslateError with non-regex error
func TestValidateAndTranslateError_ValidatorReturnsNonRegexError_ReturnsOriginal(t *testing.T) {
	// Arrange
	originalError := "field is required"
	validator := &mockValidator{
		shouldFail: true,
		errorMsg:   originalError,
	}

	// Act
	result := ValidateAndTranslateError(validator)

	// Assert
	if result != originalError {
		t.Errorf("Expected %s, got %s", originalError, result)
	}
}

// TestTranslateValidationError_EdgeCases_HandlesCorrectly tests edge cases for validation error translation
func TestTranslateValidationError_EdgeCases_HandlesCorrectly(t *testing.T) {
	// Arrange
	testCases := []struct {
		name     string
		error    error
		expected string
	}{
		{
			name:     "Empty error message",
			error:    errors.New(""),
			expected: "",
		},
		{
			name:     "Only regex pattern text",
			error:    errors.New("regex pattern"),
			expected: "regex pattern",
		},
		{
			name:     "Case sensitivity test",
			error:    errors.New("validation failed on field macaddress with regex pattern"),
			expected: "validation failed on field macaddress with regex pattern",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result := TranslateValidationError(tc.error)

			// Assert
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestValidationErrorMessages_MapExists_ContainsExpectedKeys tests that the ValidationErrorMessages map contains expected keys
func TestValidationErrorMessages_MapExists_ContainsExpectedKeys(t *testing.T) {
	// Arrange
	expectedKeys := []string{
		"MacAddress",
		"NetworkInformation.MacAddress",
		"IpmiAddress",
		"Cidr",
		"NetworkInformation.Cidr",
		"NetworkInformationCni.Cidr",
		"AddressStart",
		"AddressEnd",
	}

	// Act & Assert
	for _, key := range expectedKeys {
		if _, exists := ValidationErrorMessages[key]; !exists {
			t.Errorf("Expected key %s to exist in ValidationErrorMessages map", key)
		}
	}
}

// TestValidationErrorMessages_MessageContent_IsNotEmpty tests that all messages in the map are not empty
func TestValidationErrorMessages_MessageContent_IsNotEmpty(t *testing.T) {
	// Act & Assert
	for key, message := range ValidationErrorMessages {
		if strings.TrimSpace(message) == "" {
			t.Errorf("Expected message for key %s to be non-empty", key)
		}
	}
}
