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
	"os/exec"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"

	proto "network_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf
	"network_module/internal/server/utils" // import network utils
)

// struct of SONiC network ansible
type EdgeCoreSonicAnsible struct {
	Logger        klog.Logger
	AnsibleSubDir string // sub-directory name under implementation/ that contains the ansible directory
}

// Command execution
func (l EdgeCoreSonicAnsible) CmdExecute(ctx context.Context, remoteHost string, remoteUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (output []byte, errMsg *common.ErrorMessage) {
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

	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansible_path := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", l.AnsibleSubDir, "ansible")
	l.Logger.V(2).Info("Ansible path retrieved", "ansiblePath", ansible_path)

	// generate command line arguments
	args := []string{
		"ansible-playbook",
		ansible_path + "/" + playbook,
		"-i", remoteHost + ",",
		"-u", remoteUser,
		"--private-key", sshPrivateKeyFile,
		"-e", extrArgs,
	}

	// generate ansible command
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	l.Logger.V(2).Info("branch: ansible command generated",
		"remote_host", remoteHost,
		"cmd", strings.Join(args, " "))

	// excetute ansible comamnd
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

func (l EdgeCoreSonicAnsible) analyzeAnsibleError(err error) proto.DetailCode {
	var result proto.DetailCode
	defer func() {
		l.Logger.V(2).Info("end analyzeAnsibleError", "result", result)
	}()
	l.Logger.V(2).Info("start analyzeAnsibleError", "error", err.Error())

	// error before process start
	exitError, isExitError := err.(*exec.ExitError)
	if !isExitError {
		l.Logger.V(2).Info("branch: not exit error", "error_type", "environment")
		result = proto.DetailCode_NW_ENVIRONMENT_ERROR
		return result
	}

	// judge error type by exit code
	exitCode := exitError.ExitCode()
	switch exitCode {
	case 1, 2: // 1: general error, 2: usage error
		l.Logger.V(2).Info("branch: exit code 1-2", "error_type", "command")
		result = proto.DetailCode_NW_COMMAND_ERROR
	case 3, 4, 5: // 3: Unreachable, 4: not found playbook, 5: syntax error
		l.Logger.V(2).Info("branch: exit code 3-5", "exit_code", exitCode, "error_type", "environment")
		result = proto.DetailCode_NW_ENVIRONMENT_ERROR
	default:
		l.Logger.V(2).Info("branch: unknown exit code", "exit_code", exitCode, "error_type", "command")
		result = proto.DetailCode_NW_COMMAND_ERROR
	}
	return result
}
