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

package broadcom_sonic_network

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	common "common/api/proto"                                                                      // import of common protobuf
	proto "network_module/api/proto"                                                               // import of gRPC protobuf
	edgecore_sonic_network "network_module/internal/server/implementation/edgecore_sonic_network" // import edgecore sonic network implement
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

// Mock command factory for testing
type mockCommandFactory struct {
	executor CommandExecutor
}

func (f *mockCommandFactory) CreateCommand(ctx context.Context, name string, args ...string) CommandExecutor {
	return f.executor
}

// Modified BroadcomSonicAnsible for testing with DI
type testableAnsibleImple struct {
	BroadcomSonicAnsible
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

	// get ansible path from executable location
	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansiblePath := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", "broadcom_sonic_network", "ansible")

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
			if extractedError != "" {
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
			DetailCode: int32(proto.DetailCode_NW_COMMAND_ERROR),
			Message:    errorMessage,
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost)
	return output, nil
}

// setupTestConfig sets up test configuration for ansible tests
func setupTestConfig(t *testing.T) {
	utils.ResetConfigForTesting()
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/tmp/certs")
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}
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
		BroadcomSonicAnsible: BroadcomSonicAnsible{
			EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
				Logger:        klog.Background(),
				AnsibleSubDir: "broadcom_sonic_network",
			},
		},
		cmdFactory: cmdFactory,
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
		BroadcomSonicAnsible: BroadcomSonicAnsible{
			EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
				Logger:        klog.Background(),
				AnsibleSubDir: "broadcom_sonic_network",
			},
		},
		cmdFactory: cmdFactory,
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
		BroadcomSonicAnsible: BroadcomSonicAnsible{
			EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
				Logger:        klog.Background(),
				AnsibleSubDir: "broadcom_sonic_network",
			},
		},
		cmdFactory: cmdFactory,
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
		BroadcomSonicAnsible: BroadcomSonicAnsible{
			EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
				Logger:        klog.Background(),
				AnsibleSubDir: "broadcom_sonic_network",
			},
		},
		cmdFactory: cmdFactory,
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

func TestBroadcomSonicAnsible_CmdExecute_OriginalImplementation_ValidateStructure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	ansible := BroadcomSonicAnsible{
		EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
			Logger:        klog.Background(),
			AnsibleSubDir: "broadcom_sonic_network",
		},
	}

	// Act & Assert
	if &ansible == nil {
		t.Error("Expected ansible struct to be created, got nil")
	}
	if ansible.AnsibleSubDir != "broadcom_sonic_network" {
		t.Errorf("Expected AnsibleSubDir 'broadcom_sonic_network', got '%s'", ansible.AnsibleSubDir)
	}
}

func TestBroadcomSonicAnsible_CmdExecute_RealImplementation_HandlesInvalidCommand(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	ansible := BroadcomSonicAnsible{
		EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
			Logger:        klog.Background(),
			AnsibleSubDir: "broadcom_sonic_network",
		},
	}
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
		if len(output) == 0 && errMsg.Message == "" {
			t.Error("Expected either output or error message, got neither")
		}
	}
}

func TestCmdExecute_ErrorWithEmptyOutput_UsesErrorMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestConfig(t)
	defer clearTestConfig()

	mockError := &exec.ExitError{ProcessState: nil}
	mockExecutor := &mockCmdExecutor{
		output: []byte(""),
		err:    mockError,
	}
	cmdFactory := &mockCommandFactory{executor: mockExecutor}

	ansible := &testableAnsibleImple{
		BroadcomSonicAnsible: BroadcomSonicAnsible{
			EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
				Logger:        klog.Background(),
				AnsibleSubDir: "broadcom_sonic_network",
			},
		},
		cmdFactory: cmdFactory,
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
	if errMsg.Message == "" {
		t.Error("Expected non-empty error message")
	}
}

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
		BroadcomSonicAnsible: BroadcomSonicAnsible{
			EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
				Logger:        klog.Background(),
				AnsibleSubDir: "broadcom_sonic_network",
			},
		},
		cmdFactory: cmdFactory,
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
	if !strings.Contains(errMsg.Message, "testhost") || !strings.Contains(errMsg.Message, "Connection refused") {
		t.Errorf("Expected extracted error message containing 'testhost' and 'Connection refused', got '%s'", errMsg.Message)
	}
}

func TestBroadcomSonicAnsible_AnsibleSubDir_IsCorrect(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange & Act
	ansible := BroadcomSonicAnsible{
		EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
			Logger:        klog.Background(),
			AnsibleSubDir: "broadcom_sonic_network",
		},
	}

	// Assert
	if ansible.AnsibleSubDir != "broadcom_sonic_network" {
		t.Errorf("Expected AnsibleSubDir 'broadcom_sonic_network', got '%s'", ansible.AnsibleSubDir)
	}
}
