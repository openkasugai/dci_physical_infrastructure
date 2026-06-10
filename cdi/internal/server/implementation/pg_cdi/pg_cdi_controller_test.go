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
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"

	proto "cdi_module/api/proto"
    common "common/api/proto"    // import of common protobuf
	"cdi_module/internal/server/interfaces"
	"cdi_module/internal/server/test_utils"
	"common/models/extra_parameters"
)

// MockPgCDIAnsible is a mock implementation of PgCDIAnsible interface for testing
type MockPgCDIAnsible struct {
	CmdExecuteFunc func(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (*common.ErrorMessage, map[string]interface{})
	CallHistory    []MockCall
}

type MockCall struct {
	RemoteHost        string
	RemoteUser        string
	SshPrivateKeyFile string
	Playbook          string
	ExtraArgs         string
}

// Helper function to create string pointer
func ptrString(s string) *string {
	return &s
}

func (m *MockPgCDIAnsible) CmdExecute(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
	// Record the call
	m.CallHistory = append(m.CallHistory, MockCall{
		RemoteHost:        remoteHost,
		RemoteUser:        remotUser,
		SshPrivateKeyFile: sshPrivateKeyFile,
		Playbook:          playbook,
		ExtraArgs:         extrArgs,
	})

	if m.CmdExecuteFunc != nil {
		return m.CmdExecuteFunc(ctx, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs)
	}

	// Default success response
	return nil, map[string]interface{}{"data": "success"}
}

func TestPgCDIController_ImplementsInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: &MockPgCDIAnsible{},
		SSHKey:  "/tmp/test_key",
	}

	// Verify it implements the interface
	var _ interfaces.CDIController = controller
}

func TestPgCDIController_MachineCreate_ValidRequest_ReturnsAccept(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			if strings.Contains(playbook, "machine_show.yaml") {
				// Return error (machine not found) so creation proceeds
				return &common.ErrorMessage{
					ErrorCode:  int32(codes.NotFound),
					DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
					Message:    "machine not found",
				}, nil
			}
			// Return success for machine_create.yaml
			return nil, map[string]interface{}{"data": "success"}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.MachineCreateRequest{
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:    "test-group",
		MachineName:  "test-machine",
		ResourceList: []string{"resource1", "resource2"},
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	}

	// Execute
	reply, err := controller.MachineCreate(context.Background(), request)

	// Verify
	if err != nil {
		t.Fatalf("MachineCreate failed: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", reply.GetResult())
	}

	if reply.GetErrorMessage() != "" {
		t.Errorf("Expected empty error message, got %s", reply.GetErrorMessage())
	}

	// Verify ansible was called for both machine_show and machine_create
	if len(mockAnsible.CallHistory) != 2 {
		t.Fatalf("Expected 2 ansible calls (MachineShow + MachineCreate), got %d", len(mockAnsible.CallHistory))
	}

	// First call should be MachineShow
	showCall := mockAnsible.CallHistory[0]
	if showCall.RemoteHost != "test-host" {
		t.Errorf("Expected remote host 'test-host', got %s", showCall.RemoteHost)
	}
	if !strings.Contains(showCall.Playbook, "machine_show.yaml") {
		t.Errorf("Expected playbook to contain 'machine_show.yaml', got %s", showCall.Playbook)
	}

	// Second call should be MachineCreate
	createCall := mockAnsible.CallHistory[1]
	if createCall.RemoteHost != "test-host" {
		t.Errorf("Expected remote host 'test-host', got %s", createCall.RemoteHost)
	}
	if !strings.Contains(createCall.Playbook, "machine_create.yaml") {
		t.Errorf("Expected playbook to contain 'machine_create.yaml', got %s", createCall.Playbook)
	}

	var showArgs map[string]string
	if err := json.Unmarshal([]byte(showCall.ExtraArgs), &showArgs); err != nil {
		t.Fatalf("Extra args should be valid JSON, got '%s': %v", showCall.ExtraArgs, err)
	}
	expectedShowArgs := map[string]string{
		"cdi_user":     "cdi-user",
		"cdi_password": "cdi-pass",
		"cdi_guest":    "test-guest",
		"group_name":   "test-group",
		"machine_name": "test-machine",
	}
	for k, v := range expectedShowArgs {
		if showArgs[k] != v {
			t.Errorf("Expected extra arg %s=%s, got %s", k, v, showArgs[k])
		}
	}
}

func TestPgCDIController_MachineCreate_MachineAlreadyExists_ReturnsAccept(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - mock that machine already exists
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			if strings.Contains(playbook, "machine_show.yaml") {
				// Return existing machine data
				machineData := map[string]interface{}{
					"name":   "test-machine",
					"status": "active",
				}
				data := map[string]interface{}{
					"data": map[string]interface{}{
						"machines": []interface{}{machineData},
					},
				}
				return nil, data
			}
			// Should not reach machine_create.yaml since machine exists
			t.Error("Should not call machine_create.yaml when machine already exists")
			return nil, nil
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.MachineCreateRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:    "test-group",
		MachineName:  "test-machine",
		ResourceList: []string{"resource1"},
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	}

	// Execute
	reply, err := controller.MachineCreate(context.Background(), request)

	// Verify
	if err != nil {
		t.Fatalf("MachineCreate failed: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT for existing machine, got %v", reply.GetResult())
	}

	// Should only call MachineShow, not MachineCreate
	if len(mockAnsible.CallHistory) != 1 {
		t.Fatalf("Expected 1 ansible call (MachineShow only), got %d", len(mockAnsible.CallHistory))
	}
}

func TestPgCDIController_MachineDestroy_ValidRequest_ReturnsAccept(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	callCount := 0
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			callCount++
			if strings.Contains(playbook, "machine_power.yaml") {
				return nil, map[string]interface{}{"data": "success"}
			}
			if strings.Contains(playbook, "machine_show.yaml") {
				// Return INACTIVE POFF status to satisfy polling condition
				machineData := map[string]interface{}{
					"name":               "test-machine",
					"mach_status_detail": "INACTIVE POFF",
				}
				data := map[string]interface{}{
					"data": map[string]interface{}{
						"machines": []interface{}{machineData},
					},
				}
				return nil, data
			}
			if strings.Contains(playbook, "machine_destroy.yaml") {
				return nil, map[string]interface{}{"data": "success"}
			}
			return nil, map[string]interface{}{"data": "success"}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.MachineDestroyRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:   "test-group",
		MachineName: "test-machine",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	}

	// Execute
	reply, err := controller.MachineDestroy(context.Background(), request)

	// Verify
	if err != nil {
		t.Fatalf("MachineDestroy failed: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", reply.GetResult())
	}

	// Should call power off, status checks, and destroy
	if callCount < 3 {
		t.Errorf("Expected at least 3 calls (power off + status check + destroy), got %d", callCount)
	}
}

