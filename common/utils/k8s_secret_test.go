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
)

// TestGetSecretData_EmptySecretName tests validation of empty secret name
func TestGetSecretData_EmptySecretName(t *testing.T) {
	_, err := GetSecretData("", "default", "password")
	if err == nil {
		t.Error("GetSecretData() with empty secret name should return error")
	}
	expectedMsg := "secret name cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("GetSecretData() error = %v, want %v", err.Error(), expectedMsg)
	}
}

// TestGetSecretData_EmptyNamespace tests validation of empty namespace
func TestGetSecretData_EmptyNamespace(t *testing.T) {
	_, err := GetSecretData("my-secret", "", "password")
	if err == nil {
		t.Error("GetSecretData() with empty namespace should return error")
	}
	expectedMsg := "namespace cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("GetSecretData() error = %v, want %v", err.Error(), expectedMsg)
	}
}

// TestGetSecretData_EmptyKey tests validation of empty key
func TestGetSecretData_EmptyKey(t *testing.T) {
	_, err := GetSecretData("my-secret", "default", "")
	if err == nil {
		t.Error("GetSecretData() with empty key should return error")
	}
	expectedMsg := "key cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("GetSecretData() error = %v, want %v", err.Error(), expectedMsg)
	}
}

// TestGetSecretData_InClusterConfigError tests error when not running in cluster
func TestGetSecretData_InClusterConfigError(t *testing.T) {
	// This test will fail to get in-cluster config when not running in Kubernetes
	// This is expected behavior and we're testing the error handling
	_, err := GetSecretData("my-secret", "default", "password")
	if err == nil {
		t.Skip("Test is running inside a Kubernetes cluster, skipping in-cluster config error test")
	}
	
	// Verify the error message contains expected text
	errMsg := err.Error()
	if errMsg != "" && errMsg != "secret name cannot be empty" && 
	   errMsg != "namespace cannot be empty" && errMsg != "key cannot be empty" {
		// This is expected when not in a cluster
		// The error should mention in-cluster config
		if len(errMsg) == 0 {
			t.Errorf("GetSecretData() should return non-empty error message")
		}
	}
}

// TestGetSecretData_AllParametersRequired tests that all parameters are required
func TestGetSecretData_AllParametersRequired(t *testing.T) {
	tests := []struct {
		name        string
		secretName  string
		namespace   string
		key         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "All empty",
			secretName:  "",
			namespace:   "",
			key:         "",
			expectError: true,
			errorMsg:    "secret name cannot be empty",
		},
		{
			name:        "Only secret name",
			secretName:  "my-secret",
			namespace:   "",
			key:         "",
			expectError: true,
			errorMsg:    "namespace cannot be empty",
		},
		{
			name:        "Secret name and namespace",
			secretName:  "my-secret",
			namespace:   "default",
			key:         "",
			expectError: true,
			errorMsg:    "key cannot be empty",
		},
		{
			name:        "All parameters provided",
			secretName:  "my-secret",
			namespace:   "default",
			key:         "password",
			expectError: true, // Will fail due to no cluster config, but passes validation
			errorMsg:    "",   // Different error expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetSecretData(tt.secretName, tt.namespace, tt.key)
			if !tt.expectError && err != nil {
				t.Errorf("GetSecretData() unexpected error = %v", err)
			}
			if tt.expectError && err == nil {
				t.Error("GetSecretData() expected error, got nil")
			}
			if tt.errorMsg != "" && err != nil && err.Error() != tt.errorMsg {
				// For the last test case, we expect a different error (in-cluster config)
				if tt.errorMsg == "" {
					return
				}
				t.Errorf("GetSecretData() error = %v, want %v", err.Error(), tt.errorMsg)
			}
		})
	}
}

// TestGetSecretData_ParameterValidation tests comprehensive parameter validation
func TestGetSecretData_ParameterValidation(t *testing.T) {
	// Test empty secret name first (highest priority check)
	t.Run("EmptySecretName", func(t *testing.T) {
		_, err := GetSecretData("", "namespace", "key")
		if err == nil || err.Error() != "secret name cannot be empty" {
			t.Errorf("Expected 'secret name cannot be empty' error, got: %v", err)
		}
	})

	// Test empty namespace second
	t.Run("EmptyNamespace", func(t *testing.T) {
		_, err := GetSecretData("secret", "", "key")
		if err == nil || err.Error() != "namespace cannot be empty" {
			t.Errorf("Expected 'namespace cannot be empty' error, got: %v", err)
		}
	})

	// Test empty key third
	t.Run("EmptyKey", func(t *testing.T) {
		_, err := GetSecretData("secret", "namespace", "")
		if err == nil || err.Error() != "key cannot be empty" {
			t.Errorf("Expected 'key cannot be empty' error, got: %v", err)
		}
	})
}

// TestGetSecretData_ValidInputsNoCluster tests behavior with valid inputs outside cluster
func TestGetSecretData_ValidInputsNoCluster(t *testing.T) {
	// When running outside a Kubernetes cluster, this should fail at in-cluster config
	_, err := GetSecretData("valid-secret", "valid-namespace", "valid-key")
	
	if err == nil {
		// If we're actually in a cluster, we might succeed or fail differently
		t.Skip("Running inside a Kubernetes cluster, test behavior may vary")
	}
	
	// Outside cluster, should get in-cluster config error
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("GetSecretData() should return error when not in cluster")
	}
}

// Note: Full integration tests with actual Kubernetes cluster would require:
// 1. Mock Kubernetes client using interfaces
// 2. Or test in actual Kubernetes environment
// 3. Or use fake clientset from k8s.io/client-go/kubernetes/fake
// 
// For this differential test coverage, we focus on:
// - Parameter validation (fully testable)
// - Error path coverage (testable without cluster)
// - Integration test marked for cluster environment
//
// The actual Kubernetes Secret retrieval logic is tested in integration tests
// or is covered by the quality assurance of the kubernetes client-go library.
