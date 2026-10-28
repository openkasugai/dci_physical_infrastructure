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
