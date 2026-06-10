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
	"os"
	"os/exec"
	"testing"

	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"

	proto "cdi_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf
	"cdi_module/internal/server/interfaces"
	"cdi_module/internal/server/test_utils"
	"cdi_module/internal/server/utils"
)

func TestPgCDIAnsibleImple_ImplementsInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Verify it implements the interface
	var _ interfaces.CDIAnsible = impl
}

func TestPgCDIAnsibleImple_CmdExecute_SuccessfulAnsibleCommand_ReturnsData(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Skip if running without actual ansible environment
	if os.Getenv("SKIP_ANSIBLE_TESTS") != "" {
		t.Skip("Skipping ansible test due to SKIP_ANSIBLE_TESTS environment variable")
	}

	// Setup test environment
	setupValidTestEnv()
	defer teardownTestEnv()
	utils.ResetConfigForTesting()
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Mock successful command execution by creating a test script
	testScript := createTestAnsibleScript(t, `
echo 'msg": "RESULT_TYPE:SUCCESS\n{\"data\": {\"test\": \"success\"}}"'
`)
	defer os.Remove(testScript)

	// Execute with mocked command
	ctx := context.Background()
	errMsg, data := impl.cmdExecuteWithScript(ctx, "localhost", "testuser", "/tmp/key", testScript, "test=args")

	// Verify
	if errMsg != nil {
		t.Fatalf("CmdExecute should have succeeded, got error: %v", errMsg)
	}

	if data == nil {
		t.Fatal("CmdExecute should have returned data")
	}

	testData, ok := data["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to contain 'data' field with map")
	}

	if testData["test"] != "success" {
		t.Errorf("Expected test=success in data, got %v", testData["test"])
	}
}

func TestPgCDIAnsibleImple_CmdExecute_FailedAnsibleCommand_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Mock failed command execution
	testScript := createTestAnsibleScript(t, `
echo 'msg": "RESULT_TYPE:ERROR_V_1_0\nCommand execution failed"'
`)
	defer os.Remove(testScript)

	// Execute with mocked command
	ctx := context.Background()
	errMsg, data := impl.cmdExecuteWithScript(ctx, "localhost", "testuser", "/tmp/key", testScript, "test=args")

	// Verify
	if errMsg == nil {
		t.Fatal("CmdExecute should have returned error")
	}

	if errMsg.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected ErrorCode %d, got %d", int32(codes.Internal), errMsg.ErrorCode)
	}

	if errMsg.DetailCode != int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0) {
		t.Errorf("Expected DetailCode %d, got %d", int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0), errMsg.DetailCode)
	}

	if data != nil {
		t.Error("CmdExecute should not return data on error")
	}
}

