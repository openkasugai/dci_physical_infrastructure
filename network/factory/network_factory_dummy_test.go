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

package factory

import (
	"context"
	"testing"

	proto "network_module/api/proto"
    common "common/api/proto"    // import of common protobuf
	"network_module/internal/server/implementation/dummy_network"
	"network_module/internal/server/interfaces"
	"network_module/internal/server/test_utils"

	"k8s.io/klog/v2"
)

func TestCreateNetworkController_DummyBuild_ReturnsDummyController(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	logger := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "Dummy",
		ProductName: "DummySwitch",
		Version:     "1.0",
		Os:          &[]string{"DummyOS"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	if controller == nil {
		t.Error("Expected controller to be created, got nil")
		return
	}

	// Check if it's the correct type
	dummyController, ok := controller.(*dummy_network.DummyNetworkController)
	if !ok {
		t.Errorf("Expected DummyNetworkController, got %T", controller)
		return
	}

	// Verify the logger is set
	if dummyController == nil {
		t.Error("Expected dummy controller to be initialized")
	}
}

func TestCreateNetworkController_DummyBuild_ImplementsInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	logger := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "Dummy",
		ProductName: "DummySwitch",
		Version:     "1.0",
		Os:          &[]string{"DummyOS"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	if controller == nil {
		t.Error("Expected controller to be created, got nil")
		return
	}

	// Verify it implements the NetworkController interface
	var _ interfaces.NetworkController = controller
}

func TestCreateNetworkController_DummyBuild_NilLogger_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	var logger klog.Logger
	productInfo := &proto.ProductInformation{
		Vendor:      "Dummy",
		ProductName: "DummySwitch",
		Version:     "1.0",
		Os:          &[]string{"DummyOS"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	if controller == nil {
		t.Error("Expected controller to be created even with zero-value logger, got nil")
		return
	}

	// Verify it's still a dummy controller
	_, ok := controller.(*dummy_network.DummyNetworkController)
	if !ok {
		t.Errorf("Expected DummyNetworkController, got %T", controller)
	}
}

func TestCreateNetworkController_DummyBuild_FunctionalTest(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	logger := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "Dummy",
		ProductName: "DummySwitch",
		Version:     "1.0",
		Os:          &[]string{"DummyOS"}[0],
	}
	controller := CreateNetworkController(logger, productInfo)
	ctx := context.Background()

	vlanID := int32(100)

	// Test VlanAdd
	vlanAddReq := &proto.VlanAddRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType: "trunk",
		VlanId:   &vlanID,
	}

	// Act & Assert VlanAdd
	reply1, err1 := controller.VlanAdd(ctx, vlanAddReq)
	if err1 != nil {
		t.Errorf("Expected no error from dummy VlanAdd, got %v", err1)
	}
	if reply1 == nil {
		t.Error("Expected reply from dummy VlanAdd, got nil")
	} else if reply1.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS from dummy VlanAdd, got %v", reply1.GetResult())
	}

	// Test VlanDelete
	vlanDelReq := &proto.VlanDeleteRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
	}

	reply2, err2 := controller.VlanDelete(ctx, vlanDelReq)
	if err2 != nil {
		t.Errorf("Expected no error from dummy VlanDelete, got %v", err2)
	}
	if reply2 == nil {
		t.Error("Expected reply from dummy VlanDelete, got nil")
	} else if reply2.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS from dummy VlanDelete, got %v", reply2.GetResult())
	}

	// Test VswVlanAdd
	vswAddReq := &proto.VswVlanAddRequest{
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	reply3, err3 := controller.VswVlanAdd(ctx, vswAddReq)
	if err3 != nil {
		t.Errorf("Expected no error from dummy VswVlanAdd, got %v", err3)
	}
	if reply3 == nil {
		t.Error("Expected reply from dummy VswVlanAdd, got nil")
	} else if reply3.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS from dummy VswVlanAdd, got %v", reply3.GetResult())
	}

	// Test VswVlanDelete
	vswDelReq := &proto.VswVlanDeleteRequest{
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	reply4, err4 := controller.VswVlanDelete(ctx, vswDelReq)
	if err4 != nil {
		t.Errorf("Expected no error from dummy VswVlanDelete, got %v", err4)
	}
	if reply4 == nil {
		t.Error("Expected reply from dummy VswVlanDelete, got nil")
	} else if reply4.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS from dummy VswVlanDelete, got %v", reply4.GetResult())
	}
}

func TestCreateNetworkController_DummyBuild_MultipleCreations_ReturnsSeparateInstances(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	logger1 := klog.Background()
	logger2 := klog.Background()
	productInfo := &proto.ProductInformation{
		Vendor:      "Dummy",
		ProductName: "DummySwitch",
		Version:     "1.0",
		Os:          &[]string{"DummyOS"}[0],
	}

	// Act
	controller1 := CreateNetworkController(logger1, productInfo)
	controller2 := CreateNetworkController(logger2, productInfo)

	// Assert
	if controller1 == nil || controller2 == nil {
		t.Error("Expected both controllers to be created")
		return
	}

	if controller1 == controller2 {
		t.Error("Expected separate controller instances, got same instance")
	}

	// Both should be dummy controllers
	_, ok1 := controller1.(*dummy_network.DummyNetworkController)
	_, ok2 := controller2.(*dummy_network.DummyNetworkController)

	if !ok1 || !ok2 {
		t.Error("Expected both controllers to be DummyNetworkController instances")
	}
}

func TestCreateNetworkController_DummyBuild_LoggerIntegration(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	logger := klog.Background().WithName("test-factory")
	productInfo := &proto.ProductInformation{
		Vendor:      "Dummy",
		ProductName: "DummySwitch",
		Version:     "1.0",
		Os:          &[]string{"DummyOS"}[0],
	}

	// Act
	controller := CreateNetworkController(logger, productInfo)

	// Assert
	if controller == nil {
		t.Error("Expected controller to be created with named logger, got nil")
		return
	}

	// Verify it's a dummy controller
	dummyController, ok := controller.(*dummy_network.DummyNetworkController)
	if !ok {
		t.Errorf("Expected DummyNetworkController, got %T", controller)
		return
	}

	// Test that the controller can be used without issues
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

	reply, err := dummyController.VlanAdd(ctx, req)
	if err != nil {
		t.Errorf("Expected no error from dummy controller with named logger, got %v", err)
	}
	if reply == nil || reply.GetResult() != common.ResultCode_SUCCESS {
		t.Error("Expected successful reply from dummy controller with named logger")
	}
}
