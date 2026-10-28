package interfaces

import "context"

// interface of API
type API interface {
	APIExecuteUserAuth(ctx context.Context, method string, url string, apiname string, loginUser string, loginPass string, queryParameter string) (jsonData interface{}, err error)
	APIExecuteJWTAUth(ctx context.Context, method string, url string, apiname string, jwt string, queryParameter string) (jsonData interface{}, err error)
}
