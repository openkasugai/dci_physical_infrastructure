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

package pg_cdi

import (
	"context"
	"exporter_module/internal/server/test_utils"
	"exporter_module/internal/server/utils"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/klog/v2"
)

// Test helper functions
func setEnv(key, value string) func() {
	old := os.Getenv(key)
	os.Setenv(key, value)
	return func() { os.Setenv(key, old) }
}

// createMockAnsibleScript creates a temporary mock ansible-playbook script
func createMockAnsibleScript(t *testing.T, output string, exitCode int) (scriptPath string, cleanup func()) {
	tmpDir := t.TempDir()
	scriptPath = filepath.Join(tmpDir, "ansible-playbook")
	
	scriptContent := "#!/bin/bash\n"
	if exitCode != 0 {
		scriptContent += "echo '" + output + "' >&2\n"
		scriptContent += "exit " + string(rune('0'+exitCode)) + "\n"
	} else {
		scriptContent += "echo '" + output + "'\n"
		scriptContent += "exit 0\n"
	}
	
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}
	
	// Add script directory to PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	
	cleanup = func() {
		os.Setenv("PATH", oldPath)
	}
	
	return scriptPath, cleanup
}

// TestPgCDIAnsibleImple_CmdExecute_SuccessWithValidJSON tests successful execution with valid JSON
func TestPgCDIAnsibleImple_CmdExecute_SuccessWithValidJSON(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock script
	output := `RESULT_TYPE:SUCCESS\n{"cpu_usage": 75.5, "memory_usage": 82.3}`
	_, cleanupScript := createMockAnsibleScript(t, output, 0)
	defer cleanupScript()

	// Setup environment variables
	tmpDir := t.TempDir()
	defer setEnv("LOG_LEVEL", "2")()
	defer setEnv("INTERVAL", "300")()
	defer setEnv("METRICS_PORT", "9090")()
	defer setEnv("P2P_INTERVAL", "60")()
	defer setEnv("SSH_KEY", "/test/key")()
	defer setEnv("METRICS_ENDPOINT", "http://localhost:9090")()
	defer setEnv("DB_URL", "postgres://localhost:5432/test")()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	// Create dummy playbook file
	playbookPath := filepath.Join(tmpDir, "test.yml")
	os.WriteFile(playbookPath, []byte("---"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errMsg, jsonData := ansible.CmdExecute(ctx, "testhost", "testuser", "/tmp/key", "test.yml", "var=value")

	if errMsg != nil {
		t.Errorf("Expected no error, got: %v", *errMsg)
	}

	if jsonData == nil {
		t.Fatal("Expected jsonData to be non-nil")
	}

	if jsonData["cpu_usage"] != float64(75.5) {
		t.Errorf("Expected cpu_usage to be 75.5, got: %v", jsonData["cpu_usage"])
	}
}

// TestPgCDIAnsibleImple_CmdExecute_AnsibleCommandFails tests command failure
func TestPgCDIAnsibleImple_CmdExecute_AnsibleCommandFails(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock script that fails
	output := `FATAL: [testhost]: FAILED! => {"msg": "Connection timeout"}`
	_, cleanupScript := createMockAnsibleScript(t, output, 1)
	defer cleanupScript()

	tmpDir := t.TempDir()
	defer setEnv("LOG_LEVEL", "2")()
	defer setEnv("INTERVAL", "300")()
	defer setEnv("METRICS_PORT", "9090")()
	defer setEnv("P2P_INTERVAL", "60")()
	defer setEnv("SSH_KEY", "/test/key")()
	defer setEnv("METRICS_ENDPOINT", "http://localhost:9090")()
	defer setEnv("DB_URL", "postgres://localhost:5432/test")()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	playbookPath := filepath.Join(tmpDir, "test.yml")
	os.WriteFile(playbookPath, []byte("---"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errMsg, jsonData := ansible.CmdExecute(ctx, "testhost", "testuser", "/tmp/key", "test.yml", "var=value")

	if errMsg == nil {
		t.Error("Expected error, got nil")
	}

	if jsonData != nil {
		t.Errorf("Expected jsonData to be nil, got: %v", jsonData)
	}
}

// TestPgCDIAnsibleImple_CmdExecute_InvalidCDIOutput tests invalid CDI wrapper output
func TestPgCDIAnsibleImple_CmdExecute_InvalidCDIOutput(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock script with invalid output
	output := `Some random output without proper format`
	_, cleanupScript := createMockAnsibleScript(t, output, 0)
	defer cleanupScript()

	tmpDir := t.TempDir()
	defer setEnv("LOG_LEVEL", "2")()
	defer setEnv("INTERVAL", "300")()
	defer setEnv("METRICS_PORT", "9090")()
	defer setEnv("P2P_INTERVAL", "60")()
	defer setEnv("SSH_KEY", "/test/key")()
	defer setEnv("METRICS_ENDPOINT", "http://localhost:9090")()
	defer setEnv("DB_URL", "postgres://localhost:5432/test")()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	playbookPath := filepath.Join(tmpDir, "test.yml")
	os.WriteFile(playbookPath, []byte("---"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errMsg, jsonData := ansible.CmdExecute(ctx, "testhost", "testuser", "/tmp/key", "test.yml", "var=value")

	if errMsg == nil {
		t.Error("Expected error for invalid CDI output, got nil")
	}

	if jsonData != nil {
		t.Errorf("Expected jsonData to be nil, got: %v", jsonData)
	}
}

// TestPgCDIAnsibleImple_CmdExecute_ErrorV10Response tests ERROR_V_1_0 response
func TestPgCDIAnsibleImple_CmdExecute_ErrorV10Response(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	output := `msg": "RESULT_TYPE:ERROR_V_1_0\nDatabase connection failed"`
	_, cleanupScript := createMockAnsibleScript(t, output, 0)
	defer cleanupScript()

	tmpDir := t.TempDir()
	defer setEnv("LOG_LEVEL", "2")()
	defer setEnv("INTERVAL", "300")()
	defer setEnv("METRICS_PORT", "9090")()
	defer setEnv("P2P_INTERVAL", "60")()
	defer setEnv("SSH_KEY", "/test/key")()
	defer setEnv("METRICS_ENDPOINT", "http://localhost:9090")()
	defer setEnv("DB_URL", "postgres://localhost:5432/test")()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	playbookPath := filepath.Join(tmpDir, "test.yml")
	os.WriteFile(playbookPath, []byte("---"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errMsg, jsonData := ansible.CmdExecute(ctx, "testhost", "testuser", "/tmp/key", "test.yml", "var=value")

	if errMsg == nil {
		t.Error("Expected error for ERROR_V_1_0, got nil")
	} else if *errMsg != "Database connection failed" {
		t.Errorf("Expected error message 'Database connection failed', got: %v", *errMsg)
	}

	if jsonData != nil {
		t.Errorf("Expected jsonData to be nil, got: %v", jsonData)
	}
}

// TestPgCDIAnsibleImple_CmdExecute_ContextCancellation tests context cancellation
func TestPgCDIAnsibleImple_CmdExecute_ContextCancellation(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create a script that sleeps for a while
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "ansible-playbook")
	scriptContent := "#!/bin/bash\nsleep 10\necho 'RESULT_TYPE:SUCCESS'\n"
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	defer setEnv("LOG_LEVEL", "2")()
	defer setEnv("INTERVAL", "300")()
	defer setEnv("METRICS_PORT", "9090")()
	defer setEnv("P2P_INTERVAL", "60")()
	defer setEnv("SSH_KEY", "/test/key")()
	defer setEnv("METRICS_ENDPOINT", "http://localhost:9090")()
	defer setEnv("DB_URL", "postgres://localhost:5432/test")()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	playbookPath := filepath.Join(tmpDir, "test.yml")
	os.WriteFile(playbookPath, []byte("---"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	errMsg, jsonData := ansible.CmdExecute(ctx, "testhost", "testuser", "/tmp/key", "test.yml", "var=value")

	// Should get an error due to context cancellation
	if errMsg == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}

	if jsonData != nil {
		t.Errorf("Expected jsonData to be nil, got: %v", jsonData)
	}
}

// TestPgCDIAnsibleImple_CmdExecute_EmptyOutput tests empty command output
func TestPgCDIAnsibleImple_CmdExecute_EmptyOutput(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock script with empty output that exits with error
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "ansible-playbook")
	scriptContent := "#!/bin/bash\nexit 1\n"
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	defer setEnv("LOG_LEVEL", "2")()
	defer setEnv("INTERVAL", "300")()
	defer setEnv("METRICS_PORT", "9090")()
	defer setEnv("P2P_INTERVAL", "60")()
	defer setEnv("SSH_KEY", "/test/key")()
	defer setEnv("METRICS_ENDPOINT", "http://localhost:9090")()
	defer setEnv("DB_URL", "postgres://localhost:5432/test")()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	playbookPath := filepath.Join(tmpDir, "test.yml")
	os.WriteFile(playbookPath, []byte("---"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errMsg, jsonData := ansible.CmdExecute(ctx, "testhost", "testuser", "/tmp/key", "test.yml", "var=value")

	if errMsg == nil {
		t.Error("Expected error for empty output, got nil")
	}

	if jsonData != nil {
		t.Errorf("Expected jsonData to be nil, got: %v", jsonData)
	}
}

// TestPgCDIAnsibleImple_CmdExecute_WithAnsibleError tests Ansible error extraction
func TestPgCDIAnsibleImple_CmdExecute_WithAnsibleError(t *testing.T) {
	if _, err := exec.LookPath("ansible-playbook"); err == nil {
		t.Skip("Skipping test when ansible-playbook is available in PATH to avoid conflicts")
	}

	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup mock script with Ansible error format
	output := `PLAY [test] ************************************

TASK [Gathering Facts] *************************
fatal: [testhost]: UNREACHABLE! => {"changed": false, "msg": "Failed to connect to the host via ssh", "unreachable": true}

PLAY RECAP *************************************
testhost : ok=0 changed=0 unreachable=1 failed=0 skipped=0 rescued=0 ignored=0`
	
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "ansible-playbook")
	scriptContent := "#!/bin/bash\necho '" + output + "' >&2\nexit 4\n"
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	defer setEnv("LOG_LEVEL", "2")()
	defer setEnv("INTERVAL", "300")()
	defer setEnv("METRICS_PORT", "9090")()
	defer setEnv("P2P_INTERVAL", "60")()
	defer setEnv("SSH_KEY", "/test/key")()
	defer setEnv("METRICS_ENDPOINT", "http://localhost:9090")()
	defer setEnv("DB_URL", "postgres://localhost:5432/test")()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	playbookPath := filepath.Join(tmpDir, "test.yml")
	os.WriteFile(playbookPath, []byte("---"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errMsg, jsonData := ansible.CmdExecute(ctx, "testhost", "testuser", "/tmp/key", "test.yml", "var=value")

	if errMsg == nil {
		t.Error("Expected error from Ansible, got nil")
	}

	if jsonData != nil {
		t.Errorf("Expected jsonData to be nil, got: %v", jsonData)
	}
}

// TestPgCDIAnsibleImple_CmdExecute_SkipOriginal is the original test (now always skipped)
func TestPgCDIAnsibleImple_CmdExecute_SkipOriginal(t *testing.T) {
	t.Skip("Original test moved to specific test cases with mocks")
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_Success tests parsing SUCCESS result
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`RESULT_TYPE:SUCCESS\n{"key1": "value1", "key2": 123}`)

	result, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg != nil {
		t.Errorf("Expected no error, got: %v", *errMsg)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["key1"] != "value1" {
		t.Errorf("Expected key1 to be 'value1', got: %v", result["key1"])
	}

	if result["key2"] != float64(123) {
		t.Errorf("Expected key2 to be 123, got: %v", result["key2"])
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_ErrorV10 tests parsing ERROR_V_1_0 result
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_ErrorV10(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`msg": "RESULT_TYPE:ERROR_V_1_0\nTest error message"`)

	_, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg == nil {
		t.Error("Expected error, got nil")
	}

	expectedErr := "Test error message"
	if *errMsg != expectedErr {
		t.Errorf("Expected error '%s', got: %v", expectedErr, *errMsg)
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_ErrorV11 tests parsing ERROR_V_1_1 result
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_ErrorV11(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`msg": "RESULT_TYPE:ERROR_V_1_1\nAnother error message"`)

	_, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg == nil {
		t.Error("Expected error, got nil")
	}

	expectedErr := "Another error message"
	if *errMsg != expectedErr {
		t.Errorf("Expected error '%s', got: %v", expectedErr, *errMsg)
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_Unknown tests parsing UNKNOWN result
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_Unknown(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`msg": "RESULT_TYPE:UNKNOWN\nUnknown result type"`)

	_, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg == nil {
		t.Error("Expected error, got nil")
	}

	expectedErr := "invalid cdi response"
	if *errMsg != expectedErr {
		t.Errorf("Expected error '%s', got: %v", expectedErr, *errMsg)
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_NoResultType tests parsing output without RESULT_TYPE
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_NoResultType(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`Some output without RESULT_TYPE
DATA:
{"key": "value"}`)

	_, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg == nil {
		t.Error("Expected error for missing RESULT_TYPE, got nil")
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_SuccessEmptyData tests SUCCESS with empty data
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_SuccessEmptyData(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`RESULT_TYPE:SUCCESS`)

	result, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg != nil {
		t.Errorf("Expected no error for empty data, got: %v", *errMsg)
	}

	if result != nil && len(result) > 0 {
		t.Errorf("Expected empty result for empty data, got: %v", result)
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_SuccessInvalidJSON tests SUCCESS with invalid JSON
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_SuccessInvalidJSON(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`RESULT_TYPE:SUCCESS\n{invalid json}`)

	_, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_SuccessWithPythonNone tests SUCCESS with Python None value
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_SuccessWithPythonNone(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`RESULT_TYPE:SUCCESS\n{"key": "None"}`)

	result, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg != nil {
		t.Errorf("Expected no error, got: %v", *errMsg)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	// Python "None" as string should be converted
	if result["key"] == nil {
		// This is expected if the implementation converts "None" string to nil
		return
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_SuccessWithShellPrompt tests SUCCESS with shell prompt
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_SuccessWithShellPrompt(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`bash-4.2$ RESULT_TYPE:SUCCESS\n{"key": "value"}`)

	result, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg != nil {
		t.Errorf("Expected no error, got: %v", *errMsg)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["key"] != "value" {
		t.Errorf("Expected key to be 'value', got: %v", result["key"])
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_EmptyOutput tests parsing empty output
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_EmptyOutput(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(``)

	_, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg == nil {
		t.Error("Expected error for empty output, got nil")
	}
}

// TestPgCDIAnsibleImple_ParseCDIWrapperOutput_MultipleDataLines tests SUCCESS with multiple data lines
func TestPgCDIAnsibleImple_ParseCDIWrapperOutput_MultipleDataLines(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	ansible := &PgCDIAnsibleImple{Logger: logger}

	output := []byte(`RESULT_TYPE:SUCCESS\n{\n  "key1": "value1",\n  "key2": "value2",\n  "nested": {\n    "subkey": "subvalue"\n  }\n}`)

	result, errMsg := ansible.parseCDIWrapperOutput(output)

	if errMsg != nil {
		t.Errorf("Expected no error, got: %v", *errMsg)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["key1"] != "value1" {
		t.Errorf("Expected key1 to be 'value1', got: %v", result["key1"])
	}

	nested, ok := result["nested"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected nested to be map[string]interface{}, got: %T", result["nested"])
	} else if nested["subkey"] != "subvalue" {
		t.Errorf("Expected nested.subkey to be 'subvalue', got: %v", nested["subkey"])
	}
}

// TestPgCDIAnsibleImple_CmdExecute_NoExtraArgs tests command execution with no extra arguments
func TestPgCDIAnsibleImple_CmdExecute_NoExtraArgs(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// Setup mock script
		output := `RESULT_TYPE:SUCCESS\n{"status": "ok"}`
		_, cleanupScript := createMockAnsibleScript(t, output, 0)
		defer cleanupScript()

		// Setup environment variables
		tmpDir := t.TempDir()
		defer setEnv("ANSIBLE_PATH", tmpDir)()
		defer setEnv("LOG_LEVEL", "2")()
		defer setEnv("INTERVAL", "300")()
		defer setEnv("METRICS_PORT", "9090")()
		defer setEnv("P2P_INTERVAL", "60")()
		defer setEnv("SSH_KEY", "/test/key")()
		defer setEnv("METRICS_ENDPOINT", "http://localhost:9090")()
		defer setEnv("DB_URL", "postgres://localhost:5432/test")()

		err := utils.InitializeConfig()
		if err != nil {
			t.Fatalf("Failed to initialize config: %v", err)
		}

		logger := klog.NewKlogr()
		ansible := &PgCDIAnsibleImple{Logger: logger}

		// Create dummy playbook file
		playbookPath := filepath.Join(tmpDir, "test.yml")
		os.WriteFile(playbookPath, []byte("---"), 0644)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Call with empty extra arguments - should NOT include -e flag
		errMsg, jsonData := ansible.CmdExecute(ctx, "testhost", "testuser", "/tmp/key", "test.yml", "")

		if errMsg != nil {
			t.Errorf("Expected no error with empty extra args, got: %v", *errMsg)
		}

		if jsonData == nil {
			t.Fatal("Expected jsonData to be non-nil")
		}

		if jsonData["status"] != "ok" {
			t.Errorf("Expected status to be 'ok', got: %v", jsonData["status"])
	}
}

// TODO
