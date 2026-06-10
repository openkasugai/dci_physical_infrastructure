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
