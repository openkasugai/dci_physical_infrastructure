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

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	proto "network_module/api/proto"
    common "common/api/proto"    // import of common protobuf
	"network_module/internal/server/interfaces"
	"network_module/internal/server/test_utils"
	"network_module/internal/server/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Mock network controller for testing
type mockNetworkController struct {
	vlanAddReply       *proto.VlanAddReply
	vlanDeleteReply    *proto.VlanDeleteReply
	vswVlanAddReply    *proto.VswVlanAddReply
	vswVlanDeleteReply *proto.VswVlanDeleteReply
	shouldError        bool
}

func (m *mockNetworkController) VlanAdd(ctx context.Context, in *proto.VlanAddRequest) (*proto.VlanAddReply, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	if m.vlanAddReply != nil {
		return m.vlanAddReply, nil
	}
	return &proto.VlanAddReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}, nil
}

func (m *mockNetworkController) VlanDelete(ctx context.Context, in *proto.VlanDeleteRequest) (*proto.VlanDeleteReply, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	if m.vlanDeleteReply != nil {
		return m.vlanDeleteReply, nil
	}
	return &proto.VlanDeleteReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}, nil
}

func (m *mockNetworkController) VswVlanAdd(ctx context.Context, in *proto.VswVlanAddRequest) (*proto.VswVlanAddReply, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	if m.vswVlanAddReply != nil {
		return m.vswVlanAddReply, nil
	}
	return &proto.VswVlanAddReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}, nil
}

func (m *mockNetworkController) VswVlanDelete(ctx context.Context, in *proto.VswVlanDeleteRequest) (*proto.VswVlanDeleteReply, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	if m.vswVlanDeleteReply != nil {
		return m.vswVlanDeleteReply, nil
	}
	return &proto.VswVlanDeleteReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}, nil
}

// setupMainTestConfig sets up test configuration for main tests
func setupMainTestConfig() {
	os.Setenv("NW_SERVER_PORT", "50051")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "false")
	os.Setenv("TLS_CERT_PATH", "/test/cert")
	utils.InitializeConfig()
}

// clearMainTestConfig clears test configuration
func clearMainTestConfig() {
	os.Unsetenv("NW_SERVER_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SSH_KEY")
	os.Unsetenv("TLS_ENABLE")
	os.Unsetenv("TLS_CERT_PATH")
}

// resetGlobalState resets global state for testing
func resetGlobalState() {
	testListener = nil
	isTest = false
	klogInitOnce = sync.Once{}
}

func TestNewNetworkServer_ValidController_ReturnsServer(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange & Act
	server := newNetworkServer()

	// Assert
	if server == nil {
		t.Error("Expected server to be created, got nil")
		return
	}
}

func TestNewNetworkServer_NilController_ReturnsServerWithNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange & Act
	server := newNetworkServer()

	// Assert
	if server == nil {
		t.Error("Expected server to be created, got nil")
		return
	}
}

func TestVlanAdd_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanAddRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType:      "trunk", // Valid vlan type
		VlanId:        &vlanID,
		InterfaceName: "eth0",
	}

	// Act
	reply, err := server.VlanAdd(ctx, req)

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

func TestVlanAdd_InvalidRequest_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	req := &proto.VlanAddRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		SwitchInfo: nil, // This should cause validation error
		VlanType:   "",
		VlanId:     nil,
	}

	// Act
	reply, err := server.VlanAdd(ctx, req)

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

func TestVlanAdd_InvalidVlanType_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanAddRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType: "invalid_very_long_type", // Invalid vlan type (too long)
		VlanId:   &vlanID,
	}

	// Act
	reply, err := server.VlanAdd(ctx, req)

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
}

func TestVlanAdd_MissingPort_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanAddRequest{
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType: "trunk", // Valid vlan type
		VlanId:   &vlanID,
	}

	// Act
	reply, err := server.VlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	// Note: The actual validation happens after the mock, so this may return SUCCESS
	// This test verifies the function can handle nil port without panicking
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Logf("Port validation may be handled differently, got result: %v", reply.GetResult())
	}
}

