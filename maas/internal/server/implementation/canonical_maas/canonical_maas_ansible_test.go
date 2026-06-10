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

package canonical_maas

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"k8s.io/klog/v2"

	"maas_module/internal/server/test_utils"
	"maas_module/internal/server/utils"
)

// Mock executor for testing
type mockExecutor struct {
	output []byte
	err    error
	calls  []mockExecutorCall
	mutex  sync.Mutex
}

type mockExecutorCall struct {
	name string
	args []string
}

func (m *mockExecutor) ExecuteCommand(ctx context.Context, name string, arg ...string) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.calls = append(m.calls, mockExecutorCall{
		name: name,
		args: arg,
	})

	return m.output, m.err
}

func (m *mockExecutor) GetCalls() []mockExecutorCall {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	calls := make([]mockExecutorCall, len(m.calls))
	copy(calls, m.calls)
	return calls
}

func (m *mockExecutor) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.calls = nil
}

// Helper function to set test environment and initialize config
func setupTestEnvironment(t *testing.T) {
	// Set test environment variables
	testEnv := map[string]string{
		"LOG_LEVEL":        "2",
		"MAAS_SERVER_PORT": "8080",
		"MAAS_ACCESS_URL":  "http://test-maas.com",
		"MAAS_API_KEY":     "test-key",
		"VM_HOST_DISK":     "50",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "/test/ssh/key",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}

	for key, value := range testEnv {
		os.Setenv(key, value)
	}

	// Initialize config
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		for key := range testEnv {
			os.Unsetenv(key)
		}
	})
}

func expectedAnsiblePlaybookPath(playbook string) string {
	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansiblePath := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", "canonical_maas", "ansible")
	return ansiblePath + "/" + playbook
}

// Note: Since utils doesn't have ResetConfigForTesting, we'll work around this limitation

// TestCmdExecutor_ExecuteCommand_ReturnsOutput tests CmdExecutor.ExecuteCommand
func TestCmdExecutor_ExecuteCommand_ReturnsOutput(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	executor := &CmdExecutor{}
	ctx := context.Background()

	// Act
	output, err := executor.ExecuteCommand(ctx, "echo", "test message")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// The actual output will depend on the system
	// We just verify that the method doesn't panic and returns something
	if output == nil {
		t.Error("Expected some output, got nil")
	}
}

// TestCmdExecutor_ExecuteCommand_InvalidCommand_ReturnsError tests CmdExecutor with invalid command
func TestCmdExecutor_ExecuteCommand_InvalidCommand_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	executor := &CmdExecutor{}
	ctx := context.Background()

	// Act
	output, err := executor.ExecuteCommand(ctx, "nonexistent-command-12345", "arg1")

	// Assert
	if err == nil {
		t.Error("Expected error for nonexistent command, got nil")
	}

	// Output might be nil or empty
	_ = output // Just to use the variable
}

