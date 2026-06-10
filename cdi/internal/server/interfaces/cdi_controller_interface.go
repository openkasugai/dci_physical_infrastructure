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
