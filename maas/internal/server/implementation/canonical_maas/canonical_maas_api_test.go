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

package canonical_maas

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"k8s.io/klog/v2"

	"maas_module/internal/server/implementation/canonical_maas/mocks"
	"maas_module/internal/server/test_utils"
	"maas_module/internal/server/utils"
)

// Helper function to set test environment variables and clean up after test
func setTestEnvForAPI(t *testing.T, env map[string]string) {
	// Set test environment variables
	for key, value := range env {
		os.Setenv(key, value)
	}

	// Clean up after test
	t.Cleanup(func() {
		for key := range env {
			os.Unsetenv(key)
		}
	})
}

// TestNewCanonicalMaasAPIImple_ValidLogger_ReturnsInstance tests NewCanonicalMaasAPIImple with valid logger
func TestNewCanonicalMaasAPIImple_ValidLogger_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	logger := klog.NewKlogr()

	// Act
	result := NewCanonicalMaasAPIImple(logger, "http://test-url", "key:token:secret")

	// Assert
	if result == nil {
		t.Error("Expected CanonicalMaasAPIImple instance, got nil")
	}

	if result.Client == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	// Verify it's the default HTTP client
	if _, ok := result.Client.(*http.Client); !ok {
		t.Error("Expected default HTTP client")
	}
}

// TestNewCanonicalMaasAPIImple_NilLogger_ReturnsInstance tests NewCanonicalMaasAPIImple with nil logger
func TestNewCanonicalMaasAPIImple_NilLogger_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	var logger klog.Logger

	// Act
	result := NewCanonicalMaasAPIImple(logger, "http://test-url", "key:token:secret")

	// Assert
	if result == nil {
		t.Error("Expected CanonicalMaasAPIImple instance even with nil logger")
	}

	if result.Client == nil {
		t.Error("Expected HTTP client to be initialized even with nil logger")
	}
}

// TestAPIExecute_ValidEnvironment_ExecutesSuccessfully tests APIExecute with valid environment
func TestAPIExecute_ValidEnvironment_ExecutesSuccessfully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_ACCESS_URL": "http://test-maas.example.com/",
		"MAAS_API_KEY":    "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	mockClient := &mocks.MockHTTPClient{
		MockResponse: mocks.NewMockResponse(200, `{"success": true}`),
	}

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: mockClient,
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "consumer:token:secret",
	}

	ctx := context.Background()

	// Act
	statusCode, jsonData, err := api.APIExecute(ctx, "GET", "api/test/", "")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("Expected status code 200, got: %d", statusCode)
	}
	if string(jsonData) != `{"success": true}` {
		t.Errorf("Expected JSON data '{\"success\": true}', got: %s", string(jsonData))
	}

	// Verify request was made with correct URL
	lastRequest := mockClient.GetLastRequest()
	if lastRequest == nil {
		t.Error("Expected HTTP request to be made")
	} else {
		expectedURL := "http://test-maas.example.com/api/test/"
		if lastRequest.URL.String() != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, lastRequest.URL.String())
		}
	}
}

// TestAPIExecute_MissingAccessURL_ReturnsError tests APIExecute without MAAS_ACCESS_URL
func TestAPIExecute_MissingAccessURL_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - Set only API key, not access URL
	testEnv := map[string]string{
		"MAAS_API_KEY": "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	mockClient := &mocks.MockHTTPClient{
		MockResponse: mocks.NewMockResponse(500, "Internal Server Error"),
		MockError:    errors.New("unsupported protocol scheme"),
	}
	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: mockClient,
		AccessUrl: "",
		ApiKey:    "consumer:token:secret",
	}

	ctx := context.Background()

	// Act
	_, _, err := api.APIExecute(ctx, "GET", "api/test/", "")

	// Assert
	// The request will still be made, but with empty base URL
	// This should result in an invalid URL that causes an HTTP error
	if err == nil {
		t.Error("Expected error due to invalid URL")
	}

	// Verify it's an environment error
	if envErr, ok := err.(*utils.EnvError); ok {
		if !strings.Contains(envErr.Message, "unsupported protocol scheme") && !strings.Contains(envErr.Message, "invalid") {
			t.Errorf("Expected URL-related error, got: %s", envErr.Message)
		}
	} else {
		t.Errorf("Expected EnvError, got: %T", err)
	}
}

