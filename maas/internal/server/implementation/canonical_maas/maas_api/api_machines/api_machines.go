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

package api_machines

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

// Machines represents the API for managing machines in Canonical MAAS.
type Machines struct {
	maas_api.AbstractMaas
}

// GET /machines/
func (m *Machines) GET(ctx context.Context) (response_body.Resbody, error) {
	klog.V(2).InfoS("start GET", "api", "machines/")
	defer func() {
		klog.V(2).InfoS("end GET", "api", "machines/")
	}()

	// execute API call to get machines
	statusCode, data, err := m.API.APIExecute(ctx, "GET", "machines/", "")
	if err != nil {
		klog.V(2).InfoS("branch: API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyGetMachines
	if m.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = res.UnmarshalJSON(data)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			m.AbstractMaas.Logger.Error(err, "Failed to unmarshal machines response")
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// POST /machines/
func (m *Machines) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "machines/")
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST")
	}()

	// cast request body
	var req request_body.ReqbodyMachines
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyMachines); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "hostname", req.Hostname, "architecture", req.Architecture)

	// Idempotency check: Check if machine with same hostname already exists
	klog.V(2).InfoS("branch: checking if machine already exists", "hostname", req.Hostname)
	getRes, err := m.GET(ctx)
	if err != nil {
		return nil, err
	}
	var responseBody response_body.ResbodyGetMachines
	if responseBody, ok = getRes.(response_body.ResbodyGetMachines); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: machine list response type invalid", "error", err)
		return nil, err
	}
	for _, item := range responseBody.Machines {
		if item.HostName == req.Hostname {
			klog.V(2).InfoS("branch: machine with same hostname already exists", "hostname", req.Hostname, "systemID", item.SystemID)
			res := response_body.ResbodyPostMachines{
				ResbodyCommon: m.NewResponseCommon(200, []byte(`{"system_id":"`+item.SystemID+`"}`)),
				SystemID:      item.SystemID,
			}
			return res, nil
		}
	}

	klog.V(2).InfoS("branch: machine does not exist, proceeding with creation", "hostname", req.Hostname)

	// generate request body
	apiReqBody := fmt.Sprintf(
		"architecture=%s&mac_addresses=%s&hostname=%s&"+
			"commission=%v&enable_ssh=%v&power_type=%s&"+
			"power_parameters={ \"power_address\": \"%s\", \"power_user\": \"%s\", \"power_pass\": \"%s\" }",
		url.QueryEscape(req.Architecture),
		url.QueryEscape(req.MACAddresses),
		url.QueryEscape(req.Hostname),
		req.Commission, req.EnableSSH,
		url.QueryEscape(req.PowerType),
		url.QueryEscape(req.PowerAddress),
		url.QueryEscape(req.PowerUser),
		url.QueryEscape(req.PowerPass),
	)

	klog.V(2).InfoS("branch: API request body generated")

	// execute API call to create a machine
	statusCode, data, err := m.API.APIExecute(ctx, "POST", "machines/", apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyPostMachines
	if m.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = json.Unmarshal(data, &res)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			m.AbstractMaas.Logger.Error(err, "Failed to unmarshal machine creation response")
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// MachineSystemID represents the API for managing a specific machine by its system ID in Canonical MAAS.
type MachineSystemID struct {
	maas_api.AbstractMaas
	SystemID string
}

// GET /machines/{system_id}/
func (m *MachineSystemID) GET(ctx context.Context) (response_body.Resbody, error) {
	klog.V(2).InfoS("start GET", "api", "machines/{system_id}/", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end GET", "api", "machines/{system_id}/", "systemID", m.SystemID)
	}()

	// execute API call to get machine details
	statusCode, data, err := m.API.APIExecute(ctx, "GET", fmt.Sprintf("machines/%s/", m.SystemID), "")
	if err != nil {
		klog.V(2).InfoS("branch: API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: API execution successful", "statusCode", statusCode)

	// parse response data
	var res response_body.ResbodyGetMachine
	if m.Success(statusCode) {
		klog.V(2).InfoS("branch: parsing successful response")
		err = res.UnmarshalJSON(data)
		if err != nil {
			klog.V(2).InfoS("branch: JSON unmarshal failed", "error", err.Error())
			err = &utils.RespError{Message: err.Error()}
			m.AbstractMaas.Logger.Error(err, err.Error())
			return nil, err
		}
	}

	// set common response data
	res.ResbodyCommon = m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// PUT /machines/{system_id}/
func (m *MachineSystemID) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start PUT", "api", "machines/{system_id}/", "systemID", m.SystemID)
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end PUT", "api", "machines/{system_id}/", "systemID", m.SystemID)
	}()

	// cast request body
	var req request_body.ReqbodyMachineUpdate
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyMachineUpdate); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "description", req.Description)

	// generate request body
	apiReqBody := fmt.Sprintf("description=%s", url.QueryEscape(req.Description))

	// execute API call to update a machine
	statusCode, data, err := m.API.APIExecute(ctx, "PUT", fmt.Sprintf("machines/%s/", m.SystemID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: API execution successful", "statusCode", statusCode)

	// set common response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// Helper function to get machine status
func (m *MachineSystemID) getMachineStatus(ctx context.Context) (string, error) {
	klog.V(2).InfoS("start getMachineStatus", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end getMachineStatus", "systemID", m.SystemID)
	}()

	getRes, err := m.GET(ctx)
	if err != nil {
		return "", err
	}
	var ok bool
	var responseBody response_body.ResbodyGetMachine
	if responseBody, ok = getRes.(response_body.ResbodyGetMachine); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: machine response type invalid", "error", err)
		return "", err
	}

	return responseBody.StatusName, nil
}

// Helper function to get machine power status
func (m *MachineSystemID) getMachinePowerStatus(ctx context.Context) (string, error) {
	klog.V(2).InfoS("start getMachinePowerStatus", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end getMachinePowerStatus", "systemID", m.SystemID)
	}()

	getRes, err := m.GET(ctx)
	if err != nil {
		return "", err
	}
	var ok bool
	var responseBody response_body.ResbodyGetMachine
	if responseBody, ok = getRes.(response_body.ResbodyGetMachine); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: machine response type invalid", "error", err)
		return "", err
	}

	return responseBody.PowerStatus, nil
}