func TestPgCDIController_MachineShow_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	expectedMachine := map[string]interface{}{
		"name":   "test-machine",
		"status": "active",
		"group":  "test-group",
	}

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{expectedMachine},
				},
			}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.MachineShowRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:   "test-group",
		MachineName: "test-machine",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	}

	// Execute
	reply, err := controller.MachineShow(context.Background(), request)

	// Verify
	if err != nil {
		t.Fatalf("MachineShow failed: %v", err)
	}

	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected result SUCCESS, got %v", reply.GetResult())
	}

	// Verify JSON data
	if reply.GetData() == "" {
		t.Fatal("Expected data in response, got empty string")
	}

	var returnedMachine map[string]interface{}
	err = json.Unmarshal([]byte(reply.GetData()), &returnedMachine)
	if err != nil {
		t.Fatalf("Failed to unmarshal returned data: %v", err)
	}

	if returnedMachine["name"] != expectedMachine["name"] {
		t.Errorf("Expected name %v, got %v", expectedMachine["name"], returnedMachine["name"])
	}
}

func TestPgCDIController_MachineShow_InvalidAnsibleResponse_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - return data without expected structure
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return nil, map[string]interface{}{
				"invalid": "structure",
			}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.MachineShowRequest{
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:   "test-group",
		MachineName: "test-machine",
	}

	// Execute
	reply, err := controller.MachineShow(context.Background(), request)

	// Verify
	if err == nil {
		t.Fatal("Expected error for invalid ansible response")
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}
}

func TestPgCDIController_ResourceList_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	expectedData := map[string]interface{}{
		"resources": []interface{}{
			map[string]interface{}{"name": "resource1", "type": "cpu"},
			map[string]interface{}{"name": "resource2", "type": "memory"},
		},
	}

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": expectedData,
			}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.ResourceListRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName: "test-group",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	}

	// Execute
	reply, err := controller.ResourceList(context.Background(), request)

	// Verify
	if err != nil {
		t.Fatalf("ResourceList failed: %v", err)
	}

	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected result SUCCESS, got %v", reply.GetResult())
	}

	if reply.GetData() == "" {
		t.Fatal("Expected data in response, got empty string")
	}

	var returnedData map[string]interface{}
	err = json.Unmarshal([]byte(reply.GetData()), &returnedData)
	if err != nil {
		t.Fatalf("Failed to unmarshal returned data: %v", err)
	}

	if returnedData["resources"] == nil {
		t.Error("Expected resources field in returned data")
	}
}

func TestPgCDIController_ResourceShow_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	expectedResource := map[string]interface{}{
		"name": "test-resource",
		"type": "cpu",
		"spec": map[string]interface{}{
			"cores": 8,
			"speed": "2.4GHz",
		},
	}

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return nil, expectedResource
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.ResourceShowRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		ResourceName: "test-resource",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	}

	// Execute
	reply, err := controller.ResourceShow(context.Background(), request)

	// Verify
	if err != nil {
		t.Fatalf("ResourceShow failed: %v", err)
	}

	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected result SUCCESS, got %v", reply.GetResult())
	}

	if reply.GetData() == "" {
		t.Fatal("Expected data in response, got empty string")
	}

	var returnedResource map[string]interface{}
	err = json.Unmarshal([]byte(reply.GetData()), &returnedResource)
	if err != nil {
		t.Fatalf("Failed to unmarshal returned data: %v", err)
	}

	if returnedResource["name"] != expectedResource["name"] {
		t.Errorf("Expected name %v, got %v", expectedResource["name"], returnedResource["name"])
	}
}

func TestPgCDIController_PowerOffMachine_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return nil, map[string]interface{}{"success": true}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	cdiInfo := &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}

	extraParams := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "cdi-user",
		CDIPassword: "cdi-pass",
		CDIGuest:    "test-guest",
	}

	// Execute
	errMsg := controller.powerOffMachine(context.Background(), cdiInfo, "test-machine", "test-group", extraParams)

	// Verify
	if errMsg != nil {
		t.Fatalf("powerOffMachine failed: %v", errMsg)
	}

	// Verify correct playbook was called
	if len(mockAnsible.CallHistory) != 1 {
		t.Fatalf("Expected 1 ansible call, got %d", len(mockAnsible.CallHistory))
	}

	call := mockAnsible.CallHistory[0]
	if call.Playbook != "machine_power.yaml" {
		t.Errorf("Expected playbook 'machine_power.yaml', got %s", call.Playbook)
	}

	var powerArgs map[string]string
	if err := json.Unmarshal([]byte(call.ExtraArgs), &powerArgs); err != nil {
		t.Fatalf("Extra args should be valid JSON, got '%s': %v", call.ExtraArgs, err)
	}
	expectedPowerArgs := map[string]string{
		"cdi_user":     "cdi-user",
		"cdi_password": "cdi-pass",
		"cdi_guest":    "test-guest",
		"machine_name": "test-machine",
		"power":        "off",
		"group_name":   "test-group",
	}
	for k, v := range expectedPowerArgs {
		if powerArgs[k] != v {
			t.Errorf("Expected extra arg %s=%s, got %s", k, v, powerArgs[k])
		}
	}
}

func TestPgCDIController_GetMachineStatus_ValidRequest_ReturnsStatus(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			machineData := map[string]interface{}{
				"name":               "test-machine",
				"mach_status_detail": "INACTIVE POFF",
			}
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{machineData},
				},
			}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	cdiInfo := &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}

	productInfo := &proto.ProductInformation{}
	extraParamStr := `{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`

	// Execute
	status, err := controller.getMachineStatus(context.Background(), productInfo, cdiInfo, "test-machine", "test-group", extraParamStr)

	// Verify
	if err != nil {
		t.Fatalf("getMachineStatus failed: %v", err)
	}

	if status != "INACTIVE POFF" {
		t.Errorf("Expected status 'INACTIVE POFF', got '%s'", status)
	}
}

func TestPgCDIController_GetMachineStatus_MachineShowError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.NotFound),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_1),
				Message:    "machine not found",
			}, nil
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	cdiInfo := &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}

	productInfo := &proto.ProductInformation{}
	extraParamStr := `{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`

	// Execute
	status, err := controller.getMachineStatus(context.Background(), productInfo, cdiInfo, "test-machine", "test-group", extraParamStr)

	// Verify
	if err == nil {
		t.Fatal("Expected error for machine show failure")
	}

	if status != "" {
		t.Errorf("Expected empty status on error, got '%s'", status)
	}
}

func TestPgCDIController_PollMachineStatus_TargetReached_ReturnsSuccessfully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup
	callCount := 0
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			callCount++
			var status string
			if callCount >= 2 {
				status = "INACTIVE POFF" // Target status on second call
			} else {
				status = "ACTIVE" // Initial status
			}

			machineData := map[string]interface{}{
				"name":               "test-machine",
				"mach_status_detail": status,
			}
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{machineData},
				},
			}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	cdiInfo := &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}

	productInfo := &proto.ProductInformation{}
	extraParamStr := `{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`

	// Execute with short timeout for testing
	controller.pollMachineStatus(context.Background(), productInfo, cdiInfo, "test-machine", "test-group", extraParamStr, []string{"INACTIVE POFF"}, 10*time.Millisecond, 1*time.Second)

	// Verify - Should have been called at least twice (initial status check + target reached)
	if callCount < 2 {
		t.Errorf("Expected at least 2 calls to check status, got %d", callCount)
	}
}

