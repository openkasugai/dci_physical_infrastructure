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
	"fmt"

	"k8s.io/klog/v2"
	"log_module/internal/server/utils"
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
func (l AnsibleImplement) CmdExecute(ctx context.Context, remoteHost string, remotUser string, playbook string, extraArgs string) (output interface{}, err error) {
	l.Logger.V(3).Info("start CmdExecute", "remoteHost", remoteHost, "remotUser", remotUser, "playbook", playbook)
	defer func() {
		l.Logger.V(3).Info("end CmdExecute", "err", err)
		l.Logger.V(3).Info("CmdExecute output detail", "output", output)
	}()

	l.Logger.V(3).Info("executing Ansible playbook", "host", remoteHost, "user", remotUser, "playbook", playbook)

	// get env
	exePath, _ := os.Executable()
	filePath, _ := filepath.Abs(exePath)
	ansible_path := filepath.Join(filepath.Dir(filePath), "internal", "server", "implementation", "ansible")
	l.Logger.V(3).Info("Ansible path retrieved", "ansiblePath", ansible_path)

	sshPrivateKeyFile := os.Getenv("SSH_KEY")
	l.Logger.V(3).Info("SSH key path retrieved", "sshKeyPath", sshPrivateKeyFile)

	// generate command lisne arguments
	args := []string{
		"ansible-playbook",
		ansible_path + "/" + playbook,
		"-i", remoteHost + ",",
		"-u", remotUser,
		"--private-key", sshPrivateKeyFile,
	}
	if extraArgs != "" {
		l.Logger.V(3).Info("branch: extra arguments provided", "extraArgs", extraArgs)
		args = append(append(args, "-e"), extraArgs)
	} else {
		l.Logger.V(3).Info("branch: no extra arguments")
	}

	// generate ansible command
	cmd := l.Executor.CommandContext(ctx, args[0], args[1:]...)
	cmdStr := strings.Join(args, " ")
	l.Logger.V(2).Info("Ansible command generated", "cmd", cmdStr)

	// excetute ansible comamnd
	out, e := cmd.CombinedOutput()
	if e != nil {
		l.Logger.V(2).Info("branch: Ansible command execution failed", "error", e.Error(), "outputSize", len(out))
		l.Logger.V(3).Info("branch: Ansible command execution failed detail", "output", string(out))

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

	l.Logger.V(3).Info("branch: Ansible command executed successfully", "outputSize", len(out))

	// parse ansible result
	if (playbook == "cdi_spec_list.yaml") {
		l.Logger.V(3).Info("branch: parsing CDI wrapper output")
		_, err = l.parseCDIWrapperOutput(out)
		if err != nil {
			l.Logger.V(2).Info("branch: Ansible output parsing failed", "error", err.Error())
			return
		}
	} else {
		l.Logger.V(3).Info("branch: extracting message from Ansible output")
		output, err = l.extractMsg(string(out))
		if err != nil {
			l.Logger.V(2).Info("branch: Ansible output parsing failed", "error", err.Error())
			return
		}
	}

	l.Logger.V(3).Info("branch: Ansible output parsed successfully")
	l.Logger.V(3).Info("Ansible playbook execution completed", "host", remoteHost, "playbook", playbook)
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
	l.Logger.V(3).Info("start extractMsg", "inputSize", len(input))
	defer func() {
		l.Logger.V(3).Info("end extractMsg", "err", err)
	}()

	fixedMessage := "invalid cdi response: "

	re := regexp.MustCompile(`(?s)TASK \[result output\].*?=>[ \t]*({.*?})\n`)
	match := re.FindStringSubmatch(input)

	if len(match) > 1 {
		l.Logger.V(3).Info("branch: regex match found", "matchCount", len(match))
		jsonStr := fixJSON(match[1])

		var result map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			l.Logger.V(2).Info("branch: JSON unmarshal failed", "error", err.Error())
			return
		}

		l.Logger.V(3).Info("branch: JSON unmarshal successful", "fieldsCount", len(result))

		msg, ok := result["msg"]
		if !ok {
			l.Logger.V(2).Info("branch: 'msg' key not found in result")
			err = errors.New(fixedMessage + "'msg' key not found")
			return
		}

		l.Logger.V(3).Info("branch: 'msg' key extracted successfully")
		message = msg
	} else {
		l.Logger.V(2).Info("branch: no regex match found for result output")
		err = errors.New(fixedMessage + "no match found for result output")
	}

	return
}

// parseCdiWrapperOutput Parse CdiWrapper.sh output directly
func (l AnsibleImplement) parseCDIWrapperOutput(output []byte) (jsonData map[string]interface{}, err error) {
	defer func() {
		l.Logger.V(2).Info("end parseCdiWrapperOutput", "err", err)
		l.Logger.V(3).Info("parseCdiWrapperOutput detail", "jsonData", jsonData)
	}()
	l.Logger.V(3).Info("start parseCdiWrapperOutput",
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
		l.Logger.V(3).Info("branch: result type is SUCCESS")
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
				l.Logger.V(3).Info("branch: parsing JSON data")
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
					l.Logger.V(3).Info("branch: JSON parsing successful")
					return result, nil
				} else {
					l.Logger.Error(err, fixedMessage+": "+err.Error())
					return nil, err
				}
			}
		}
		return nil, nil

	case "ERROR_V_1_0":
		l.Logger.Error(nil, errorMessage)
		return nil, errors.New(errorMessage)

	case "ERROR_V_1_1":
		l.Logger.Error(nil, errorMessage)
		return nil, errors.New(errorMessage)

	case "UNKNOWN":
		l.Logger.Error(nil, fixedMessage+errorMessage)
		return nil, errors.New(fixedMessage+errorMessage)

	default:
		l.Logger.Error(nil, fixedMessage+fmt.Sprintf(": No valid result type found in output: %s", outputStr))
		return nil, errors.New(fixedMessage+fmt.Sprintf(": No valid result type found in output: %s", outputStr))
	}
}