func TestPgCDIAnsibleImple_CmdExecute_InvalidCommandExecution_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Execute with invalid command that will fail to execute
	ctx := context.Background()
	errMsg, data := impl.CmdExecute(ctx, "localhost", "testuser", "/tmp/key", "nonexistent-playbook.yaml", "test=args")

	// Verify
	if errMsg == nil {
		t.Fatal("CmdExecute should have returned error for invalid command")
	}

	if errMsg.ErrorCode != int32(codes.Internal) {
		t.Errorf("Expected ErrorCode %d, got %d", int32(codes.Internal), errMsg.ErrorCode)
	}

	if errMsg.DetailCode != int32(proto.DetailCode_CDI_ENVIRONMENT_ERROR) {
		t.Errorf("Expected DetailCode %d, got %d", int32(proto.DetailCode_CDI_ENVIRONMENT_ERROR), errMsg.DetailCode)
	}

	if data != nil {
		t.Error("CmdExecute should not return data on error")
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_SuccessWithJSON_ReturnsData(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	testCases := []struct {
		name     string
		output   string
		expected map[string]interface{}
	}{
		{
			name: "Valid JSON success",
			output: `Some ansible output
msg": "RESULT_TYPE:SUCCESS\n{\"data\": {\"test\": \"value\"}}`,
			expected: map[string]interface{}{
				"data": map[string]interface{}{
					"test": "value",
				},
			},
		},
		{
			name: "Complex JSON success",
			output: `Verbose ansible output
TASK [some task] ***
ok: [host]
msg": "RESULT_TYPE:SUCCESS\n{\"data\": {\"machines\": [{\"name\": \"test1\", \"status\": \"active\"}]}}`,
			expected: map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"name":   "test1",
							"status": "active",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Execute
			data, errMsg := impl.parseCDIWrapperOutput([]byte(tc.output))

			// Verify
			if errMsg != nil {
				t.Fatalf("parseCDIWrapperOutput should have succeeded, got error: %v", errMsg)
			}

			if data == nil {
				t.Fatal("parseCDIWrapperOutput should have returned data")
			}

			// Deep comparison of expected data structure
			verifyDataStructure(t, tc.expected, data)
		})
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_ErrorResponse_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	testCases := []struct {
		name           string
		output         string
		expectedDetail int32
	}{
		{
			name: "Error with message",
			output: `Some ansible output
msg": "RESULT_TYPE:ERROR_V_1_0\nCommand execution failed"`,
			expectedDetail: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
		},
		{
			name: "Error with message",
			output: `Some ansible output
msg": "RESULT_TYPE:ERROR_V_1_1\nCommand execution failed"`,
			expectedDetail: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_1),
		},
		{
			name: "Unknown result type",
			output: `Some ansible output
RESULT_TYPE:UNKNOWN
Unknown error occurred`,
			expectedDetail: int32(proto.DetailCode_CDI_RESPONSE_INVALID),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Execute
			data, errMsg := impl.parseCDIWrapperOutput([]byte(tc.output))

			// Verify
			if errMsg == nil {
				t.Fatal("parseCDIWrapperOutput should have returned error")
			}

			if data != nil {
				t.Error("parseCDIWrapperOutput should not return data on error")
			}

			if errMsg.ErrorCode != int32(codes.Internal) {
				t.Errorf("Expected ErrorCode %d, got %d", int32(codes.Internal), errMsg.ErrorCode)
			}

			if errMsg.DetailCode != tc.expectedDetail {
				t.Errorf("Expected DetailCode %d, got %d", tc.expectedDetail, errMsg.DetailCode)
			}
		})
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_InvalidOutput_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	testCases := []struct {
		name   string
		output string
	}{
		{
			name:   "No result type marker",
			output: `Some ansible output without result marker`,
		},
		{
			name:   "Empty output",
			output: ``,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Execute
			data, errMsg := impl.parseCDIWrapperOutput([]byte(tc.output))

			// Verify
			if errMsg == nil {
				t.Fatal("parseCDIWrapperOutput should have returned error for invalid output")
			}

			if data != nil {
				t.Error("parseCDIWrapperOutput should not return data on error")
			}

			if errMsg.ErrorCode != int32(codes.Internal) {
				t.Errorf("Expected ErrorCode %d, got %d", int32(codes.Internal), errMsg.ErrorCode)
			}

			if errMsg.DetailCode != int32(proto.DetailCode_CDI_RESPONSE_INVALID) {
				t.Errorf("Expected DetailCode %d, got %d", int32(proto.DetailCode_CDI_RESPONSE_INVALID), errMsg.DetailCode)
			}
		})
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_SuccessWithEmptyData_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	testCases := []struct {
		name        string
		output      string
		expectError bool
	}{
		{
			name: "Success but invalid JSON",
			output: `msg": "RESULT_TYPE:SUCCESS\ninvalid json content"`,
			expectError: true, // Invalid JSON should return error
		},
		{
			name: "Success but empty response",
			output: `RESULT_TYPE:SUCCESS
`,
			expectError: false, // Empty SUCCESS returns nil, nil
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Execute
			data, errMsg := impl.parseCDIWrapperOutput([]byte(tc.output))

			if tc.expectError {
				if errMsg == nil {
					t.Error("parseCDIWrapperOutput should have returned error for invalid JSON")
				}
			} else {
				if errMsg != nil {
					t.Errorf("parseCDIWrapperOutput should have returned nil error, got: %v", errMsg)
				}
			}

			if data != nil {
				t.Error("parseCDIWrapperOutput should return nil data")
			}
		})
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_SuccessWithAnsibleCleanup_ReturnsCleanData(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	output := `TASK [Execute CDI command] *****
ok: [host] => changed=false
msg": "RESULT_TYPE:SUCCESS\n{\"data\": {\"test\": \"value\"}}"
cdi: some trailing ansible output
more ansible output`

	// Execute
	data, errMsg := impl.parseCDIWrapperOutput([]byte(output))

	// Verify
	if errMsg != nil {
		t.Fatalf("parseCDIWrapperOutput should have succeeded, got error: %v", errMsg)
	}

	if data == nil {
		t.Fatal("parseCDIWrapperOutput should have returned data")
	}

	// Verify the data structure
	testData, ok := data["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to contain 'data' field with map")
	}

	if testData["test"] != "value" {
		t.Errorf("Expected test=value in data, got %v", testData["test"])
	}
}