func TestPgCDIController_MachineCreate_MachineNotExists_CreatesNewMachine(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup - mock machine does not exist and should be created
	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			if strings.Contains(playbook, "machine_show.yaml") {
				// Return nil data to force extraction error (will cause err != nil)
				return nil, map[string]interface{}{
					"wrong": "structure",
				}
			}
			// Return success for machine_create.yaml
			return nil, map[string]interface{}{"data": "success"}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.MachineCreateRequest{
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:    "test-group",
		MachineName:  "test-machine",
		ResourceList: []string{"resource1", "resource2"},
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	}

	// Execute
	reply, err := controller.MachineCreate(context.Background(), request)

	// Verify
	if err != nil {
		t.Fatalf("MachineCreate failed: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", reply.GetResult())
	}

	if reply.GetErrorMessage() != "" {
		t.Errorf("Expected empty error message, got %s", reply.GetErrorMessage())
	}

	// Verify ansible was called twice (MachineShow + MachineCreate)
	if len(mockAnsible.CallHistory) != 2 {
		t.Fatalf("Expected 2 ansible calls (MachineShow + MachineCreate), got %d", len(mockAnsible.CallHistory))
	}

	// First call should be MachineShow
	showCall := mockAnsible.CallHistory[0]
	if !strings.Contains(showCall.Playbook, "machine_show.yaml") {
		t.Errorf("Expected first playbook to contain 'machine_show.yaml', got %s", showCall.Playbook)
	}

	// Second call should be MachineCreate
	createCall := mockAnsible.CallHistory[1]
	if !strings.Contains(createCall.Playbook, "machine_create.yaml") {
		t.Errorf("Expected second playbook to contain 'machine_create.yaml', got %s", createCall.Playbook)
	}

	var createArgs map[string]string
	if err := json.Unmarshal([]byte(createCall.ExtraArgs), &createArgs); err != nil {
		t.Fatalf("Extra args should be valid JSON, got '%s': %v", createCall.ExtraArgs, err)
	}
	expectedCreateArgs := map[string]string{
		"cdi_user":      "cdi-user",
		"cdi_password":  "cdi-pass",
		"cdi_guest":     "test-guest",
		"group_name":    "test-group",
		"machine_name":  "test-machine",
		"resource_enum": "resource1,resource2",
	}
	for k, v := range expectedCreateArgs {
		if createArgs[k] != v {
			t.Errorf("Expected extra arg %s=%s, got %s", k, v, createArgs[k])
		}
	}
}

// Test for error scenarios in controller methods
func TestPgCDIController_MachineCreate_AnsibleError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return error for machine_create
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			// Return error for machine_show to simulate machine not existing
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "machine not found",
			}, nil
		}
		if strings.Contains(playbook, "machine_create") {
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_1),
				Message:    "ansible command failed",
			}, nil
		}
		// Default case
		return nil, map[string]interface{}{"data": "success"}
	}

	// Execute
	reply, err := controller.MachineCreate(context.Background(), &proto.MachineCreateRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:    "test-group",
		MachineName:  "test-machine",
		ResourceList: []string{"resource1"},
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	})

	// Verify
	if err != nil {
		t.Errorf("MachineCreate should not return go error, got: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}

	if reply.GetErrorMessage() == "" {
		t.Error("Expected error message to be set")
	}
}

func TestPgCDIController_MachineDestroy_PowerOffError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return error for machine_power
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_power") {
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "poweroff command failed",
			}, nil
		}
		return nil, map[string]interface{}{"status": "success"}
	}

	// Execute
	reply, err := controller.MachineDestroy(context.Background(), &proto.MachineDestroyRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:   "test-group",
		MachineName: "test-machine",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	})

	// Verify
	if err != nil {
		t.Errorf("MachineDestroy should not return go error, got: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}
}

func TestPgCDIController_MachineDestroy_DestroyError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return success for poweroff but error for destroy
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_power") {
			return nil, map[string]interface{}{"status": "success"}
		}
		if strings.Contains(playbook, "machine_destroy") {
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "destroy command failed",
			}, nil
		}
		if strings.Contains(playbook, "machine_show") {
			machineData := map[string]interface{}{
				"name":               "test-machine",
				"mach_status_detail": "INACTIVE POFF",
			}
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{machineData},
				},
			}
		}
		return nil, map[string]interface{}{"status": "success"}
	}

	// Execute
	reply, err := controller.MachineDestroy(context.Background(), &proto.MachineDestroyRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:   "test-group",
		MachineName: "test-machine",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	})

	// Verify
	if err != nil {
		t.Errorf("MachineDestroy should not return go error, got: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}
}

// TestPgCDIController_MachineDestroy_InvalidExtraParameter tests MachineDestroy with invalid extra parameter
func TestPgCDIController_MachineDestroy_InvalidExtraParameter(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Execute with invalid ExtraParameter
	reply, err := controller.MachineDestroy(context.Background(), &proto.MachineDestroyRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "test-host",
			RemoteUser: "test-user",
		},
		GroupName:      "test-group",
		MachineName:    "test-machine",
		ExtraParameter: stringPtr(`invalid-json`),
	})

	// Verify
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}

	if !strings.Contains(reply.GetErrorMessage(), "invalid character") {
		t.Errorf("Expected error message to contain 'invalid character', got: %s", reply.GetErrorMessage())
	}
}

// TestPgCDIController_MachineDestroy_MachineShowError tests MachineDestroy when MachineShow fails
func TestPgCDIController_MachineDestroy_MachineShowError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			// Return error for machine_show
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "machine show failed",
			}, nil
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	reply, err := controller.MachineDestroy(context.Background(), &proto.MachineDestroyRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "test-host",
			RemoteUser: "test-user",
		},
		GroupName:      "test-group",
		MachineName:    "test-machine",
		ExtraParameter: stringPtr(string(extraParamJSON)),
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}

	if !strings.Contains(reply.GetErrorMessage(), "machine show failed") {
		t.Errorf("Expected error message to contain 'machine show failed', got: %s", reply.GetErrorMessage())
	}
}