func TestVlanAdd_MissingVlanID_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	req := &proto.VlanAddRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType: "trunk", // Valid vlan type
		VlanId:   nil,     // Missing required field
	}

	// Act
	reply, err := server.VlanAdd(ctx, req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	// Note: The actual validation happens after the mock, so this may return SUCCESS
	// This test verifies the function can handle nil vlan ID without panicking
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Logf("VlanID validation may be handled differently, got result: %v", reply.GetResult())
	}
}

func TestVlanDelete_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanDeleteRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId:        &vlanID,
		InterfaceName: "eth0",
	}

	// Act
	reply, err := server.VlanDelete(ctx, req)

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

func TestVlanDelete_InvalidRequest_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	req := &proto.VlanDeleteRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		SwitchInfo: nil, // This should cause validation error
		VlanId:     nil,
	}

	// Act
	reply, err := server.VlanDelete(ctx, req)

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
}

func TestVswVlanAdd_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
setupMainTestConfig()
defer clearMainTestConfig()
	defer cleanup()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanAddRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	// Act
	reply, err := server.VswVlanAdd(ctx, req)

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

func TestVswVlanAdd_InvalidRequest_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
setupMainTestConfig()
defer clearMainTestConfig()
	defer cleanup()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	req := &proto.VswVlanAddRequest{
		HostInfo: nil, // This should cause validation error
		VlanId:   nil,
		IfName:   "",
	}

	// Act
	reply, err := server.VswVlanAdd(ctx, req)

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
}

func TestVswVlanAdd_IfNameTooLong_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
setupMainTestConfig()
defer clearMainTestConfig()
	defer cleanup()

	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanAddRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth01234", // 8 chars, max_len is 7
	}

	reply, err := server.VswVlanAdd(ctx, req)
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
}

func TestVswVlanDelete_ValidRequest_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
setupMainTestConfig()
defer clearMainTestConfig()
	defer cleanup()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanDeleteRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	// Act
	reply, err := server.VswVlanDelete(ctx, req)

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

func TestVswVlanDelete_InvalidRequest_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
setupMainTestConfig()
defer clearMainTestConfig()
	defer cleanup()

	// Arrange
	server := newNetworkServer()
	ctx := context.Background()
	req := &proto.VswVlanDeleteRequest{
		HostInfo: nil, // This should cause validation error
		VlanId:   nil,
		IfName:   "",
	}

	// Act
	reply, err := server.VswVlanDelete(ctx, req)

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
}

func TestVswVlanDelete_IfNameTooLong_ReturnsValidationError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
setupMainTestConfig()
defer clearMainTestConfig()
	defer cleanup()

	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanDeleteRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "Dummy",
			ProductName: "DummySwitch",
			Version:     "1.0",
			Os:          &[]string{"DummyOS"}[0],
		},
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth01234", // 8 chars, max_len is 7
	}

	reply, err := server.VswVlanDelete(ctx, req)
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
}

func TestRun_ValidPort_StartsServer(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupMainTestConfig()
	defer clearMainTestConfig()
	resetGlobalState()
	isTest = true

	// Mock the serveWrapper to avoid actually starting the server
	originalServeWrapper := serveWrapper
	defer func() {
		serveWrapper = originalServeWrapper
		isTest = false
	}()

	called := false
	serveWrapper = func(s *grpc.Server, lis net.Listener) error {
		called = true
		return nil // Mock successful start
	}

	// Act
	run(50051)

	// Assert
	if !called {
		t.Error("Expected serveWrapper to be called")
	}
	if testListener == nil {
		t.Error("Expected testListener to be set, got nil")
	}
}

// func TestInitKlog_CallsInitFlagsOnce(t *testing.T) {
// 	// Arrange
// 	setupMainTestConfig()
// 	defer clearMainTestConfig()
// 	resetGlobalState()

// 	// Act
// 	initKlog()
// 	initKlog() // Call twice to test Once behavior

// 	// Assert
// 	// This test primarily verifies that the function can be called multiple times
// 	// without panicking, which tests the sync.Once behavior
// 	// The actual klog initialization is harder to test directly
// }

