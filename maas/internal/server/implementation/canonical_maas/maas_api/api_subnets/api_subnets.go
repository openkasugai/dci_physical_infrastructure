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

package api_subnets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/interfaces/maas_api"
	"maas_module/internal/server/utils"
	"net/url"

	"k8s.io/klog/v2"
)

// Subnets represents the API for managing subnets in Canonical MAAS.
type Subnets struct {
	maas_api.AbstractMaas
}

// GET /subnets/
func (s *Subnets) GET(ctx context.Context) (response_body.Resbody, error) {
	klog.V(2).InfoS("start GET", "api", "subnets/")
	defer func() {
		klog.V(2).InfoS("end GET", "api", "subnets/")
	}()

	// execute API call to get subnets
	statusCode, data, err := s.API.APIExecute(ctx, "GET", "subnets/", "")
	if err != nil {
		klog.V(2).InfoS("branch: subnets API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: subnets API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyGetSubnets
	if s.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = res.UnmarshalJSON(data)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			s.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = s.NewResponseCommon(statusCode, data)
	return res, s.HTTPError(statusCode, data)
}

// POST /subnets/
func (s *Subnets) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "subnets/")
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "subnets/")
	}()

	// cast request body
	var req request_body.ReqbodySubnets
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodySubnets); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "cidr", req.Cidr, "fabricID", req.FabricID, "vid", req.Vid)

	// execute API call to create a subnet
	apiReqBody := fmt.Sprintf("cidr=%s&fabric=%v&vid=%v", url.QueryEscape(req.Cidr), req.FabricID, req.Vid)
	statusCode, data, err := s.API.APIExecute(ctx, "POST", "subnets/", apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: subnet creation API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: subnet creation API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyPostSubnets
	if s.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = json.Unmarshal(data, &res)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			s.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = s.NewResponseCommon(statusCode, data)
	return res, s.HTTPError(statusCode, data)
}

// SubnetUnreservedIPRanges represents the API for getting unreserved IP ranges in a subnet.
type SubnetUnreservedIPRanges struct {
	maas_api.AbstractMaas
	SubnetID int
}

// GET /subnets/{id}/op-unreserved_ip_ranges
func (s *SubnetUnreservedIPRanges) GET(ctx context.Context) (response_body.Resbody, error) {
	klog.V(2).InfoS("start GET", "api", "subnets/{id}/op-unreserved_ip_ranges", "subnetID", s.SubnetID)
	defer func() {
		klog.V(2).InfoS("end GET", "api", "subnets/{id}/op-unreserved_ip_ranges")
	}()

	// execute API call to get unreserved IP ranges
	apiPath := fmt.Sprintf("subnets/%d/op-unreserved_ip_ranges", s.SubnetID)
	statusCode, data, err := s.API.APIExecute(ctx, "GET", apiPath, "")
	if err != nil {
		klog.V(2).InfoS("branch: unreserved IP ranges API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: unreserved IP ranges API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodySubnetUnreservedIPRanges
	if s.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = res.UnmarshalJSON(data)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			s.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = s.NewResponseCommon(statusCode, data)
	return res, s.HTTPError(statusCode, data)
}
