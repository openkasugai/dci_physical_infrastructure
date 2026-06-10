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
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"k8s.io/klog/v2"
	"exporter_module/internal/server/utils"
)

// CommandExecutor provides an abstraction for executing OS commands
type CommandExecutor interface {
	CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd
}

// RealCommandExecutor provides the actual command execution
type RealCommandExecutor struct{}

func (r RealCommandExecutor) CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}

// struct of ansible
type AnsibleImplement struct {
	Logger   klog.Logger
	Executor CommandExecutor
}

// NewAnsibleImplement creates a new AnsibleImplement instance with default executor
func NewAnsibleImplement(logger klog.Logger) *AnsibleImplement {
	return &AnsibleImplement{
		Logger:   logger,
		Executor: RealCommandExecutor{},
	}
}

// command execution
func (l AnsibleImplement) CmdExecute(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extraArgs string) (output interface{}, err error) {
	l.Logger.V(2).Info("start CmdExecute", "remoteHost", remoteHost, "remotUser", remotUser, "playbook", playbook, "extraArgs", extraArgs)
	defer func() {
		l.Logger.V(2).Info("end CmdExecute", "output", output, "err", err)
	}()

	l.Logger.Info("executing Ansible playbook", "host", remoteHost, "user", remotUser, "playbook", playbook)

	// get env
	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansible_path := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", "ansible")
	l.Logger.V(2).Info("Ansible path retrieved", "ansiblePath", ansible_path)

	// generate command lisne arguments
	args := []string{
		"ansible-playbook",
		ansible_path + "/" + playbook,
		"-i", remoteHost + ",",
		"-u", remotUser,
		"--private-key", sshPrivateKeyFile,
	}
	if extraArgs != "" {
		l.Logger.V(2).Info("branch: extra arguments provided", "extraArgs", extraArgs)
		args = append(append(args, "-e"), extraArgs)
	} else {
		l.Logger.V(2).Info("branch: no extra arguments")
	}

	// generate ansible command
	cmd := l.Executor.CommandContext(ctx, args[0], args[1:]...)
	cmdStr := strings.Join(args, " ")
	l.Logger.V(2).Info("Ansible command generated", "cmd", cmdStr)

	// excetute ansible comamnd
	out, e := cmd.CombinedOutput()
	if e != nil {
		l.Logger.V(2).Info("branch: Ansible command execution failed", "error", e.Error(), "output", string(out))

        // Extract meaningful error message from Ansible output
        var errorMessage string
        if len(out) > 0 {
            extractedError := utils.ExtractAnsibleError(out)
			if (extractedError != "") {
				errorMessage = extractedError
			} else {
				errorMessage = e.Error()
			}
        } else {
            errorMessage = e.Error()
        }
		err = errors.New(errorMessage)
		return
	}

	l.Logger.V(2).Info("branch: Ansible command executed successfully", "outputSize", len(out))

	// parse ansible result
	output, err = l.extractMsg(string(out))
	if err != nil {
		l.Logger.V(2).Info("branch: Ansible output parsing failed", "error", err.Error())
		return
	}

	l.Logger.V(2).Info("branch: Ansible output parsed successfully")
	l.Logger.Info("Ansible playbook execution completed", "host", remoteHost, "playbook", playbook)
	return
}

func fixJSON(input string) string {
	input = strings.TrimSpace(input)

	openBraceCount := strings.Count(input, "{")
	closeBraceCount := strings.Count(input, "}")

	if closeBraceCount < openBraceCount {
		diff := openBraceCount - closeBraceCount
		input += strings.Repeat("}", diff)
	}

	return input
}

func (l AnsibleImplement) extractMsg(input string) (message interface{}, err error) {
	l.Logger.V(2).Info("start extractMsg", "inputSize", len(input))
	defer func() {
		l.Logger.V(2).Info("end extractMsg", "err", err)
	}()

	re := regexp.MustCompile(`(?s)TASK \[.*?\].*?=>[ \t]*({.*?})\n`)
	match := re.FindStringSubmatch(input)

	if len(match) > 1 {
		l.Logger.V(2).Info("branch: regex match found", "matchCount", len(match))
		jsonStr := fixJSON(match[1])
		l.Logger.V(2).Info("branch: JSON string fixed", "jsonStr", jsonStr)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			l.Logger.V(2).Info("branch: JSON unmarshal failed", "error", err.Error())
			return
		}

		l.Logger.V(2).Info("branch: JSON unmarshal successful", "fieldsCount", len(result))

		msg, ok := result["msg"]
		if !ok {
			l.Logger.V(2).Info("branch: 'msg' key not found in result")
			err = errors.New("'msg' key not found")
			return
		}

		l.Logger.V(2).Info("branch: 'msg' key extracted successfully")
		message = msg
		return

	} else {
		l.Logger.V(2).Info("branch: no regex match found for result output")
		err = errors.New("no match found for result output")
		return
	}
}
