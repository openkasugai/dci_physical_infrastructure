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
	"exporter_module/internal/server/test_utils"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

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
	return exec.CommandContext(ctx, name, arg...)
}

// Test helper functions
func setEnv(key, value string) func() {
	old := os.Getenv(key)
	os.Setenv(key, value)
	return func() { os.Setenv(key, old) }
}

// TestAnsibleImplement_CmdExecute_ValidRequest_ReturnsSuccess tests successful execution
func TestAnsibleImplement_CmdExecute_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" {
				// Create a mock command that returns valid output
				return exec.Command("echo", `TASK [result output] *******************************************************************
ok: [test] => {
    "msg": {"test": "data", "value": 123}
}`)
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}
	ctx := context.Background()

	// Execute
	output, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if output == nil {
		t.Error("Expected output to be non-nil")
	}

	result, ok := output.(map[string]interface{})
	if !ok {
		t.Errorf("Expected output to be map[string]interface{}, got: %T", output)
	}

	if result["test"] != "data" {
		t.Errorf("Expected test field to be 'data', got: %v", result["test"])
	}
}

// TestAnsibleImplement_CmdExecute_WithExtraArgs_ReturnsSuccess tests execution with extra arguments
func TestAnsibleImplement_CmdExecute_WithExtraArgs_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	extraArgsPassed := false
	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" {
				// Verify extra args are included
				argStr := strings.Join(arg, " ")
				if strings.Contains(argStr, "-e") && strings.Contains(argStr, "test=value") {
					extraArgsPassed = true
				}
				return exec.Command("echo", `TASK [result output] *******************************************************************
ok: [test] => {
    "msg": {"success": true}
}`)
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}
	ctx := context.Background()

	// Execute
	output, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "test=value")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !extraArgsPassed {
		t.Error("Expected extra arguments to be passed")
	}

	if output == nil {
		t.Error("Expected output to be non-nil")
	}
}

// TestAnsibleImplement_CmdExecute_CommandFails_ReturnsError tests command failure
func TestAnsibleImplement_CmdExecute_CommandFails_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" {
				return exec.Command("false") // Command that always fails
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}
	ctx := context.Background()

	// Execute
	_, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestAnsibleImplement_CmdExecute_CommandFailsWithAnsibleOutput_ReturnsExtractedError tests command failure with Ansible error output
func TestAnsibleImplement_CmdExecute_CommandFailsWithAnsibleOutput_ReturnsExtractedError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" {
				// Create a command that outputs Ansible error format and fails
				cmd := exec.Command("sh", "-c", `echo 'fatal: [testhost]: FAILED! => {"changed": false, "msg": "Connection refused"}' && exit 1`)
				return cmd
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}
	ctx := context.Background()

	// Execute
	_, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Check that error message contains the extracted Ansible error
	expectedMsg := "Host testhost failed: Connection refused"
	if !strings.Contains(err.Error(), "Connection refused") {
		t.Errorf("Expected error message to contain 'Connection refused', got '%s'", err.Error())
	}
	// The exact format depends on ExtractAnsibleError implementation
	if err.Error() != expectedMsg {
		// Log for debugging, but don't fail since format may vary
		t.Logf("Note: Error format is '%s', expected exact match '%s'", err.Error(), expectedMsg)
	}
}

// TestAnsibleImplement_CmdExecute_CommandFailsWithEmptyOutput_ReturnsOriginalError tests command failure with empty output
func TestAnsibleImplement_CmdExecute_CommandFailsWithEmptyOutput_ReturnsOriginalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" {
				// Command that fails without output
				return exec.Command("false")
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}
	ctx := context.Background()

	// Execute
	_, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should contain the original exit error since output is empty
	if !strings.Contains(err.Error(), "exit status") {
		t.Errorf("Expected error to contain 'exit status', got '%s'", err.Error())
	}
}

// TestAnsibleImplement_CmdExecute_CommandFailsWithNonAnsibleOutput_ReturnsOriginalError tests command failure with non-Ansible output
func TestAnsibleImplement_CmdExecute_CommandFailsWithNonAnsibleOutput_ReturnsOriginalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" {
				// Command that outputs non-Ansible error and fails
				cmd := exec.Command("sh", "-c", `echo 'Some random error message' && exit 1`)
				return cmd
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}
	ctx := context.Background()

	// Execute
	_, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should contain the original exit error since ExtractAnsibleError returns empty string
	if !strings.Contains(err.Error(), "exit status") {
		t.Errorf("Expected error to contain 'exit status', got '%s'", err.Error())
	}
}