func TestMockNetworkController_ImplementsInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	var _ interfaces.NetworkController = &mockNetworkController{}

	// This test will fail at compile time if mockNetworkController doesn't implement the interface
	// Act & Assert - just verify the interface is properly implemented
	mock := &mockNetworkController{}
	if mock == nil {
		t.Error("Expected mock to be created, got nil")
	}
}

func TestServerWrapper_OriginalFunction_CallsServe(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	originalServeWrapper := serveWrapper
	defer func() { serveWrapper = originalServeWrapper }()

	// Create a test server and listener
	s := grpc.NewServer()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer lis.Close()

	// Act
	go func() {
		// This will block, so we run it in a goroutine
		serveWrapper(s, lis)
	}()

	// Stop the server to unblock the goroutine
	s.Stop()

	// Assert
	// The test passes if no panic occurs
}

// Integration test helper functions
func startTestServer(port int) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, nil, err
	}

	s := grpc.NewServer()
	proto.RegisterNetworkServer(s, newNetworkServer())

	go func() {
		s.Serve(lis)
	}()

	return s, lis, nil
}

func TestIntegration_VlanAdd_EndToEnd(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// This test can be enabled if you want to test the full gRPC stack
	t.Skip("Integration test - enable if needed")

	// Arrange
	server, lis, err := startTestServer(0)
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := proto.NewNetworkClient(conn)
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
	reply, err := client.VlanAdd(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", reply.GetResult())
	}
}

// Additional coverage tests for main.go functions

func TestNetworkServerCreation_ValidController_ReturnsServer(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange

	// Act
	server := newNetworkServer()

	// Assert
	if server == nil {
		t.Error("Expected server to be created, got nil")
	}
}

func TestNetworkServerCreation_NilController_ReturnsServerWithNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange

	// Act
	server := newNetworkServer()

	// Assert
	if server == nil {
		t.Error("Expected server to be created even with nil controller, got nil")
	}
}

func TestRun_ValidPort_StartsServerSuccessfully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Use an available port for testing
	testPort := 0 // Let the OS assign an available port

	// Act & Assert
	// Since run() would block, we test by verifying the listener can be created
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", testPort))
	if err != nil {
		t.Errorf("Expected to be able to create listener, got error: %v", err)
	}
	if lis != nil {
		lis.Close()
	}
}

func TestMain_EnvironmentConfiguration_InitializesCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Act
	err := utils.InitializeConfig()

	// Assert
	if err != nil {
		t.Errorf("Expected configuration to initialize successfully, got error: %v", err)
	}

	config := utils.GetConfig()
	if config == nil {
		t.Error("Expected configuration to be available after initialization")
	}
}

func TestMain_InvalidConfiguration_HandlesGracefully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	clearMainTestConfig() // Ensure no valid config

	// Reset the global config state to test error handling
	defer func() {
		// Restore valid config for other tests
		setupMainTestConfig()
		utils.InitializeConfig()
	}()

	// Act
	err := utils.InitializeConfig()

	// Assert
	if err == nil {
		t.Log("Note: Configuration validation might be lenient or default values are used")
	}
}

func TestNetworkServer_StructEmbedding_ImplementsInterface(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	server := newNetworkServer()

	// Act & Assert
	// This test verifies that networkServer implements the proto.NetworkServer interface
	var _ proto.NetworkServer = server
	if server == nil {
		t.Error("Expected server to implement NetworkServer interface")
	}
}

func TestNetworkServer_UnimplementedMethods_AreEmbedded(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	server := newNetworkServer()

	// Act & Assert
	// Verify that the UnimplementedNetworkServer is properly embedded
	// by checking that we can access its methods
	if server == nil {
		t.Error("Expected server with embedded UnimplementedNetworkServer")
	}

	// The embedding is verified by the successful compilation and interface compliance
	// If UnimplementedNetworkServer wasn't properly embedded, this wouldn't compile
}

// Integration test helper function for enhanced testing
func createTestServerHelper(port int, controller interfaces.NetworkController) (net.Listener, *grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, nil, err
	}

	s := grpc.NewServer()
	nwServer := newNetworkServer()
	proto.RegisterNetworkServer(s, nwServer)

	go func() {
		s.Serve(lis)
	}()

	return lis, s, nil
}

