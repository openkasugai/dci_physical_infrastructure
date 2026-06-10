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

package api_ipaddresses

import (
	"context"
	"errors"
	"fmt"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/interfaces/maas_api"
	"net/url"

	"k8s.io/klog/v2"
)

// IPAddressReserve represents the API for reserving IP addresses in Canonical MAAS.
type IPAddressReserve struct {
	maas_api.AbstractMaas
}

// POST /ipaddresses/op-reserve
func (i *IPAddressReserve) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "ipaddresses/op-reserve")
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "ipaddresses/op-reserve")
	}()

	// cast request body
	var req request_body.ReqbodyIPAddressReserve
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyIPAddressReserve); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "ip", req.IP, "subnet", req.Subnet)

	// execute API call to reserve IP address
	apiReqBody := fmt.Sprintf("ip=%s&subnet=%s", url.QueryEscape(req.IP), url.QueryEscape(req.Subnet))
	statusCode, data, err := i.API.APIExecute(ctx, "POST", "ipaddresses/op-reserve", apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: IP address reserve API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: IP address reserve API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyIPAddressReserve
	if i.Success(statusCode) {
		klog.V(2).InfoS("branch: IP address reserved successfully")
	}

	// set common response data
	res.ResbodyCommon = i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}

// IPAddressRelease represents the API for releasing IP addresses in Canonical MAAS.
type IPAddressRelease struct {
	maas_api.AbstractMaas
}

// POST /ipaddresses/op-release
func (i *IPAddressRelease) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "ipaddresses/op-release")
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "ipaddresses/op-release")
	}()

	// cast request body
	var req request_body.ReqbodyIPAddressRelease
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyIPAddressRelease); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "ip", req.IP, "force", req.Force)

	// execute API call to release IP address
	apiReqBody := fmt.Sprintf("ip=%s&force=%t", url.QueryEscape(req.IP), req.Force)
	statusCode, data, err := i.API.APIExecute(ctx, "POST", "ipaddresses/op-release", apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: IP address release API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: IP address release API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyIPAddressRelease
	if i.Success(statusCode) {
		klog.V(2).InfoS("branch: IP address released successfully")
	}

	// set common response data
	res.ResbodyCommon = i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}
