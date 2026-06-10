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
	"context"
	"exporter_module/internal/server/test_utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/klog/v2"
)

// TestAPIImplement_ApiExecute_ValidRequest_ReturnsSuccess tests successful API execution
func TestAPIImplement_ApiExecute_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify basic auth
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify headers
		if r.Header.Get("Accept") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success", "data": {"value": 123}}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	resultInterface, err := api.APIExecuteUserAuth(ctx, "GET", url, "api/test", "testuser", "testpass", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if resultInterface == nil {
		t.Error("Expected result to be non-nil")
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["result"] != "success" {
		t.Errorf("Expected result field to be 'success', got: %v", result["result"])
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected data field to be map[string]interface{}, got: %T", result["data"])
	}

	if data["value"] != float64(123) {
		t.Errorf("Expected value field to be 123, got: %v", data["value"])
	}
}

// TestAPIImplement_ApiExecute_Unauthorized_ReturnsError tests API execution with unauthorized response
func TestAPIImplement_ApiExecute_Unauthorized_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	_, err := api.APIExecuteUserAuth(ctx, "GET", url, "api/test", "wronguser", "wrongpass", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	customErr, ok := err.(*CustomError)
	if !ok {
		t.Errorf("Expected CustomError, got: %T", err)
	}

	if customErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, customErr.StatusCode)
	}
}

// TestAPIImplement_ApiExecute_InternalServerError_ReturnsError tests API execution with server error
func TestAPIImplement_ApiExecute_InternalServerError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	_, err := api.APIExecuteUserAuth(ctx, "GET", url, "api/test", "testuser", "testpass", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	customErr, ok := err.(*CustomError)
	if !ok {
		t.Errorf("Expected CustomError, got: %T", err)
	}

	if customErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, customErr.StatusCode)
	}
}

