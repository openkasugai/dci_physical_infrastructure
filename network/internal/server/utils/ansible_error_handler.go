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
    
    // Check for stderr messages
    if stderrError := extractStderrMessage(outputStr); stderrError != "" {
        return stderrError
    }

    // Parse JSON-formatted task results
    if errorInfo := parseAnsibleJSON(outputStr); errorInfo != nil {
        return formatErrorMessage(errorInfo)
    }
    
    // Parse text-formatted task results (fallback)
    if errorInfo := parseAnsibleText(outputStr); errorInfo != nil {
        return formatErrorMessage(errorInfo)
    }
    
    // Extract PLAY RECAP for summary
    if recapError := parsePlayRecap(outputStr); recapError != "" {
        return recapError
    }
    
    // Other -> return empty
    return ""
}

// extractStderrMessage extracts error messages from stderr in Ansible output
func extractStderrMessage(output string) string {
    // Corrected regex to handle escaped quotes within the stderr string
    // This pattern captures the content within the first double quotes after "stderr":
    stderrPattern := regexp.MustCompile(`"stderr":\s*"((?:[^"\\]|\\.)*)"`)
    matches := stderrPattern.FindAllStringSubmatch(output, -1)
    
    for _, match := range matches {
        if len(match) > 1 {
            // The captured string might still contain escaped newlines and quotes
            stderr := match[1]
            stderr = strings.ReplaceAll(stderr, "\\n", "\n") // Convert escaped \n to actual newline
            stderr = strings.ReplaceAll(stderr, "\\\"", "\"") // Convert escaped \" to actual quote
            stderr = strings.TrimSpace(stderr)
            
            // Return if stderr contains meaningful error information
            if stderr != "" { // Check only for content, then prioritize within extractMostImportantError
                return extractMostImportantError(stderr)
            }
        }
    }
    
    return ""
}

// extractMostImportantError extracts the most important error message from stderr
func extractMostImportantError(stderr string) string {
    lines := strings.Split(stderr, "\n")
    
    // Priority 1: Look for "Error:" lines - these are the most specific
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "Error: ") {
            return line
        }
    }
    
    // Priority 2: Look for "Usage:" lines for command help related messages
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "Usage: ") {
            return cleanErrorMessage(line) // Clean up if it's a usage line
        }
    }

    // Priority 3: Return the complete stderr cleaned up if no specific pattern found
    return cleanErrorMessage(stderr)
}

// cleanErrorMessage cleans and formats error messages
func cleanErrorMessage(message string) string {
    lines := strings.Split(message, "\n")
    var cleanLines []string
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" {
            cleanLines = append(cleanLines, line)
        }
    }
    
    // Join with space for better readability, preserving line breaks if they convey meaning for multi-line errors.
    result := strings.Join(cleanLines, " ")
    
    // Clean up any remaining escape sequences (e.g., if there were `\\u003c` from earlier mis-parsing)
    // Given the regex fix, these might be less necessary, but good for robustness.
    result = strings.ReplaceAll(result, "\\\"", "\"")
    result = strings.ReplaceAll(result, "\\\\", "\\") // Convert `\\` to `\` (e.g., if JSON had `\\\\` for a literal `\`)
    
    return result
}

// parseAnsibleJSON parses JSON-formatted Ansible task results
func parseAnsibleJSON(output string) *AnsibleErrorInfo {
    // Pattern for JSON task results: fatal: [host]: FAILED! => {"changed": false, "msg": "..."}
    // Corrected to handle nested braces within the JSON string for more robustness
    jsonPattern := regexp.MustCompile(`(fatal|UNREACHABLE): \[([^\]]+)\]: (FAILED!|UNREACHABLE!) => (\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\})`)
    matches := jsonPattern.FindStringSubmatch(output)
    
    if len(matches) >= 5 {
        errorType := matches[1]
        hostName := matches[2]
        status := matches[3]
        jsonStr := matches[4] // Correctly captured JSON string
        
        var result map[string]interface{}
        // Use json.Unmarshal to robustly parse the JSON string, which handles all escape sequences.
        if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
            errorInfo := &AnsibleErrorInfo{
                HostName:    hostName,
                ErrorType:   status,
                Unreachable: strings.Contains(errorType, "UNREACHABLE"),
            }
            
            if msg, ok := result["msg"].(string); ok {
                // If the 'msg' field is present and is a string, use it.
                // This is often a concise message from Ansible itself.
                errorInfo.Message = msg
            } else if stderr, ok := result["stderr"].(string); ok {
                // If 'msg' is not there or not a string, check for 'stderr'.
                // json.Unmarshal will have already unescaped this string.
                if cleanedStderr := extractMostImportantError(stderr); cleanedStderr != "" {
                     errorInfo.Message = cleanedStderr
                } else {
                     errorInfo.Message = stderr // Fallback to raw stderr if nothing specific extracted
                }
            } else if stdout, ok := result["stdout"].(string); ok {
                 // As a last resort, if stderr is empty, check stdout (though less likely to contain primary errors)
                if strings.Contains(stdout, "Error:") { // Simple check for "Error:" in stdout
                    errorInfo.Message = stdout
                }
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
    } else if errorInfo.ErrorType != "" && errorInfo.ErrorType != "FAILED" { // Only add "failed" if it's the specific type
         parts = append(parts, strings.ToLower(errorInfo.ErrorType))
    } else if !errorInfo.Unreachable && errorInfo.ErrorType == "FAILED" {
        parts = append(parts, "failed")
    }
    
    result := strings.Join(parts, " ")
    
    if errorInfo.Message != "" {
        // If the result string is empty because host and type aren't informative, just return message.
        if result == "" {
            return errorInfo.Message
        }
        result += ": " + errorInfo.Message
    }
    
    return result
}
