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

package maas_api

import (
	"context"

	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
)

// BasisMaasAPI defines the interface for the basic operations of the Canonical MAAS API.
type BasisMaasAPI interface {
	GET(ctx context.Context) (response_body.Resbody, error)
	POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error)
	PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error)
	DELETE(ctx context.Context) (response_body.Resbody, error)
}
