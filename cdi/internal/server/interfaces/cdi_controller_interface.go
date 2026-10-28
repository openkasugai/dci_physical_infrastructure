package interfaces

import (
	"context"

	proto "cdi_module/api/proto" // import for gRPC protobuf
)

// interface of CDI controller
type CDIController interface {
	MachineCreate(ctx context.Context, in *proto.MachineCreateRequest) (*proto.MachineCreateReply, error)
	MachineDestroy(ctx context.Context, in *proto.MachineDestroyRequest) (*proto.MachineDestroyReply, error)
	MachineShow(ctx context.Context, in *proto.MachineShowRequest) (*proto.MachineShowReply, error)
	ResourceList(ctx context.Context, in *proto.ResourceListRequest) (*proto.ResourceListReply, error)
	ResourceShow(ctx context.Context, in *proto.ResourceShowRequest) (*proto.ResourceShowReply, error)
	CardScaling(ctx context.Context, in *proto.CardScalingRequest) (*proto.CardScalingReply, error)
}