// DELETE /machines/{system_id}/
func (m *MachineSystemID) DELETE(ctx context.Context) (response_body.Resbody, error) {
	klog.V(2).InfoS("start DELETE", "api", "machines/{system_id}/", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end DELETE", "api", "machines/{system_id}/", "systemID", m.SystemID)
	}()

	// execute API call to delete a machine
	statusCode, data, err := m.API.APIExecute(ctx, "DELETE", fmt.Sprintf("machines/%s/", m.SystemID), "")
	if err != nil {
		klog.V(2).InfoS("branch: API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: API execution successful", "statusCode", statusCode)

	// generate response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// MachineCommission represents the API for commissioning a machine in Canonical MAAS.
type MachineCommission struct {
	MachineSystemID
}

// POST /machines/{system_id}/op-commission
func (m *MachineCommission) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "machines/op-commission", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "machines/op-commission", "systemID", m.SystemID)
	}()

	// Idempotency check: Check if machine is already commissioning or ready
	klog.V(2).InfoS("branch: checking machine status for idempotency", "systemID", m.SystemID)
	status, err := m.getMachineStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status == "Commissioning" || status == "Ready" || status == "Testing" {
		klog.V(2).InfoS("branch: machine already commissioned or ready, returning success", "systemID", m.SystemID, "status", status)
		// Return success without executing commission
		res := m.NewResponseCommon(200, nil)
		return res, nil
	}

	klog.V(2).InfoS("branch: machine status allows commission", "systemID", m.SystemID, "status", status)

	// execute API call to commission a machine
	statusCode, data, err := m.API.APIExecute(ctx, "POST", fmt.Sprintf(`machines/%s/op-commission`, m.SystemID), "")
	if err != nil {
		klog.V(2).InfoS("branch: commission API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: commission API execution successful", "statusCode", statusCode)

	// generate response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// MachineDeploy represents the API for deploying a machine in Canonical MAAS.
type MachineDeploy struct {
	MachineSystemID
}

// POST /machines/{system_id}/op-deploy
func (m *MachineDeploy) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "machines/deploy", "systemID", m.SystemID)
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "machines/deploy", "systemID", m.SystemID)
	}()

	// Idempotency check: Check if machine is already deploying or deployed
	klog.V(2).InfoS("branch: checking machine status for idempotency", "systemID", m.SystemID)
	status, err := m.getMachineStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status == "Deploying" || status == "Deployed" {
		klog.V(2).InfoS("branch: machine already deploying or deployed, returning success", "systemID", m.SystemID, "status", status)
		// Return success without executing commission
		res := m.NewResponseCommon(200, nil)
		return res, nil
	}

	klog.V(2).InfoS("branch: machine status allows deploy", "systemID", m.SystemID, "status", status)

	// cast request body
	var req request_body.ReqbodyMachineDeploy
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyMachineDeploy); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	klog.V(3).InfoS("branch: request body cast successful", "distribution", req.Distribution, "version", req.Version)

	// execute API call to deploy a machine
	apiReqBody := fmt.Sprintf("bridge_all=%v&distro_series=%s/%s&user_data=%s",
		req.BridgeAll,
		url.QueryEscape(req.Distribution),
		url.QueryEscape(req.Version),
		url.QueryEscape(req.UserData),
	)

	klog.V(2).InfoS("branch: deploying machine", "bridgeAll", req.BridgeAll)
	statusCode, data, err := m.API.APIExecute(ctx, "POST", fmt.Sprintf("machines/%s/op-deploy", m.SystemID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: deploy API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: deploy API execution successful", "statusCode", statusCode)

	// generate response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// MachineRelease represents the API for releasing a machine in Canonical MAAS.
type MachineRelease struct {
	MachineSystemID
}

// POST /machines/{system_id}/op-release
func (m *MachineRelease) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "machines/op-release", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "machines/op-release", "systemID", m.SystemID)
	}()

	// cast request body
	var req request_body.ReqbodyMachineRelease
	var ok bool
	if req, ok = reqBody.(request_body.ReqbodyMachineRelease); !ok {
		klog.V(2).InfoS("branch: invalid request body type")
		return nil, errors.New("invalid call")
	}

	// Idempotency check: Check if machine is already releasing or ready
	klog.V(2).InfoS("branch: checking machine status for idempotency", "systemID", m.SystemID)
	status, err := m.getMachineStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status == "Releasing" || status == "Ready" {
		klog.V(2).InfoS("branch: machine already releasing or ready, returning success", "systemID", m.SystemID, "status", status)
		// Return success without executing commission
		res := m.NewResponseCommon(200, nil)
		return res, nil
	}

	klog.V(2).InfoS("branch: machine status allows release", "systemID", m.SystemID, "status", status)

	// execute API call to release a machine
	apiReqBody := fmt.Sprintf("erase=%v&quick_erase=%v&secure_erase=%v", req.Erase, req.QuickErase, req.SecureErase)
	statusCode, data, err := m.API.APIExecute(ctx, "POST", fmt.Sprintf("machines/%s/op-release", m.SystemID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: release API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: release API execution successful", "statusCode", statusCode)

	// generate response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// MachineAbort represents the API for aborting a machine operation in Canonical MAAS.
type MachineAbort struct {
	MachineSystemID
}

// POST /machines/{system_id}/op-abort
func (m *MachineAbort) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "machines/op-abort", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "machines/op-abort", "systemID", m.SystemID)
	}()

	// execute API call to abort a machine operation
	statusCode, data, err := m.API.APIExecute(ctx, "POST", fmt.Sprintf("machines/%s/op-abort", m.SystemID), "")
	if err != nil {
		klog.V(2).InfoS("branch: abort API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: abort API execution successful", "statusCode", statusCode)

	// generate response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// MachineMarkBroken represents the API for marking a machine as broken in Canonical MAAS.
type MachineMarkBroken struct {
	MachineSystemID
}

// POST /machines/{system_id}/op-mark_broken
func (m *MachineMarkBroken) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "machines/op-mark_broken", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "machines/op-mark_broken", "systemID", m.SystemID)
	}()

	// Idempotency check: Check if machine is already marked as broken
	klog.V(2).InfoS("branch: checking machine status for idempotency", "systemID", m.SystemID)
	status, err := m.getMachineStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status == "Broken" {
		klog.V(2).InfoS("branch: machine already marked as broken, returning success", "systemID", m.SystemID, "status", status)
		// Return success without executing commission
		res := m.NewResponseCommon(200, nil)
		return res, nil
	}

	klog.V(2).InfoS("branch: machine status allows mark broken", "systemID", m.SystemID, "status", status)

	// cast request body (optional comment parameter)
	var req request_body.ReqbodyMachineMarkBroken
	var ok bool
	apiReqBody := ""

	if req, ok = reqBody.(request_body.ReqbodyMachineMarkBroken); ok {
		klog.V(2).InfoS("branch: comment provided", "comment", req.Comment)
		if req.Comment != "" {
			apiReqBody = fmt.Sprintf("comment=%s", url.QueryEscape(req.Comment))
		}
	} else {
		klog.V(2).InfoS("branch: no comment provided")
	}

	// execute API call to mark a machine as broken
	statusCode, data, err := m.API.APIExecute(ctx, "POST", fmt.Sprintf("machines/%s/op-mark_broken", m.SystemID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: mark broken API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: mark broken API execution successful", "statusCode", statusCode)

	// generate response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// MachinePowerON represents the API for powering on a machine in Canonical MAAS.
type MachinePowerON struct {
	MachineSystemID
}

// POST /machines/{system_id}/op-power_on
func (m *MachinePowerON) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "machines/op-power_on", "systemID", m.SystemID)
	klog.V(3).InfoS("request body", "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "machines/op-power_on", "systemID", m.SystemID)
	}()

	// Idempotency check: Check if machine is already powered on
	klog.V(2).InfoS("branch: checking machine status for idempotency", "systemID", m.SystemID)
	status, err := m.getMachinePowerStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status == "on" {
		klog.V(2).InfoS("branch: machine already powered on, returning success", "systemID", m.SystemID, "status", status)
		// Return success without executing power operation
		res := m.NewResponseCommon(200, nil)
		return res, nil
	}

	apiReqBody := ""
	if reqBody != nil {
		var req request_body.ReqbodyMachinePowerON
		var ok bool
		if req, ok = reqBody.(request_body.ReqbodyMachinePowerON); !ok {
			klog.V(2).InfoS("branch: invalid request body type")
			return nil, errors.New("invalid call")
		}
		apiReqBody = fmt.Sprintf("user_data=%s", url.QueryEscape(req.UserData))
	}

	// execute API call to power-on a machine operation
	statusCode, data, err := m.API.APIExecute(ctx, "POST", fmt.Sprintf("machines/%s/op-power_on", m.SystemID), apiReqBody)
	if err != nil {
		klog.V(2).InfoS("branch: power-on API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: power-on API execution successful", "statusCode", statusCode)

	// generate response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}

// MachinePowerOFF represents the API for powering off a machine in Canonical MAAS.
type MachinePowerOFF struct {
	MachineSystemID
}

// POST /machines/{system_id}/op-power_off
func (m *MachinePowerOFF) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	klog.V(2).InfoS("start POST", "api", "machines/op-power_off", "systemID", m.SystemID)
	defer func() {
		klog.V(2).InfoS("end POST", "api", "machines/op-power_off", "systemID", m.SystemID)
	}()

	// Idempotency check: Check if machine is already powered off
	klog.V(2).InfoS("branch: checking machine status for idempotency", "systemID", m.SystemID)
	status, err := m.getMachinePowerStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status == "off" {
		klog.V(2).InfoS("branch: machine already powered off, returning success", "systemID", m.SystemID, "status", status)
		// Return success without executing power operation
		res := m.NewResponseCommon(200, nil)
		return res, nil
	}

	// execute API call to power-off a machine operation
	statusCode, data, err := m.API.APIExecute(ctx, "POST", fmt.Sprintf("machines/%s/op-power_off", m.SystemID), "")
	if err != nil {
		klog.V(2).InfoS("branch: power-off API execution failed", "error", err.Error())
		return nil, err
	}

	klog.V(2).InfoS("branch: power-off API execution successful", "statusCode", statusCode)

	// generate response data
	res := m.NewResponseCommon(statusCode, data)
	return res, m.HTTPError(statusCode, data)
}
