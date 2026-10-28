package interfaces

import (
	"context"
)

// interface of Canonical Maas API
type MaasAPI interface {

	// API execution
	APIExecute(ctx context.Context, method string, apiname string, reqBody string, withRetry bool) (statusCode int, jsonData []byte, err error)
}
