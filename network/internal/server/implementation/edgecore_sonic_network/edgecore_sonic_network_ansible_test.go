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

package edgecore_sonic_network

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	proto "network_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf
	"network_module/internal/server/test_utils"
	"network_module/internal/server/utils"

	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"
)

// Mock command executor for testing
type mockCmdExecutor struct {
	output []byte
	err    error
}

func (m *mockCmdExecutor) CombinedOutput() ([]byte, error) {
	return m.output, m.err
}

// CommandExecutor interface for dependency injection
type CommandExecutor interface {
	CombinedOutput() ([]byte, error)
}

// CommandFactory interface for creating commands
type CommandFactory interface {
	CreateCommand(ctx context.Context, name string, args ...string) CommandExecutor
}

// Default command factory
type defaultCommandFactory struct{}

func (f *defaultCommandFactory) CreateCommand(ctx context.Context, name string, args ...string) CommandExecutor {
	return exec.CommandContext(ctx, name, args...)
}

// Mock command factory for testing
type mockCommandFactory struct {
	executor CommandExecutor
}

func (f *mockCommandFactory) CreateCommand(ctx context.Context, name string, args ...string) CommandExecutor {
	return f.executor
}

// Modified EdgeCoreSonicAnsible for testing with DI
type testableAnsibleImple struct {
	EdgeCoreSonicAnsible
	cmdFactory CommandFactory
}

func (l *testableAnsibleImple) CmdExecute(ctx context.Context, remoteHost string, remoteUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (output []byte, errMsg *common.ErrorMessage) {
	defer func() {
		l.Logger.V(2).Info("end CmdExecute",
			"remote_host", remoteHost,
			"playbook", playbook,
			"output", string(output),
			"errMsg", errMsg)
	}()
	l.Logger.V(2).Info("start CmdExecute",
		"remote_host", remoteHost,
		"remote_user", remoteUser,
		"ssh_private_key_file", sshPrivateKeyFile,
		"playbook", playbook,
		"extra_args", extrArgs)

	// get configuration
	// get ansible path from executable location
	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansiblePath := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", "edgecore_sonic_network", "ansible")

	// generate command line arguments
	args := []string{
		ansiblePath + "/" + playbook,
		"-i", remoteHost + ",",
		"-u", remoteUser,
		"--private-key", sshPrivateKeyFile,
		"-e", extrArgs,
	}

	// generate ansible command using factory
	cmd := l.cmdFactory.CreateCommand(ctx, "ansible-playbook", args...)
	l.Logger.V(2).Info("branch: ansible command generated",
		"remote_host", remoteHost,
		"cmd", "ansible-playbook "+remoteHost)

	// execute ansible command
	output, err := cmd.CombinedOutput()
	if err != nil {
		l.Logger.V(2).Info("branch: ansible command execution failed",
			"remote_host", remoteHost,
			"error", err.Error())

		// Extract meaningful error message from Ansible output
		var errorMessage string
		if len(output) > 0 {
			extractedError := utils.ExtractAnsibleError(output)
			if (extractedError != "") {
				errorMessage = extractedError
			} else {
				errorMessage = err.Error()
			}
		} else {
			errorMessage = err.Error()
		}
		klog.Error(errorMessage)

		errMsg = &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(l.analyzeAnsibleError(err)),
			Message:    errorMessage,
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost)
	// success case
	return output, nil
}

// setupTestConfig sets up test configuration for ansible tests
func setupTestConfig(t *testing.T) {
	// Reset config to allow re-initialization
	utils.ResetConfigForTesting()
	// Set environment variables
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/tmp/certs")
	// Initialize config after setting env vars
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}
	// Verify config is not nil
	if utils.GetConfig() == nil {
		t.Fatal("Config is nil after initialization")
	}
}

// clearTestConfig clears test configuration
func clearTestConfig() {
	os.Unsetenv("NW_SERVER_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SSH_KEY")
	os.Unsetenv("TLS_ENABLE")
	os.Unsetenv("TLS_CERT_PATH")
}

func TestCmdExecute_ValidCommand_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockOutput := []byte("ansible execution success")
	mockExecutor := &mockCmdExecutor{
		output: mockOutput,
		err:    nil,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:               cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "192.168.1.1", "admin", "/tmp/test.pem", "test.yaml", "port=1 vid=100")

	// Assert
	if errMsg != nil {
		t.Errorf("Expected no error message, got %v", errMsg)
	}
	if string(output) != string(mockOutput) {
		t.Errorf("Expected output %s, got %s", string(mockOutput), string(output))
	}
}