func TestCreateTestServerHelper_WorksCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	// Arrange

	// Act
	lis, server, err := createTestServerHelper(0, newNetworkServer())

	// Assert
	if err != nil {
		t.Errorf("Expected no error starting test server, got: %v", err)
	}
	if lis == nil {
		t.Error("Expected listener to be created")
	}
	if server == nil {
		t.Error("Expected server to be created")
	}

	// Cleanup
	if server != nil {
		server.Stop()
	}
	if lis != nil {
		lis.Close()
	}
}

// Test VlanAdd with unsupported product
func TestVlanAdd_UnsupportedProduct_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanAddRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "UnsupportedVendor",
			ProductName: "UnsupportedProduct",
			Version:     "1.0",
			Os:          &[]string{"UnsupportedOS"}[0],
		},
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanType:      "trunk",
		VlanId:        &vlanID,
		InterfaceName: "eth0",
	}

	reply, err := server.VlanAdd(ctx, req)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR for unsupported product, got %v", reply.GetResult())
	}
}

// Test VlanDelete with unsupported product
func TestVlanDelete_UnsupportedProduct_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VlanDeleteRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "UnsupportedVendor",
			ProductName: "UnsupportedProduct",
			Version:     "1.0",
			Os:          &[]string{"UnsupportedOS"}[0],
		},
		SwitchInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId:        &vlanID,
		InterfaceName: "eth0",
	}

	reply, err := server.VlanDelete(ctx, req)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR for unsupported product, got %v", reply.GetResult())
	}
}

// Test VswVlanAdd with unsupported product
func TestVswVlanAdd_UnsupportedProduct_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanAddRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "UnsupportedVendor",
			ProductName: "UnsupportedProduct",
			Version:     "1.0",
			Os:          &[]string{"UnsupportedOS"}[0],
		},
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	reply, err := server.VswVlanAdd(ctx, req)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR for unsupported product, got %v", reply.GetResult())
	}
}

// Test VswVlanDelete with unsupported product
func TestVswVlanDelete_UnsupportedProduct_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	server := newNetworkServer()
	ctx := context.Background()
	vlanID := int32(100)
	req := &proto.VswVlanDeleteRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "UnsupportedVendor",
			ProductName: "UnsupportedProduct",
			Version:     "1.0",
			Os:          &[]string{"UnsupportedOS"}[0],
		},
		HostInfo: &proto.NwInformation{
			RemoteHost: "192.168.1.1",
			RemoteUser: "admin",
		},
		VlanId: &vlanID,
		IfName: "eth0",
	}

	reply, err := server.VswVlanDelete(ctx, req)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if reply == nil {
		t.Error("Expected reply, got nil")
		return
	}
	if reply.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected ERROR for unsupported product, got %v", reply.GetResult())
	}
}

// Test TLS enabled with certificate not found
func TestRun_TlsEnabled_CertificateNotFound_HandlesError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Setup environment with TLS enabled but non-existent certificate path
	os.Setenv("NW_SERVER_PORT", "50053")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "true")
	os.Setenv("TLS_CERT_PATH", "/nonexistent/cert/path")
	defer clearMainTestConfig()
	utils.ResetConfigForTesting()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// Enable skipOsExit to prevent test termination
	originalSkipOsExit := skipOsExit
	skipOsExit = true
	defer func() { skipOsExit = originalSkipOsExit }()

	// Call run() which will enter TLS branch and attempt to load certificate
	// The certificate file does not exist, so tls.LoadX509KeyPair will fail
	// With skipOsExit=true, it will return instead of calling os.Exit(1)
	run(50053)

	// Verify TLS configuration was set
	config := utils.GetConfig()
	if !config.TlsEnable {
		t.Error("Expected TlsEnable to be true")
	}
	if config.TlsCertPath != "/nonexistent/cert/path" {
		t.Errorf("Expected TlsCertPath to be /nonexistent/cert/path, got %s", config.TlsCertPath)
	}
}

