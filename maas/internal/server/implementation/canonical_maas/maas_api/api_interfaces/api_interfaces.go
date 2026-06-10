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

package api_interfaces

import (
	"context"
	"errors"
	"fmt"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/interfaces/maas_api"
	"maas_module/internal/server/utils"

	"k8s.io/klog/v2"
)

// Interfaces represents the API for managing interfaces in Canonical MAAS.
type Interfaces struct {
	maas_api.AbstractMaas
	SystemID string
}

// GET /interfaces/{system_id}/
func (i *Interfaces) GET(ctx context.Context) (response_body.Resbody, error) {
	klog.V(2).InfoS("start GET", "api", "interfaces/{system_id}/", "systemID", i.SystemID)
	defer func() {
		klog.V(2).InfoS("end GET", "api", "interfaces/{system_id}/", "systemID", i.SystemID)
	}()

	// execute API call to get interfaces
	statusCode, data, err := i.API.APIExecute(ctx, "GET", fmt.Sprintf(`nodes/%s/interfaces/`, i.SystemID), "")
	if err != nil {
		klog.V(2).InfoS("branch: interfaces API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: interfaces API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyGetInterfaces
	if i.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = res.UnmarshalJSON(data)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			i.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// generate response data
	res.ResbodyCommon = i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}

// InterfaceLinkSubnet represents the API for linking a subnet to an interface in Canonical MAAS.
type InterfaceLinkSubnet struct {
	Interfaces
	InterfaceID int
}

// POST /node/{system_id}/interface/{interface_id}/op-link_subnet
func (i *InterfaceLinkSubnet) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "interface/op-link_subnet", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "interface/op-link_subnet", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	}()

	// cast request body
	var req request_body.ReqbodyIFLinkSubnet
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyIFLinkSubnet); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "mode", req.Mode, "subnetID", req.SubnetID)

	// execute API call to link subnet
	apiReqBody := fmt.Sprintf("mode=%s&subnet=%d", req.Mode, req.SubnetID)
	statusCode, data, err := i.API.APIExecute(ctx, "POST", fmt.Sprintf(`nodes/%s/interfaces/%d/op-link_subnet`, i.SystemID, i.InterfaceID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: link subnet API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: link subnet API execution successful", "statusCode", statusCode)

	// generate response data
	res := i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}

// InterfaceDisconnect represents the API for disconnecting an interface in Canonical MAAS.
type InterfaceDisconnect struct {
	Interfaces
	InterfaceID int
}

// POST /node/{system_id}/interface/{interface_id}/op-disconnect
func (i *InterfaceDisconnect) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "interface/op-disconnect", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "interface/op-disconnect", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	}()

	// execute API call to disconnect interface
	statusCode, data, err := i.API.APIExecute(ctx, "POST", fmt.Sprintf(`nodes/%s/interfaces/%d/op-disconnect`, i.SystemID, i.InterfaceID), "")
	if err != nil {
		klog.V(2).InfoS("branch: disconnect interface API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: disconnect interface API execution successful", "statusCode", statusCode)

	// generate response data
	res := i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}

// InterfaceAddTag represents the API for adding a tag to an interface in Canonical MAAS.
type InterfaceAddTag struct {
	Interfaces
	InterfaceID int
}

// POST /node/{system_id}/interface/{interface_id}/op-add_tag
func (i *InterfaceAddTag) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "interface/op-add_tag", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "interface/op-add_tag", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	}()

	// cast request body
	var req request_body.ReqbodyInterfaceTag
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyInterfaceTag); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "tag", req.Tag)

	// execute API call to add tag
	apiReqBody := fmt.Sprintf("tag=%s", req.Tag)
	statusCode, data, err := i.API.APIExecute(ctx, "POST", fmt.Sprintf(`nodes/%s/interfaces/%d/op-add_tag`, i.SystemID, i.InterfaceID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: add tag API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: add tag API execution successful", "statusCode", statusCode)

	// generate response data
	res := i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}

// InterfaceRemoveTag represents the API for removing a tag from an interface in Canonical MAAS.
type InterfaceRemoveTag struct {
	Interfaces
	InterfaceID int
}

// POST /node/{system_id}/interface/{interface_id}/op-remove_tag
func (i *InterfaceRemoveTag) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "interface/op-remove_tag", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "interface/op-remove_tag", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	}()

	// cast request body
	var req request_body.ReqbodyInterfaceTag
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyInterfaceTag); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "tag", req.Tag)

	// execute API call to remove tag
	apiReqBody := fmt.Sprintf("tag=%s", req.Tag)
	statusCode, data, err := i.API.APIExecute(ctx, "POST", fmt.Sprintf(`nodes/%s/interfaces/%d/op-remove_tag`, i.SystemID, i.InterfaceID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: remove tag API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: remove tag API execution successful", "statusCode", statusCode)

	// generate response data
	res := i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}

// InterfaceUpdate represents the API for updating an interface in Canonical MAAS.
type InterfaceUpdate struct {
	Interfaces
	InterfaceID int
}

// PUT /nodes/{system_id}/interfaces/{interface_id}/
func (i *InterfaceUpdate) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start PUT", "api", "nodes/{system_id}/interfaces/{interface_id}/", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end PUT", "api", "nodes/{system_id}/interfaces/{interface_id}/", "systemID", i.SystemID, "interfaceID", i.InterfaceID)
	}()

	// cast request body
	var req request_body.ReqbodyInterfaceUpdate
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyInterfaceUpdate); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "name", req.Name)

	// execute API call to update interface name
	apiReqBody := fmt.Sprintf("name=%s", req.Name)
	statusCode, data, err := i.API.APIExecute(ctx, "PUT", fmt.Sprintf(`nodes/%s/interfaces/%d/`, i.SystemID, i.InterfaceID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: update interface API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: update interface API execution successful", "statusCode", statusCode)

	// generate response data
	res := i.NewResponseCommon(statusCode, data)
	return res, i.HTTPError(statusCode, data)
}
