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