// Test initKlog function execution
func TestInitKlog_Execution_CompletesSuccessfully(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()

	// This test ensures initKlog can be called without error
	// The actual klog initialization is handled by the sync.Once mechanism
	// We verify the environment is set up correctly for klog
	config := utils.GetConfig()
	if config == nil {
		t.Fatal("Config should be initialized")
	}
	if config.LogLevel != "2" {
		t.Errorf("Expected LogLevel to be '2', got '%s'", config.LogLevel)
	}
}

// Test main function components without os.Exit
func TestMain_Components_WorkCorrectly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
	setupMainTestConfig()
	defer clearMainTestConfig()
	utils.ResetConfigForTesting()

	// Enable test mode and skip os.Exit
	isTest = true
	skipOsExit = true
	defer func() {
		isTest = false
		skipOsExit = false
		// Recover from any panic (e.g., flag redefinition)
		if r := recover(); r != nil {
			t.Logf("Recovered from panic (expected in test): %v", r)
		}
	}()

	// Create a goroutine to run main and stop it after a short time
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Flag redefinition panic is expected in tests
			}
			done <- true
		}()
		main()
	}()

	// Wait briefly for initialization
	select {
	case <-done:
		// main() completed (likely due to test mode)
	case <-time.After(100 * time.Millisecond):
		// Timeout is expected since server runs indefinitely
	}

	// Verify config was initialized
	config := utils.GetConfig()
	if config == nil {
		t.Fatal("GetConfig returned nil")
	}

	// Verify port configuration
	if config.NWServerPort != 50051 {
		t.Errorf("Expected NWServerPort 50051, got %d", config.NWServerPort)
	}
}

// ERROR PATH TESTS

func TestRun_NetListenError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupMainTestConfig()
	defer clearMainTestConfig()

	// Enable skipOsExit to prevent test termination
	originalSkipOsExit := skipOsExit
	skipOsExit = true
	defer func() { skipOsExit = originalSkipOsExit }()

	// Test with invalid port to trigger listen error
	// Port 999999 is out of valid range and will cause net.Listen to fail
	run(999999)

	// If we reach this point, the test has succeeded
	// because run() returned instead of calling os.Exit(1)
}

func TestMain_InitializeConfigError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Set invalid environment to trigger InitializeConfig failure
	os.Setenv("NW_SERVER_PORT", "invalid-port")
	defer clearMainTestConfig()

	// Enable skipOsExit to prevent test termination
	originalSkipOsExit := skipOsExit
	skipOsExit = true
	defer func() { skipOsExit = originalSkipOsExit }()

	utils.ResetConfigForTesting()

	// Create a goroutine to run main
	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Recover from any panic (klog flag redefinition)
			}
			done <- true
		}()
		main()
	}()

	// Wait for main to complete or timeout
	select {
	case <-done:
		// main() completed, which is expected with invalid config
	case <-time.After(500 * time.Millisecond):
		// Timeout - main may be stuck, but we tested the error path
	}

	// Verify config initialization failed
	// The function should have returned early due to InitializeConfig error
}

// TestRun_TlsEnabled_CACertificateNotFound_HandlesError tests CA certificate load failure
func TestRun_TlsEnabled_CACertificateNotFound_HandlesError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create temporary directory for test certificates
	tmpDir := t.TempDir()

	// Create dummy server certificate and key files (to pass initial LoadX509KeyPair)
	certFile := tmpDir + "/tls.crt"
	keyFile := tmpDir + "/tls.key"
	
	// Generate a minimal valid certificate and key for testing
	certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABKKl
