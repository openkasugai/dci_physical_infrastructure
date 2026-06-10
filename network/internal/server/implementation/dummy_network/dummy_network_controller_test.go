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

package dummy_network

import (
	"context"
	"testing"

	proto "network_module/api/proto"
    common "common/api/proto"    // import of common protobuf
	"network_module/internal/server/test_utils"

	"k8s.io/klog/v2"
)

func TestVlanAdd_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
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

func TestVlanAdd_NilSwitchInfo_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
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

func TestVlanAdd_NilPort_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
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
}

func TestVlanAdd_NilVlanId_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
	ctx := context.Background()
	req := &proto.VlanAddRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType: "access",
		VlanId:   nil,
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
	controller := DummyNetworkController{Logger: klog.Background()}
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

func TestVlanDelete_NilSwitchInfo_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
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

func TestVlanDelete_NilPort_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
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
}

func TestVlanDelete_NilVlanId_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
	ctx := context.Background()
	req := &proto.VlanDeleteRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: nil,
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
	controller := DummyNetworkController{Logger: klog.Background()}
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
	controller := DummyNetworkController{Logger: klog.Background()}
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

func TestVswVlanAdd_NilHostInfo_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
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
	controller := DummyNetworkController{Logger: klog.Background()}
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
	controller := DummyNetworkController{Logger: klog.Background()}
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

func TestVswVlanDelete_NilHostInfo_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := DummyNetworkController{Logger: klog.Background()}
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
