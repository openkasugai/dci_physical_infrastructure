package interfaces

import (
	"context"

	proto "maas_module/api/proto" // import for gRPC protobuf
)

// MaasController defines the interface for the MaaS controller operations.
type MaasController interface {
	MachineRegister(ctx context.Context, in *proto.MachineRegisterRequest) (*proto.MachineRegisterResponse, error)
	MachineDelete(ctx context.Context, in *proto.MachineDeleteRequest) (*proto.MachineDeleteResponse, error)
	OsDeploy(ctx context.Context, in *proto.OsDeployRequest) (*proto.OsDeployResponse, error)
	OsRelease(ctx context.Context, in *proto.OsReleaseRequest) (*proto.OsReleaseResponse, error)
	VMCompose(ctx context.Context, in *proto.VmComposeRequest) (*proto.VmComposeResponse, error)
	VMDelete(ctx context.Context, in *proto.VmDeleteRequest) (*proto.VmDeleteResponse, error)
	MachineList(ctx context.Context, in *proto.MachineListRequest) (*proto.MachineListResponse, error)
	MachineShow(ctx context.Context, in *proto.MachineShowRequest) (*proto.MachineShowResponse, error)
	Cancel(ctx context.Context, in *proto.CancelRequest) (*proto.CancelResponse, error)
	PowerON(ctx context.Context, in *proto.PowerOnRequest) (*proto.PowerOnResponse, error)
	PowerOFF(ctx context.Context, in *proto.PowerOffRequest) (*proto.PowerOffResponse, error)
	KubeadmReset(ctx context.Context, in *proto.KubeadmResetRequest) (*proto.KubeadmResetResponse, error)
	KubeadmJoin(ctx context.Context, in *proto.KubeadmJoinRequest) (*proto.KubeadmJoinResponse, error)
	NetworkUpdate(ctx context.Context, in *proto.NetworkUpdateRequest) (*proto.NetworkUpdateResponse, error)
}