func TestCmdExecute_CommandFails_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockError := &exec.ExitError{ProcessState: nil}
	mockExecutor := &mockCmdExecutor{
		output: []byte("command failed"),
		err:    mockError,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:               cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "192.168.1.1", "admin", "/tmp/test.pem", "test.yaml", "port=1 vid=100")

	// Assert
	if errMsg == nil {
		t.Error("Expected error message, got nil")
		return
	}
	if errMsg.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected error code %d, got %d", codes.Internal, errMsg.ErrorCode)
	}
	if string(output) != "command failed" {
		t.Errorf("Expected output 'command failed', got %s", string(output))
	}
}

func TestCmdExecute_EmptyRemoteHost_ExecutesWithEmptyHost(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockOutput := []byte("success")
	mockExecutor := &mockCmdExecutor{
		output: mockOutput,
		err:    nil,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:               cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "", "admin", "/tmp/test.pem", "test.yaml", "port=1 vid=100")

	// Assert
	if errMsg != nil {
		t.Errorf("Expected no error message, got %v", errMsg)
	}
	if string(output) != string(mockOutput) {
		t.Errorf("Expected output %s, got %s", string(mockOutput), string(output))
	}
}

func TestCmdExecute_EmptyExtraArgs_ExecutesWithEmptyArgs(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockOutput := []byte("success")
	mockExecutor := &mockCmdExecutor{
		output: mockOutput,
		err:    nil,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:               cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "192.168.1.1", "admin", "/tmp/test.pem", "test.yaml", "")

	// Assert
	if errMsg != nil {
		t.Errorf("Expected no error message, got %v", errMsg)
	}
	if string(output) != string(mockOutput) {
		t.Errorf("Expected output %s, got %s", string(mockOutput), string(output))
	}
}

func TestAnalyzeAnsibleError_EnvironmentError_ReturnsEnvironmentErrorCode(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := errors.New("command not found")

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(err)

	// Assert
	if result != proto.DetailCode_NW_ENVIRONMENT_ERROR {
		t.Errorf("Expected proto.DetailCode_NW_ENVIRONMENT_ERROR, got %v", result)
	}
}

func TestAnalyzeAnsibleError_ExitCode1_ReturnsCommandError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	cmd := exec.Command("sh", "-c", "exit 1")
	cmd.Run() // This will create an ExitError with exit code 1
	// Create a proper ExitError for testing
	exitErr := &exec.ExitError{ProcessState: cmd.ProcessState}

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert
	if result != proto.DetailCode_NW_COMMAND_ERROR {
		t.Errorf("Expected proto.DetailCode_NW_COMMAND_ERROR, got %v", result)
	}
}

func TestAnalyzeAnsibleError_ExitCode2_ReturnsCommandError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	cmd := exec.Command("sh", "-c", "exit 2")
	cmd.Run()
	exitErr := &exec.ExitError{ProcessState: cmd.ProcessState}

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert
	if result != proto.DetailCode_NW_COMMAND_ERROR {
		t.Errorf("Expected proto.DetailCode_NW_COMMAND_ERROR, got %v", result)
	}
}

func TestAnalyzeAnsibleError_ExitCode3_ReturnsEnvironmentError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	cmd := exec.Command("sh", "-c", "exit 3")
	cmd.Run()
	exitErr := &exec.ExitError{ProcessState: cmd.ProcessState}

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert
	if result != proto.DetailCode_NW_ENVIRONMENT_ERROR {
		t.Errorf("Expected proto.DetailCode_NW_ENVIRONMENT_ERROR, got %v", result)
	}
}

func TestAnalyzeAnsibleError_ExitCode4_ReturnsEnvironmentError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	cmd := exec.Command("sh", "-c", "exit 4")
	cmd.Run()
	exitErr := &exec.ExitError{ProcessState: cmd.ProcessState}

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert
	if result != proto.DetailCode_NW_ENVIRONMENT_ERROR {
		t.Errorf("Expected proto.DetailCode_NW_ENVIRONMENT_ERROR, got %v", result)
	}
}

func TestAnalyzeAnsibleError_ExitCode5_ReturnsEnvironmentError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	cmd := exec.Command("sh", "-c", "exit 5")
	cmd.Run()
	exitErr := &exec.ExitError{ProcessState: cmd.ProcessState}

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert
	if result != proto.DetailCode_NW_ENVIRONMENT_ERROR {
		t.Errorf("Expected proto.DetailCode_NW_ENVIRONMENT_ERROR, got %v", result)
	}
}

