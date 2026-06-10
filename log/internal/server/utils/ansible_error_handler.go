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

package utils

import (
	"encoding/json"
	"regexp"
	"strings"
)

// AnsibleErrorInfo represents extracted error information
type AnsibleErrorInfo struct {
    HostName    string
    ErrorType   string // FAILED, UNREACHABLE, etc.
    Message     string
    Unreachable bool
}


// ExtractAnsibleError extracts meaningful error messages from Ansible output
func ExtractAnsibleError(output []byte) string {
    outputStr := string(output)
    
    // 1. Parse JSON-formatted task results
    if errorInfo := parseAnsibleJSON(outputStr); errorInfo != nil {
        return formatErrorMessage(errorInfo)
    }
    
    // 2. Parse text-formatted task results (fallback)
    if errorInfo := parseAnsibleText(outputStr); errorInfo != nil {
        return formatErrorMessage(errorInfo)
    }
    
    // 3. Extract PLAY RECAP for summary
    if recapError := parsePlayRecap(outputStr); recapError != "" {
        return recapError
    }
    
    // Other -> return empty
    return ""
}

// parseAnsibleJSON parses JSON-formatted Ansible task results
func parseAnsibleJSON(output string) *AnsibleErrorInfo {
    // Pattern for JSON task results: fatal: [host]: FAILED! => {"changed": false, "msg": "..."}
    jsonPattern := regexp.MustCompile(`(fatal|UNREACHABLE): \[([^\]]+)\]: (FAILED!|UNREACHABLE!) => ({[^}]+(?:\}[^}]*)*})`)
    matches := jsonPattern.FindStringSubmatch(output)
    
    if len(matches) >= 5 {
        errorType := matches[1]
        hostName := matches[2]
        status := matches[3]
        jsonStr := matches[4]
        
        var result map[string]interface{}
        if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
            errorInfo := &AnsibleErrorInfo{
                HostName:    hostName,
                ErrorType:   status,
                Unreachable: strings.Contains(errorType, "UNREACHABLE"),
            }
            
            if msg, ok := result["msg"].(string); ok {
                errorInfo.Message = msg
            }
            
            return errorInfo
        }
    }
    
    return nil
}

// parseAnsibleText parses text-formatted Ansible errors
func parseAnsibleText(output string) *AnsibleErrorInfo {
    lines := strings.Split(output, "\n")
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        
        // Look for error patterns
        if strings.Contains(line, "fatal:") || strings.Contains(line, "FAILED!") || strings.Contains(line, "UNREACHABLE!") {
            // Extract host and error type
            hostPattern := regexp.MustCompile(`\[([^\]]+)\]`)
            hostMatches := hostPattern.FindStringSubmatch(line)
            
            errorInfo := &AnsibleErrorInfo{
                ErrorType:   "FAILED",
                Unreachable: strings.Contains(line, "UNREACHABLE"),
            }
            
            if len(hostMatches) > 1 {
                errorInfo.HostName = hostMatches[1]
            }
            
            // Extract error message from the line itself
            if colonIndex := strings.Index(line, ":"); colonIndex != -1 {
                if remaining := strings.TrimSpace(line[colonIndex+1:]); remaining != "" {
                    // Remove common prefixes like "FAILED!" or "UNREACHABLE!"
                    remaining = strings.TrimPrefix(remaining, "FAILED!")
                    remaining = strings.TrimPrefix(remaining, "UNREACHABLE!")
                    remaining = strings.TrimSpace(remaining)
                    if remaining != "" {
                        errorInfo.Message = remaining
                    }
                }
            }
            
            return errorInfo
        }
    }
    
    return nil
}

// parsePlayRecap extracts error summary from PLAY RECAP
func parsePlayRecap(output string) string {
    recapPattern := regexp.MustCompile(`PLAY RECAP[^*]*\n([^*]+)`)
    matches := recapPattern.FindStringSubmatch(output)
    
    if len(matches) > 1 {
        recapLines := strings.Split(strings.TrimSpace(matches[1]), "\n")
        for _, line := range recapLines {
            line = strings.TrimSpace(line)
            if line != "" {
                // Look for failed or unreachable counts
                if strings.Contains(line, "failed=") && !strings.Contains(line, "failed=0") {
                    return "Task execution failed: " + line
                }
                if strings.Contains(line, "unreachable=") && !strings.Contains(line, "unreachable=0") {
                    return "Host unreachable: " + line
                }
            }
        }
    }
    
    return ""
}

// formatErrorMessage formats extracted error information
func formatErrorMessage(errorInfo *AnsibleErrorInfo) string {
    var parts []string
    
    if errorInfo.HostName != "" {
        parts = append(parts, "Host "+errorInfo.HostName)
    }
    
    if errorInfo.Unreachable {
        parts = append(parts, "is unreachable")
    } else {
        parts = append(parts, "failed")
    }
    
    result := strings.Join(parts, " ")
    
    if errorInfo.Message != "" {
        result += ": " + errorInfo.Message
    }
    
    return result
}
