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
	// Set default environment variables
	originalEnvVars := make(map[string]string)
	defaultEnvVars := map[string]string{
		"LOG_LEVEL":                         "2",
		"MAAS_SERVER_PORT":                  "50051",
		"VM_HOST_DISK":                      "50",
		"LXD_PORT":                          "8443",
		"SSH_KEY":                           "/test/ssh_key",
		"TLS_ENABLE":                        "false",
		"TLS_CERT_PATH":                     "/test/cert",
		"K8S_JOB_WORKER_IMAGE":              "worker:latest",
		"K8S_JOB_SERVICE_ACCOUNT":           "test-sa",
		"K8S_JOB_SSH_SECRET_NAME":           "test-ssh-secret",
		"K8S_JOB_TLS_SECRET_NAME":           "test-tls-secret",
		"K8S_JOB_BACKOFF_LIMIT":             "1",
		"K8S_JOB_TTL_SECONDS":               "600",
		"K8S_JOB_ACTIVE_DEADLINE_SECONDS":   "7200",
	}

	// Save original values and set defaults
	for key, defaultValue := range defaultEnvVars {
		originalEnvVars[key] = os.Getenv(key)
		os.Setenv(key, defaultValue)
	}

	// Initialize klog
	InitKlogForTest(t)

	// Return cleanup function
	return func() {
		for key, originalValue := range originalEnvVars {
			if originalValue != "" {
				os.Setenv(key, originalValue)
			} else {
				os.Unsetenv(key)
			}
		}
		klog.Flush()
	}
}
