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
	"encoding/json"
	"fmt"
	"os/exec"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"

	proto "cdi_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf
	"cdi_module/internal/server/utils"
)

// PgCDIAnsibleImple struct of PRIMAGY-CDI ansible
type PgCDIAnsibleImple struct {
	Logger klog.Logger
}

// CmdExecute command execution
func (l PgCDIAnsibleImple) CmdExecute(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (errMsg *common.ErrorMessage, jsonData map[string]interface{}) {
	defer func() {
		l.Logger.V(2).Info("end CmdExecute",
			"remote_host", remoteHost,
			"playbook", playbook,
			"errMsg", errMsg,
			"jsonData", jsonData)
	}()
	l.Logger.V(2).Info("start CmdExecute",
		"remote_host", remoteHost,
		"remote_user", remotUser,
		"ssh_private_key_file", sshPrivateKeyFile,
		"playbook", playbook,
		"extra_args", extrArgs)

	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansible_path := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", "pg_cdi", "ansible")
	l.Logger.V(2).Info("Ansible path retrieved", "ansiblePath", ansible_path)

	// Generate command line arguments
	args := []string{
		"ansible-playbook",
		ansible_path + "/" + playbook,
		"-i", remoteHost + ",",
		"-u", remotUser,
		"--private-key", sshPrivateKeyFile,
		"-e", extrArgs,
	}

	// Generate ansible command
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	l.Logger.V(2).Info("branch: ansible command generated",
		"remote_host", remoteHost,
		"cmd", strings.Join(args, " "))

	// Execute ansible command
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
		l.Logger.Error(err, errorMessage)

		errMsg = &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_CDI_ENVIRONMENT_ERROR),
			Message:    errorMessage,
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost)
	// Parse CdiWrapper.sh output directly
	data, errorMessage := l.parseCDIWrapperOutput(output)
	if errorMessage != nil {
		l.Logger.V(2).Info("branch: cdi wrapper output parsing failed",
			"remote_host", remoteHost)
		errMsg = errorMessage
		return
	}

	l.Logger.V(2).Info("branch: cdi wrapper output parsing successful",
		"remote_host", remoteHost)
	jsonData = data
	return
}

// parseCdiWrapperOutput Parse CdiWrapper.sh output directly
func (l PgCDIAnsibleImple) parseCDIWrapperOutput(output []byte) (jsonData map[string]interface{}, errMsg *common.ErrorMessage) {
	defer func() {
		l.Logger.V(2).Info("end parseCdiWrapperOutput",
			"jsonData", jsonData,
			"errMsg", errMsg)
	}()
	l.Logger.V(2).Info("start parseCdiWrapperOutput",
		"output_length", len(output))
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	var resultType string
	var dataLines []string
	var errorMessage string

	// Look for the result type marker
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "RESULT_TYPE:SUCCESS") {
			resultType = "SUCCESS"
			// Collect data lines after the marker
			resultLines := strings.Split(line, "\\n")
			if 1 < len(resultLines) {
				dataLines = resultLines[1:]
			}
			break
		} else if strings.Contains(line, "msg\": \"RESULT_TYPE:ERROR_V_1_0") {
			resultType = "ERROR_V_1_0"
			// Get error message from next line
			resultLines := strings.Split(line, "\\n")
			if 1 < len(resultLines) {
				errorMessage = strings.TrimSpace(resultLines[1])
				errorMessage = strings.TrimSuffix(errorMessage, "\"")
			}
			break
		} else if strings.Contains(line, "msg\": \"RESULT_TYPE:ERROR_V_1_1") {
			resultType = "ERROR_V_1_1"
			// Get error message from next line
			resultLines := strings.Split(line, "\\n")
			if 1 < len(resultLines) {
				errorMessage = strings.TrimSpace(resultLines[1])
				errorMessage = strings.TrimSuffix(errorMessage, "\"")
			}
			break
		} else if strings.Contains(line, "RESULT_TYPE:UNKNOWN") {
			resultType = "UNKNOWN"
			resultLines := strings.Split(line, "\\n")
			if 1 < len(resultLines) {
				errorMessage = strings.TrimSpace(resultLines[1])
				errorMessage = strings.TrimSuffix(errorMessage, "\"")
			}
			break
		}
	}

	fixedMessage := "invalid cdi response"
	switch resultType {
	case "SUCCESS":
		l.Logger.V(2).Info("branch: result type is SUCCESS")
		// Parse JSON from data lines
		if len(dataLines) > 0 {
			jsonStr := strings.Join(dataLines, "\n")
			jsonStr = strings.TrimSpace(jsonStr)
			jsonStr = strings.Replace(jsonStr, "\\n", "\n", -1)
			jsonStr = strings.Replace(jsonStr, "\\\"", "\"", -1)
			jsonStr = strings.Replace(jsonStr, "}\"", "}", -1)

			// Remove any trailing ansible output or shell prompts
			if idx := strings.Index(jsonStr, "cdi:"); idx != -1 {
				jsonStr = jsonStr[:idx]
				jsonStr = strings.TrimSpace(jsonStr)
			}

			if jsonStr != "" {
				l.Logger.V(2).Info("branch: parsing JSON data")
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
					l.Logger.V(2).Info("branch: JSON parsing successful")
					return result, nil
				} else {
					l.Logger.Error(err, fixedMessage+": "+err.Error())
					return nil, &common.ErrorMessage{
						ErrorCode:  int32(codes.Internal),
						DetailCode: int32(proto.DetailCode_CDI_RESPONSE_INVALID),
						Message:    fixedMessage,
					}
				}
			}
		}
		return nil, nil

	case "ERROR_V_1_0":
		l.Logger.Error(nil, errorMessage)
		return nil, &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
			Message:    errorMessage,
		}

	case "ERROR_V_1_1":
		l.Logger.Error(nil, errorMessage)
		return nil, &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_1),
			Message:    errorMessage,
		}

	case "UNKNOWN":
		l.Logger.Error(nil, fixedMessage+errorMessage)
		return nil, &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_CDI_RESPONSE_INVALID),
			Message:    fixedMessage,
		}

	default:
		l.Logger.Error(nil, fixedMessage+fmt.Sprintf(": No valid result type found in output: %s", outputStr))
		return nil, &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_CDI_RESPONSE_INVALID),
			Message:    fixedMessage,
		}
	}
}
