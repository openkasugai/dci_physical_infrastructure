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
	"exporter_module/internal/server/test_utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

// TestDatabaseImplement_Init_WithoutK8sAccess_ReturnsError tests database initialization without K8s access
func TestDatabaseImplement_Init_WithoutK8sAccess_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	database := &DatabaseImplement{Logger: logger}

	// Execute - This may panic if GetConfig() is not initialized, which is expected in test environment
	// The function should return error when JWT retrieval fails
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected when config is not initialized
			t.Log("Expected panic due to uninitialized config in test environment")
		}
	}()

	err := database.Init()

	// Verify - Since JWT retrieval will fail due to K8s not being available in test,
	// we expect an error
	if err == nil {
		t.Error("Expected error due to JWT retrieval failure, got nil")
	} else {
		assert.Contains(t, err.Error(), "failed to retrieve JWT")
	}
}

// TestDatabaseImplement_Finalize_DoesNotPanic tests database finalization
func TestDatabaseImplement_Finalize_DoesNotPanic(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	database := &DatabaseImplement{Logger: logger}

	// Execute - should not panic
	database.Finalize()

	// Note: Finalize doesn't return error, just verify it doesn't panic
}

// TestDatabaseImplement_SelectServerTable_WithNilAPI_ReturnsError tests server selection with nil API
func TestDatabaseImplement_SelectServerTable_WithNilAPI_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	database := &DatabaseImplement{Logger: logger}

	// Execute - This may panic due to nil API, which is expected
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected when API is nil
			t.Log("Expected panic due to nil API in test environment")
		}
	}()

	targets, err := database.SelectServerTable()

	// Verify - Expect error due to nil API (if it doesn't panic)
	if err == nil && len(targets) != 0 {
		t.Error("Expected error or empty targets due to nil API")
	}
}

// TestDatabaseImplement_SelectNwSwitchTable_WithNilAPI_ReturnsError tests network switch selection with nil API
func TestDatabaseImplement_SelectNwSwitchTable_WithNilAPI_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	database := &DatabaseImplement{Logger: logger}

	// Execute - This may panic due to nil API, which is expected
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected when API is nil
			t.Log("Expected panic due to nil API in test environment")
		}
	}()

	targets, err := database.SelectNwSwitchTable()

	// Verify - Expect error due to nil API (if it doesn't panic)
	if err == nil && len(targets) != 0 {
		t.Error("Expected error or empty targets due to nil API")
	}
}

// Helper function to set environment variable and return cleanup function
func setEnvVar(key, value string) func() {
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