func TestPgCDIController_ResourceList_AnsibleError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return error
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
			Message:    "resource list command failed",
		}, nil
	}

	// Execute
	reply, err := controller.ResourceList(context.Background(), &proto.ResourceListRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName: "test-group",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	})

	// Verify
	if err != nil {
		t.Errorf("ResourceList should not return go error, got: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}
}

func TestPgCDIController_ResourceShow_AnsibleError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return error
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
			Message:    "resource show command failed",
		}, nil
	}

	// Execute
	reply, err := controller.ResourceShow(context.Background(), &proto.ResourceShowRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		ResourceName: "test-resource",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	})

	// Verify
	if err != nil {
		t.Errorf("ResourceShow should not return go error, got: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}
}

func TestPgCDIController_PowerOffMachine_AnsibleError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return error
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
			Message:    "poweroff command failed",
		}, nil
	}

	extraParams := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "cdi-user",
		CDIPassword: "cdi-pass",
		CDIGuest:    "test-guest",
	}

	// Execute
	errMsg := controller.powerOffMachine(context.Background(), &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}, "test-machine", "test-group", extraParams)

	// Verify
	if errMsg == nil {
		t.Error("Expected error from powerOffMachine")
	}

	if errMsg.GetDetailCode() != int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0) {
		t.Errorf("Expected detail code CDI_COMMAND_ERROR, got %d", errMsg.GetDetailCode())
	}
}

func TestPgCDIController_GetMachineStatus_JsonParsingError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return malformed JSON data
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return nil, map[string]interface{}{
			"malformed": "data without status field",
		}
	}

	productInfo := &proto.ProductInformation{}
	extraParamStr := `{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`

	// Execute
	status, err := controller.getMachineStatus(context.Background(), productInfo, &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}, "test-machine", "test-group", extraParamStr)

	// Verify
	if err == nil {
		t.Error("Expected error from getMachineStatus due to malformed data")
	}

	if status != "" {
		t.Errorf("Expected empty status on error, got %s", status)
	}
}

func TestPgCDIController_PollMachineStatus_TimeoutReached_CompletesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	callCount := 0
	// Configure mock to always return a different status than target
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
		callCount++
		machineData := map[string]interface{}{
			"name":               "test-machine",
			"mach_status_detail": "ACTIVE PON", // Different from target "INACTIVE POFF"
		}
		return nil, map[string]interface{}{
			"data": map[string]interface{}{
				"machines": []interface{}{machineData},
			},
		}
	}

	productInfo := &proto.ProductInformation{}
	extraParamStr := `{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`

	// Execute with very short timeout to force timeout
	controller.pollMachineStatus(context.Background(), productInfo, &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}, "test-machine", "test-group", extraParamStr, []string{"INACTIVE POFF"}, 10*time.Millisecond, 50*time.Millisecond)

	// Verify - should have been called multiple times before timeout
	if callCount < 2 {
		t.Errorf("Expected multiple calls before timeout, got %d", callCount)
	}
}

// Additional JSON Marshal error tests for full coverage

func TestPgCDIController_MachineShow_JSONMarshalError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return data that will cause JSON marshal error
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return nil, map[string]interface{}{
			"data": map[string]interface{}{
				"machines": []interface{}{
					map[string]interface{}{
						"invalid": func() {}, // This will cause JSON marshal to fail
					},
				},
			},
		}
	}

	// Execute
	reply, err := controller.MachineShow(context.Background(), &proto.MachineShowRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName:   "test-group",
		MachineName: "test-machine",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	})

	// Verify
	if err == nil {
		t.Error("Expected error from JSON marshal failure")
	}

	if reply == nil {
		t.Fatal("Expected reply even with error")
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}
}

func TestPgCDIController_ResourceList_JSONMarshalError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return data that will cause JSON marshal error
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return nil, map[string]interface{}{
			"data": func() {}, // This will cause JSON marshal to fail
		}
	}

	// Execute
	reply, err := controller.ResourceList(context.Background(), &proto.ResourceListRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		GroupName: "test-group",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	})

	// Verify
	if err == nil {
		t.Error("Expected error from JSON marshal failure")
	}

	if reply == nil {
		t.Fatal("Expected reply even with error")
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}
}

func TestPgCDIController_ResourceShow_JSONMarshalError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return data that will cause JSON marshal error
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return nil, map[string]interface{}{
			"invalid": func() {}, // This will cause JSON marshal to fail
		}
	}

	// Execute
	reply, err := controller.ResourceShow(context.Background(), &proto.ResourceShowRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost:  "test-host",
			RemoteUser:  "test-user",
			
			
			
		},
		ResourceName: "test-resource",
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	})

	// Verify
	if err == nil {
		t.Error("Expected error from JSON marshal failure")
	}

	if reply == nil {
		t.Fatal("Expected reply even with error")
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}
}

func TestPgCDIController_GetMachineStatus_JSONUnmarshalError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return data that will cause JSON unmarshal error
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		// Return success but with data that causes JSON marshal to fail
		return nil, map[string]interface{}{
			"data": map[string]interface{}{
				"machines": []interface{}{
					map[string]interface{}{
						"invalid": func() {}, // This will cause JSON marshal to fail
					},
				},
			},
		}
	}

	cdiInfo := &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}

	productInfo := &proto.ProductInformation{}
	extraParamStr := `{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`

	// Execute
	status, err := controller.getMachineStatus(context.Background(), productInfo, cdiInfo, "test-machine", "test-group", extraParamStr)

	// Verify
	if err == nil {
		t.Error("Expected error from JSON marshal failure")
	}

	if status != "" {
		t.Error("Expected empty status on error")
	}
}

func TestPgCDIController_GetMachineStatus_MissingStatusField_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return data without status field
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return nil, map[string]interface{}{
			"data": map[string]interface{}{
				"machines": []interface{}{
					map[string]interface{}{
						"name": "test-machine",
						// Missing "status" field
					},
				},
			},
		}
	}

	cdiInfo := &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}

	productInfo := &proto.ProductInformation{}
	extraParamStr := `{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`

	// Execute
	status, err := controller.getMachineStatus(context.Background(), productInfo, cdiInfo, "test-machine", "test-group", extraParamStr)

	// Verify
	if err == nil {
		t.Error("Expected error for missing status field")
	}

	if status != "" {
		t.Error("Expected empty status on error")
	}
}