// TestCanonicalMaasAnsibleImple_CmdExecute_ValidInput_ExecutesSuccessfully tests CmdExecute with valid input
func TestCanonicalMaasAnsibleImple_CmdExecute_ValidInput_ExecutesSuccessfully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("ansible execution success"),
		err:    nil,
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()
	remoteHost := "192.168.1.100"
	playbook := "test-playbook.yml"
	extraArgs := "var1=value1"

	// Act
	_, err := ansible.CmdExecute(ctx, remoteHost, playbook, extraArgs)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify command was called correctly
	calls := mockExec.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 command call, got %d", len(calls))
	} else {
		call := calls[0]
		if call.name != "ansible-playbook" {
			t.Errorf("Expected command 'ansible-playbook', got: %s", call.name)
		}

		// Verify arguments
		expectedArgs := []string{
			expectedAnsiblePlaybookPath("test-playbook.yml"),
			"-i", "192.168.1.100,",
			"-u", "cloud-user",
			"--private-key", "/test/ssh/key",
			"-e", "var1=value1",
		}

		if len(call.args) != len(expectedArgs) {
			t.Errorf("Expected %d args, got %d", len(expectedArgs), len(call.args))
		} else {
			for i, expected := range expectedArgs {
				if call.args[i] != expected {
					t.Errorf("Arg %d: expected %s, got %s", i, expected, call.args[i])
				}
			}
		}
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_NoExtraArgs_ExecutesWithoutExtraArgs tests CmdExecute without extra arguments
func TestCanonicalMaasAnsibleImple_CmdExecute_NoExtraArgs_ExecutesWithoutExtraArgs(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("ansible success"),
		err:    nil,
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()
	remoteHost := "test-host"
	playbook := "simple.yml"
	extraArgs := "" // Empty extra args

	// Act
	_, err := ansible.CmdExecute(ctx, remoteHost, playbook, extraArgs)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify command was called without extra args
	calls := mockExec.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 command call, got %d", len(calls))
	} else {
		call := calls[0]

		// Expected arguments without -e flag
		expectedArgs := []string{
			expectedAnsiblePlaybookPath("simple.yml"),
			"-i", "test-host,",
			"-u", "cloud-user",
			"--private-key", "/test/ssh/key",
		}

		if len(call.args) != len(expectedArgs) {
			t.Errorf("Expected %d args, got %d", len(expectedArgs), len(call.args))
		} else {
			for i, expected := range expectedArgs {
				if call.args[i] != expected {
					t.Errorf("Arg %d: expected %s, got %s", i, expected, call.args[i])
				}
			}
		}
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_ExecutorError_ReturnsEnvError tests CmdExecute when executor returns error
func TestCanonicalMaasAnsibleImple_CmdExecute_ExecutorError_ReturnsEnvError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("error output"),
		err:    errors.New("execution failed"),
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()

	// Act
	output, err := ansible.CmdExecute(ctx, "host", "playbook.yml", "")

	// Assert
	if err == nil {
		t.Error("Expected error from executor failure")
	}

	// Verify it's wrapped as EnvError
	envErr, ok := err.(*utils.EnvError)
	if !ok {
		t.Errorf("Expected EnvError, got: %T", err)
	} else {
		if !strings.Contains(envErr.Message, "execution failed") {
			t.Errorf("Expected error message to contain 'execution failed', got: %s", envErr.Message)
		}
	}

	// Output should still be returned even on error
	if string(output) != "error output" {
		t.Errorf("Expected 'error output', got: %s", string(output))
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_ExecutorError_WithAnsibleError_ExtractsMessage tests CmdExecute with Ansible-formatted error
func TestCanonicalMaasAnsibleImple_CmdExecute_ExecutorError_WithAnsibleError_ExtractsMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	// Simulate Ansible JSON error output with proper format
	ansibleErrorOutput := []byte(`fatal: [test-host]: FAILED! => {"changed": false, "msg": "Unable to connect to host: Connection timed out", "unreachable": true}`)

	mockExec := &mockExecutor{
		output: ansibleErrorOutput,
		err:    errors.New("ansible-playbook failed"),
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()

	// Act
	output, err := ansible.CmdExecute(ctx, "host", "playbook.yml", "")

	// Assert
	if err == nil {
		t.Error("Expected error from executor failure")
	}

	// Verify it's wrapped as EnvError
	envErr, ok := err.(*utils.EnvError)
	if !ok {
		t.Errorf("Expected EnvError, got: %T", err)
	} else {
		// Should contain extracted Ansible error message
		if !strings.Contains(envErr.Message, "Unable to connect to host") {
			t.Errorf("Expected error message to contain extracted Ansible error, got: %s", envErr.Message)
		}
	}

	// Output should still be returned even on error
	if string(output) != string(ansibleErrorOutput) {
		t.Errorf("Expected ansible error output, got: %s", string(output))
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_ExecutorError_EmptyOutput_UsesOriginalError tests CmdExecute with empty output
func TestCanonicalMaasAnsibleImple_CmdExecute_ExecutorError_EmptyOutput_UsesOriginalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte(""),
		err:    errors.New("command execution failed"),
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()

	// Act
	output, err := ansible.CmdExecute(ctx, "host", "playbook.yml", "")

	// Assert
	if err == nil {
		t.Error("Expected error from executor failure")
	}

	// Verify it's wrapped as EnvError
	envErr, ok := err.(*utils.EnvError)
	if !ok {
		t.Errorf("Expected EnvError, got: %T", err)
	} else {
		// Should use original error message when output is empty
		if !strings.Contains(envErr.Message, "command execution failed") {
			t.Errorf("Expected error message to contain original error, got: %s", envErr.Message)
		}
	}

	// Output should be empty
	if len(output) != 0 {
		t.Errorf("Expected empty output, got: %s", string(output))
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_ExecutorError_NoExtractableError_UsesOriginalError tests when Ansible error cannot be extracted
func TestCanonicalMaasAnsibleImple_CmdExecute_ExecutorError_NoExtractableError_UsesOriginalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	// Output that doesn't contain extractable Ansible error
	nonAnsibleOutput := []byte("Some generic error message without Ansible format")

	mockExec := &mockExecutor{
		output: nonAnsibleOutput,
		err:    errors.New("generic execution error"),
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()

	// Act
	output, err := ansible.CmdExecute(ctx, "host", "playbook.yml", "")

	// Assert
	if err == nil {
		t.Error("Expected error from executor failure")
	}

	// Verify it's wrapped as EnvError
	envErr, ok := err.(*utils.EnvError)
	if !ok {
		t.Errorf("Expected EnvError, got: %T", err)
	} else {
		// Should use original error message when Ansible error cannot be extracted
		if !strings.Contains(envErr.Message, "generic execution error") {
			t.Errorf("Expected error message to contain original error, got: %s", envErr.Message)
		}
	}

	// Output should still be returned
	if string(output) != string(nonAnsibleOutput) {
		t.Errorf("Expected non-ansible output, got: %s", string(output))
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_EmptyPlaybook_ExecutesWithEmptyPlaybook tests CmdExecute with empty playbook
func TestCanonicalMaasAnsibleImple_CmdExecute_EmptyPlaybook_ExecutesWithEmptyPlaybook(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("executed"),
		err:    nil,
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()

	// Act
	_, err := ansible.CmdExecute(ctx, "host", "", "")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify command includes empty playbook path
	calls := mockExec.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 command call, got %d", len(calls))
	} else {
		call := calls[0]

		// First arg should be the playbook path (which will end with "/ansible/")
		expectedPlaybook := expectedAnsiblePlaybookPath("")
		if call.args[0] != expectedPlaybook {
			t.Errorf("Expected playbook path '%s', got: %s", expectedPlaybook, call.args[0])
		}
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_EmptyRemoteHost_ExecutesWithEmptyHost tests CmdExecute with empty remote host
func TestCanonicalMaasAnsibleImple_CmdExecute_EmptyRemoteHost_ExecutesWithEmptyHost(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("executed"),
		err:    nil,
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()

	// Act
	_, err := ansible.CmdExecute(ctx, "", "test.yml", "")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify command includes empty host inventory
	calls := mockExec.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 command call, got %d", len(calls))
	} else {
		call := calls[0]

		// Should find the inventory argument (,)
		foundInventory := false
		for _, arg := range call.args {
			if arg == "," {
				foundInventory = true
				break
			}
		}
		if !foundInventory {
			t.Error("Expected to find inventory argument ','")
		}
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_SpecialCharactersInArgs_HandlesCorrectly tests CmdExecute with special characters
func TestCanonicalMaasAnsibleImple_CmdExecute_SpecialCharactersInArgs_HandlesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("executed"),
		err:    nil,
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()
	remoteHost := "host.example.com"
	playbook := "playbook with spaces.yml"
	extraArgs := "var='value with spaces'"

	// Act
	_, err := ansible.CmdExecute(ctx, remoteHost, playbook, extraArgs)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify arguments are passed correctly
	calls := mockExec.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 command call, got %d", len(calls))
	} else {
		call := calls[0]

		// Check playbook path includes spaces
		expectedPlaybook := expectedAnsiblePlaybookPath("playbook with spaces.yml")
		if call.args[0] != expectedPlaybook {
			t.Errorf("Expected playbook %s, got %s", expectedPlaybook, call.args[0])
		}

		// Check extra args with spaces
		foundExtraArgs := false
		for i, arg := range call.args {
			if arg == "-e" && i+1 < len(call.args) {
				if call.args[i+1] == "var='value with spaces'" {
					foundExtraArgs = true
					break
				}
			}
		}
		if !foundExtraArgs {
			t.Error("Expected to find extra args with spaces")
		}
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_ContextCancellation_HandlesCorrectly tests CmdExecute with context cancellation
func TestCanonicalMaasAnsibleImple_CmdExecute_ContextCancellation_HandlesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("executed before cancellation"),
		err:    nil,
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act
	_, err := ansible.CmdExecute(ctx, "host", "playbook.yml", "")

	// Assert
	// Note: The current implementation doesn't actually use context for cancellation in the executor
	// The mock executor will still execute successfully
	if err != nil {
		t.Errorf("Expected no error (context not implemented in executor), got: %v", err)
	}

	// Verify command was still called
	calls := mockExec.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 command call, got %d", len(calls))
	}
}

// TestCanonicalMaasAnsibleImple_CmdExecute_MultipleExtraArgs_HandlesCorrectly tests CmdExecute with multiple extra arguments
func TestCanonicalMaasAnsibleImple_CmdExecute_MultipleExtraArgs_HandlesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("executed with multiple args"),
		err:    nil,
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()
	extraArgs := "var1=value1 var2=value2 var3='value with spaces'"

	// Act
	_, err := ansible.CmdExecute(ctx, "host", "playbook.yml", extraArgs)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the entire extra args string is passed as one argument to -e
	calls := mockExec.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 command call, got %d", len(calls))
	} else {
		call := calls[0]

		// Find -e argument and its value
		foundExtraArgs := false
		for i, arg := range call.args {
			if arg == "-e" && i+1 < len(call.args) {
				if call.args[i+1] == extraArgs {
					foundExtraArgs = true
					break
				}
			}
		}
		if !foundExtraArgs {
			t.Errorf("Expected to find extra args '%s'", extraArgs)
		}
	}
}

// TestCanonicalMaasAnsibleImple_LoginUser_UsesCorrectUser tests that the correct login user is used
func TestCanonicalMaasAnsibleImple_LoginUser_UsesCorrectUser(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironment(t)

	mockExec := &mockExecutor{
		output: []byte("executed"),
		err:    nil,
	}

	ansible := &CanonicalMaasAnsibleImple{
		Logger:   klog.NewKlogr(),
		Executor: mockExec,
	}

	ctx := context.Background()

	// Act
	_, err := ansible.CmdExecute(ctx, "host", "playbook.yml", "")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the correct user is used
	calls := mockExec.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 command call, got %d", len(calls))
	} else {
		call := calls[0]

		// Find -u argument and verify it's followed by cloud-user
		foundUser := false
		for i, arg := range call.args {
			if arg == "-u" && i+1 < len(call.args) {
				if call.args[i+1] == "cloud-user" {
					foundUser = true
					break
				}
			}
		}
		if !foundUser {
			t.Error("Expected to find user 'cloud-user'")
		}
	}
}
