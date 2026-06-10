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

package test_utils

import (
	"flag"
	"os"
	"sync"
	"testing"

	"k8s.io/klog/v2"
)

var testKlogInitOnce sync.Once

// InitKlogForTest initializes klog for testing with appropriate log level
func InitKlogForTest(t *testing.T) {
	testKlogInitOnce.Do(func() {
		// Initialize klog for testing
		klog.InitFlags(nil)

		// Set log level for test environment
		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			logLevel = "2" // Default to V(2) level for testing
		}

		err := flag.Set("v", logLevel)
		if err != nil {
			t.Logf("Failed to set log level: %v", err)
		}

		// Integrate log output with test logs
		flag.Set("logtostderr", "true")
		flag.Set("stderrthreshold", "INFO")

		flag.Parse()
	})
}

// SetupTestEnvironmentWithKlog sets up test environment with klog initialization
func SetupTestEnvironmentWithKlog(t *testing.T) func() {
	// Set environment variables
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalProductMappings := os.Getenv("PRODUCT_MAPPINGS")
	
	os.Setenv("LOG_LEVEL", "2")
	// Set product mappings for test - supports Dummy, EdgeCore, Broadcom
	os.Setenv("PRODUCT_MAPPINGS", `{"nw_products":[{"vendor":"Dummy","type":"Dummy"},{"vendor":"EdgeCore","type":"EdgeCoreSonic"},{"vendor":"Broadcom","type":"BroadcomSonic"}]}`)

	// Initialize klog
	InitKlogForTest(t)

	// Return cleanup function
	return func() {
		if originalLogLevel != "" {
			os.Setenv("LOG_LEVEL", originalLogLevel)
		} else {
			os.Unsetenv("LOG_LEVEL")
		}
		if originalProductMappings != "" {
			os.Setenv("PRODUCT_MAPPINGS", originalProductMappings)
		} else {
			os.Unsetenv("PRODUCT_MAPPINGS")
		}
		klog.Flush()
	}
}