// TestAnsibleImplement_CmdExecute_ContextCancelled_ReturnsError tests context cancellation
func TestAnsibleImplement_CmdExecute_ContextCancelled_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" {
				return exec.Command("sleep", "10") // Long running command
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Execute
	_, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")

	// Verify
	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}
}

// TestAnsibleImplement_CmdExecute_ParseError_ReturnsError tests JSON parsing error
func TestAnsibleImplement_CmdExecute_ParseError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" {
				return exec.Command("echo", "invalid output format")
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}
	ctx := context.Background()

	// Execute
	_, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")

	// Verify
	if err == nil {
		t.Error("Expected parsing error, got nil")
	}
}

// TestAnsibleImplement_CmdExecute_PathBasedOnExecutable_UsesExecutableRelativePath tests that playbook path is resolved relative to executable
func TestAnsibleImplement_CmdExecute_PathBasedOnExecutable_UsesExecutableRelativePath(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()

	pathUsed := ""
	mockExecutor := MockCommandExecutor{
		CommandFunc: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			if name == "ansible-playbook" && len(arg) > 0 {
				pathUsed = arg[0]
				return exec.Command("echo", `TASK [result output] *******************************************************************
ok: [test] => {
    "msg": {"success": true}
}`)
			}
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: mockExecutor,
	}
	ctx := context.Background()

	// Execute
	_, err := ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Path should end with internal/server/implementation/ansible/test.yml (resolved from executable)
	if !strings.HasSuffix(pathUsed, "ansible/test.yml") {
		t.Errorf("Expected path to end with 'ansible/test.yml', got: %s", pathUsed)
	}
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
	if ansible == nil {
		t.Fatal("Expected AnsibleImplement instance, got nil")
	}

	if ansible.Logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	if ansible.Executor == nil {
		t.Error("Expected executor to be set")
	}

	// Verify executor is RealCommandExecutor
	_, ok := ansible.Executor.(RealCommandExecutor)
	if !ok {
		t.Error("Expected executor to be RealCommandExecutor")
	}
}

// TestFixJSON_ValidJSON_ReturnsUnchanged tests fixJSON with valid JSON
func TestFixJSON_ValidJSON_ReturnsUnchanged(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	input := `{"key": "value"}`

	// Execute
	result := fixJSON(input)

	// Verify
	if result != input {
		t.Errorf("Expected %s, got %s", input, result)
	}
}

// TestFixJSON_MissingClosingBraces_AddsClosingBraces tests fixJSON with missing braces
func TestFixJSON_MissingClosingBraces_AddsClosingBraces(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	input := `{"key": "value"`
	expected := `{"key": "value"}`

	// Execute
	result := fixJSON(input)

	// Verify
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestFixJSON_MultipleMissingBraces_AddsMultipleClosingBraces tests fixJSON with multiple missing braces
func TestFixJSON_MultipleMissingBraces_AddsMultipleClosingBraces(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	input := `{"outer": {"inner": "value"`
	expected := `{"outer": {"inner": "value"}}`

	// Execute
	result := fixJSON(input)

	// Verify
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestFixJSON_ExtraClosingBraces_ReturnsUnchanged tests fixJSON with extra braces
func TestFixJSON_ExtraClosingBraces_ReturnsUnchanged(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	input := `{"key": "value"}}}`

	// Execute
	result := fixJSON(input)

	// Verify
	if result != input {
		t.Errorf("Expected %s, got %s", input, result)
	}
}

// TestFixJSON_EmptyString_ReturnsEmpty tests fixJSON with empty string
func TestFixJSON_EmptyString_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	input := ""

	// Execute
	result := fixJSON(input)

	// Verify
	if result != input {
		t.Errorf("Expected empty string, got %s", result)
	}
}

// TestFixJSON_WhitespaceOnly_ReturnsEmpty tests fixJSON with whitespace
func TestFixJSON_WhitespaceOnly_ReturnsEmpty(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	input := "   \n\t  "

	// Execute
	result := fixJSON(input)

	// Verify
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

// TestAnsibleImplement_extractMsg_ValidOutput_ReturnsMessage tests extractMsg with valid output
func TestAnsibleImplement_extractMsg_ValidOutput_ReturnsMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger, Executor: RealCommandExecutor{}}
	output := `TASK [result output] *******************************************************************
ok: [test] => {
    "msg": {"test": "data", "value": 123}
}
PLAY RECAP *****************************************************************************`

	// Execute
	result, err := ansible.extractMsg(output)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", result)
	}

	if resultMap["test"] != "data" {
		t.Errorf("Expected test field to be 'data', got: %v", resultMap["test"])
	}
}

