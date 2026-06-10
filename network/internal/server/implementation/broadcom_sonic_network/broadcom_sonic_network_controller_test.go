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

package broadcom_sonic_network

import (
	"context"
	"encoding/json"
	"testing"

	proto "network_module/api/proto"
	common "common/api/proto" // import of common protobuf
	edgecore_sonic_network "network_module/internal/server/implementation/edgecore_sonic_network" // import edgecore sonic network implement
	"network_module/internal/server/interfaces"
	"network_module/internal/server/test_utils"

	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"
)

// Mock Ansible implementation for testing
type mockAnsible struct {
	output   []byte
	errorMsg *common.ErrorMessage
}

func (m *mockAnsible) CmdExecute(ctx context.Context, remoteHost string, remoteUser string, sshPrivateKeyFile string, playbook string, extrArgs string) ([]byte, *common.ErrorMessage) {
	return m.output, m.errorMsg
}

// newController is a helper to create a BroadcomSonicNetworkController with a mock ansible
func newController(ansible interfaces.NetworkAnsible) BroadcomSonicNetworkController {
	return BroadcomSonicNetworkController{
		EdgeCoreSonicNetworkController: edgecore_sonic_network.EdgeCoreSonicNetworkController{
			Logger:  klog.Background(),
			Ansible: ansible,
			SSHKey:  "/tmp/test.pem",
		},
	}
}

func TestVlanAdd_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanAddRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType: "access",
		VlanId:   &vlanID,
	}

	// Act
	reply, err := controller.VlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
	if reply.ErrorMessage != "" {
		t.Errorf("Expected empty error message, got %s", reply.ErrorMessage)
	}
}

func TestVlanAdd_AnsibleCommandFails_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorMsg := &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: 71,
		Message:    "command execution failed",
	}
	controller := newController(&mockAnsible{output: []byte("failed"), errorMsg: errorMsg})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanAddRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType: "access",
		VlanId:   &vlanID,
	}

	// Act
	reply, err := controller.VlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR, got %v", reply.GetResult())
	}
	if reply.ErrorMessage == "" {
		t.Error("Expected error message, got empty string")
	}
}

func TestVlanAdd_NilSwitchInfo_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanAddRequest{
		SwitchInfo: nil,
		VlanType:   "access",
		VlanId:     &vlanID,
	}

	// Act
	reply, err := controller.VlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
}

func TestVlanAdd_TrunkVlanType_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanAddRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType: "trunk",
		VlanId:   &vlanID,
	}

	// Act
	reply, err := controller.VlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
}

func TestVlanDelete_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanDeleteRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
	}

	// Act
	reply, err := controller.VlanDelete(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
	if reply.ErrorMessage != "" {
		t.Errorf("Expected empty error message, got %s", reply.ErrorMessage)
	}
}

func TestVlanDelete_AnsibleCommandFails_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorMsg := &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: 71,
		Message:    "command execution failed",
	}
	controller := newController(&mockAnsible{output: []byte("failed"), errorMsg: errorMsg})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanDeleteRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
	}

	// Act
	reply, err := controller.VlanDelete(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR, got %v", reply.GetResult())
	}
	if reply.ErrorMessage == "" {
		t.Error("Expected error message, got empty string")
	}
}

func TestVlanDelete_NilSwitchInfo_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanDeleteRequest{
		SwitchInfo: nil,
		VlanId:     &vlanID,
	}

	// Act
	reply, err := controller.VlanDelete(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
}

func TestVswVlanAdd_ValidRequestWithVlanID_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanAddRequest{
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	// Act
	reply, err := controller.VswVlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
	if reply.ErrorMessage != "" {
		t.Errorf("Expected empty error message, got %s", reply.ErrorMessage)
	}
}

func TestVswVlanAdd_ValidRequestWithoutVlanID_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	req := &proto.VswVlanAddRequest{
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: nil,
		IfName: "eth0",
	}

	// Act
	reply, err := controller.VswVlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
}

func TestVswVlanAdd_AnsibleCommandFails_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorMsg := &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: 71,
		Message:    "command execution failed",
	}
	controller := newController(&mockAnsible{output: []byte("failed"), errorMsg: errorMsg})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanAddRequest{
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	// Act
	reply, err := controller.VswVlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR, got %v", reply.GetResult())
	}
	if reply.ErrorMessage == "" {
		t.Error("Expected error message, got empty string")
	}
	// Verify that proto.DetailCode_NW_COMMAND_ERROR (71) is converted to proto.DetailCode_VSW_VLAN_DUPLICATE (73)
	var errMsg common.ErrorMessage
	if err := json.Unmarshal([]byte(reply.ErrorMessage), &errMsg); err != nil {
		t.Errorf("Failed to parse error message: %v", err)
	}
	if errMsg.DetailCode != 73 {
		t.Errorf("Expected DetailCode 73 (proto.DetailCode_VSW_VLAN_DUPLICATE), got %d", errMsg.DetailCode)
	}
}

func TestVswVlanAdd_NilHostInfo_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanAddRequest{
		HostInfo: nil,
		VlanId:   &vlanID,
		IfName:   "eth0",
	}

	// Act
	reply, err := controller.VswVlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
}

func TestVswVlanDelete_ValidRequestWithVlanID_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanDeleteRequest{
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	// Act
	reply, err := controller.VswVlanDelete(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
	if reply.ErrorMessage != "" {
		t.Errorf("Expected empty error message, got %s", reply.ErrorMessage)
	}
}

func TestVswVlanDelete_ValidRequestWithoutVlanID_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	req := &proto.VswVlanDeleteRequest{
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: nil,
		IfName: "eth0",
	}

	// Act
	reply, err := controller.VswVlanDelete(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
}

func TestVswVlanDelete_AnsibleCommandFails_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	errorMsg := &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: 71,
		Message:    "command execution failed",
	}
	controller := newController(&mockAnsible{output: []byte("failed"), errorMsg: errorMsg})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanDeleteRequest{
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	// Act
	reply, err := controller.VswVlanDelete(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR, got %v", reply.GetResult())
	}
	if reply.ErrorMessage == "" {
		t.Error("Expected error message, got empty string")
	}
	// Verify that proto.DetailCode_NW_COMMAND_ERROR (71) is converted to proto.DetailCode_VSW_VLAN_NOTFOUND (74)
	var errMsg common.ErrorMessage
	if err := json.Unmarshal([]byte(reply.ErrorMessage), &errMsg); err != nil {
		t.Errorf("Failed to parse error message: %v", err)
	}
	if errMsg.DetailCode != 74 {
		t.Errorf("Expected DetailCode 74 (proto.DetailCode_VSW_VLAN_NOTFOUND), got %d", errMsg.DetailCode)
	}
}

func TestVswVlanDelete_NilHostInfo_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := newController(&mockAnsible{output: []byte("success"), errorMsg: nil})
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanDeleteRequest{
		HostInfo: nil,
		VlanId:   &vlanID,
		IfName:   "eth0",
	}

	// Act
	reply, err := controller.VswVlanDelete(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
}

// Test to ensure the mockAnsible implements the interface
func TestMockAnsible_ImplementsInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	var _ interfaces.NetworkAnsible = &mockAnsible{}

	mock := &mockAnsible{}
	if mock == nil {
		t.Error("Expected mock to be created, got nil")
	}
}
