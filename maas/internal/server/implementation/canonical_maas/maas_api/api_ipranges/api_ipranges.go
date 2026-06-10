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

package api_ipranges

import (
	"context"
	"errors"
	"fmt"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/interfaces/maas_api"

	"k8s.io/klog/v2"
)

// IPranges represents the API for managing IP ranges in Canonical MAAS.
type IPranges struct {
	maas_api.AbstractMaas
}

// POST /ipranges/
func (i *IPranges) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "ipranges/")
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "ipranges/")
	}()

	// cast request body
	var req request_body.ReqbodyIPRanges
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyIPRanges); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "subnetID", req.SubnetID, "startIP", req.StartIP, "endIP", req.EndIP, "type", req.Type)

	// execute API call to create an IP range
	apiReqBody := fmt.Sprintf("subnet_id=%d&start_ip=%s&end_ip=%s&type=%s", req.SubnetID, req.StartIP, req.EndIP, req.Type)
	statusCode, data, err := i.API.APIExecute(ctx, "POST", "ipranges/", apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: IP range creation API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: IP range creation API execution successful", "statusCode", statusCode)

	// generate response data
	res := i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}