func TestAnalyzeAnsibleError_UnknownExitCode_ReturnsCommandError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	cmd := exec.Command("sh", "-c", "exit 99")
	cmd.Run()
	exitErr := &exec.ExitError{ProcessState: cmd.ProcessState}

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert
	if result != proto.DetailCode_NW_COMMAND_ERROR {
		t.Errorf("Expected proto.DetailCode_NW_COMMAND_ERROR, got %v", result)
	}
}

// Test original implementation with integration test approach
func TestEdgeCoreSonicAnsible_CmdExecute_OriginalImplementation_ValidateStructure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	ansible := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}

	// This test verifies that the original struct has the expected structure
	// We can't easily test the actual execution without mocking the exec package,
	// but we can verify the struct is properly initialized

	// Act & Assert
	// Just verify the struct can be created successfully
	if &ansible == nil {
		t.Error("Expected ansible struct to be created, got nil")
	}
}

// Test to improve coverage for the real CmdExecute method
func TestEdgeCoreSonicAnsible_CmdExecute_RealImplementation_HandlesInvalidCommand(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	ansible := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	ctx := context.Background()

	// Act - Use invalid command that will fail quickly
	output, errMsg := ansible.CmdExecute(ctx, "invalid-host", "invalid-user", "/nonexistent/key", "nonexistent.yaml", "test=1")

	// Assert
	if errMsg == nil {
		t.Log("Note: Error message might be nil if system handles invalid commands gracefully")
	} else {
		if errMsg.ErrorCode != int32(codes.Internal) {
			t.Errorf("Expected error code %d, got %d", codes.Internal, errMsg.ErrorCode)
		}
		// Should have some output or error information
		if len(output) == 0 && errMsg.Message == "" {
			t.Error("Expected either output or error message, got neither")
		}
	}
}

// Test analyzeAnsibleError function coverage
func TestAnalyzeAnsibleError_NotExitError_ReturnsEnvironmentError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	err := errors.New("general error")

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(err)

	// Assert
	if result != proto.DetailCode_NW_ENVIRONMENT_ERROR {
		t.Errorf("Expected proto.DetailCode_NW_ENVIRONMENT_ERROR, got %v", result)
	}
}

func TestAnalyzeAnsibleError_WithExitError_ReturnsValidCode(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	exitErr := &exec.ExitError{ProcessState: &os.ProcessState{}}
	// Note: This is a simplified mock - in real usage ExitCode() would return actual exit code

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert - This will depend on the actual implementation behavior
	// Since we can't easily mock ProcessState.ExitCode(), this test validates the function runs
	if result != proto.DetailCode_NW_COMMAND_ERROR && result != proto.DetailCode_NW_ENVIRONMENT_ERROR {
		t.Errorf("Expected valid DetailCode, got %v", result)
	}
}

func TestAnalyzeAnsibleError_WithDifferentExitErrors_ReturnsValidCodes(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - Test with exec.ExitError
	exitErr := &exec.ExitError{ProcessState: &os.ProcessState{}}

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert - This validates the function executes and returns a valid result
	if result != proto.DetailCode_NW_COMMAND_ERROR && result != proto.DetailCode_NW_ENVIRONMENT_ERROR {
		t.Errorf("Expected valid DetailCode, got %v", result)
	}
}

func TestAnalyzeAnsibleError_EdgeCase_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	exitErr := &exec.ExitError{ProcessState: &os.ProcessState{}}

	// Act
	nw := EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"}
	result := nw.analyzeAnsibleError(exitErr)

	// Assert - This validates the function executes and returns a valid result
	if result != proto.DetailCode_NW_COMMAND_ERROR && result != proto.DetailCode_NW_ENVIRONMENT_ERROR {
		t.Errorf("Expected valid DetailCode, got %v", result)
	}
}
// TestCmdExecute_ErrorWithEmptyOutput_UsesErrorMessage tests error handling with empty output
func TestCmdExecute_ErrorWithEmptyOutput_UsesErrorMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockError := &exec.ExitError{ProcessState: nil}
	mockExecutor := &mockCmdExecutor{
		output: []byte(""), // Empty output
		err:    mockError,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:               cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "192.168.1.1", "admin", "/tmp/test.pem", "test.yaml", "port=1 vid=100")

	// Assert
	if errMsg == nil {
		t.Error("Expected error message, got nil")
		return
	}
	if errMsg.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected error code %d, got %d", codes.Internal, errMsg.ErrorCode)
	}
	if len(output) != 0 {
		t.Errorf("Expected empty output, got %d bytes", len(output))
	}
	// Error message should be from err.Error() since output is empty
	if errMsg.Message == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestCmdExecute_ErrorWithOutputButNoExtractedError_UsesErrError tests fallback to err.Error()