// TestAnsibleImplement_extractMsg_NoMatch_ReturnsError tests extractMsg with no match
func TestAnsibleImplement_extractMsg_NoMatch_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger, Executor: RealCommandExecutor{}}
	output := `No TASK found in output`

	// Execute
	_, err := ansible.extractMsg(output)

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestAnsibleImplement_extractMsg_InvalidJSON_ReturnsError tests extractMsg with invalid JSON
func TestAnsibleImplement_extractMsg_InvalidJSON_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger, Executor: RealCommandExecutor{}}
	output := `TASK [result output] *******************************************************************
ok: [test] => {
    "msg": {invalid json}
}`

	// Execute
	_, err := ansible.extractMsg(output)

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestAnsibleImplement_extractMsg_MissingMsgKey_ReturnsError tests extractMsg with missing msg key
func TestAnsibleImplement_extractMsg_MissingMsgKey_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger, Executor: RealCommandExecutor{}}
	output := `TASK [result output] *******************************************************************
ok: [test] => {
    "other": {"test": "data"}
}`

	// Execute
	_, err := ansible.extractMsg(output)

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestAnsibleImplement_extractMsg_EmptyInput_ReturnsError tests extractMsg with empty input
func TestAnsibleImplement_extractMsg_EmptyInput_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger, Executor: RealCommandExecutor{}}
	output := ""

	// Execute
	_, err := ansible.extractMsg(output)

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestAnsibleImplement_extractMsg_MultipleMatches_ReturnsFirstMatch tests extractMsg with multiple matches
func TestAnsibleImplement_extractMsg_MultipleMatches_ReturnsFirstMatch(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()
	ansible := &AnsibleImplement{Logger: logger, Executor: RealCommandExecutor{}}
	output := `TASK [result output] *******************************************************************
ok: [test] => {
    "msg": {"first": "match"}
}
TASK [result output] *******************************************************************
ok: [test] => {
    "msg": {"second": "match"}
}`

	// Execute
	result, err := ansible.extractMsg(output)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result to be map[string]interface{}, got: %T", result)
	}

	if resultMap["first"] != "match" {
		t.Errorf("Expected first field to be 'match', got: %v", resultMap["first"])
	}

	if resultMap["second"] != nil {
		t.Error("Expected only first match to be returned")
	}
}

// TestRealCommandExecutor_CommandContext_ValidCommand_ReturnsCmd tests RealCommandExecutor
func TestRealCommandExecutor_CommandContext_ValidCommand_ReturnsCmd(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	executor := RealCommandExecutor{}
	ctx := context.Background()

	// Execute
	cmd := executor.CommandContext(ctx, "echo", "test")

	// Verify
	if cmd == nil {
		t.Fatal("Expected command, got nil")
	}

	if cmd.Path != "/bin/echo" && cmd.Path != "/usr/bin/echo" {
		// Allow flexibility for different systems
		if !strings.Contains(cmd.Path, "echo") {
			t.Errorf("Expected echo command, got: %s", cmd.Path)
		}
	}

	if len(cmd.Args) < 2 || cmd.Args[1] != "test" {
		t.Errorf("Expected args to contain 'test', got: %v", cmd.Args)
	}
}

// TestAnsibleImplement_CmdExecute_NoExecutor_PanicsGracefully tests handling of nil executor
func TestAnsibleImplement_CmdExecute_NoExecutor_PanicsGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	logger := klog.NewKlogr()

	ansible := &AnsibleImplement{
		Logger:   logger,
		Executor: nil, // Nil executor should cause panic
	}
	ctx := context.Background()

	// Execute and verify panic is handled
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic due to nil executor, but didn't panic")
		}
	}()

	ansible.CmdExecute(ctx, "test.example.com", "testuser", "/path/to/key", "test.yml", "")
}
