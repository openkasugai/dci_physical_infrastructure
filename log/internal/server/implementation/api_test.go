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
	"encoding/json"
	"log_module/internal/server/test_utils"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

func Test_Api(t *testing.T) {
	if os.Getenv("RUN_HW_API_TESTS") != "1" {
		t.Skip("Skipping hardware connectivity test in CI/local by default. Set RUN_HW_API_TESTS=1 to run.")
	}

	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	/*********************************
	 * Setting up configuration
	 *********************************/
	_ = os.Setenv("LOG_LEVEL", "0")

	/*********************************
	 * Mock configuration and Prerequisites
	 *********************************/
	ctx := context.Background()
	logger := klog.Background()
	api := &APIImplement{Logger: logger}

	testCases := []struct {
		name           string
		method         string
		url            string
		apiname        string
		loginUser      string
		loginPass      string
		queryParameter string
		success        bool
	}{
		{
			name:           "GET HW system info",
			method:         "GET",
			url:            "https://172.31.16.100",
			apiname:        "redfish/v1/Systems/0",
			loginUser:      "admin",
			loginPass:      "6g-infra-poc",
			queryParameter: "",
			success:        true,
		},
	}

	var failed bool
	for _, tc := range testCases {
		if failed {
			break
		}
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			t.Log("---" + tc.name + " started-----------------------")
			defer func() { t.Log("---" + tc.name + " finished----------------------") }()

			/*********************************
			 * Executing the target method
			 *********************************/
			output, err := api.APIExecuteUserAuth(ctx,
				tc.method, tc.url, tc.apiname, tc.loginUser, tc.loginPass, tc.queryParameter)

			/*********************************
			 * Expected value check
			 *********************************/
			if tc.success {
				if err != nil {
					t.Error("!! FAILED !!", "Unexpected error:", err, "expected: nil")
					failed = true
				}
				jsonStr, _ := json.MarshalIndent(output, "", "  ")
				t.Log("jsonStr:", string(jsonStr))
			} else {
				if err == nil {
					t.Error("!! FAILED !!", "Unexpected error:", "nil", "expected: not nil")
					failed = true
				}
				t.Log("err:", err)
				jsonStr, _ := json.MarshalIndent(output, "", "  ")
				t.Log("jsonStr:", string(jsonStr))
			}
		})
	}

	// Cleanup
	_ = os.Unsetenv("LOG_LEVEL")
}

// Helper function to set environment variable and return cleanup function
func setEnvAPI(key, value string) func() {
	original := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		if original != "" {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	}
}

// TestAPIImplement_APIExecute_ValidRequest_ReturnsSuccess tests successful API execution with mock server
func TestAPIImplement_APIExecute_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()

	// Create mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/redfish/v1/Systems/0", r.URL.Path)

		// Check basic auth
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "admin", username)
		assert.Equal(t, "password", password)

		// Return valid JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"@odata.type": "#ComputerSystem.v1_0_0.ComputerSystem",
			"Id":          "System",
			"Name":        "System",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Extract host from server URL (remove https://)
	// URL already includes https://

	// Execute
	resp, err := api.APIExecuteUserAuth(context.Background(), "GET", server.URL, "redfish/v1/Systems/0", "admin", "password", "")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	respMap, ok := resp.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "#ComputerSystem.v1_0_0.ComputerSystem", respMap["@odata.type"])
}

// TestAPIImplement_APIExecute_InvalidMethod_ReturnsError tests invalid HTTP method
func TestAPIImplement_APIExecute_InvalidMethod_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Execute with invalid HTTP method
	_, err := api.APIExecuteUserAuth(context.Background(), "INVALID METHOD", "https://localhost", "api/test", "user", "pass", "")

	// Verify
	assert.Error(t, err)
}

