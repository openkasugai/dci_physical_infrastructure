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

	proto "network_module/api/proto" // import for gRPC protobuf
)

// interface of network controller
type NetworkController interface {
	VlanAdd(ctx context.Context, in *proto.VlanAddRequest) (*proto.VlanAddReply, error)
	VlanDelete(ctx context.Context, in *proto.VlanDeleteRequest) (*proto.VlanDeleteReply, error)
	VswVlanAdd(ctx context.Context, in *proto.VswVlanAddRequest) (*proto.VswVlanAddReply, error)
	VswVlanDelete(ctx context.Context, in *proto.VswVlanDeleteRequest) (*proto.VswVlanDeleteReply, error)
}
