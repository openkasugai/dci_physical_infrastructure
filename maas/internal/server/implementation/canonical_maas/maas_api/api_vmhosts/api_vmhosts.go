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

package api_vmhosts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_machines"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/interfaces/maas_api"
	"maas_module/internal/server/utils"
	"net/url"

	"k8s.io/klog/v2"
)

// VMhosts represents the API for managing VM hosts in Canonical MAAS.
type VMhosts struct {
	maas_api.AbstractMaas
}

// GET /vm-hosts/
func (v *VMhosts) GET(ctx context.Context) (response_body.Resbody, error) {
	klog.V(2).InfoS("start GET", "api", "vm-hosts/")
	defer func() {
		klog.V(2).InfoS("end GET", "api", "vm-hosts/")
	}()

	// execute API call to get VM hosts
	statusCode, data, err := v.API.APIExecute(ctx, "GET", "vm-hosts/", "")
	if err != nil {
		klog.V(2).InfoS("branch: VM hosts API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: VM hosts API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyGetVMHosts
	if v.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = res.UnmarshalJSON(data)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			v.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = v.NewResponseCommon(statusCode, data)
	return res, v.HTTPError(statusCode, data)
}

// POST /vm-hosts/
func (v *VMhosts) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "vm-hosts/")
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "vm-hosts/")
	}()

	// cast request body
	var req request_body.ReqbodyVMhosts
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyVMhosts); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "powerAddress", req.PowerAddress, "type", req.Type)

	// execute API call to create a VM host
	apiReqBody := fmt.Sprintf("power_address=%s&type=%s", req.PowerAddress, req.Type)
	statusCode, data, err := v.API.APIExecute(ctx, "POST", "vm-hosts/", apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: VM host creation API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: VM host creation API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyPostVMHost
	if v.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = json.Unmarshal(data, &res)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			v.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = v.NewResponseCommon(statusCode, data)
	return res, v.HTTPError(statusCode, data)
}

// VMhostHostID represents the API for managing a specific VM host in Canonical MAAS.
type VMhostHostID struct {
	maas_api.AbstractMaas
	HostID int
}

