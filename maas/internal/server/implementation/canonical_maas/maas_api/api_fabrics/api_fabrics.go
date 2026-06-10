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

package api_fabrics

import (
	"context"
	"encoding/json"

	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/interfaces/maas_api"
	"maas_module/internal/server/utils"

	"k8s.io/klog/v2"
)

// Fabrics represents the API for managing fabrics in Canonical MAAS.
type Fabrics struct {
	maas_api.AbstractMaas
}

// POST /fabrics/
func (f *Fabrics) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "fabrics/")
	defer func() {
		klog.V(2).InfoS("end POST", "api", "fabrics/")
	}()

	// execute API call to create a fabric
	statusCode, data, err := f.API.APIExecute(ctx, "POST", "fabrics/", "")
	if err != nil {
		klog.V(2).InfoS("branch: fabric creation API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: fabric creation API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyPostFabrics
	if f.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = json.Unmarshal(data, &res)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			f.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = f.NewResponseCommon(statusCode, data)
	return res, f.HTTPError(statusCode, data)
}