func TestPgCDIController_GetMachineStatus_NonStringStatusField_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return data with non-string status field
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		return nil, map[string]interface{}{
			"data": map[string]interface{}{
				"machines": []interface{}{
					map[string]interface{}{
						"name":   "test-machine",
						"status": 123, // Non-string status will fail type assertion
					},
				},
			},
		}
	}

	cdiInfo := &proto.CdiInformation{
		RemoteHost:  "test-host",
		RemoteUser:  "test-user",
		
		
		
	}

	productInfo := &proto.ProductInformation{}
	extraParamStr := `{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`

	// Execute
	status, err := controller.getMachineStatus(context.Background(), productInfo, cdiInfo, "test-machine", "test-group", extraParamStr)

	// Verify
	if err == nil {
		t.Error("Expected error for non-string status field")
	}

	if status != "" {
		t.Error("Expected empty status on error")
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// TestGetMachineResources_Success tests successful resource retrieval
func TestGetMachineResources_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			if playbook == "machine_show.yaml" {
				responseData := map[string]interface{}{
					"data": map[string]interface{}{
						"machines": []interface{}{
							map[string]interface{}{
								"name":   "test-machine",
								"status": "Active",
								"resources": []interface{}{
									map[string]interface{}{"res_name": "GPU-1"},
									map[string]interface{}{"res_name": "GPU-2"},
								},
							},
						},
					},
				}
				return nil, responseData
			}
			return &common.ErrorMessage{
				ErrorCode: int32(codes.Internal),
				Message:   "unexpected playbook",
			}, nil
		},
	}

	controller := PgCDIController{
		Ansible: mockAnsible,
	}

	productInfo := &proto.ProductInformation{
		Vendor:      "test-vendor",
		ProductName: "test-product",
		Version:     "1.0",
	}
	cdiInfo := &proto.CdiInformation{
		RemoteHost: "192.168.1.100",
		RemoteUser: "testuser",
	}
	extraParam := `{"remote_host":"192.168.1.100","remote_user":"testuser","ssh_private_key_file":"/path/to/key","cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`

	resources, err := controller.getMachineResources(context.Background(), productInfo, cdiInfo, "test-machine", "test-group", extraParam)
	if err != nil {
		t.Fatalf("getMachineResources returned error: %v", err)
	}
	if len(resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(resources))
	}
}

// TestCardScaling_WithDelay tests CardScaling with goroutine execution
func TestCardScaling_WithDelay(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			if playbook == "machine_show.yaml" {
				responseData := map[string]interface{}{
					"data": map[string]interface{}{
						"machines": []interface{}{
							map[string]interface{}{
								"name":               "test-machine",
								"mach_status_detail": "INACTIVE POFF",
								"resources": []interface{}{
									map[string]interface{}{"res_name": "GPU-1"},
								},
							},
						},
					},
				}
				return nil, responseData
			}
			if playbook == "machine_modify.yaml" {
				return nil, map[string]interface{}{"result": "success"}
			}
			return &common.ErrorMessage{
				ErrorCode: int32(codes.Internal),
				Message:   "unexpected playbook",
			}, nil
		},
	}

	controller := PgCDIController{
		Ansible: mockAnsible,
	}

	remoteHost := "192.168.1.100"
	remoteUser := "testuser"

	req := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      remoteHost,
			ProductName: remoteUser,
			Version:     "1.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: remoteHost,
			RemoteUser: remoteUser,
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{
				Op:           "add",
				ResourceName: "GPU-2",
			},
		},
		ExtraParameter: stringPtr(`{"remote_host":"192.168.1.100","remote_user":"testuser","ssh_private_key_file":"/path/to/key","cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`),
	}

	reply, err := controller.CardScaling(context.Background(), req)
	if err != nil {
		t.Fatalf("CardScaling returned error: %v", err)
	}
	if reply == nil {
		t.Fatal("reply is nil")
	}

	// Wait for goroutine to execute including polling (1 poll cycle = ~1s)
	time.Sleep(1500 * time.Millisecond)
}

// TestExtractResourceNames_Success tests successful resource name extraction
func TestExtractResourceNames_Success(t *testing.T) {
	resources := []interface{}{
		map[string]interface{}{"res_name": "GPU-1", "status": "available"},
		map[string]interface{}{"res_name": "GPU-2", "status": "in-use"},
	}

	names, err := extractResourceNames(resources)
	if err != nil {
		t.Fatalf("extractResourceNames returned error: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
	found1, found2 := false, false
	for _, name := range names {
		if name == "GPU-1" {
			found1 = true
		}
		if name == "GPU-2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("expected names GPU-1 and GPU-2, got %v", names)
	}
}

// TestExtractResourceNames_NotArray tests extraction when resources is not an array
func TestExtractResourceNames_NotArray(t *testing.T) {
	resources := "not-an-array"
	names, err := extractResourceNames(resources)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if names != nil {
		t.Errorf("expected nil names, got %v", names)
	}
}

// TestExtractResourceNames_ItemNotMap tests extraction when resource item is not a map
func TestExtractResourceNames_ItemNotMap(t *testing.T) {
	resources := []interface{}{"string-item", 123}
	names, err := extractResourceNames(resources)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if names != nil {
		t.Errorf("expected nil names, got %v", names)
	}
}

// TestExtractResourceNames_ResNameNotString tests extraction when res_name is not string
func TestExtractResourceNames_ResNameNotString(t *testing.T) {
	resources := []interface{}{
		map[string]interface{}{"res_name": 123},
	}
	names, err := extractResourceNames(resources)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if names != nil {
		t.Errorf("expected nil names, got %v", names)
	}
}

// TestTransformResourceModels_WithMappings tests model transformation with mappings
func TestTransformResourceModels_WithMappings(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Set environment variable for model mappings
	os.Setenv("PG_CDI_MODEL_MAPPINGS", `{"NVIDIA-A100":"GPU-Type-A","NVIDIA-H100":"GPU-Type-B"}`)
	defer os.Unsetenv("PG_CDI_MODEL_MAPPINGS")

	// Reset the once so the environment variable is re-read
	modelMappingsOnce = sync.Once{}
	modelMappings = make(map[string]string)

	// Create test data with resspecs array
	data := map[string]interface{}{
		"resspecs": []interface{}{
			map[string]interface{}{"resspec_model": "NVIDIA-A100", "resspec_type": "GPU"},
			map[string]interface{}{"resspec_model": "NVIDIA-H100", "resspec_type": "GPU"},
			map[string]interface{}{"resspec_model": "Unknown-Model", "resspec_type": "GPU"},
		},
	}

	logger := klog.NewKlogr()
	transformed := transformResourceModels(data, logger)

	resspecs, ok := transformed["resspecs"].([]interface{})
	if !ok {
		t.Fatal("resspecs not found in transformed data")
	}

	resspec0 := resspecs[0].(map[string]interface{})
	if resspec0["resspec_model"] != "GPU-Type-A" {
		t.Errorf("expected GPU-Type-A, got %v", resspec0["resspec_model"])
	}

	resspec1 := resspecs[1].(map[string]interface{})
	if resspec1["resspec_model"] != "GPU-Type-B" {
		t.Errorf("expected GPU-Type-B, got %v", resspec1["resspec_model"])
	}

	resspec2 := resspecs[2].(map[string]interface{})
	if resspec2["resspec_model"] != "Unknown-Model" {
		t.Errorf("expected Unknown-Model, got %v", resspec2["resspec_model"])
	}
}

// TestTransformResourceModels_NoMappings tests model transformation without mappings
func TestTransformResourceModels_NoMappings(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	os.Unsetenv("PG_CDI_MODEL_MAPPINGS")

	// Reset the once so the environment variable is re-read
	modelMappingsOnce = sync.Once{}
	modelMappings = make(map[string]string)

	data := map[string]interface{}{
		"resspecs": []interface{}{
			map[string]interface{}{"resspec_model": "NVIDIA-A100", "resspec_type": "GPU"},
		},
	}

	logger := klog.NewKlogr()
	transformed := transformResourceModels(data, logger)

	resspecs, ok := transformed["resspecs"].([]interface{})
	if !ok {
		t.Fatal("resspecs not found in transformed data")
	}

	resspec0 := resspecs[0].(map[string]interface{})
	if resspec0["resspec_model"] != "NVIDIA-A100" {
		t.Errorf("expected NVIDIA-A100, got %v", resspec0["resspec_model"])
	}
}

// TestLoadModelMappings_ValidJSON tests loading valid JSON model mappings
func TestLoadModelMappings_ValidJSON(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	os.Setenv("PG_CDI_MODEL_MAPPINGS", `{"Model1":"Hardware1","Model2":"Hardware2"}`)
	defer os.Unsetenv("PG_CDI_MODEL_MAPPINGS")

	// Reset the once so the environment variable is re-read
	modelMappingsOnce = sync.Once{}
	modelMappings = make(map[string]string)

	mappings := loadModelMappings()
	if len(mappings) != 2 {
		t.Errorf("expected 2 mappings, got %d", len(mappings))
	}
	if mappings["Model1"] != "Hardware1" {
		t.Errorf("expected Hardware1, got %v", mappings["Model1"])
	}
	if mappings["Model2"] != "Hardware2" {
		t.Errorf("expected Hardware2, got %v", mappings["Model2"])
	}
}

// TestLoadModelMappings_EmptyEnv tests loading when environment variable is empty
func TestLoadModelMappings_EmptyEnv(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	os.Unsetenv("PG_CDI_MODEL_MAPPINGS")

	// Reset the once so the environment variable is re-read
	modelMappingsOnce = sync.Once{}
	modelMappings = make(map[string]string)

	mappings := loadModelMappings()
	if len(mappings) != 0 {
		t.Errorf("expected empty mappings, got %v", mappings)
	}
}

// TestLoadModelMappings_InvalidJSON tests loading invalid JSON
func TestLoadModelMappings_InvalidJSON(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	os.Setenv("PG_CDI_MODEL_MAPPINGS", `{invalid-json}`)
	defer os.Unsetenv("PG_CDI_MODEL_MAPPINGS")

	// Reset the once so the environment variable is re-read
	modelMappingsOnce = sync.Once{}
	modelMappings = make(map[string]string)

	mappings := loadModelMappings()
	if len(mappings) != 0 {
		t.Errorf("expected empty mappings on error, got %v", mappings)
	}
}

// TestGetMachineResources_MachineShowFailure tests getMachineResources when MachineShow fails
func TestGetMachineResources_MachineShowFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "machine show command failed",
			}, nil
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	productInfo := &proto.ProductInformation{
		Vendor:      "TestVendor",
		ProductName: "TestProduct",
		Version:     "1.0.0",
	}
	cdiInfo := &proto.CdiInformation{
		RemoteHost: "test-host",
		RemoteUser: "test-user",
	}
	extraParam := `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`

	resources, err := controller.getMachineResources(context.Background(), productInfo, cdiInfo, "machine1", "group1", extraParam)

	if err == nil {
		t.Error("Expected error from getMachineResources when MachineShow fails")
	}
	if len(resources) != 0 {
		t.Errorf("Expected empty resources, got %v", resources)
	}
}

