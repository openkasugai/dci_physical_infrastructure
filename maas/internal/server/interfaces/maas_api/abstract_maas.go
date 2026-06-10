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
	"encoding/json"
	"errors"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/interfaces" // import for MaaS API interface
	"maas_module/internal/server/utils"

	"k8s.io/klog/v2"
)

// AbstractMaas is an abstract struct that implements common methods for interacting with the Canonical MaaS API.
type AbstractMaas struct {
	API    interfaces.MaasAPI
	Logger klog.Logger
}

// Success checks if the HTTP status code indicates a successful response
func (b *AbstractMaas) Success(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// HTTPError checks if the HTTP status code indicates an error and returns a custom error if it does
func (b *AbstractMaas) HTTPError(statusCode int, result []byte) error {
	var err error
	if !b.Success(statusCode) {
		err = &utils.HttpError{StatusCode: statusCode, Message: string(result)}
		b.Logger.Error(err, err.Error())
	} else {
		err = nil
	}
	return err
}

// NewResponseCommon creates a common response structure
func (b *AbstractMaas) NewResponseCommon(statusCode int, respData []byte) response_body.ResbodyCommon {
	data := ""
	errorMessage := ""
	if statusCode < 200 || statusCode >= 300 { // not 2xx -> error status
		errorMessage = string(respData)
	} else { // success status
		data = string(respData)
	}

	return response_body.ResbodyCommon{
		HTTPStatus:   statusCode,
		ErrorMessage: errorMessage,
		RawJSONData:  data,
	}
}

// ExtractValue extracts a value from a JSON byte slice based on a given key.
func (b *AbstractMaas) ExtractValue(jsonBytes []byte, key string) (value interface{}, result bool) {
	var data interface{}
	err := json.Unmarshal(jsonBytes, &data)
	if err != nil {
		result = false
		return
	}

	value, result = b.FindValue(data, key)
	if !result {
		return
	}

	return
}

// FindValue recursively searches for a key in a nested data structure (map or slice).
func (b *AbstractMaas) FindValue(data interface{}, key string) (value interface{}, result bool) {
	switch v := data.(type) {
	case map[string]interface{}:
		if val, ok := v[key]; ok {
			value = val
			result = true
			return
		}
		for _, val := range v {
			if value, result = b.FindValue(val, key); result {
				return
			}
		}
	case []interface{}:
		for _, item := range v {
			if value, result = b.FindValue(item, key); result {
				return
			}
		}
	default:
		// Nothing to do.
	}

	result = false
	return
}

// GET method retrieves a resource and returns a response body or an error.
func (b *AbstractMaas) GET(ctx context.Context) (response_body.Resbody, error) {
	return nil, errors.New("not implementation")
}

// POST method takes a request body and returns a response body or an error.
func (b *AbstractMaas) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return nil, errors.New("not implementation")
}

// PUT method updates a resource and returns a response body or an error.
func (b *AbstractMaas) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return nil, errors.New("not implementation")
}

// DELETE method deletes a resource and returns a response body or an error.
func (b *AbstractMaas) DELETE(ctx context.Context) (response_body.Resbody, error) {
	return nil, errors.New("not implementation")
}