// TestAPIExecute_WithRequestBody_SendsBodyCorrectly tests APIExecute with request body
func TestAPIExecute_WithRequestBody_SendsBodyCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_ACCESS_URL": "http://test-maas.example.com/",
		"MAAS_API_KEY":    "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	mockClient := &mocks.MockHTTPClient{
		MockResponse: mocks.NewMockResponse(201, `{"created": true}`),
	}

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: mockClient,
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "consumer:token:secret",
	}

	ctx := context.Background()
	requestBody := "name=test&value=123"

	// Act
	statusCode, _, err := api.APIExecute(ctx, "POST", "api/create/", requestBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if statusCode != 201 {
		t.Errorf("Expected status code 201, got: %d", statusCode)
	}

	// Verify request was made with correct method and headers
	lastRequest := mockClient.GetLastRequest()
	if lastRequest == nil {
		t.Error("Expected HTTP request to be made")
	} else {
		if lastRequest.Method != "POST" {
			t.Errorf("Expected POST method, got: %s", lastRequest.Method)
		}

		contentType := lastRequest.Header.Get("Content-Type")
		expectedContentType := "application/x-www-form-urlencoded"
		if contentType != expectedContentType {
			t.Errorf("Expected Content-Type %s, got: %s", expectedContentType, contentType)
		}

		// Verify Authorization header is set
		authHeader := lastRequest.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "OAuth") {
			t.Errorf("Expected OAuth authorization header, got: %s", authHeader)
		}

		// Verify body
		body, _ := io.ReadAll(lastRequest.Body)
		if string(body) != requestBody {
			t.Errorf("Expected request body %s, got: %s", requestBody, string(body))
		}
	}
}

// TestExecRequest_MissingAPIKey_ReturnsEnvError tests execRequest without MAAS_API_KEY
func TestExecRequest_MissingAPIKey_ReturnsEnvError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - Don't set MAAS_API_KEY
	setTestEnvForAPI(t, map[string]string{})

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: &http.Client{},
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "",
	}

	// Act
	_, _, err := api.execRequest("GET", "http://test.com", "")

	// Assert
	if err == nil {
		t.Error("Expected error for missing API key")
	}

	envErr, ok := err.(*utils.EnvError)
	if !ok {
		t.Errorf("Expected EnvError, got: %T", err)
	} else {
		if envErr.Message != "invalid API key format" {
			t.Errorf("Expected invalid API key format error message, got: %s", envErr.Message)
		}
	}

	// Note: statusCode and resp are not checked since we're only testing error case
}

// TestExecRequest_InvalidAPIKeyFormat_ReturnsEnvError tests execRequest with invalid API key format
func TestExecRequest_InvalidAPIKeyFormat_ReturnsEnvError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_API_KEY": "invalid-key-format",
	}
	setTestEnvForAPI(t, testEnv)

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: &http.Client{},
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "invalid-key",
	}

	// Act
	_, _, err := api.execRequest("GET", "http://test.com", "")

	// Assert
	if err == nil {
		t.Error("Expected error for invalid API key format")
	}

	if envErr, ok := err.(*utils.EnvError); ok {
		if envErr.Message != "invalid API key format" {
			t.Errorf("Expected invalid API key format error, got: %s", envErr.Message)
		}
	} else {
		t.Errorf("Expected EnvError, got: %T", err)
	}
}

