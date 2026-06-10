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

package interfaces

import (
	"context"
	"testing"

    common "common/api/proto"    // import of common protobuf
)

// Test interface contract with mock implementation
type mockCDIAnsible struct {
	executeFunc func(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (*common.ErrorMessage, map[string]interface{})
}

func (m *mockCDIAnsible) CmdExecute(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs)
	}
	return nil, map[string]interface{}{"test": "data"}
}

func TestCDIAnsible_MockImplementation_WorksCorrectly(t *testing.T) {
	mock := &mockCDIAnsible{
		executeFunc: func(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return nil, map[string]interface{}{
				"mock_result": "success",
				"host":        remoteHost,
			}
		},
	}

	// Verify it implements the interface
	var _ CDIAnsible = mock

	// Test the mock
	errMsg, data := mock.CmdExecute(context.Background(), "test-host", "test-user", "/tmp/key", "test.yaml", "test-args")

	if errMsg != nil {
		t.Errorf("Expected no error, got %v", errMsg)
	}

	if data == nil {
		t.Fatal("Expected data to be returned")
	}

	if data["mock_result"] != "success" {
		t.Errorf("Expected mock_result to be 'success', got %v", data["mock_result"])
	}

	if data["host"] != "test-host" {
		t.Errorf("Expected host to be 'test-host', got %v", data["host"])
	}
}

func TestCDIAnsible_InterfaceMethodSignature_IsCorrect(t *testing.T) {
	// Test that the interface method signature is as expected
	var ansible CDIAnsible = &mockCDIAnsible{}

	// Test CmdExecute method signature by calling it
	errMsg, data := ansible.CmdExecute(context.Background(), "test-host", "test-user", "/tmp/key", "test.yaml", "test-args")

	// Verify the method returns the expected types
	if errMsg != nil {
		// errMsg should be *common.ErrorMessage or nil
		if errMsg.GetMessage() == "" && errMsg.GetErrorCode() == 0 {
			t.Log("ErrorMessage structure is correct")
		}
	}

	if data != nil {
		// data should be map[string]interface{}
		if _, ok := data["test"]; ok {
			t.Log("Data map structure is correct")
		}
	}
}
