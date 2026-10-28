package interfaces

import (
	"context"

	proto "network_module/api/proto" // import for gRPC protobuf
)

// interface of network controller
type NetworkController interface {
	VlanAdd(ctx context.Context, in *proto.VlanAddRequest) (*proto.VlanAddReply, error)
	VlanDelete(ctx context.Context, in *proto.VlanDeleteRequest) (*proto.VlanDeleteReply, error)
	VswVlanAdd(ctx context.Context, in *proto.VswVlanAddRequest) (*proto.VswVlanAddReply, error)
	VswVlanDelete(ctx context.Context, in *proto.VswVlanDeleteRequest) (*proto.VswVlanDeleteReply, error)
}