// TestExecRequest_ValidAPIKey_GeneratesOAuthHeader tests execRequest with valid API key
func TestExecRequest_ValidAPIKey_GeneratesOAuthHeader(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_API_KEY": "consumer123:token456:secret789",
	}
	setTestEnvForAPI(t, testEnv)

	mockClient := &mocks.MockHTTPClient{
		MockResponse: mocks.NewMockResponse(200, "OK"),
	}

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: mockClient,
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "consumer123:token456:secret789",
	}

	// Act
	statusCode, resp, err := api.execRequest("GET", "http://test.com/api", "")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("Expected status code 200, got: %d", statusCode)
	}
	if string(resp) != "OK" {
		t.Errorf("Expected response 'OK', got: %s", string(resp))
	}

	// Verify OAuth header components
	lastRequest := mockClient.GetLastRequest()
	if lastRequest == nil {
		t.Error("Expected HTTP request to be made")
	} else {
		authHeader := lastRequest.Header.Get("Authorization")

		// Check OAuth header format and components
		if !strings.HasPrefix(authHeader, "OAuth") {
			t.Error("Expected OAuth authorization header")
		}
		if !strings.Contains(authHeader, "oauth_consumer_key=\"consumer123\"") {
			t.Error("Expected consumer key in OAuth header")
		}
		if !strings.Contains(authHeader, "oauth_token=\"token456\"") {
			t.Error("Expected token in OAuth header")
		}
		if !strings.Contains(authHeader, "oauth_signature=\"%26secret789\"") { // & is URL encoded
			t.Error("Expected signature in OAuth header")
		}
		if !strings.Contains(authHeader, "oauth_version=\"1.0\"") {
			t.Error("Expected version in OAuth header")
		}
		if !strings.Contains(authHeader, "oauth_signature_method=\"PLAINTEXT\"") {
			t.Error("Expected signature method in OAuth header")
		}
	}
}

// TestExecRequest_HTTPClientError_ReturnsEnvError tests execRequest when HTTP client returns error
func TestExecRequest_HTTPClientError_ReturnsEnvError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_API_KEY": "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	mockClient := &mocks.MockHTTPClient{
		MockError: errors.New("network error"),
	}

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: mockClient,
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "consumer:token:secret",
	}

	// Act
	_, _, err := api.execRequest("GET", "http://test.com", "")

	// Assert
	if err == nil {
		t.Error("Expected error from HTTP client")
	}

	if envErr, ok := err.(*utils.EnvError); ok {
		if !strings.Contains(envErr.Message, "network error") {
			t.Errorf("Expected network error in message, got: %s", envErr.Message)
		}
	} else {
		t.Errorf("Expected EnvError, got: %T", err)
	}
}

// TestExecRequest_HTTPErrorResponse_ReturnsCorrectStatus tests execRequest with HTTP error responses
func TestExecRequest_HTTPErrorResponse_ReturnsCorrectStatus(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_API_KEY": "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	testCases := []struct {
		name         string
		statusCode   int
		responseBody string
	}{
		{"BadRequest", 400, "Bad Request"},
		{"NotFound", 404, "Not Found"},
		{"InternalServerError", 500, "Internal Server Error"},
		{"ServiceUnavailable", 503, "Service Unavailable"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockClient := &mocks.MockHTTPClient{
				MockResponse: mocks.NewMockResponse(tc.statusCode, tc.responseBody),
			}

			api := &CanonicalMaasAPIImple{
				Logger: klog.NewKlogr(),
				Client: mockClient,
				AccessUrl: "http://test-maas.example.com/",
				ApiKey:    "consumer:token:secret",
			}

			// Act
			statusCode, resp, err := api.execRequest("GET", "http://test.com", "")

			// Assert
			if err != nil {
				t.Errorf("Expected no error for HTTP response, got: %v", err)
			}
			if statusCode != tc.statusCode {
				t.Errorf("Expected status code %d, got: %d", tc.statusCode, statusCode)
			}
			if string(resp) != tc.responseBody {
				t.Errorf("Expected response %s, got: %s", tc.responseBody, string(resp))
			}
		})
	}
}