// Test for CmdExecute with valid configuration but command execution error
func TestPgCDIAnsibleImple_CmdExecute_ValidConfig_CommandExecutionHandled(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup valid configuration
	setupValidTestEnv()
	defer teardownTestEnv()

	utils.ResetConfigForTesting()
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Execute with parameters that will result in command not found (expected in test environment)
	errMsg, jsonData := impl.CmdExecute(context.Background(), "test-host", "test-user", "/tmp/key", "test.yaml", "test-args")

	// In test environment, ansible-playbook command is not available, so we expect an error
	if errMsg == nil {
		t.Error("Expected error due to ansible-playbook not found in PATH")
	}

	if jsonData != nil {
		t.Error("Expected nil jsonData on command execution error")
	}

	// The error should be related to command execution
	if errMsg != nil {
		t.Logf("Got expected error: %s", errMsg.Message)
	}
}

func TestPgCDIAnsibleImple_CmdExecute_CommandFailure_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupValidTestEnv()
	defer teardownTestEnv()
	utils.ResetConfigForTesting()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Execute with non-existent playbook to force command failure
	errMsg, jsonData := impl.CmdExecute(context.Background(), "test-host", "test-user", "/tmp/key", "non-existent.yaml", "test-args")

	// Verify error is returned
	if errMsg == nil {
		t.Error("Expected error due to command failure")
	}

	if jsonData != nil {
		t.Error("Expected nil jsonData on command failure")
	}
}