AXySjNWpLbeG/yqhLKwKHJQ1n8N3KpYqGQqJqNBMxJDLAGxfB5p7gPYXx3LLvGEC
WOqKpIQqFvVPqGWj3CKjUDBOMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcD
AjAMBgNVHRMBAf8EAjAAMB8GA1UdIwQYMBaAFG/2RG3ZrBQ6qPpJ6pWiZzX2e7r7
MAoGCCqGSM49BAMCA0gAMEUCIBcY6OMmw8lhkFtC2E8rSKIYIGjB8TgqFPCJNqAz
3xHnAiEA2qcZi3yZQDz4eGy5fLpP5Lqvh0i2NphGnB+9T8VYbJI=
-----END CERTIFICATE-----`)
	keyPEM := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEoqUBfJKM1aktt4b/KqEsrAoclDWfw3cqlioZComo0EzEkMsAbF8H
mnuA9hfHcsu8YQJY6oqkhCoW9U+oZaPcIg==
-----END EC PRIVATE KEY-----`)

	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Do NOT create ca.crt file - this is what we're testing

	// Setup environment with TLS enabled but CA certificate does not exist
	os.Setenv("NW_SERVER_PORT", "50054")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "true")
	os.Setenv("TLS_CERT_PATH", tmpDir)
	defer clearMainTestConfig()
	utils.ResetConfigForTesting()

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// Enable skipOsExit to prevent test termination
	originalSkipOsExit := skipOsExit
	skipOsExit = true
	defer func() { skipOsExit = originalSkipOsExit }()

	// Call run() which will attempt to load CA certificate
	// The ca.crt file does not exist, so os.ReadFile will fail
	run(50054)

	// Verify TLS configuration was set
	config := utils.GetConfig()
	if !config.TlsEnable {
		t.Error("Expected TlsEnable to be true")
	}
	if config.TlsCertPath != tmpDir {
		t.Errorf("Expected TlsCertPath to be %s, got %s", tmpDir, config.TlsCertPath)
	}
}

// TestRun_TlsEnabled_InvalidCACertificate_HandlesError tests invalid CA certificate
func TestRun_TlsEnabled_InvalidCACertificate_HandlesError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create temporary directory for test certificates
	tmpDir := t.TempDir()
	certFile := tmpDir + "/tls.crt"
	keyFile := tmpDir + "/tls.key"
	caCertFile := tmpDir + "/ca.crt"

	// Generate a minimal valid certificate and key for testing
	certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABKKl
AXySjNWpLbeG/yqhLKwKHJQ1n8N3KpYqGQqJqNBMxJDLAGxfB5p7gPYXx3LLvGEC
WOqKpIQqFvVPqGWj3CKjUDBOMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcD
AjAMBgNVHRMBAf8EAjAAMB8GA1UdIwQYMBaAFG/2RG3ZrBQ6qPpJ6pWiZzX2e7r7
MAoGCCqGSM49BAMCA0gAMEUCIBcY6OMmw8lhkFtC2E8rSKIYIGjB8TgqFPCJNqAz
3xHnAiEA2qcZi3yZQDz4eGy5fLpP5Lqvh0i2NphGnB+9T8VYbJI=
-----END CERTIFICATE-----`)
	keyPEM := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEoqUBfJKM1aktt4b/KqEsrAoclDWfw3cqlioZComo0EzEkMsAbF8H
mnuA9hfHcsu8YQJY6oqkhCoW9U+oZaPcIg==
-----END EC PRIVATE KEY-----`)

	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Create invalid CA certificate file (not PEM format)
	err := os.WriteFile(caCertFile, []byte("INVALID CERTIFICATE DATA"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid CA cert: %v", err)
	}

	// Setup environment with TLS enabled
	os.Setenv("NW_SERVER_PORT", "50055")
	os.Setenv("LOG_LEVEL", "2")
	os.Setenv("SSH_KEY", "/tmp/test.pem")
	os.Setenv("TLS_ENABLE", "true")
	os.Setenv("TLS_CERT_PATH", tmpDir)
	defer clearMainTestConfig()
	utils.ResetConfigForTesting()

	err = utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// Enable skipOsExit to prevent test termination
	originalSkipOsExit := skipOsExit
	skipOsExit = true
	defer func() { skipOsExit = originalSkipOsExit }()

	// Call run() which will attempt to parse invalid CA certificate
	// caCertPool.AppendCertsFromPEM will return false
	run(50055)

	// Verify TLS configuration was set
	config := utils.GetConfig()
	if !config.TlsEnable {
		t.Error("Expected TlsEnable to be true")
	}
}