// TestAPIImplement_ApiExecute_InvalidJSON_ReturnsError tests API execution with invalid JSON response
func TestAPIImplement_ApiExecute_InvalidJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json format`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	_, err := api.APIExecuteUserAuth(ctx, "GET", url, "api/test", "testuser", "testpass", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "response body is invalid for json format" {
		t.Errorf("Expected specific JSON error message, got: %v", err.Error())
	}
}

// TestAPIImplement_ApiExecute_InvalidHost_ReturnsError tests API execution with invalid host
func TestAPIImplement_ApiExecute_InvalidHost_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Execute with invalid host
	_, err := api.APIExecuteUserAuth(ctx, "GET", "https://invalid-host:99999", "api/test", "testuser", "testpass", "")

	// Verify
	if err == nil {
		t.Error("Expected error due to invalid host, got nil")
	}
}

// TestAPIImplement_ApiExecute_PostMethod_ReturnsSuccess tests API execution with POST method
func TestAPIImplement_ApiExecute_PostMethod_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"result": "created", "id": 456}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	resultInterface, err := api.APIExecuteUserAuth(ctx, "POST", url, "api/create", "testuser", "testpass", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["result"] != "created" {
		t.Errorf("Expected result field to be 'created', got: %v", result["result"])
	}

	if result["id"] != float64(456) {
		t.Errorf("Expected id field to be 456, got: %v", result["id"])
	}
}

// TestAPIImplement_execRequest_ValidRequest_ReturnsSuccess tests execRequest with valid request
func TestAPIImplement_execRequest_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "test response"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Create HTTP request
	req, err := http.NewRequest("GET", server.URL+"/api/status", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth("testuser", "testpass")
	req.Header.Set("Accept", "application/json")

	// Execute
	resultInterface, err := api.execRequest(req)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status field to be 'ok', got: %v", result["status"])
	}

	if result["message"] != "test response" {
		t.Errorf("Expected message field to be 'test response', got: %v", result["message"])
	}
}

// TestAPIImplement_execRequest_InvalidMethod_ReturnsError tests execRequest with invalid HTTP method
func TestAPIImplement_execRequest_InvalidMethod_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Execute with invalid HTTP method - NewRequest will fail
	req, err := http.NewRequest("INVALID METHOD", "https://localhost/api/test", nil)
	if err == nil {
		req.SetBasicAuth("user", "pass")
		_, err = api.execRequest(req)
	}

	// Verify
	if err == nil {
		t.Error("Expected error due to invalid HTTP method, got nil")
	}
}

// TestAPIImplement_execRequest_BadRequest_ReturnsError tests execRequest with bad request
func TestAPIImplement_execRequest_BadRequest_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Bad Request", "details": "Invalid parameters"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Create HTTP request
	req, err := http.NewRequest("GET", server.URL+"/api/bad", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth("testuser", "testpass")

	// Execute
	_, err = api.execRequest(req)

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	customErr, ok := err.(*CustomError)
	if !ok {
		t.Errorf("Expected CustomError, got: %T", err)
	}

	if customErr.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, customErr.StatusCode)
	}
}

// TestAPIImplement_execRequest_EmptyResponse_ReturnsError tests execRequest with empty response
func TestAPIImplement_execRequest_EmptyResponse_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Empty response body
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Create HTTP request
	req, err := http.NewRequest("GET", server.URL+"/api/empty", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth("testuser", "testpass")

	// Execute
	_, err = api.execRequest(req)

	// Verify
	if err == nil {
		t.Error("Expected error due to empty JSON, got nil")
	}
}

// TestAPIImplement_extracJsonResponse_ValidJSON_ReturnsMap tests extracJsonResponse with valid JSON
func TestAPIImplement_extracJsonResponse_ValidJSON_ReturnsMap(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	jsonBytes := []byte(`{"test": "value", "number": 42, "nested": {"key": "nested_value"}}`)

	// Execute
	resultInterface, err := api.extracJsonResponse(jsonBytes)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["test"] != "value" {
		t.Errorf("Expected test field to be 'value', got: %v", result["test"])
	}

	if result["number"] != float64(42) {
		t.Errorf("Expected number field to be 42, got: %v", result["number"])
	}

	nested, ok := result["nested"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected nested field to be map[string]interface{}, got: %T", result["nested"])
	}

	if nested["key"] != "nested_value" {
		t.Errorf("Expected nested key to be 'nested_value', got: %v", nested["key"])
	}
}

// TestAPIImplement_extracJsonResponse_InvalidJSON_ReturnsError tests extracJsonResponse with invalid JSON
func TestAPIImplement_extracJsonResponse_InvalidJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	invalidJsonBytes := []byte(`{invalid json format}`)

	// Execute
	_, err := api.extracJsonResponse(invalidJsonBytes)

	// Verify
	if err == nil {
		t.Error("Expected error due to invalid JSON, got nil")
	}

	if err.Error() != "response body is invalid for json format" {
		t.Errorf("Expected specific error message, got: %v", err.Error())
	}
}

// TestAPIImplement_extracJsonResponse_EmptyJSON_ReturnsError tests extracJsonResponse with empty input
func TestAPIImplement_extracJsonResponse_EmptyJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	emptyBytes := []byte(``)

	// Execute
	_, err := api.extracJsonResponse(emptyBytes)

	// Verify
	if err == nil {
		t.Error("Expected error due to empty JSON, got nil")
	}

	if err.Error() != "response body is invalid for json format" {
		t.Errorf("Expected specific error message, got: %v", err.Error())
	}
}

// TestCustomError_Error_ReturnsFormattedMessage tests CustomError.Error method
func TestCustomError_Error_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	err := &CustomError{
		StatusCode: 404,
		Message:    "Resource not found",
	}

	expected := "<404> Resource not found"
	actual := err.Error()

	if actual != expected {
		t.Errorf("Expected '%s', got '%s'", expected, actual)
	}
}

// TestCustomError_Error_EmptyMessage_ReturnsFormattedMessage tests CustomError.Error with empty message
func TestCustomError_Error_EmptyMessage_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	err := &CustomError{
		StatusCode: 500,
		Message:    "",
	}

	expected := "<500> "
	actual := err.Error()

	if actual != expected {
		t.Errorf("Expected '%s', got '%s'", expected, actual)
	}
}

// TestCustomError_Error_ZeroStatusCode_ReturnsFormattedMessage tests CustomError.Error with zero status code
func TestCustomError_Error_ZeroStatusCode_ReturnsFormattedMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	err := &CustomError{
		StatusCode: 0,
		Message:    "Unknown error",
	}

	expected := "<0> Unknown error"
	actual := err.Error()

	if actual != expected {
		t.Errorf("Expected '%s', got '%s'", expected, actual)
	}
}

// TestAPIImplement_TLSConfiguration_UsesCorrectSettings tests that TLS configuration is properly set
func TestAPIImplement_TLSConfiguration_UsesCorrectSettings(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server with specific TLS requirements
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tls": "configured"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute - this should work with our TLS configuration
	resultInterface, err := api.APIExecuteUserAuth(ctx, "GET", url, "api/tls", "testuser", "testpass", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error with TLS configuration, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["tls"] != "configured" {
		t.Errorf("Expected tls field to be 'configured', got: %v", result["tls"])
	}
}

// TestAPIImplement_ApiExecute_LongApiName_ReturnsSuccess tests API execution with long API name
func TestAPIImplement_ApiExecute_LongApiName_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	longApiName := "api/v1/very/long/path/with/many/segments/test"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+longApiName {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"path": "long"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	resultInterface, err := api.APIExecuteUserAuth(ctx, "GET", url, longApiName, "testuser", "testpass", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["path"] != "long" {
		t.Errorf("Expected path field to be 'long', got: %v", result["path"])
	}
}

// TestAPIImplement_ApiExecuteJWT_ValidRequest_ReturnsSuccess tests successful API execution with JWT authentication
func TestAPIImplement_ApiExecuteJWT_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify JWT Bearer token
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-jwt-token-12345" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify headers
		if r.Header.Get("Accept") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success", "data": {"value": 456}}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	resultInterface, err := api.APIExecuteJWTAUth(ctx, "GET", url, "api/test", "test-jwt-token-12345", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if resultInterface == nil {
		t.Error("Expected result to be non-nil")
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["result"] != "success" {
		t.Errorf("Expected result field to be 'success', got: %v", result["result"])
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected data field to be map[string]interface{}, got: %T", result["data"])
	}

	if data["value"] != float64(456) {
		t.Errorf("Expected value field to be 456, got: %v", data["value"])
	}
}

// TestAPIImplement_ApiExecuteJWT_Unauthorized_ReturnsError tests API execution with unauthorized JWT
func TestAPIImplement_ApiExecuteJWT_Unauthorized_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized - Invalid JWT"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	_, err := api.APIExecuteJWTAUth(ctx, "GET", url, "api/test", "invalid-jwt-token", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	customErr, ok := err.(*CustomError)
	if !ok {
		t.Errorf("Expected CustomError, got: %T", err)
	}

	if customErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, customErr.StatusCode)
	}
}

// TestAPIImplement_ApiExecuteJWT_InternalServerError_ReturnsError tests API execution with server error
func TestAPIImplement_ApiExecuteJWT_InternalServerError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	_, err := api.APIExecuteJWTAUth(ctx, "GET", url, "api/test", "test-jwt-token", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	customErr, ok := err.(*CustomError)
	if !ok {
		t.Errorf("Expected CustomError, got: %T", err)
	}

	if customErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, customErr.StatusCode)
	}
}

// TestAPIImplement_ApiExecuteJWT_InvalidJSON_ReturnsError tests API execution with invalid JSON response
func TestAPIImplement_ApiExecuteJWT_InvalidJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json format`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	_, err := api.APIExecuteJWTAUth(ctx, "GET", url, "api/test", "test-jwt-token", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "response body is invalid for json format" {
		t.Errorf("Expected specific JSON error message, got: %v", err.Error())
	}
}

