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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/klog/v2"

	"log_module/internal/server/test_utils"
)

// TestAPIImplement_APIExecuteJWTAuth_ValidRequest_ReturnsSuccess tests APIExecuteJWTAUth with valid input
func TestAPIImplement_APIExecuteJWTAuth_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify JWT token
		authHeader := r.Header.Get("Authorization")
		assert.Contains(t, authHeader, "Bearer test-jwt-token")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Create API instance
	api := APIImplement{Logger: klog.Background()}

	// Execute
	result, err := api.APIExecuteJWTAUth(context.Background(), "GET", server.URL, "test-api", "test-jwt-token", "")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestAPIImplement_APIExecuteJWTAuth_WithQueryParameter_ReturnsSuccess tests APIExecuteJWTAUth with query parameter
func TestAPIImplement_APIExecuteJWTAuth_WithQueryParameter_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameter
		assert.Equal(t, "value1", r.URL.Query().Get("param1"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Create API instance
	api := APIImplement{Logger: klog.Background()}

	// Execute
	result, err := api.APIExecuteJWTAUth(context.Background(), "GET", server.URL, "test-api", "test-jwt-token", "param1=value1")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestAPIImplement_APIExecuteJWTAuth_InvalidURL_ReturnsError tests APIExecuteJWTAUth with invalid URL
func TestAPIImplement_APIExecuteJWTAuth_InvalidURL_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create API instance
	api := APIImplement{Logger: klog.Background()}

	// Execute with invalid URL (missing protocol)
	result, err := api.APIExecuteJWTAUth(context.Background(), "GET", "://invalid-url", "test-api", "test-jwt-token", "")

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestAPIImplement_APIExecuteJWTAuth_ServerError_ReturnsError tests APIExecuteJWTAUth with server error
func TestAPIImplement_APIExecuteJWTAuth_ServerError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	// Create API instance
	api := APIImplement{Logger: klog.Background()}

	// Execute
	result, err := api.APIExecuteJWTAUth(context.Background(), "GET", server.URL, "test-api", "test-jwt-token", "")

	// Verify - should return error for 500 status
	assert.Error(t, err)
	assert.Nil(t, result)
}
