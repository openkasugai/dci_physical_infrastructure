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
	"context"
	"log_module/internal/server/test_utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

// MockCommandExecutor for testing
type MockCommandExecutor struct {
	CommandFunc func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

func (m MockCommandExecutor) CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if m.CommandFunc != nil {
		return m.CommandFunc(ctx, name, arg...)
	}
	return exec.Command("echo", "mock output")
}

// Helper function to set environment variable and return cleanup function
func setEnv(key, value string) func() {
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

func expectedAnsiblePlaybookPath(playbook string) string {
	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansiblePath := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", "ansible")
	return ansiblePath + "/" + playbook
}

// TestNewAnsibleImplement_ValidLogger_ReturnsInstance tests constructor
func TestNewAnsibleImplement_ValidLogger_ReturnsInstance(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	// Execute
	ansible := NewAnsibleImplement(logger)

	// Verify
	assert.NotNil(t, ansible)
	assert.Equal(t, logger, ansible.Logger)
	assert.NotNil(t, ansible.Executor)
	assert.IsType(t, RealCommandExecutor{}, ansible.Executor)
}

// TestRealCommandExecutor_CommandContext_ReturnsExecCmd tests real executor
func TestRealCommandExecutor_CommandContext_ReturnsExecCmd(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	executor := RealCommandExecutor{}
	ctx := context.Background()

	// Execute
	cmd := executor.CommandContext(ctx, "echo", "test")

	// Verify
	assert.NotNil(t, cmd)
	// パスの末尾が"echo"で終わることを確認（環境に依存しない）
	assert.True(t, strings.HasSuffix(cmd.Path, "echo"))
	// または、引数が正しく設定されているかをチェック
	assert.Equal(t, []string{"echo", "test"}, cmd.Args)
}

// TestAnsibleImplement_CmdExecute_ValidInput_ReturnsSuccess tests successful execution
func TestAnsibleImplement_CmdExecute_ValidInput_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			// Return a command that outputs valid Ansible result
			validOutput := `TASK [result output] ****************************************************
ok: [test.example.com] => {"changed": false, "msg": {"status": "success", "data": "test_data"}}`
			return exec.Command("echo", validOutput)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}

	// Execute
	output, err := ansible.CmdExecute(context.Background(), "test.example.com", "testuser", "test.yml", "")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, output)
}

// TestAnsibleImplement_CmdExecute_WithExtraArgs_ReturnsSuccess tests execution with extra arguments
func TestAnsibleImplement_CmdExecute_WithExtraArgs_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	defer setEnv("SSH_KEY", "/path/to/ssh/key")()
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			// Verify args include playbook, inventory, user, private-key, and extra args
			// Expected: executable relative ansible path + arguments
			expectedArgs := []string{expectedAnsiblePlaybookPath("test.yml"), "-i", "test.example.com,", "-u", "testuser", "--private-key", "/path/to/ssh/key", "-e", "extra=value"}
			if len(arg) >= len(expectedArgs) {
				for i, expected := range expectedArgs {
					if arg[i] != expected {
						t.Errorf("Expected arg[%d] to be %s, got %s", i, expected, arg[i])
					}
				}
			} else {
				t.Errorf("Expected %d args, got %d", len(expectedArgs), len(arg))
			}

			validOutput := `TASK [result output] ****************************************************
ok: [test.example.com] => {"changed": false, "msg": {"status": "success"}}`
			return exec.Command("echo", validOutput)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}

	// Execute
	output, err := ansible.CmdExecute(context.Background(), "test.example.com", "testuser", "test.yml", "extra=value")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, output)
}

// TestAnsibleImplement_CmdExecute_CommandFails_ReturnsError tests command execution failure
func TestAnsibleImplement_CmdExecute_CommandFails_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			// Return a command that will fail
			return exec.Command("false")
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}

	// Execute
	_, err := ansible.CmdExecute(context.Background(), "test.example.com", "testuser", "test.yml", "")

	// Verify
	assert.Error(t, err)
}

// TestAnsibleImplement_CmdExecute_CommandFailsWithAnsibleError_ReturnsExtractedError tests error extraction from Ansible output
func TestAnsibleImplement_CmdExecute_CommandFailsWithAnsibleError_ReturnsExtractedError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			// Return a command that simulates Ansible error output
			errorOutput := `fatal: [test.example.com]: FAILED! => {"changed": false, "msg": "Connection refused"}`
			cmd := exec.Command("bash", "-c", "echo '"+errorOutput+"' && exit 1")
			return cmd
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}

	// Execute
	_, err := ansible.CmdExecute(context.Background(), "test.example.com", "testuser", "test.yml", "")

	// Verify
	assert.Error(t, err)
	// Error message should contain the extracted Ansible error
	assert.Contains(t, err.Error(), "test.example.com")
}

