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
	"os/exec"
	"strings"
	"os"
	"path/filepath"

	// import of gRPC protobuf
	"maas_module/internal/server/utils"

	"k8s.io/klog/v2"
)

// struct of Ansible
type CanonicalMaasAnsibleImple struct {
	Logger   klog.Logger
	Executor cmdExecutor
}

var loginUser = "cloud-user"

// cmdExecutor interface for command execution.  This allows us to mock the execution.
type cmdExecutor interface {
	ExecuteCommand(ctx context.Context, name string, arg ...string) ([]byte, error)
}

// CmdExecutor is the real implementation of the command executor.
type CmdExecutor struct{}

// ExecuteCommand executes a command.
func (r *CmdExecutor) ExecuteCommand(ctx context.Context, name string, arg ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd.CombinedOutput()
}

// command execution
func (l CanonicalMaasAnsibleImple) CmdExecute(ctx context.Context, remoteHost string, playbook string, extrArgs string) (output []byte, err error) {
	klog.V(2).InfoS("start CmdExecute", "remoteHost", remoteHost, "playbook", playbook, "extrArgs", extrArgs)
	defer func() {
		klog.V(2).InfoS("end CmdExecute", "output", output, "err", err)
	}()

	// get config
	config := utils.GetConfig()
	klog.V(2).InfoS("branch: retrieved config", "config", config)
	
	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansible_path := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", "canonical_maas", "ansible")
	l.Logger.V(2).Info("Ansible path retrieved", "ansiblePath", ansible_path)

	// generate command lisne arguments
	args := []string{
		"ansible-playbook",
		ansible_path + "/" + playbook,
		"-i", remoteHost + ",",
		"-u", loginUser,
		"--private-key", config.SshKey,
	}
	if extrArgs != "" {
		klog.V(2).InfoS("branch: extra arguments provided", "extrArgs", extrArgs)
		args = append(append(args, "-e"), extrArgs)
	}

	// generate ansible command
	klog.InfoS("Ansible command generated", "cmd", strings.Join(args, " "))

	// excetute ansible comamnd
	output, err = l.Executor.ExecuteCommand(ctx, args[0], args[1:]...)
	if err != nil {
		klog.V(2).InfoS("branch: command execution failed", "error", err.Error())

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

		err = &utils.EnvError{Message: errorMessage}
		klog.Error(err, errorMessage)
		return
	}

	klog.V(2).InfoS("branch: command execution successful")
	return
}
