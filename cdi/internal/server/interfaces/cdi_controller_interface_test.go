package interfaces

import (
	"context"
	"testing"

	proto "cdi_module/api/proto"
    common "common/api/proto"    // import of common protobuf
)

// Mock implementation for testing interface compliance
type mockCDIController struct{}

func (m *mockCDIController) MachineCreate(ctx context.Context, in *proto.MachineCreateRequest) (*proto.MachineCreateReply, error) {
	return &proto.MachineCreateReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}, nil
}

func (m *mockCDIController) MachineDestroy(ctx context.Context, in *proto.MachineDestroyRequest) (*proto.MachineDestroyReply, error) {
	return &proto.MachineDestroyReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}, nil
}

func (m *mockCDIController) MachineShow(ctx context.Context, in *proto.MachineShowRequest) (*proto.MachineShowReply, error) {
	return &proto.MachineShowReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
		Data:         `{"status": "active"}`,
	}, nil
}

func (m *mockCDIController) ResourceList(ctx context.Context, in *proto.ResourceListRequest) (*proto.ResourceListReply, error) {
	return &proto.ResourceListReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
		Data:         `{"resources": []}`,
	}, nil
}

func (m *mockCDIController) ResourceShow(ctx context.Context, in *proto.ResourceShowRequest) (*proto.ResourceShowReply, error) {
	return &proto.ResourceShowReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
		Data:         `{"resource": "test"}`,
	}, nil
}

func (m *mockCDIController) CardScaling(ctx context.Context, in *proto.CardScalingRequest) (*proto.CardScalingReply, error) {
	return &proto.CardScalingReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}, nil
}

// Test interface method signatures
func TestCDIController_InterfaceMethodSignatures_AreCorrect(t *testing.T) {
	// This test ensures all required methods exist with correct signatures
	// by attempting to assign method values

	var controller CDIController = &mockCDIController{}

	// Test all methods by calling them
	ctx := context.Background()

	// Test MachineCreate
	createReply, err := controller.MachineCreate(ctx, &proto.MachineCreateRequest{})
	if err != nil {
		t.Errorf("MachineCreate should not return error, got: %v", err)
	}
	if createReply == nil {
		t.Error("MachineCreate should return a reply")
	}

	// Test MachineDestroy
	destroyReply, err := controller.MachineDestroy(ctx, &proto.MachineDestroyRequest{})
	if err != nil {
		t.Errorf("MachineDestroy should not return error, got: %v", err)
	}
	if destroyReply == nil {
		t.Error("MachineDestroy should return a reply")
	}

	// Test MachineShow
	showReply, err := controller.MachineShow(ctx, &proto.MachineShowRequest{})
	if err != nil {
		t.Errorf("MachineShow should not return error, got: %v", err)
	}
	if showReply == nil {
		t.Error("MachineShow should return a reply")
	}

	// Test ResourceList
	listReply, err := controller.ResourceList(ctx, &proto.ResourceListRequest{})
	if err != nil {
		t.Errorf("ResourceList should not return error, got: %v", err)
	}
	if listReply == nil {
		t.Error("ResourceList should return a reply")
	}

	// Test ResourceShow
	resourceReply, err := controller.ResourceShow(ctx, &proto.ResourceShowRequest{})
	if err != nil {
		t.Errorf("ResourceShow should not return error, got: %v", err)
	}
	if resourceReply == nil {
		t.Error("ResourceShow should return a reply")
	}

	// Test CardScaling
	scalingReply, err := controller.CardScaling(ctx, &proto.CardScalingRequest{})
	if err != nil {
		t.Errorf("CardScaling should not return error, got: %v", err)
	}
	if scalingReply == nil {
		t.Error("CardScaling should return a reply")
	}
}

func TestCDIController_MockImplementation_WorksCorrectly(t *testing.T) {
	mock := &mockCDIController{}

	// Verify it implements the interface
	var _ CDIController = mock

	// Test a specific method
	ctx := context.Background()
	reply, err := mock.MachineShow(ctx, &proto.MachineShowRequest{})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if reply == nil {
		t.Fatal("Expected reply to be returned")
	}

	if reply.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected result SUCCESS, got %v", reply.GetResult())
	}

	if reply.GetData() != `{"status": "active"}` {
		t.Errorf("Expected specific data, got %s", reply.GetData())
	}
}