func TestPgCDIAnsibleImple_CmdExecute_CancelledContext_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupValidTestEnv()
	defer teardownTestEnv()
	utils.ResetConfigForTesting()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute with cancelled context
	errMsg, jsonData := impl.CmdExecute(ctx, "test-host", "test-user", "/tmp/key", "test.yaml", "test-args")

	// Verify error is returned for cancelled context
	if errMsg == nil {
		t.Error("Expected error due to cancelled context")
	}

	if jsonData != nil {
		t.Error("Expected nil jsonData on cancelled context")
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_UnknownResultType_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Test output with UNKNOWN result type
	output := `Some initial output
RESULT_TYPE:UNKNOWN
Unknown error occurred`

	// Execute
	jsonData, errMsg := impl.parseCDIWrapperOutput([]byte(output))

	// Verify
	if errMsg == nil {
		t.Error("Expected error for UNKNOWN result type")
	}

	if jsonData != nil {
		t.Error("Expected nil jsonData for UNKNOWN result type")
	}

	if errMsg.GetDetailCode() != int32(proto.DetailCode_CDI_RESPONSE_INVALID) {
		t.Errorf("Expected detail code proto.DetailCode_CDI_RESPONSE_INVALID, got %d", errMsg.GetDetailCode())
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_EmptySuccessData_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Test output with SUCCESS but no data
	output := `Some initial output
RESULT_TYPE:SUCCESS
`

	// Execute
	jsonData, errMsg := impl.parseCDIWrapperOutput([]byte(output))

	// Verify - according to diff, empty SUCCESS returns nil, nil
	if errMsg != nil {
		t.Errorf("Expected nil error for empty success data, got: %v", errMsg)
	}

	if jsonData != nil {
		t.Error("Expected nil jsonData for empty success data")
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_InvalidJson_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Test output with SUCCESS but invalid JSON
	output := `Some initial output
msg": "RESULT_TYPE:SUCCESS\n{invalid json format}"
`

	// Execute
	jsonData, errMsg := impl.parseCDIWrapperOutput([]byte(output))

	// Verify - invalid JSON should return error
	if errMsg == nil {
		t.Error("Expected error for invalid JSON")
	}

	if jsonData != nil {
		t.Error("Expected nil jsonData for invalid JSON")
	}

	if errMsg.GetDetailCode() != int32(proto.DetailCode_CDI_RESPONSE_INVALID) {
		t.Errorf("Expected detail code proto.DetailCode_CDI_RESPONSE_INVALID, got %d", errMsg.GetDetailCode())
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_NoResultTypeMarker_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Test output without any result type marker
	output := `Some output without result type
More output lines
No markers here`

	// Execute
	jsonData, errMsg := impl.parseCDIWrapperOutput([]byte(output))

	// Verify
	if errMsg == nil {
		t.Error("Expected error for no result type marker")
	}

	if jsonData != nil {
		t.Error("Expected nil jsonData for no result type marker")
	}

	if errMsg.GetDetailCode() != int32(proto.DetailCode_CDI_RESPONSE_INVALID) {
		t.Errorf("Expected detail code proto.DetailCode_CDI_RESPONSE_INVALID, got %d", errMsg.GetDetailCode())
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_JsonWithAnsibleCleanup_ReturnsCleanData(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Test output with JSON followed by ansible cleanup output
	output := `Some initial output
msg": "RESULT_TYPE:SUCCESS\n{\"test_key\": \"test_value\", \"status\": \"active\"}"
cdi: cleanup output that should be removed`

	// Execute
	jsonData, errMsg := impl.parseCDIWrapperOutput([]byte(output))

	// Verify
	if errMsg != nil {
		t.Errorf("Unexpected error: %v", errMsg)
	}

	if jsonData == nil {
		t.Fatal("Expected jsonData to be returned")
	}

	if jsonData["test_key"] != "test_value" {
		t.Errorf("Expected test_key to be 'test_value', got %v", jsonData["test_key"])
	}

	if jsonData["status"] != "active" {
		t.Errorf("Expected status to be 'active', got %v", jsonData["status"])
	}
}

// Test actual CmdExecute method to improve coverage
func TestPgCDIAnsibleImple_CmdExecute_ActualImplementation_CommandNotFound(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup environment
	setupValidTestEnv()
	defer teardownTestEnv()

	utils.ResetConfigForTesting()
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Execute actual CmdExecute - this will fail because ansible-playbook is not available in test environment
	// but it will cover the actual implementation code paths
	ctx := context.Background()
	errMsg, data := impl.CmdExecute(ctx, "localhost", "testuser", "/tmp/testkey", "test.yaml", "test=value")

	// Verify error handling
	if errMsg == nil {
		t.Error("Expected error due to ansible-playbook command not found")
	} else {
		// Should get proto.DetailCode_CDI_ENVIRONMENT_ERROR due to command execution failure
		if errMsg.GetDetailCode() != int32(proto.DetailCode_CDI_ENVIRONMENT_ERROR) {
			t.Errorf("Expected DetailCode %d (proto.DetailCode_CDI_ENVIRONMENT_ERROR), got %d",
				int32(proto.DetailCode_CDI_ENVIRONMENT_ERROR), errMsg.GetDetailCode())
		}

		if errMsg.GetErrorCode() != int32(codes.Internal) {
			t.Errorf("Expected ErrorCode %d (Internal), got %d",
				int32(codes.Internal), errMsg.GetErrorCode())
		}
	}

	if data != nil {
		t.Error("Expected nil data on command execution error")
	}
}

func TestPgCDIAnsibleImple_CmdExecute_ActualImplementation_CancelledContext(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup environment
	setupValidTestEnv()
	defer teardownTestEnv()

	utils.ResetConfigForTesting()
	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute with cancelled context
	errMsg, data := impl.CmdExecute(ctx, "localhost", "testuser", "/tmp/testkey", "test.yaml", "test=value")

	// Verify error handling
	if errMsg == nil {
		t.Error("Expected error due to cancelled context")
	}

	if data != nil {
		t.Error("Expected nil data on cancelled context")
	}
}

func TestPgCDIAnsibleImple_parseCDIWrapperOutput_Coverage_AllBranches(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Test coverage for various parseCDIWrapperOutput scenarios
	testCases := []struct {
		name           string
		output         string
		expectError    bool
		expectData     bool
		expectedDetail int32
	}{
		{
			name: "Valid SUCCESS with JSON",
			output: `Some ansible output
msg": "RESULT_TYPE:SUCCESS\n{\"data\": \"test\"}"`,
			expectError: false,
			expectData:  true,
		},
		{
			name: "Valid ERROR response",
			output: `Some ansible output
msg": "RESULT_TYPE:ERROR_V_1_0\nCommand execution failed"`,
			expectError:    true,
			expectData:     false,
			expectedDetail: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
		},
		{
			name: "Valid ERROR response",
			output: `Some ansible output
msg": "RESULT_TYPE:ERROR_V_1_1\nCommand execution failed"`,
			expectError:    true,
			expectData:     false,
			expectedDetail: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_1),
		},
		{
			name: "UNKNOWN result type",
			output: `Some ansible output
RESULT_TYPE:UNKNOWN
Unknown error occurred`,
			expectError:    true,
			expectData:     false,
			expectedDetail: int32(proto.DetailCode_CDI_RESPONSE_INVALID),
		},
		{
			name: "No result type marker",
			output: `Some ansible output
No markers here`,
			expectError:    true,
			expectData:     false,
			expectedDetail: int32(proto.DetailCode_CDI_RESPONSE_INVALID),
		},
		{
			name: "SUCCESS with invalid JSON",
			output: `Some ansible output
msg": "RESULT_TYPE:SUCCESS\n{invalid json"`,
			expectError: true, // Invalid JSON should return error
			expectData:  false,
			expectedDetail: int32(proto.DetailCode_CDI_RESPONSE_INVALID),
		},
		{
			name: "SUCCESS with empty data",
			output: `Some ansible output
RESULT_TYPE:SUCCESS
`,
			expectError: false, // According to diff, empty SUCCESS returns nil, nil (not error)
			expectData:  false,
		},
		{
			name: "SUCCESS with JSON and ansible cleanup",
			output: `Some ansible output
msg": "RESULT_TYPE:SUCCESS\n{\"data\": \"test\"}"
cdi: cleanup output`,
			expectError: false,
			expectData:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			data, errMsg := impl.parseCDIWrapperOutput([]byte(tc.output))

			if tc.expectError {
				if errMsg == nil {
					t.Errorf("Expected error for case: %s", tc.name)
				} else if tc.expectedDetail != 0 && errMsg.GetDetailCode() != tc.expectedDetail {
					t.Errorf("Expected detail code %d, got %d for case: %s",
						tc.expectedDetail, errMsg.GetDetailCode(), tc.name)
				}
				if data != nil {
					t.Errorf("Expected nil data on error for case: %s", tc.name)
				}
			} else {
				if errMsg != nil {
					t.Errorf("Unexpected error for case: %s - %v", tc.name, errMsg)
				}
				if tc.expectData && data == nil {
					t.Errorf("Expected data for successful case: %s", tc.name)
				}
				if !tc.expectData && data != nil {
					t.Errorf("Expected nil data for case: %s", tc.name)
				}
			}
		})
	}
}

// Test to cover successful command execution path
func TestPgCDIAnsibleImple_CmdExecute_MockedSuccessfulCommand_ReturnsData(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// This test uses a workaround to test the success path without actually running ansible-playbook
	// We'll create a minimal executable script that simulates ansible-playbook behavior

	// Setup environment
	setupValidTestEnv()
	defer teardownTestEnv()

	// Create a temporary directory for our mock ansible-playbook
	tmpDir, err := os.MkdirTemp("", "mock-ansible-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock ansible-playbook executable
	mockAnsiblePlaybook := tmpDir + "/ansible-playbook"
	// Simplified output that matches the expected format with escaped JSON
	err = os.WriteFile(mockAnsiblePlaybook, []byte(`#!/bin/bash
echo '"msg": "RESULT_TYPE:SUCCESS\n{\"data\": {\"machine_id\": \"test123\", \"status\": \"active\"}}"'
`), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock ansible-playbook: %v", err)
	}

	// Temporarily modify PATH to include our mock directory
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	utils.ResetConfigForTesting()
	err = utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Execute with our mocked ansible-playbook
	ctx := context.Background()
	errMsg, data := impl.CmdExecute(ctx, "localhost", "testuser", "/tmp/testkey", "test.yaml", "test=value")

	// Verify success path
	if errMsg != nil {
		t.Errorf("Expected successful execution, got error: %v", errMsg)
	}

	if data == nil {
		t.Error("Expected data to be returned")
	} else {
		// Verify the JSON data structure
		if dataField, ok := data["data"].(map[string]interface{}); ok {
			if dataField["machine_id"] != "test123" {
				t.Errorf("Expected machine_id=test123, got %v", dataField["machine_id"])
			}
			if dataField["status"] != "active" {
				t.Errorf("Expected status=active, got %v", dataField["status"])
			}
		} else {
			t.Error("Expected data field to contain proper structure")
		}
	}
}

func TestPgCDIAnsibleImple_CmdExecute_MockedSuccessfulCommand_WithCleanup_ReturnsCleanData(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test successful command with ansible cleanup output

	// Setup environment
	setupValidTestEnv()
	defer teardownTestEnv()

	// Create a temporary directory for our mock ansible-playbook
	tmpDir, err := os.MkdirTemp("", "mock-ansible-cleanup-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock ansible-playbook executable with cleanup output
	mockAnsiblePlaybook := tmpDir + "/ansible-playbook"
	// Simplified output that matches the expected format with escaped JSON
	err = os.WriteFile(mockAnsiblePlaybook, []byte(`#!/bin/bash
echo '"msg": "RESULT_TYPE:SUCCESS\n{\"resources\": [{\"id\": \"res1\", \"type\": \"cpu\"}]}"'
echo "cdi: cleanup output that should be ignored"
`), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock ansible-playbook: %v", err)
	}

	// Temporarily modify PATH to include our mock directory
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	utils.ResetConfigForTesting()
	err = utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	impl := &PgCDIAnsibleImple{Logger: klog.Background()}

	// Execute with our mocked ansible-playbook
	ctx := context.Background()
	errMsg, data := impl.CmdExecute(ctx, "localhost", "testuser", "/tmp/testkey", "test.yaml", "test=value")

	// Verify success path
	if errMsg != nil {
		t.Errorf("Expected successful execution, got error: %v", errMsg)
	}

	if data == nil {
		t.Error("Expected data to be returned")
	} else {
		// Verify the JSON data structure (should be cleaned from ansible output)
		if resources, ok := data["resources"].([]interface{}); ok {
			if len(resources) != 1 {
				t.Errorf("Expected 1 resource, got %d", len(resources))
			} else {
				if resource, ok := resources[0].(map[string]interface{}); ok {
					if resource["id"] != "res1" {
						t.Errorf("Expected id=res1, got %v", resource["id"])
					}
					if resource["type"] != "cpu" {
						t.Errorf("Expected type=cpu, got %v", resource["type"])
					}
				} else {
					t.Error("Expected resource to be proper structure")
				}
			}
		} else {
			t.Error("Expected resources field to be array")
		}
	}
}

// Helper functions for testing

// cmdExecuteWithScript is a helper method for testing that uses a custom script instead of ansible-playbook
func (l PgCDIAnsibleImple) cmdExecuteWithScript(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, script string, extrArgs string) (errMsg *common.ErrorMessage, jsonData map[string]interface{}) {
	defer func() {
		l.Logger.V(2).Info("end cmdExecuteWithScript",
			"remote_host", remoteHost,
			"errMsg", errMsg,
			"jsonData", jsonData)
	}()
	l.Logger.V(2).Info("start cmdExecuteWithScript",
		"remote_host", remoteHost,
		"script", script)

	// Execute the test script directly
	cmd := exec.CommandContext(ctx, "bash", script)
	l.Logger.V(2).Info("branch: test script command generated",
		"remote_host", remoteHost,
		"cmd", "bash "+script)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		l.Logger.V(2).Info("branch: test script command execution failed",
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

	l.Logger.V(2).Info("branch: test script command execution successful",
		"remote_host", remoteHost)
	// Parse output
	data, errorMessage := l.parseCDIWrapperOutput(output)
	if errorMessage != nil {
		l.Logger.V(2).Info("branch: test script output parsing failed",
			"remote_host", remoteHost)
		errMsg = errorMessage
		return
	}

	l.Logger.V(2).Info("branch: test script output parsing successful",
		"remote_host", remoteHost)
	jsonData = data
	return
}

func createTestAnsibleScript(t *testing.T, content string) string {
	// Create a temporary script file
	file, err := os.CreateTemp("", "test-ansible-*.sh")
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	_, err = file.WriteString("#!/bin/bash\n" + content)
	if err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}

	err = file.Close()
	if err != nil {
		t.Fatalf("Failed to close test script: %v", err)
	}

	// Make it executable
	err = os.Chmod(file.Name(), 0755)
	if err != nil {
		t.Fatalf("Failed to make test script executable: %v", err)
	}

	return file.Name()
}

func verifyDataStructure(t *testing.T, expected, actual map[string]interface{}) {
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			t.Errorf("Expected key '%s' not found in actual data", key)
			continue
		}

		switch expectedVal := expectedValue.(type) {
		case map[string]interface{}:
			actualVal, ok := actualValue.(map[string]interface{})
			if !ok {
				t.Errorf("Expected key '%s' to be map[string]interface{}, got %T", key, actualValue)
				continue
			}
			verifyDataStructure(t, expectedVal, actualVal)
		case []interface{}:
			actualVal, ok := actualValue.([]interface{})
			if !ok {
				t.Errorf("Expected key '%s' to be []interface{}, got %T", key, actualValue)
				continue
			}
			if len(expectedVal) != len(actualVal) {
				t.Errorf("Expected array length %d for key '%s', got %d", len(expectedVal), key, len(actualVal))
				continue
			}
			for i, expectedItem := range expectedVal {
				if expectedItemMap, ok := expectedItem.(map[string]interface{}); ok {
					if actualItemMap, ok := actualVal[i].(map[string]interface{}); ok {
						verifyDataStructure(t, expectedItemMap, actualItemMap)
					} else {
						t.Errorf("Expected array item %d to be map[string]interface{} for key '%s', got %T", i, key, actualVal[i])
					}
				} else {
					if expectedItem != actualVal[i] {
						t.Errorf("Expected array item %d to be %v for key '%s', got %v", i, expectedItem, key, actualVal[i])
					}
				}
			}
		default:
			if expectedValue != actualValue {
				t.Errorf("Expected key '%s' to be %v, got %v", key, expectedValue, actualValue)
			}
		}
	}
}

func setupValidTestEnv() {
	os.Setenv("CDI_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test_key")
	os.Setenv("TLS_ENABLE", "true")
	os.Setenv("TLS_CERT_PATH", "/tmp/certs")
}

func teardownTestEnv() {
	os.Unsetenv("CDI_SERVER_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SSH_KEY")
	os.Unsetenv("TLS_ENABLE")
	os.Unsetenv("TLS_CERT_PATH")
}