// DELETE /vm-hosts/{host_id}/
func (v *VMhostHostID) DELETE(ctx context.Context) (response_body.Resbody, error) {
	klog.V(2).InfoS("start DELETE", "api", "vm-hosts/{host_id}/", "hostID", v.HostID)
	defer func() {
		klog.V(2).InfoS("end DELETE", "api", "vm-hosts/{host_id}/", "hostID", v.HostID)
	}()

	// execute API call to delete a VM host
	statusCode, data, err := v.API.APIExecute(ctx, "DELETE", fmt.Sprintf("vm-hosts/%d/", v.HostID), "")
	if err != nil {
		klog.V(2).InfoS("branch: VM host deletion API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: VM host deletion API execution successful", "statusCode", statusCode)

	// generate response data
	res := v.NewResponseCommon(statusCode, data)
	return res, v.HTTPError(statusCode, data)
}

// VMhostCompose represents the API for composing a VM host in Canonical MAAS.
type VMhostCompose struct {
	VMhostHostID
}

// POST /vm-hosts/{host_id}/op-compose
func (v *VMhostCompose) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "vm-hosts/op-compose", "hostID", v.HostID)
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "vm-hosts/op-compose", "hostID", v.HostID)
	}()

	// cast request body
	var req request_body.ReqbodyVMhostCompose
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyVMhostCompose); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "cores", req.Cores, "hostName", req.HostName, "memory", req.Memory, "storage", req.Storage)

    // Idempotency check: Check if machine with same hostname already exists
    klog.V(2).InfoS("branch: checking if machine already exists", "hostname", req.HostName)
    if existingSystemID, exists, err := v.checkMachineExists(ctx, req.HostName); err != nil {
        klog.V(2).InfoS("branch: failed to check machine existence", "error", err.Error())
        return nil, err
    } else if exists {
        klog.V(2).InfoS("branch: machine with same hostname already exists", "hostname", req.HostName, "systemID", existingSystemID)
        // Return success response with existing machine's system_id
        res := response_body.ResbodyPostVMCompose{
            ResbodyCommon: v.NewResponseCommon(200, []byte(`{"system_id":"`+existingSystemID+`"}`)),
            SystemID:      existingSystemID,
        }
        return res, nil
    }

    klog.V(2).InfoS("branch: machine does not exist, proceeding with composition", "hostname", req.HostName)

	// execute API call to create a VM host
	apiReqBody := fmt.Sprintf("cores=%d&"+
		"hostname=%s&memory=%d&"+
		"storage=storage:%d&interfaces=%s",
		req.Cores, url.QueryEscape(req.HostName),
		req.Memory, req.Storage, req.Interfaces)
	statusCode, data, err := v.API.APIExecute(ctx, "POST", fmt.Sprintf("vm-hosts/%d/op-compose", v.HostID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: VM host compose API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: VM host compose API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyPostVMCompose
	if v.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = json.Unmarshal(data, &res)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			v.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = v.NewResponseCommon(statusCode, data)
	return res, v.HTTPError(statusCode, data)
}

// checkMachineExists checks if a machine with the given hostname already exists
// Returns (systemID, exists, error)
func (v *VMhostCompose) checkMachineExists(ctx context.Context, hostname string) (string, bool, error) {
    klog.V(2).InfoS("start checkMachineExists", "hostname", hostname)
    defer func() {
        klog.V(2).InfoS("end checkMachineExists", "hostname", hostname)
    }()

    // Create machines API instance to check existing machines
    machines := &api_machines.Machines{
        AbstractMaas: v.AbstractMaas,
    }

    // Get all machines
    getRes, err := machines.GET(ctx)
    if err != nil {
        klog.V(2).InfoS("branch: failed to get machines list", "error", err.Error())
        return "", false, err
    }

    // Cast response to correct type
    var responseBody response_body.ResbodyGetMachines
    var ok bool
    if responseBody, ok = getRes.(response_body.ResbodyGetMachines); !ok {
        err = &utils.RespError{Message: "response type is invalid"}
        klog.V(2).InfoS("branch: machine list response type invalid", "error", err)
        return "", false, err
    }

    // Check if machine with same hostname exists
    for _, item := range responseBody.Machines {
        if item.HostName == hostname {
            klog.V(2).InfoS("branch: found existing machine", "hostname", hostname, "systemID", item.SystemID)
            return item.SystemID, true, nil
        }
    }

    klog.V(2).InfoS("branch: no existing machine found", "hostname", hostname)
    return "", false, nil
}

// VMhostParameters represents the API for getting operation parameters for a VM host in Canonical MAAS.
type VMhostParameters struct {
	VMhostHostID
}

// POST /vm-hosts/{host_id}/op-parameters
func (v *VMhostParameters) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "vm-hosts/op-parameters", "hostID", v.HostID)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "vm-hosts/op-parameters", "hostID", v.HostID)
	}()

	// execute API call to get operation parameters for a VM host
	statusCode, data, err := v.API.APIExecute(ctx, "GET", fmt.Sprintf("vm-hosts/%d/op-parameters", v.HostID), "")
	if err != nil {
		klog.V(2).InfoS("branch: VM host parameters API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: VM host parameters API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyGetOpParameter
	if v.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = json.Unmarshal(data, &res)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			v.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = v.NewResponseCommon(statusCode, data)
	return res, v.HTTPError(statusCode, data)
}

// VMhostRefresh represents the API for refreshing a VM host in Canonical MAAS.
type VMhostRefresh struct {
	VMhostHostID
}

// POST /vm-hosts/{host_id}/op-refresh
func (v *VMhostRefresh) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "vm-hosts/op-refresh", "hostID", v.HostID)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "vm-hosts/op-refresh", "hostID", v.HostID)
	}()

	// execute API call to refresh a VM host
	statusCode, data, err := v.API.APIExecute(ctx, "POST", fmt.Sprintf("vm-hosts/%d/op-refresh", v.HostID), "")
	if err != nil {
		klog.V(2).InfoS("branch: VM host refresh API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: VM host refresh API execution successful", "statusCode", statusCode)

	// generate response data
	res := v.NewResponseCommon(statusCode, data)
	return res, v.HTTPError(statusCode, data)
}
