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
	os.Setenv("LOG_LEVEL", "2")

	// Initialize klog
	InitKlogForTest(t)

	// Return cleanup function
	return func() {
		if originalLogLevel != "" {
			os.Setenv("LOG_LEVEL", originalLogLevel)
		} else {
			os.Unsetenv("LOG_LEVEL")
		}
		klog.Flush()
	}
}

// SetupProductMappings sets up PRODUCT_MAPPINGS environment variable for tests
func SetupProductMappings() func() {
	originalMappings := os.Getenv("PRODUCT_MAPPINGS")
	
	// Set product mappings for all tests
	productMappings := `{
		"server_products":[
			{"vendor":"dell","product_name":"PowerEdge","version":"","type":"Dell"},
			{"vendor":"fujitsu","product_name":"PRIMERGY","version":"","type":"Primergy"}
		],
		"cdi_products":[
			{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1","type":"PG_CDI_1_1"},
			{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.0","type":"PG_CDI_1_0"}
		],
		"maas_products":[
			{"vendor":"canonical","product_name":"MAAS","version":"3.6.2","type":"Canonical"}
		]
	}`
	os.Setenv("PRODUCT_MAPPINGS", productMappings)
	
	// Return cleanup function
	return func() {
		if originalMappings != "" {
			os.Setenv("PRODUCT_MAPPINGS", originalMappings)
		} else {
			os.Unsetenv("PRODUCT_MAPPINGS")
		}
	}
}