// TestAnsibleImplement_CmdExecute_CommandFailsNoExtractableError_ReturnsOriginalError tests fallback to original error
func TestAnsibleImplement_CmdExecute_CommandFailsNoExtractableError_ReturnsOriginalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			// Return a command that fails with non-Ansible error output
			cmd := exec.Command("bash", "-c", "echo 'some generic error' && exit 1")
			return cmd
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}

	// Execute
	_, err := ansible.CmdExecute(context.Background(), "test.example.com", "testuser", "test.yml", "")

	// Verify
	assert.Error(t, err)
	// Should contain the original error message
	assert.NotEmpty(t, err.Error())
}

// TestAnsibleImplement_CmdExecute_InvalidOutput_ReturnsError tests parsing failure
func TestAnsibleImplement_CmdExecute_InvalidOutput_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			// Return invalid Ansible output
			return exec.Command("echo", "invalid output")
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}

	// Execute
	_, err := ansible.CmdExecute(context.Background(), "test.example.com", "testuser", "test.yml", "")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no match found for result output")
}

// TestAnsibleImplement_CmdExecute_NoExecutor_PanicsGracefully tests nil executor handling
func TestAnsibleImplement_CmdExecute_NoExecutor_PanicsGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: nil, // Nil executor should cause panic
	}

	// Execute and verify panic is handled
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic due to nil executor, but didn't panic")
		}
	}()

	ansible.CmdExecute(context.Background(), "test.example.com", "testuser", "test.yml", "")
}

// TestAnsibleImplement_extractMsg_ValidJSON_ReturnsMessage tests successful message extraction
func TestAnsibleImplement_extractMsg_ValidJSON_ReturnsMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger}

	input := `TASK [result output] ****************************************************
ok: [test.example.com] => {"changed": false, "msg": {"status": "success", "data": "test_data"}}
`

	// Execute
	output, err := ansible.extractMsg(input)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, output)

	// Verify the extracted message
	expectedMsg := map[string]interface{}{
		"status": "success",
		"data":   "test_data",
	}
	assert.Equal(t, expectedMsg, output)
}

// TestAnsibleImplement_extractMsg_InvalidJSON_ReturnsError tests JSON parsing failure
func TestAnsibleImplement_extractMsg_InvalidJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger}

	input := `TASK [result output] ****************************************************
ok: [test.example.com] => {invalid json}
`

	// Execute
	_, err := ansible.extractMsg(input)

	// Verify
	assert.Error(t, err)
}

// TestAnsibleImplement_extractMsg_NoMsgKey_ReturnsError tests missing msg key
func TestAnsibleImplement_extractMsg_NoMsgKey_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger}

	input := `TASK [result output] ****************************************************
ok: [test.example.com] => {"changed": false, "status": "success"}
`

	// Execute
	_, err := ansible.extractMsg(input)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'msg' key not found")
}

// TestAnsibleImplement_extractMsg_NoMatch_ReturnsError tests no regex match
func TestAnsibleImplement_extractMsg_NoMatch_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger}

	input := "no matching pattern here"

	// Execute
	_, err := ansible.extractMsg(input)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no match found for result output")
}

// TestFixJSON_MissingClosingBrace_AddsClosingBrace tests JSON fixing function
func TestFixJSON_MissingClosingBrace_AddsClosingBrace(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test cases
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No missing braces",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "One missing closing brace",
			input:    `{"key": "value"`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "Multiple missing closing braces",
			input:    `{"key": {"nested": "value"`,
			expected: `{"key": {"nested": "value"}}`,
		},
		{
			name:     "With whitespace",
			input:    `  {"key": "value"  `,
			expected: `{"key": "value"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Execute
			result := fixJSON(tc.input)

			// Verify
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestAnsibleImplement_extractMsg_EmptyInput_ReturnsError tests empty input
func TestAnsibleImplement_extractMsg_EmptyInput_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger}

	// Execute
	_, err := ansible.extractMsg("")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no match found for result output")
}

// TestAnsibleImplement_extractMsg_MultipleMatches_UsesFirst tests multiple regex matches
func TestAnsibleImplement_extractMsg_MultipleMatches_UsesFirst(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger}

	input := `TASK [result output] ****************************************************
ok: [test1.example.com] => {"changed": false, "msg": {"first": "match"}}

TASK [result output] ****************************************************
ok: [test2.example.com] => {"changed": false, "msg": {"second": "match"}}
`

	// Execute
	output, err := ansible.extractMsg(input)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, output)

	// Should use the first match
	expectedMsg := map[string]interface{}{
		"first": "match",
	}
	assert.Equal(t, expectedMsg, output)
}