// TestAPIImplement_ApiExecuteJWT_InvalidHost_ReturnsError tests API execution with invalid host
func TestAPIImplement_ApiExecuteJWT_InvalidHost_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Execute with invalid host
	_, err := api.APIExecuteJWTAUth(ctx, "GET", "https://invalid-host:99999", "api/test", "test-jwt-token", "")

	// Verify
	if err == nil {
		t.Error("Expected error due to invalid host, got nil")
	}
}

// TestAPIImplement_ApiExecuteJWT_PostMethod_ReturnsSuccess tests API execution with POST method
func TestAPIImplement_ApiExecuteJWT_PostMethod_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Verify JWT token
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-jwt-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"result": "created", "id": 789}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	resultInterface, err := api.APIExecuteJWTAUth(ctx, "POST", url, "api/create", "test-jwt-token", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["result"] != "created" {
		t.Errorf("Expected result field to be 'created', got: %v", result["result"])
	}

	if result["id"] != float64(789) {
		t.Errorf("Expected id field to be 789, got: %v", result["id"])
	}
}

// TestAPIImplement_ApiExecuteJWT_WithQueryParameter_ReturnsSuccess tests API execution with query parameter
func TestAPIImplement_ApiExecuteJWT_WithQueryParameter_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameter
		if r.URL.RawQuery != "status=eq.1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"query": "received"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}
	ctx := context.Background()

	// Get server URL
	url := server.URL

	// Execute
	resultInterface, err := api.APIExecuteJWTAUth(ctx, "GET", url, "api/test", "test-jwt-token", "status=eq.1")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["query"] != "received" {
		t.Errorf("Expected query field to be 'received', got: %v", result["query"])
	}
}

