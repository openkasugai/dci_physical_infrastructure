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
	// Save original environment variables
	originalVars := make(map[string]string)
	envVars := []string{
		"LOG_LEVEL",
		"INTERVAL",
		"P2P_ENABLE",
		"P2P_INTERVAL",
		"ANSIBLE_PATH",
		"SSH_KEY",
		"METRICS_PORT",
		"METRICS_ENDPOINT",
		"DB_URL",
	}

	for _, key := range envVars {
		originalVars[key] = os.Getenv(key)
	}

	// Set test environment variables with defaults
	testEnvVars := map[string]string{
		"LOG_LEVEL":        "2",
		"INTERVAL":         "60",
		"P2P_ENABLE":       "true",
		"P2P_INTERVAL":     "300",
		"ANSIBLE_PATH":     "/usr/bin/ansible-playbook",
		"SSH_KEY":          "/path/to/ssh/key",
		"METRICS_PORT":     "9090",
		"METRICS_ENDPOINT": "/metrics",
		"DB_URL":           "https://postgrest:3000",
	}

	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}

	// Initialize klog
	InitKlogForTest(t)

	// Return cleanup function
	return func() {
		for key, originalValue := range originalVars {
			if originalValue != "" {
				os.Setenv(key, originalValue)
			} else {
				os.Unsetenv(key)
			}
		}
		klog.Flush()
	}
}