// TestGetMachineResources_InvalidJSON tests getMachineResources with invalid JSON response
func TestGetMachineResources_InvalidJSON(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			return nil, map[string]interface{}{
				"data": "invalid-json-string", // This will cause JSON unmarshal to fail in MachineShow
			}
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	productInfo := &proto.ProductInformation{
		Vendor:      "TestVendor",
		ProductName: "TestProduct",
		Version:     "1.0.0",
	}
	cdiInfo := &proto.CdiInformation{
		RemoteHost: "test-host",
		RemoteUser: "test-user",
	}
	extraParam := `{"cdi_user":"user","cdi_password":"pass","cdi_guest":"guest"}`

	resources, err := controller.getMachineResources(context.Background(), productInfo, cdiInfo, "machine1", "group1", extraParam)

	// Should fail because of JSON parse error
	if err == nil {
		t.Error("Expected error from getMachineResources with invalid JSON")
	}
	if len(resources) != 0 {
		t.Errorf("Expected empty resources, got %v", resources)
	}
}

// TestTransformResourceModels_NoResspecs tests transformResourceModels when resspecs is missing
func TestTransformResourceModels_NoResspecs(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup test mappings
	os.Setenv("PG_CDI_MODEL_MAPPINGS", `{"Model1":"Hardware1"}`)
	defer os.Unsetenv("PG_CDI_MODEL_MAPPINGS")

	// Reset the once so the environment variable is re-read
	modelMappingsOnce = sync.Once{}
	modelMappings = make(map[string]string)

	data := map[string]interface{}{
		"other_field": "value",
		// No "resspecs" field
	}

	result := transformResourceModels(data, klog.Background())

	// Should return original data unchanged
	if result["other_field"] != "value" {
		t.Errorf("Expected unchanged data, got %v", result)
	}
	if _, exists := result["resspecs"]; exists {
		t.Error("Did not expect resspecs field")
	}
}

// TestTransformResourceModels_InvalidResspecItem tests transformResourceModels with invalid resspec item
func TestTransformResourceModels_InvalidResspecItem(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup test mappings
	os.Setenv("PG_CDI_MODEL_MAPPINGS", `{"Model1":"Hardware1"}`)
	defer os.Unsetenv("PG_CDI_MODEL_MAPPINGS")

	// Reset the once so the environment variable is re-read
	modelMappingsOnce = sync.Once{}
	modelMappings = make(map[string]string)

	data := map[string]interface{}{
		"resspecs": []interface{}{
			"not_a_map", // Invalid item
			map[string]interface{}{
				"resspec_model": 123, // Invalid type (not string)
			},
		},
	}

	result := transformResourceModels(data, klog.Background())

	// Should return data with resspecs unchanged (branches for invalid items)
	resspecs := result["resspecs"].([]interface{})
	if len(resspecs) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resspecs))
	}
}

// TestPgCDIController_CardScaling_GetMachineResourcesError tests CardScaling goroutine error handling
func TestPgCDIController_CardScaling_GetMachineResourcesError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{
		CmdExecuteFunc: func(ctx context.Context, remoteHost, remotUser, sshPrivateKeyFile, playbook, extrArgs string) (*common.ErrorMessage, map[string]interface{}) {
			// Return error for machine_show to trigger getMachineResources error
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "failed to get machine resources",
			}, nil
		},
	}

	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{
				ResourceName: "fpga-0",
				Op:           "add",
			},
		},
		ExtraParameter: ptrString(string(extraParamJSON)),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	// Immediate response should be success (goroutine handles error)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %v", reply.GetResult())
	}

	// Wait for goroutine to execute and log error
	time.Sleep(200 * time.Millisecond)

	// Error is logged but doesn't affect response (covered in goroutine)
}

