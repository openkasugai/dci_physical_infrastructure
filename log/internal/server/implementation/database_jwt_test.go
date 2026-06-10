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