// TestExecRequest_EmptyRequestBody_DoesNotSetContentType tests execRequest without request body
func TestExecRequest_EmptyRequestBody_DoesNotSetContentType(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_API_KEY": "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	mockClient := &mocks.MockHTTPClient{
		MockResponse: mocks.NewMockResponse(200, "OK"),
	}

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: mockClient,
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "consumer:token:secret",
	}

	// Act
	_, _, err := api.execRequest("GET", "http://test.com", "")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify Content-Type header is not set
	lastRequest := mockClient.GetLastRequest()
	if lastRequest == nil {
		t.Error("Expected HTTP request to be made")
	} else {
		contentType := lastRequest.Header.Get("Content-Type")
		if contentType != "" {
			t.Errorf("Expected no Content-Type header, got: %s", contentType)
		}
	}
}

// TestExecRequest_InvalidURL_ReturnsEnvError tests execRequest with invalid URL
func TestExecRequest_InvalidURL_ReturnsEnvError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_API_KEY": "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: &http.Client{},
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "consumer:token:secret",
	}

	// Act
	_, _, err := api.execRequest("GET", "://invalid-url", "")

	// Assert
	if err == nil {
		t.Error("Expected error for invalid URL")
	}

	if _, ok := err.(*utils.EnvError); !ok {
		t.Errorf("Expected EnvError, got: %T", err)
	}
}

// TestAPIExecute_ContextCancellation_HandlesCorrectly tests APIExecute with context cancellation
func TestAPIExecute_ContextCancellation_HandlesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_ACCESS_URL": "http://test-maas.example.com/",
		"MAAS_API_KEY":    "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	mockClient := &mocks.MockHTTPClient{
		MockResponse: mocks.NewMockResponse(200, "OK"),
	}

	api := &CanonicalMaasAPIImple{
		Logger: klog.NewKlogr(),
		Client: mockClient,
		AccessUrl: "http://test-maas.example.com/",
		ApiKey:    "consumer:token:secret",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act
	statusCode, _, err := api.APIExecute(ctx, "GET", "api/test/", "")

	// Assert
	// Note: The current implementation doesn't actually use the context for cancellation
	// But we can still verify the request proceeds normally
	// In a real implementation, this should be enhanced to support context cancellation

	if err != nil {
		t.Errorf("Expected no error (context cancellation not implemented), got: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("Expected status code 200, got: %d", statusCode)
	}
}

// TestAPIExecute_DifferentHTTPMethods_WorksCorrectly tests APIExecute with different HTTP methods
func TestAPIExecute_DifferentHTTPMethods_WorksCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testEnv := map[string]string{
		"MAAS_ACCESS_URL": "http://test-maas.example.com/",
		"MAAS_API_KEY":    "consumer:token:secret",
	}
	setTestEnvForAPI(t, testEnv)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run("Method_"+method, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockClient := &mocks.MockHTTPClient{
				MockResponse: mocks.NewMockResponse(200, `{"method": "`+method+`"}`),
			}

			api := &CanonicalMaasAPIImple{
				Logger: klog.NewKlogr(),
				Client: mockClient,
				AccessUrl: "http://test-maas.example.com/",
				ApiKey:    "consumer:token:secret",
			}

			ctx := context.Background()

			// Act
			statusCode, _, err := api.APIExecute(ctx, method, "api/test/", "")

			// Assert
			if err != nil {
				t.Errorf("Expected no error for method %s, got: %v", method, err)
			}
			if statusCode != 200 {
				t.Errorf("Expected status code 200 for method %s, got: %d", method, statusCode)
			}

			// Verify request method
			lastRequest := mockClient.GetLastRequest()
			if lastRequest == nil {
				t.Errorf("Expected HTTP request for method %s", method)
			} else {
				if lastRequest.Method != method {
					t.Errorf("Expected method %s, got %s", method, lastRequest.Method)
				}
			}
		})
	}
}