// TestPgCDIController_MachineCreate_InvalidExtraParameter_ReturnsError tests invalid extra parameter
func TestPgCDIController_MachineCreate_InvalidExtraParameter_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return error for machine_show (machine doesn't exist)
	// This allows us to reach ParseExtraParameter in MachineCreate
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.NotFound),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "machine not found",
			}, nil
		}
		return nil, map[string]interface{}{"data": "success"}
	}

	request := &proto.MachineCreateRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "test-host",
			RemoteUser: "test-user",
		},
		GroupName:      "test-group",
		MachineName:    "test-machine",
		ResourceList:   []string{"resource1"},
		ExtraParameter: stringPtr(`invalid json`), // Invalid JSON
	}

	// Execute
	reply, err := controller.MachineCreate(context.Background(), request)

	// Verify - MachineCreate returns error through reply, not err
	if err == nil {
		t.Error("Expected error to be returned")
	}

	if reply == nil {
		t.Fatal("Expected reply to be non-nil")
	}

	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", reply.GetResult())
	}

	if reply.GetErrorMessage() == "" {
		t.Error("Expected error message to be set")
	}

	// Verify error is about invalid parameter
	if !strings.Contains(reply.GetErrorMessage(), "invalid character") {
		t.Errorf("Expected error message to contain 'invalid character', got: %s", reply.GetErrorMessage())
	}
}

// TestPgCDIController_MachineCreate_MachineShowError_ProceedsToCreate tests that MachineShow error leads to create
func TestPgCDIController_MachineCreate_MachineShowError_ProceedsToCreate(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock: MachineShow fails, MachineCreate succeeds
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			// Return error for machine_show
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_0),
				Message:    "machine show failed",
			}, nil
		}
		if strings.Contains(playbook, "machine_create") {
			// Create succeeds
			return nil, map[string]interface{}{"data": "created"}
		}
		return nil, map[string]interface{}{}
	}

	request := &proto.MachineCreateRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "TestVendor",
			ProductName: "TestProduct",
			Version:     "1.0.0",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "test-host",
			RemoteUser: "test-user",
		},
		GroupName:      "test-group",
		MachineName:    "test-machine",
		ResourceList:   []string{"resource1"},
		ExtraParameter: stringPtr(`{"cdi_user":"cdi-user","cdi_password":"cdi-pass","cdi_guest":"test-guest"}`),
	}

	// Execute
	reply, err := controller.MachineCreate(context.Background(), request)

	// Verify
	if err != nil {
		t.Fatalf("MachineCreate failed: %v", err)
	}

	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", reply.GetResult())
	}

	if reply.GetErrorMessage() != "" {
		t.Errorf("Expected empty error message, got %s", reply.GetErrorMessage())
	}

	// Verify ansible was called twice (MachineShow + MachineCreate)
	if len(mockAnsible.CallHistory) != 2 {
		t.Fatalf("Expected 2 ansible calls, got %d", len(mockAnsible.CallHistory))
	}
}

// TestPgCDIController_CardScaling_InvalidExtraParameter_ReturnsError tests invalid extra parameter
func TestPgCDIController_CardScaling_InvalidExtraParameter_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{
				ResourceName: "fpga-0",
				Op:           "add",
			},
		},
		ExtraParameter: ptrString(`invalid json`),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	// Verify - error should be returned through reply and err
	if err == nil {
		t.Error("Expected error to be returned")
	}
	if reply == nil {
		t.Fatal("Expected reply to be non-nil")
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR result, got %v", reply.GetResult())
	}
}

// TestPgCDIController_CardScaling_ResourceAlreadyExists_SkipsAddition tests add operation when resource exists
func TestPgCDIController_CardScaling_ResourceAlreadyExists_SkipsAddition(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return existing resources
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			// Return machine with fpga-0 already attached (proper MachineShow format)
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
								"mach_status_detail": "INACTIVE POFF",
								"resources": []interface{}{
									map[string]interface{}{"res_name": "fpga-0"},
									map[string]interface{}{"res_name": "fpga-1"},
								},
						},
					},
				},
			}
		}
		t.Error("Expected machine_modify to NOT be called when resource already exists")
		return nil, map[string]interface{}{}
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{
				ResourceName: "fpga-0", // Already exists
				Op:           "add",
			},
		},
		ExtraParameter: ptrString(string(extraParamJSON)),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	// Verify - should return success immediately
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %v", reply.GetResult())
	}

	// Wait for goroutine
	time.Sleep(200 * time.Millisecond)
}

// TestPgCDIController_CardScaling_ResourceNotExists_SkipsRemoval tests remove operation when resource doesn't exist
func TestPgCDIController_CardScaling_ResourceNotExists_SkipsRemoval(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to return existing resources without fpga-2
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			// Return machine without fpga-2 (proper MachineShow format)
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
								"mach_status_detail": "INACTIVE POFF",
								"resources": []interface{}{
									map[string]interface{}{"res_name": "fpga-0"},
									map[string]interface{}{"res_name": "fpga-1"},
								},
						},
					},
				},
			}
		}
		t.Error("Expected machine_modify to NOT be called when resource doesn't exist")
		return nil, map[string]interface{}{}
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{
				ResourceName: "fpga-2", // Doesn't exist
				Op:           "remove",
			},
		},
		ExtraParameter: ptrString(string(extraParamJSON)),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %v", reply.GetResult())
	}

	// Wait for goroutine
	time.Sleep(200 * time.Millisecond)
}

// TestPgCDIController_CardScaling_AddAndRemoveOperations_ExecutesBoth tests mixed add/remove operations
func TestPgCDIController_CardScaling_AddAndRemoveOperations_ExecutesBoth(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	addCalled := false
	removeCalled := false

	// Configure mock
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			// Return machine with fpga-1 only (proper MachineShow format with res_name)
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"mach_status_detail": "INACTIVE POFF",
							"resources": []interface{}{
								map[string]interface{}{"res_name": "fpga-1"},
							},
						},
					},
				},
			}
		}
		if strings.Contains(playbook, "machine_modify") {
			if strings.Contains(extraArgs, `"operation":"add"`) && strings.Contains(extraArgs, "fpga-0") {
				addCalled = true
			}
			if strings.Contains(extraArgs, `"operation":"remove"`) && strings.Contains(extraArgs, "fpga-1") {
				removeCalled = true
			}
			return nil, map[string]interface{}{"result": "success"}
		}
		return nil, map[string]interface{}{}
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{
				ResourceName: "fpga-0",
				Op:           "add",
			},
			{
				ResourceName: "fpga-1",
				Op:           "remove",
			},
		},
		ExtraParameter: ptrString(string(extraParamJSON)),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %v", reply.GetResult())
	}

	// Wait for goroutine including 2 polling cycles (add poll + remove poll, ~1s each)
	time.Sleep(3000 * time.Millisecond)

	// Verify both operations were called
	if !addCalled {
		t.Error("Expected add operation to be called")
	}
	if !removeCalled {
		t.Error("Expected remove operation to be called")
	}
}