// TestAPIImplement_execRequestJWTAuth_ValidRequest_ReturnsSuccess tests execRequestJWTAuth with valid request
func TestAPIImplement_execRequestJWTAuth_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer my-jwt-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "jwt authenticated"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Execute
	resultInterface, err := api.execRequestJWTAuth("GET", server.URL, "api/status", "my-jwt-token", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status field to be 'ok', got: %v", result["status"])
	}

	if result["message"] != "jwt authenticated" {
		t.Errorf("Expected message field to be 'jwt authenticated', got: %v", result["message"])
	}
}

// TestAPIImplement_execRequestJWTAuth_WithQueryParameter_ReturnsSuccess tests execRequestJWTAuth with query parameter
func TestAPIImplement_execRequestJWTAuth_WithQueryParameter_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameter
		if r.URL.RawQuery != "id=123&name=test" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "invalid query"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"query_ok": true}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Execute
	resultInterface, err := api.execRequestJWTAuth("GET", server.URL, "api/query", "test-jwt", "id=123&name=test")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	result, ok := resultInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", resultInterface)
		return
	}

	if result["query_ok"] != true {
		t.Errorf("Expected query_ok field to be true, got: %v", result["query_ok"])
	}
}

// TestAPIImplement_execRequestJWTAuth_BadRequest_ReturnsError tests execRequestJWTAuth with bad request
func TestAPIImplement_execRequestJWTAuth_BadRequest_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Bad Request", "details": "Invalid parameters"}`))
	}))
	defer server.Close()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Execute
	_, err := api.execRequestJWTAuth("GET", server.URL, "api/bad", "test-jwt", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	customErr, ok := err.(*CustomError)
	if !ok {
		t.Errorf("Expected CustomError, got: %T", err)
	}

	if customErr.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, customErr.StatusCode)
	}
}

// TestAPIImplement_execRequestJWTAuth_InvalidMethod_ReturnsError tests execRequestJWTAuth with invalid HTTP method
func TestAPIImplement_execRequestJWTAuth_InvalidMethod_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Execute with invalid HTTP method that will cause NewRequest to fail
	_, err := api.execRequestJWTAuth("INVALID METHOD WITH SPACES", "https://localhost", "api/test", "test-jwt", "")

	// Verify
	if err == nil {
		t.Error("Expected error due to invalid HTTP method, got nil")
	}
}
