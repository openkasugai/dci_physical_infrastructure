package implementation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	"log_module/internal/server/test_utils"
	"log_module/internal/server/utils"
)

// TestDatabaseImplement_Init_GetSecretDataError_ReturnsError tests JWT retrieval failure
// This test covers the error path at database.go:51-54 where GetSecretData fails
func TestDatabaseImplement_Init_GetSecretDataError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - set all required environment variables
	defer setEnvDB("LOG_LEVEL", "2")()
	defer setEnvDB("INTERVAL", "60")()
	defer setEnvDB("IPMI_LOGFILE", "ipmi.log")()
	defer setEnvDB("IPMI_LOGPATH", "/var/log/ipmi")()
	defer setEnvDB("IPMI_MAXSIZE", "100")()
	defer setEnvDB("IPMI_MAXBACKUPS", "5")()
	defer setEnvDB("IPMI_MAXAGE", "7")()
	defer setEnvDB("CDI_LOGFILE", "cdi.log")()
	defer setEnvDB("CDI_LOGPATH", "/var/log/cdi")()
	defer setEnvDB("CDI_MAXSIZE", "200")()
	defer setEnvDB("CDI_MAXBACKUPS", "10")()
	defer setEnvDB("CDI_MAXAGE", "14")()
	defer setEnvDB("DB_URL", "https://postgrest:3000")()

	// Reset and initialize config
	utils.ResetConfigForTesting()
	err := utils.InitializeConfig()
	assert.NoError(t, err)

	logger := klog.NewKlogr()
	db := &DatabaseImplement{Logger: logger}

	// Execute - In test environment (not in a Kubernetes cluster),
	// GetSecretData will fail because rest.InClusterConfig() fails
	err = db.Init()

	// Verify - should fail with JWT retrieval error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve JWT")
}