// TestAPIImplement_APIExecute_InvalidURL_ReturnsError tests invalid URL
func TestAPIImplement_APIExecute_InvalidURL_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Execute with invalid URL (contains invalid characters)
	_, err := api.APIExecuteUserAuth(context.Background(), "GET", "invalid url with spaces", "api/test", "user", "pass", "")

	// Verify
	assert.Error(t, err)
}

// TestAPIImplement_APIExecute_HTTPError_ReturnsError tests HTTP error responses
func TestAPIImplement_APIExecute_HTTPError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()

	// Create mock server that returns 404
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Extract host from server URL
	// URL already includes https://

	// Execute
	_, err := api.APIExecuteUserAuth(context.Background(), "GET", server.URL, "api/notfound", "admin", "password", "")

	// Verify
	assert.Error(t, err)
	customErr, ok := err.(*CustomError)
	assert.True(t, ok)
	assert.Equal(t, 404, customErr.StatusCode)
}

// TestAPIImplement_APIExecute_InvalidJSON_ReturnsError tests invalid JSON response
func TestAPIImplement_APIExecute_InvalidJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()

	// Create mock server that returns invalid JSON
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Extract host from server URL
	// URL already includes https://

	// Execute
	_, err := api.APIExecuteUserAuth(context.Background(), "GET", server.URL, "api/invalid", "admin", "password", "")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response body is invalid for json format")
}

// TestAPIImplement_execRequestUserAuth_ValidRequest_ReturnsSuccess tests execRequestUserAuth method directly
func TestAPIImplement_execRequestUserAuth_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()

	// Create mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"status": "success",
			"data":   "test_data",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	// Execute
	resp, err := api.execRequestUserAuth("GET", server.URL, "api/test", "admin", "password", "")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	respMap, ok := resp.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "success", respMap["status"])
}

// TestAPIImplement_extracJsonResponse_ValidJSON_ReturnsMap tests JSON extraction
func TestAPIImplement_extracJsonResponse_ValidJSON_ReturnsMap(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	validJSON := []byte(`{"status": "success", "data": {"key": "value"}}`)

	// Execute
	result, err := api.extracJsonResponse(validJSON)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "success", resultMap["status"])

	data, ok := resultMap["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "value", data["key"])
}

// TestAPIImplement_extracJsonResponse_InvalidJSON_ReturnsError tests invalid JSON
func TestAPIImplement_extracJsonResponse_InvalidJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	invalidJSON := []byte(`invalid json content`)

	// Execute
	_, err := api.extracJsonResponse(invalidJSON)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response body is invalid for json format")
}

// TestAPIImplement_extracJsonResponse_EmptyJSON_ReturnsError tests empty JSON
func TestAPIImplement_extracJsonResponse_EmptyJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	emptyJSON := []byte(``)

	// Execute
	_, err := api.extracJsonResponse(emptyJSON)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response body is invalid for json format")
}

// TestAPIImplement_extracJsonResponse_NullJSON_ReturnsError tests null JSON
func TestAPIImplement_extracJsonResponse_NullJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()
	logger := klog.NewKlogr()
	api := &APIImplement{Logger: logger}

	nullJSON := []byte(`null`)

	// Execute
	result, err := api.extracJsonResponse(nullJSON)

	// Verify - null JSON should be valid but result should be nil
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestAPIImplement_APIExecute_HTTPClientDoError_ReturnsError tests HTTP client.Do() error
func TestAPIImplement_APIExecute_HTTPClientDoError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnvAPI("LOG_LEVEL", "2")()
	logger := klog.NewKlogr()

	// Create and immediately close a server to get a closed port
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Close immediately to make the port unavailable

	api := &APIImplement{Logger: logger}

	// Execute with closed server to trigger connection error
	_, err := api.APIExecuteUserAuth(context.Background(), "GET", serverURL, "api/test", "admin", "password", "")

	// Verify
	assert.Error(t, err)
	// Should be a connection error
	_, isCustomError := err.(*CustomError)
	assert.False(t, isCustomError, "Should not be a CustomError, but a network error")
}