// TestPgCDIController_CardScaling_AddOperationError_LogsError tests add operation failure
func TestPgCDIController_CardScaling_AddOperationError_LogsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to fail on add operation
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			// Return machine without fpga-0 (proper MachineShow format)
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"status":    "ACTIVE PON",
							"resources": []interface{}{},
						},
					},
				},
			}
		}
		if strings.Contains(playbook, "machine_modify") && strings.Contains(extraArgs, `"operation":"add"`) {
			// Add operation fails
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_1),
				Message:    "failed to add resource",
			}, nil
		}
		return nil, map[string]interface{}{}
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{
				ResourceName: "fpga-0",
				Op:           "add",
			},
		},
		ExtraParameter: ptrString(string(extraParamJSON)),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	// Verify - immediate response is success, error is logged in goroutine
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %v", reply.GetResult())
	}

	// Wait for goroutine to execute and log error
	time.Sleep(200 * time.Millisecond)
}

// TestPgCDIController_CardScaling_RemoveOperationError_LogsError tests remove operation failure
func TestPgCDIController_CardScaling_RemoveOperationError_LogsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	// Configure mock to fail on remove operation
	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			// Return machine with fpga-0 (proper MachineShow format)
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"mach_status_detail": "INACTIVE POFF",
							"resources": []interface{}{
								map[string]interface{}{"res_name": "fpga-0"},
							},
						},
					},
				},
			}
		}
		if strings.Contains(playbook, "machine_modify") && strings.Contains(extraArgs, `"operation":"remove"`) {
			// Remove operation fails
			return &common.ErrorMessage{
				ErrorCode:  int32(codes.Internal),
				DetailCode: int32(proto.DetailCode_CDI_COMMAND_ERROR_V_1_1),
				Message:    "failed to remove resource",
			}, nil
		}
		return nil, map[string]interface{}{}
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{
				ResourceName: "fpga-0",
				Op:           "remove",
			},
		},
		ExtraParameter: ptrString(string(extraParamJSON)),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	// Verify - immediate response is success, error is logged in goroutine
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %v", reply.GetResult())
	}

	// Wait for goroutine to execute and log error
	time.Sleep(200 * time.Millisecond)
}

// TestPgCDIController_CardScaling_AddOperationPollingErrorStatus_AbortsBeforeRemove
// tests that when polling after add returns ERROR status, the remove operation is not executed
func TestPgCDIController_CardScaling_AddOperationPollingErrorStatus_AbortsBeforeRemove(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	machineShowCallCount := 0
	removeCalled := false

	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			machineShowCallCount++
			if machineShowCallCount == 1 {
				// First call: getMachineResources - return fpga-1 (so fpga-0 is added, fpga-1 removed)
				return nil, map[string]interface{}{
					"data": map[string]interface{}{
						"machines": []interface{}{
							map[string]interface{}{
								"mach_status_detail": "INACTIVE POFF",
								"resources": []interface{}{
									map[string]interface{}{"res_name": "fpga-1"},
								},
							},
						},
					},
				}
			}
			// Subsequent calls: polling - return ERROR to simulate failure
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"mach_status_detail": "ERROR",
							"resources":          []interface{}{},
						},
					},
				},
			}
		}
		if strings.Contains(playbook, "machine_modify") {
			if strings.Contains(extraArgs, `"operation":"remove"`) {
				removeCalled = true
			}
			return nil, map[string]interface{}{}
		}
		return nil, map[string]interface{}{}
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{ResourceName: "fpga-0", Op: "add"},
			{ResourceName: "fpga-1", Op: "remove"},
		},
		ExtraParameter: ptrString(string(extraParamJSON)),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %v", reply.GetResult())
	}

	// Wait for goroutine: add(~0ms) + poll(~1s, returns ERROR) → abort
	time.Sleep(1500 * time.Millisecond)

	// Remove must NOT be executed when add polling returns ERROR
	if removeCalled {
		t.Error("Expected remove operation NOT to be called when add polling returns ERROR")
	}
}

// TestPgCDIController_CardScaling_RemoveOperationPollingErrorStatus_LogsError
// tests that when polling after remove returns ERROR status, the error is logged
func TestPgCDIController_CardScaling_RemoveOperationPollingErrorStatus_LogsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAnsible := &MockPgCDIAnsible{}
	controller := &PgCDIController{
		Logger:  klog.Background(),
		Ansible: mockAnsible,
		SSHKey:  "/tmp/test_key",
	}

	machineShowCallCount := 0

	mockAnsible.CmdExecuteFunc = func(ctx context.Context, remoteHost, remoteUser, sshPrivateKeyFile, playbook, extraArgs string) (*common.ErrorMessage, map[string]interface{}) {
		if strings.Contains(playbook, "machine_show") {
			machineShowCallCount++
			if machineShowCallCount == 1 {
				// First call: getMachineResources - return fpga-0 so it can be removed
				return nil, map[string]interface{}{
					"data": map[string]interface{}{
						"machines": []interface{}{
							map[string]interface{}{
								"mach_status_detail": "INACTIVE POFF",
								"resources": []interface{}{
									map[string]interface{}{"res_name": "fpga-0"},
								},
							},
						},
					},
				}
			}
			// Subsequent calls: polling after remove - return ERROR
			return nil, map[string]interface{}{
				"data": map[string]interface{}{
					"machines": []interface{}{
						map[string]interface{}{
							"mach_status_detail": "ERROR",
							"resources":          []interface{}{},
						},
					},
				},
			}
		}
		if strings.Contains(playbook, "machine_modify") {
			return nil, map[string]interface{}{}
		}
		return nil, map[string]interface{}{}
	}

	extraParam := &extra_parameters.PgCDIExtraParameters{
		CDIUser:     "test_user",
		CDIPassword: "test_password",
		CDIGuest:    "test_guest",
	}
	extraParamJSON, _ := json.Marshal(extraParam)

	request := &proto.CardScalingRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "NEC",
			ProductName: "test-product",
			Version:     "test-model",
		},
		CdiInfo: &proto.CdiInformation{
			RemoteHost: "192.168.1.100",
			RemoteUser: "cdi_user",
		},
		MachineName: "test-machine",
		GroupName:   "test-group",
		ResourceModifyRequests: []*proto.ResourceModifyRequests{
			{ResourceName: "fpga-0", Op: "remove"},
		},
		ExtraParameter: ptrString(string(extraParamJSON)),
	}

	reply, err := controller.CardScaling(context.Background(), request)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %v", reply.GetResult())
	}

	// Wait for goroutine: remove(~0ms) + poll(~1s, returns ERROR) → log error and abort
	time.Sleep(1500 * time.Millisecond)
}