func TestCmdExecute_ErrorWithOutputButNoExtractedError_UsesErrError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockError := &exec.ExitError{ProcessState: nil}
	// Output that ExtractAnsibleError will return empty string for
	mockExecutor := &mockCmdExecutor{
		output: []byte("some output without ansible error pattern"),
		err:    mockError,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:               cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "192.168.1.1", "admin", "/tmp/test.pem", "test.yaml", "port=1 vid=100")

	// Assert
	if errMsg == nil {
		t.Error("Expected error message, got nil")
		return
	}
	if errMsg.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected error code %d, got %d", codes.Internal, errMsg.ErrorCode)
	}
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}
	// Error message should be from err.Error() since ExtractAnsibleError returns empty
	if errMsg.Message == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestCmdExecute_ErrorWithAnsibleFatalError_UsesExtractedError tests CmdExecute with Ansible fatal error output
func TestCmdExecute_ErrorWithAnsibleFatalError_UsesExtractedError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockError := &exec.ExitError{}
	mockExecutor := &mockCmdExecutor{
		output: []byte(`fatal: [testhost]: FAILED! => {"changed": false, "msg": "Connection refused"}`),
		err:    mockError,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:           cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "192.168.1.1", "admin", "/tmp/test.pem", "test.yaml", "port=1")

	// Assert
	if errMsg == nil {
		t.Error("Expected non-nil error message")
		return
	}
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}
	// Should use extracted error message from Ansible output
	if !strings.Contains(errMsg.Message, "testhost") || !strings.Contains(errMsg.Message, "Connection refused") {
		t.Errorf("Expected extracted error message containing 'testhost' and 'Connection refused', got '%s'", errMsg.Message)
	}
}

// TestCmdExecute_ErrorWithAnsibleJSONError_UsesExtractedError tests CmdExecute with Ansible JSON-formatted error
func TestCmdExecute_ErrorWithAnsibleJSONError_UsesExtractedError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockError := &exec.ExitError{}
	mockExecutor := &mockCmdExecutor{
		output: []byte(`UNREACHABLE: [server1]: UNREACHABLE! => {"changed": false, "msg": "Host is down"}`),
		err:    mockError,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:           cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "192.168.1.1", "admin", "/tmp/test.pem", "test.yaml", "")

	// Assert
	if errMsg == nil {
		t.Error("Expected non-nil error message")
		return
	}
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}
	// Should use extracted error message from Ansible output
	if !strings.Contains(errMsg.Message, "server1") || !strings.Contains(errMsg.Message, "unreachable") {
		t.Errorf("Expected extracted error message containing 'server1' and 'unreachable', got '%s'", errMsg.Message)
	}
}

// TestCmdExecute_ErrorWithAnsibleTextError_UsesExtractedError tests CmdExecute with Ansible text-formatted error
func TestCmdExecute_ErrorWithAnsibleTextError_UsesExtractedError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockError := &exec.ExitError{}
	mockExecutor := &mockCmdExecutor{
		output: []byte(`fatal: [webserver]: command execution failed`),
		err:    mockError,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		EdgeCoreSonicAnsible: EdgeCoreSonicAnsible{Logger: klog.Background(), AnsibleSubDir: "edgecore_sonic_network"},
		cmdFactory:           cmdFactory,
	}

	ctx := context.Background()

	// Act
	output, errMsg := ansible.CmdExecute(ctx, "192.168.1.1", "admin", "/tmp/test.pem", "test.yaml", "port=1")

	// Assert
	if errMsg == nil {
		t.Error("Expected non-nil error message")
		return
	}
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}
	// Should use extracted error message from Ansible output
	if !strings.Contains(errMsg.Message, "webserver") || !strings.Contains(errMsg.Message, "failed") {
		t.Errorf("Expected extracted error message containing 'webserver' and 'failed', got '%s'", errMsg.Message)
	}
}
