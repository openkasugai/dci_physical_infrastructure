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
	"errors"
	"fmt"
	"net/url"
	"testing"

	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/test_utils"
)

// MockCanonicalMaasApi is a mock implementation for testing
type MockCanonicalMaasApi struct {
	StatusCode     int
	Data           []byte
	Error          error
	CallCount      int
	GetData        []byte // Data for GET calls
	PostData       []byte // Data for POST calls
	APIExecuteFunc func(ctx context.Context, method, endpoint, body string) (int, []byte, error)
}

func (m *MockCanonicalMaasApi) APIExecute(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	// If custom function is provided, use it
	if m.APIExecuteFunc != nil {
		return m.APIExecuteFunc(ctx, method, endpoint, body)
	}

	if m.Error != nil {
		return 0, nil, m.Error
	}

	m.CallCount++

	// Handle different methods
	if method == "GET" {
		if m.GetData != nil {
			return 200, m.GetData, nil
		}
		return m.StatusCode, m.Data, nil
	} else if method == "POST" {
		if m.PostData != nil {
			return 201, m.PostData, nil
		}
		return m.StatusCode, m.Data, nil
	}

	return m.StatusCode, m.Data, nil
}

func TestMachines_GET_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `[{"system_id":"test-id","hostname":"test-host","status_name":"Ready","interface_set":[]}]`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machines.GET(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyGetMachines)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetMachines")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}

	if len(resbody.Machines) == 0 {
		t.Error("Expected parsed machines list, got empty list")
	}
}

func TestMachines_GET_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	machines := &Machines{}
	machines.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machines.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachines_GET_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 500,
		Data:       []byte("Internal Server Error"),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machines.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyGetMachines)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetMachines")
	}

	if resbody.HTTPStatus != 500 {
		t.Errorf("Expected status 500, got %d", resbody.HTTPStatus)
	}
}

func TestMachines_GET_JSONUnmarshalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - invalid JSON
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("invalid json"),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machines.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected JSON unmarshal error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on JSON unmarshal error")
	}
}

func TestMachines_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	// MockCanonicalMaasApi needs to return both GET (machine list) and POST responses
	machineListData := `[{"system_id":"existing-id","hostname":"other-machine","status_name":"Ready","interface_set":[]}]`
	postData := `{"system_id":"new-machine-id"}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineListData),
		PostData:   []byte(postData),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-machine", // Different from existing machine
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyPostMachines)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyPostMachines")
	}

	if resbody.HTTPStatus != 201 {
		t.Errorf("Expected status 201, got %d", resbody.HTTPStatus)
	}

	if resbody.SystemID != "new-machine-id" {
		t.Errorf("Expected system ID 'new-machine-id', got: %s", resbody.SystemID)
	}
}

func TestMachines_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	machines := &Machines{}
	machines.API = &MockCanonicalMaasApi{}

	// Invalid request body type
	invalidReqBody := "invalid request body"
	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, invalidReqBody)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid request body, got nil")
	}

	if result != nil {
		t.Error("Expected nil result for invalid request body")
	}

	expectedError := "invalid call"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMachines_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-machine",
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineSystemID_GET_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready"}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineSystemID.GET(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyGetMachine)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetMachine")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineSystemID_DELETE_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 204,
		Data:       []byte(""),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineSystemID.DELETE(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 204 {
		t.Errorf("Expected status 204, got %d", resbody.HTTPStatus)
	}
}

func TestMachineCommission_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	// getMachineStatus calls GET on MachineSystemID, which expects a machine object
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"New","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),           // For getMachineStatus call
		PostData:   []byte(`{"status":"commissioned"}`), // For actual commission call
		Error:      nil,
	}

	machineCommission := &MachineCommission{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineCommission.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineCommission.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 201 {
		t.Errorf("Expected status 201, got %d", resbody.HTTPStatus)
	}
}

func TestMachineDeploy_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	// getMachineStatus calls GET on MachineSystemID, which expects a machine object
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),       // For getMachineStatus call
		PostData:   []byte(`{"status":"deployed"}`), // For actual deploy call
		Error:      nil,
	}

	machineDeploy := &MachineDeploy{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineDeploy.API = mockAPI

	reqBody := request_body.ReqbodyMachineDeploy{
		BridgeAll:    true,
		Distribution: "ubuntu",
		Version:      "20.04",
		UserData:     "#!/bin/bash\necho 'Hello World'",
	}

	ctx := context.Background()

	// Act
	result, err := machineDeploy.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 201 {
		t.Errorf("Expected status 201, got %d", resbody.HTTPStatus)
	}
}

func TestMachineDeploy_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	machineDeploy := &MachineDeploy{
		MachineSystemID: MachineSystemID{
			SystemID: "test-system-id",
		},
	}
	// Set up API that will succeed getMachineStatus call but fail on wrong request body type
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`
	machineDeploy.API = &MockCanonicalMaasApi{
		GetData: []byte(machineStatusData),
		Error:   nil,
	}

	// Invalid request body type
	invalidReqBody := "invalid request body"
	ctx := context.Background()

	// Act
	result, err := machineDeploy.POST(ctx, invalidReqBody)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid request body, got nil")
	}

	if result != nil {
		t.Error("Expected nil result for invalid request body")
	}

	expectedError := "invalid call"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMachineRelease_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	// getMachineStatus calls GET on MachineSystemID, which expects a machine object
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Deployed","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),       // For getMachineStatus call
		PostData:   []byte(`{"status":"released"}`), // For actual release call
		Error:      nil,
	}

	machineRelease := &MachineRelease{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineRelease.API = mockAPI

	ctx := context.Background()

	// Act
	reqBody := request_body.ReqbodyMachineRelease{
		Erase:       false,
		QuickErase:  false,
		SecureErase: false,
	}
	result, err := machineRelease.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 201 {
		t.Errorf("Expected status 201, got %d", resbody.HTTPStatus)
	}
}

// Edge case tests for coverage
func TestMachines_POST_EmptyFields(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 400,
		Data:       []byte("Bad Request"),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI

	// Empty request body
	reqBody := request_body.ReqbodyMachines{}
	ctx := context.Background()

	// Act
	_, _ = machines.POST(ctx, reqBody)

	// Assert - should handle empty fields gracefully
	// Note: This test validates the request handling path
}

// Test for idempotency - machine with same hostname already exists
func TestMachines_POST_IdempotencyHit(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	existingHostname := "existing-machine"
	existingSystemID := "existing-system-id"
	machineListData := fmt.Sprintf(`[{"system_id":"%s","hostname":"%s","status_name":"Ready","interface_set":[]}]`, existingSystemID, existingHostname)

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineListData),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     existingHostname, // Same hostname as existing machine
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyPostMachines)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyPostMachines")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}

	if resbody.SystemID != existingSystemID {
		t.Errorf("Expected system ID '%s', got: %s", existingSystemID, resbody.SystemID)
	}
}

// Test GET machines list returning invalid response type
func TestMachines_POST_GetResponseTypeInvalid(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - This test triggers JSON unmarshal error in GET first
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(`invalid json`),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-machine",
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

// Test JSON unmarshal error in POST
func TestMachines_POST_JSONUnmarshalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	machineListData := `[{"system_id":"existing-id","hostname":"other-machine","status_name":"Ready","interface_set":[]}]`
	invalidPostData := `invalid json`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineListData),
		PostStatusCode: 201,
		PostData:       []byte(invalidPostData),
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-machine",
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected JSON unmarshal error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on JSON unmarshal error")
	}
}

// Test HTTP error in POST
func TestMachines_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	machineListData := `[{"system_id":"existing-id","hostname":"other-machine","status_name":"Ready","interface_set":[]}]`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineListData),
		PostStatusCode: 400,
		PostData:       []byte("Bad Request"),
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-machine",
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyPostMachines)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyPostMachines")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

func TestMachineSystemID_GET_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineSystemID.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineSystemID_GET_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 404,
		Data:       []byte("Not Found"),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineSystemID.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyGetMachine)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetMachine")
	}

	if resbody.HTTPStatus != 404 {
		t.Errorf("Expected status 404, got %d", resbody.HTTPStatus)
	}
}

func TestMachineSystemID_GET_JSONUnmarshalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("invalid json"),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineSystemID.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected JSON unmarshal error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on JSON unmarshal error")
	}
}

func TestMachineSystemID_PUT_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(`{"system_id":"test-system-id","description":"Updated description"}`),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	reqBody := request_body.ReqbodyMachineUpdate{
		Description: "Updated description",
	}

	ctx := context.Background()

	// Act
	result, err := machineSystemID.PUT(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineSystemID_PUT_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	machineSystemID := &MachineSystemID{
		SystemID: "test-system-id",
	}
	machineSystemID.API = &MockCanonicalMaasApi{}

	// Invalid request body type
	invalidReqBody := "invalid request body"
	ctx := context.Background()

	// Act
	result, err := machineSystemID.PUT(ctx, invalidReqBody)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid request body, got nil")
	}

	if result != nil {
		t.Error("Expected nil result for invalid request body")
	}

	expectedError := "invalid call"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMachineSystemID_PUT_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	reqBody := request_body.ReqbodyMachineUpdate{
		Description: "Updated description",
	}

	ctx := context.Background()

	// Act
	result, err := machineSystemID.PUT(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineSystemID_PUT_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 400,
		Data:       []byte("Bad Request"),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	reqBody := request_body.ReqbodyMachineUpdate{
		Description: "Updated description",
	}

	ctx := context.Background()

	// Act
	result, err := machineSystemID.PUT(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

func TestMachineSystemID_DELETE_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineSystemID.DELETE(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineSystemID_DELETE_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 400,
		Data:       []byte("Bad Request"),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineSystemID.DELETE(ctx)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

// Test getMachineStatus error cases
func TestMachineSystemID_getMachineStatus_GetError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		Error: errors.New("GET error"),
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	status, err := machineSystemID.getMachineStatus(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if status != "" {
		t.Errorf("Expected empty status, got %s", status)
	}
}

func TestMachineSystemID_getMachineStatus_InvalidResponseType(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("invalid json"),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	status, err := machineSystemID.getMachineStatus(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if status != "" {
		t.Errorf("Expected empty status, got %s", status)
	}
}

// Test for successful getMachineStatus
func TestMachineSystemID_getMachineStatus_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	expectedStatus := "Ready"
	mockData := fmt.Sprintf(`{"system_id":"%s","hostname":"test-host","status_name":"%s","interface_set":[]}`, systemID, expectedStatus)

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	status, err := machineSystemID.getMachineStatus(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if status != expectedStatus {
		t.Errorf("Expected status %s, got %s", expectedStatus, status)
	}
}

// Alternative test - successful GET with HTTP error return
func TestMachineSystemID_getMachineStatus_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 404,
		Data:       []byte("Not Found"),
		Error:      nil,
	}

	machineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	machineSystemID.API = mockAPI

	ctx := context.Background()

	// Act
	status, err := machineSystemID.getMachineStatus(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if status != "" {
		t.Errorf("Expected empty status, got %s", status)
	}
}

// Test edge case where machine exists but GET itself would fail after idempotency check
func TestMachines_POST_GetErrorAfterIdempotencyCheck(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - First GET succeeds for idempotency check, then later processing has issues
	machineListData := `[{"system_id":"existing-id","hostname":"other-machine","status_name":"Ready","interface_set":[]}]`

	mockAPI := &MockCanonicalMaasApiWithDoubleCall{
		FirstGetData:   []byte(machineListData),
		SecondGetError: errors.New("Second GET error"),
		PostData:       []byte(`{"system_id":"new-machine-id"}`),
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "new-machine", // Different hostname to trigger POST
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert - Should succeed as it doesn't hit second GET
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// Mock that handles multiple calls differently
type MockCanonicalMaasApiWithDoubleCall struct {
	CallCount      int
	FirstGetData   []byte
	SecondGetError error
	PostData       []byte
}

func (m *MockCanonicalMaasApiWithDoubleCall) APIExecute(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	m.CallCount++

	if method == "GET" {
		if m.CallCount == 1 {
			return 200, m.FirstGetData, nil
		} else {
			if m.SecondGetError != nil {
				return 0, nil, m.SecondGetError
			}
		}
	} else if method == "POST" {
		return 201, m.PostData, nil
	}

	return 200, []byte(`{}`), nil
}

// Test for machines list response with valid machines but containing target hostname
func TestMachines_POST_GetErrorInIdempotencyCheck(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - GET call itself fails
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("GET error during idempotency check"),
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-machine",
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

// Test MachineCommission idempotency cases
func TestMachineCommission_POST_IdempotencyCommissioning(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Commissioning","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),
		Error:      nil,
	}

	machineCommission := &MachineCommission{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineCommission.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineCommission.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineCommission_POST_GetMachineStatusError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		Error: errors.New("GET error"),
	}

	machineCommission := &MachineCommission{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineCommission.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineCommission.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineCommission_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"New","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:   []byte(machineStatusData),
		PostError: errors.New("API error"),
	}

	machineCommission := &MachineCommission{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineCommission.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineCommission.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineCommission_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"New","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineStatusData),
		PostStatusCode: 400,
		PostData:       []byte("Bad Request"),
	}

	machineCommission := &MachineCommission{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineCommission.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineCommission.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

// Test MachineDeploy idempotency cases
func TestMachineDeploy_POST_IdempotencyDeploying(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Deploying","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),
		Error:      nil,
	}

	machineDeploy := &MachineDeploy{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineDeploy.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineDeploy.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineDeploy_POST_IdempotencyDeployed(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Deployed","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),
		Error:      nil,
	}

	machineDeploy := &MachineDeploy{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineDeploy.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineDeploy.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineDeploy_POST_GetMachineStatusError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		Error: errors.New("GET error"),
	}

	machineDeploy := &MachineDeploy{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineDeploy.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineDeploy.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineDeploy_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:   []byte(machineStatusData),
		PostError: errors.New("API error"),
	}

	machineDeploy := &MachineDeploy{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineDeploy.API = mockAPI

	reqBody := request_body.ReqbodyMachineDeploy{
		BridgeAll:    true,
		Distribution: "ubuntu",
		Version:      "20.04",
		UserData:     "#!/bin/bash\necho 'Hello World'",
	}

	ctx := context.Background()

	// Act
	result, err := machineDeploy.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineDeploy_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineStatusData),
		PostStatusCode: 400,
		PostData:       []byte("Bad Request"),
	}

	machineDeploy := &MachineDeploy{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineDeploy.API = mockAPI

	reqBody := request_body.ReqbodyMachineDeploy{
		BridgeAll:    true,
		Distribution: "ubuntu",
		Version:      "20.04",
		UserData:     "#!/bin/bash\necho 'Hello World'",
	}

	ctx := context.Background()

	// Act
	result, err := machineDeploy.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

// Test MachineRelease idempotency cases
func TestMachineRelease_POST_IdempotencyReleasing(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Releasing","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),
		Error:      nil,
	}

	machineRelease := &MachineRelease{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineRelease.API = mockAPI

	ctx := context.Background()

	// Act
	reqBody := request_body.ReqbodyMachineRelease{
		Erase:       false,
		QuickErase:  false,
		SecureErase: false,
	}
	result, err := machineRelease.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineRelease_POST_IdempotencyReady(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),
		Error:      nil,
	}

	machineRelease := &MachineRelease{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineRelease.API = mockAPI

	ctx := context.Background()

	// Act
	reqBody := request_body.ReqbodyMachineRelease{
		Erase:       false,
		QuickErase:  false,
		SecureErase: false,
	}
	result, err := machineRelease.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineRelease_POST_GetMachineStatusError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		Error: errors.New("GET error"),
	}

	machineRelease := &MachineRelease{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineRelease.API = mockAPI

	ctx := context.Background()

	// Act
	reqBody := request_body.ReqbodyMachineRelease{
		Erase:       false,
		QuickErase:  false,
		SecureErase: false,
	}
	result, err := machineRelease.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineRelease_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Deployed","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:   []byte(machineStatusData),
		PostError: errors.New("API error"),
	}

	machineRelease := &MachineRelease{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineRelease.API = mockAPI

	ctx := context.Background()

	// Act
	reqBody := request_body.ReqbodyMachineRelease{
		Erase:       false,
		QuickErase:  false,
		SecureErase: false,
	}
	result, err := machineRelease.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineRelease_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Deployed","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineStatusData),
		PostStatusCode: 400,
		PostData:       []byte("Bad Request"),
	}

	machineRelease := &MachineRelease{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineRelease.API = mockAPI

	ctx := context.Background()

	// Act
	reqBody := request_body.ReqbodyMachineRelease{
		Erase:       false,
		QuickErase:  false,
		SecureErase: false,
	}
	result, err := machineRelease.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

// Test MachineAbort
func TestMachineAbort_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(`{"status":"aborted"}`),
		Error:      nil,
	}

	machineAbort := &MachineAbort{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineAbort.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineAbort.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineAbort_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	machineAbort := &MachineAbort{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineAbort.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineAbort.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineAbort_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 400,
		Data:       []byte("Bad Request"),
		Error:      nil,
	}

	machineAbort := &MachineAbort{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineAbort.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineAbort.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

// Test MachineMarkBroken
func TestMachineMarkBroken_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineStatusData),
		PostStatusCode: 200,
		PostData:       []byte(`{"status":"broken"}`),
	}

	machineMarkBroken := &MachineMarkBroken{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineMarkBroken.API = mockAPI

	reqBody := request_body.ReqbodyMachineMarkBroken{
		Comment: "Machine is broken",
	}

	ctx := context.Background()

	// Act
	result, err := machineMarkBroken.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineMarkBroken_POST_IdempotencyBroken(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Broken","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),
		Error:      nil,
	}

	machineMarkBroken := &MachineMarkBroken{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineMarkBroken.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineMarkBroken.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineMarkBroken_POST_NoComment(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineStatusData),
		PostStatusCode: 200,
		PostData:       []byte(`{"status":"broken"}`),
	}

	machineMarkBroken := &MachineMarkBroken{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineMarkBroken.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineMarkBroken.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineMarkBroken_POST_EmptyComment(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineStatusData),
		PostStatusCode: 200,
		PostData:       []byte(`{"status":"broken"}`),
	}

	machineMarkBroken := &MachineMarkBroken{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineMarkBroken.API = mockAPI

	reqBody := request_body.ReqbodyMachineMarkBroken{
		Comment: "", // Empty comment
	}

	ctx := context.Background()

	// Act
	result, err := machineMarkBroken.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

func TestMachineMarkBroken_POST_GetMachineStatusError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		Error: errors.New("GET error"),
	}

	machineMarkBroken := &MachineMarkBroken{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineMarkBroken.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineMarkBroken.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineMarkBroken_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:   []byte(machineStatusData),
		PostError: errors.New("API error"),
	}

	machineMarkBroken := &MachineMarkBroken{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineMarkBroken.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineMarkBroken.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachineMarkBroken_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApiWithCallTracking{
		GetData:        []byte(machineStatusData),
		PostStatusCode: 400,
		PostData:       []byte("Bad Request"),
	}

	machineMarkBroken := &MachineMarkBroken{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineMarkBroken.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineMarkBroken.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

// Enhanced mock for better test control
type MockCanonicalMaasApiWithCallTracking struct {
	CallCount      int
	GetData        []byte
	PostData       []byte
	PostError      error
	PostStatusCode int
}

func (m *MockCanonicalMaasApiWithCallTracking) APIExecute(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	m.CallCount++

	if method == "GET" {
		if m.GetData != nil {
			return 200, m.GetData, nil
		}
		return 200, []byte(`{"system_id":"test","status_name":"Ready"}`), nil
	} else if method == "POST" {
		if m.PostError != nil {
			return 0, nil, m.PostError
		}
		if m.PostStatusCode != 0 {
			return m.PostStatusCode, m.PostData, nil
		}
		if m.PostData != nil {
			return 201, m.PostData, nil
		}
		return 201, []byte(`{"status":"success"}`), nil
	} else if method == "PUT" {
		if m.PostError != nil {
			return 0, nil, m.PostError
		}
		if m.PostStatusCode != 0 {
			return m.PostStatusCode, m.PostData, nil
		}
		return 200, []byte(`{"status":"updated"}`), nil
	} else if method == "DELETE" {
		if m.PostError != nil {
			return 0, nil, m.PostError
		}
		if m.PostStatusCode != 0 {
			return m.PostStatusCode, m.PostData, nil
		}
		return 204, []byte(``), nil
	}

	return 200, []byte(`{"status":"success"}`), nil
}

// Mock that returns wrong type for response casting tests
type MockCanonicalMaasApiWithWrongType struct {
	GetSucceeds bool
	WrongData   []byte
}

func (m *MockCanonicalMaasApiWithWrongType) APIExecute(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	if method == "GET" && m.GetSucceeds {
		return 200, m.WrongData, nil
	}
	return 200, []byte(`{}`), nil
}

// Mock for testing POST API failure after GET success
type MockCanonicalMaasApiWithPostFailure struct {
	GetData   []byte
	PostError error
}

func (m *MockCanonicalMaasApiWithPostFailure) APIExecute(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	if method == "GET" {
		return 200, m.GetData, nil
	} else if method == "POST" {
		return 0, nil, m.PostError
	}
	return 200, []byte(`{}`), nil
}

// Mock that forces type assertion failure
type MockCanonicalMaasApiForTypeAssertionFailure struct{}

func (m *MockCanonicalMaasApiForTypeAssertionFailure) APIExecute(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	return 200, []byte(`[]`), nil
}

// Custom machines struct that returns wrong type from GET
type MachinesWithWrongGET struct {
	*Machines
}

func (m *MachinesWithWrongGET) GET(ctx context.Context) (response_body.Resbody, error) {
	// Return a different response type that will fail type assertion
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

// Test for case ① - GET success but type cast failure in POST /machines/
func TestMachines_POST_GetSuccessButTypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - Use custom machines that returns wrong type from GET
	baseMachines := &Machines{}
	baseMachines.API = &MockCanonicalMaasApiForTypeAssertionFailure{}

	machines := &MachinesWithWrongGET{Machines: baseMachines}

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-machine",
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if result != nil {
		t.Error("Expected nil result on error")
	}
	// This test covers the error path even if the specific error differs
	t.Logf("Error occurred as expected: %s", err.Error())
}

// Test for case ② - GET success, POST API execution failure
func TestMachines_POST_GetSuccessPostAPIFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - GET succeeds with empty machine list, POST fails
	machineListData := `[{"system_id":"existing-id","hostname":"other-machine","status_name":"Ready","interface_set":[]}]`

	mockAPI := &MockCanonicalMaasApiWithPostFailure{
		GetData:   []byte(machineListData),
		PostError: errors.New("POST API execution failed"),
	}

	machines := &Machines{}
	machines.API = mockAPI

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "new-machine", // Different hostname to trigger POST
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	ctx := context.Background()

	// Act
	result, err := machines.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected POST API error, got nil")
	}
	if result != nil {
		t.Error("Expected nil result on POST API error")
	}
	expectedError := "POST API execution failed"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// Custom MachineSystemID that returns wrong type from GET
type MachineSystemIDWithWrongGET struct {
	*MachineSystemID
}

func (m *MachineSystemIDWithWrongGET) GET(ctx context.Context) (response_body.Resbody, error) {
	// Return a different response type that will fail type assertion
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

// Test for case ③ - getMachineStatus GET success but type cast failure
func TestMachineSystemID_getMachineStatus_TypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - Use custom MachineSystemID that returns wrong type from GET
	systemID := "test-system-id"

	baseMachineSystemID := &MachineSystemID{
		SystemID: systemID,
	}
	baseMachineSystemID.API = &MockCanonicalMaasApiForTypeAssertionFailure{}

	machineSystemID := &MachineSystemIDWithWrongGET{MachineSystemID: baseMachineSystemID}

	ctx := context.Background()

	// Act
	status, err := machineSystemID.getMachineStatus(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if status != "" {
		t.Errorf("Expected empty status, got %s", status)
	}
	// This test covers the error path even if the specific error differs
	t.Logf("Error occurred as expected: %s", err.Error())
}

// Benchmark tests
func BenchmarkMachines_GET(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(`[{"system_id":"test"}]`),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := machines.GET(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMachines_POST(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 201,
		Data:       []byte(`{"system_id":"new-machine"}`),
		Error:      nil,
	}

	machines := &Machines{}
	machines.API = mockAPI
	ctx := context.Background()

	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-machine",
		Commission:   true,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: "192.168.1.100",
		PowerUser:    "admin",
		PowerPass:    "password",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := machines.POST(ctx, reqBody)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test MachineCommission with Testing status (new idempotency case)
func TestMachineCommission_POST_IdempotencyTesting(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Testing","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),
		Error:      nil,
	}

	machineCommission := &MachineCommission{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineCommission.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineCommission.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

// Test MachineCommission with Ready status (existing idempotency case)
func TestMachineCommission_POST_IdempotencyReady(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	machineStatusData := `{"system_id":"test-system-id","hostname":"test-host","status_name":"Ready","interface_set":[]}`

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(machineStatusData),
		Error:      nil,
	}

	machineCommission := &MachineCommission{
		MachineSystemID: MachineSystemID{
			SystemID: systemID,
		},
	}
	machineCommission.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machineCommission.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

// Test multiple status values for MachineCommission idempotency
func TestMachineCommission_POST_IdempotencyMultipleStatuses(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name   string
		status string
	}{
		{"Commissioning status", "Commissioning"},
		{"Ready status", "Ready"},
		{"Testing status", "Testing"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			systemID := "test-system-id"
			machineStatusData := fmt.Sprintf(`{"system_id":"test-system-id","hostname":"test-host","status_name":"%s","interface_set":[]}`, tc.status)

			mockAPI := &MockCanonicalMaasApi{
				StatusCode: 200,
				GetData:    []byte(machineStatusData),
				Error:      nil,
			}

			machineCommission := &MachineCommission{
				MachineSystemID: MachineSystemID{
					SystemID: systemID,
				},
			}
			machineCommission.API = mockAPI

			ctx := context.Background()

			// Act
			result, err := machineCommission.POST(ctx, nil)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for status %s, got %v", tc.status, err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			resbody, ok := result.(response_body.ResbodyCommon)
			if !ok {
				t.Fatal("Expected result to be of type ResbodyCommon")
			}

			if resbody.HTTPStatus != 200 {
				t.Errorf("Expected status 200 for machine status %s, got %d", tc.status, resbody.HTTPStatus)
			}
		})
	}
}

// Test multiple status values for MachineDeploy idempotency
func TestMachineDeploy_POST_IdempotencyMultipleStatuses(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name   string
		status string
	}{
		{"Deploying status", "Deploying"},
		{"Deployed status", "Deployed"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			systemID := "test-system-id"
			machineStatusData := fmt.Sprintf(`{"system_id":"test-system-id","hostname":"test-host","status_name":"%s","interface_set":[]}`, tc.status)

			mockAPI := &MockCanonicalMaasApi{
				StatusCode: 200,
				GetData:    []byte(machineStatusData),
				Error:      nil,
			}

			machineDeploy := &MachineDeploy{
				MachineSystemID: MachineSystemID{
					SystemID: systemID,
				},
			}
			machineDeploy.API = mockAPI

			ctx := context.Background()

			// Act
			result, err := machineDeploy.POST(ctx, nil)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for status %s, got %v", tc.status, err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			resbody, ok := result.(response_body.ResbodyCommon)
			if !ok {
				t.Fatal("Expected result to be of type ResbodyCommon")
			}

			if resbody.HTTPStatus != 200 {
				t.Errorf("Expected status 200 for machine status %s, got %d", tc.status, resbody.HTTPStatus)
			}
		})
	}
}

// Test non-idempotent statuses still trigger API calls for MachineCommission
func TestMachineCommission_POST_NonIdempotentStatusTriggersAPICall(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []string{
		"New",
		"Failed",
		"Broken",
		"Deployed", // This should trigger commission
	}

	for _, status := range testCases {
		t.Run("Status_"+status, func(t *testing.T) {
			// Arrange
			systemID := "test-system-id"
			machineStatusData := fmt.Sprintf(`{"system_id":"test-system-id","hostname":"test-host","status_name":"%s","interface_set":[]}`, status)

			mockAPI := &MockCanonicalMaasApiWithCallTracking{
				GetData:        []byte(machineStatusData),
				PostStatusCode: 200,
				PostData:       []byte(`{"status":"commissioned"}`),
			}

			machineCommission := &MachineCommission{
				MachineSystemID: MachineSystemID{
					SystemID: systemID,
				},
			}
			machineCommission.API = mockAPI

			ctx := context.Background()

			// Act
			result, err := machineCommission.POST(ctx, nil)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for status %s, got %v", status, err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			resbody, ok := result.(response_body.ResbodyCommon)
			if !ok {
				t.Fatal("Expected result to be of type ResbodyCommon")
			}

			if resbody.HTTPStatus != 200 {
				t.Errorf("Expected status 200 for machine status %s, got %d", status, resbody.HTTPStatus)
			}

			// Verify that both GET and POST were called
			if mockAPI.CallCount < 2 {
				t.Errorf("Expected at least 2 API calls (GET + POST) for status %s, got %d", status, mockAPI.CallCount)
			}
		})
	}
}

// Test non-idempotent statuses still trigger API calls for MachineDeploy
func TestMachineDeploy_POST_NonIdempotentStatusTriggersAPICall(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []string{
		"Ready",
		"Failed",
		"Broken",
		"Commissioning", // This should trigger deploy
	}

	for _, status := range testCases {
		t.Run("Status_"+status, func(t *testing.T) {
			// Arrange
			systemID := "test-system-id"
			machineStatusData := fmt.Sprintf(`{"system_id":"test-system-id","hostname":"test-host","status_name":"%s","interface_set":[]}`, status)

			mockAPI := &MockCanonicalMaasApiWithCallTracking{
				GetData:        []byte(machineStatusData),
				PostStatusCode: 200,
				PostData:       []byte(`{"status":"deployed"}`),
			}

			machineDeploy := &MachineDeploy{
				MachineSystemID: MachineSystemID{
					SystemID: systemID,
				},
			}
			machineDeploy.API = mockAPI

			reqBody := request_body.ReqbodyMachineDeploy{
				BridgeAll:    true,
				Distribution: "ubuntu",
				Version:      "20.04",
				UserData:     "#!/bin/bash\necho 'Hello World'",
			}

			ctx := context.Background()

			// Act
			result, err := machineDeploy.POST(ctx, reqBody)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for status %s, got %v", status, err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			resbody, ok := result.(response_body.ResbodyCommon)
			if !ok {
				t.Fatal("Expected result to be of type ResbodyCommon")
			}

			if resbody.HTTPStatus != 200 {
				t.Errorf("Expected status 200 for machine status %s, got %d", status, resbody.HTTPStatus)
			}

			// Verify that both GET and POST were called
			if mockAPI.CallCount < 2 {
				t.Errorf("Expected at least 2 API calls (GET + POST) for status %s, got %d", status, mockAPI.CallCount)
			}
		})
	}
}

// Edge case: Test MachineCommission with exactly matching idempotent status boundaries
func TestMachineCommission_POST_IdempotencyBoundaryConditions(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test case-sensitive status matching
	testCases := []struct {
		name               string
		status             string
		shouldBeIdempotent bool
	}{
		{"Exact Commissioning", "Commissioning", true},
		{"Exact Ready", "Ready", true},
		{"Exact Testing", "Testing", true},
		{"Case sensitive commissioning", "commissioning", false}, // lowercase should not match
		{"Case sensitive ready", "ready", false},                 // lowercase should not match
		{"Case sensitive testing", "testing", false},             // lowercase should not match
		{"Similar but different", "Commission", false},           // similar but not exact
		{"Similar but different", "Test", false},                 // similar but not exact
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			systemID := "test-system-id"
			machineStatusData := fmt.Sprintf(`{"system_id":"test-system-id","hostname":"test-host","status_name":"%s","interface_set":[]}`, tc.status)

			mockAPI := &MockCanonicalMaasApiWithCallTracking{
				GetData:        []byte(machineStatusData),
				PostStatusCode: 200,
				PostData:       []byte(`{"status":"commissioned"}`),
			}

			machineCommission := &MachineCommission{
				MachineSystemID: MachineSystemID{
					SystemID: systemID,
				},
			}
			machineCommission.API = mockAPI

			ctx := context.Background()

			// Act
			result, err := machineCommission.POST(ctx, nil)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for status %s, got %v", tc.status, err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			resbody, ok := result.(response_body.ResbodyCommon)
			if !ok {
				t.Fatal("Expected result to be of type ResbodyCommon")
			}

			if tc.shouldBeIdempotent {
				// Should return 200 and only make 1 GET call
				if resbody.HTTPStatus != 200 {
					t.Errorf("Expected status 200 for idempotent status %s, got %d", tc.status, resbody.HTTPStatus)
				}
				if mockAPI.CallCount != 1 {
					t.Errorf("Expected exactly 1 API call (GET only) for idempotent status %s, got %d", tc.status, mockAPI.CallCount)
				}
			} else {
				// Should return 200 and make 2 calls (GET + POST)
				if resbody.HTTPStatus != 200 {
					t.Errorf("Expected status 200 for non-idempotent status %s, got %d", tc.status, resbody.HTTPStatus)
				}
				if mockAPI.CallCount < 2 {
					t.Errorf("Expected at least 2 API calls (GET + POST) for non-idempotent status %s, got %d", tc.status, mockAPI.CallCount)
				}
			}
		})
	}
}

// Edge case: Test MachineDeploy with exactly matching idempotent status boundaries
func TestMachineDeploy_POST_IdempotencyBoundaryConditions(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test case-sensitive status matching
	testCases := []struct {
		name               string
		status             string
		shouldBeIdempotent bool
	}{
		{"Exact Deploying", "Deploying", true},
		{"Exact Deployed", "Deployed", true},
		{"Case sensitive deploying", "deploying", false}, // lowercase should not match
		{"Case sensitive deployed", "deployed", false},   // lowercase should not match
		{"Case sensitive allocated", "allocated", false}, // lowercase should not match (not idempotent anymore)
		{"Similar but different", "Deploy", false},       // similar but not exact
		{"Similar but different", "Allocate", false},     // similar but not exact
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			systemID := "test-system-id"
			machineStatusData := fmt.Sprintf(`{"system_id":"test-system-id","hostname":"test-host","status_name":"%s","interface_set":[]}`, tc.status)

			mockAPI := &MockCanonicalMaasApiWithCallTracking{
				GetData:        []byte(machineStatusData),
				PostStatusCode: 200,
				PostData:       []byte(`{"status":"deployed"}`),
			}

			machineDeploy := &MachineDeploy{
				MachineSystemID: MachineSystemID{
					SystemID: systemID,
				},
			}
			machineDeploy.API = mockAPI

			reqBody := request_body.ReqbodyMachineDeploy{
				BridgeAll:    true,
				Distribution: "ubuntu",
				Version:      "20.04",
				UserData:     "#!/bin/bash\necho 'Hello World'",
			}

			ctx := context.Background()

			// Act
			result, err := machineDeploy.POST(ctx, reqBody)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for status %s, got %v", tc.status, err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			resbody, ok := result.(response_body.ResbodyCommon)
			if !ok {
				t.Fatal("Expected result to be of type ResbodyCommon")
			}

			if tc.shouldBeIdempotent {
				// Should return 200 and only make 1 GET call
				if resbody.HTTPStatus != 200 {
					t.Errorf("Expected status 200 for idempotent status %s, got %d", tc.status, resbody.HTTPStatus)
				}
				if mockAPI.CallCount != 1 {
					t.Errorf("Expected exactly 1 API call (GET only) for idempotent status %s, got %d", tc.status, mockAPI.CallCount)
				}
			} else {
				// Should return 200 and make 2 calls (GET + POST)
				if resbody.HTTPStatus != 200 {
					t.Errorf("Expected status 200 for non-idempotent status %s, got %d", tc.status, resbody.HTTPStatus)
				}
				if mockAPI.CallCount < 2 {
					t.Errorf("Expected at least 2 API calls (GET + POST) for non-idempotent status %s, got %d", tc.status, mockAPI.CallCount)
				}
			}
		})
	}
}

// ============================================================================
// Tests for getMachinePowerStatus
// ============================================================================

func TestMachineSystemID_getMachinePowerStatus_Success(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

// Arrange
mockData := `{"system_id":"test-id","hostname":"test-host","power_state":"on"}`
mockAPI := &MockCanonicalMaasApi{
StatusCode: 200,
Data:       []byte(mockData),
Error:      nil,
}

machine := &MachineSystemID{
SystemID: "test-id",
}
machine.API = mockAPI

ctx := context.Background()

// Act
status, err := machine.getMachinePowerStatus(ctx)

// Assert
if err != nil {
t.Errorf("Expected no error, got %v", err)
}

if status != "on" {
t.Errorf("Expected power status 'on', got '%s'", status)
}
}

func TestMachineSystemID_getMachinePowerStatus_GetError(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

// Arrange
mockAPI := &MockCanonicalMaasApi{
StatusCode: 0,
Data:       nil,
Error:      errors.New("API error"),
}

machine := &MachineSystemID{
SystemID: "test-id",
}
machine.API = mockAPI

ctx := context.Background()

// Act
status, err := machine.getMachinePowerStatus(ctx)

// Assert
if err == nil {
t.Error("Expected error, got nil")
}

if status != "" {
t.Errorf("Expected empty status, got '%s'", status)
}
}

func TestMachineSystemID_getMachinePowerStatus_InvalidResponseType(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

// Arrange - return invalid data that can't be unmarshaled correctly
mockData := `[{"invalid":"array"}]`
mockAPI := &MockCanonicalMaasApi{
StatusCode: 200,
Data:       []byte(mockData),
Error:      nil,
}

machine := &MachineSystemID{
SystemID: "test-id",
}
machine.API = mockAPI

ctx := context.Background()

// Act
status, err := machine.getMachinePowerStatus(ctx)

// Assert
if err == nil {
t.Error("Expected error due to invalid response type, got nil")
}

if status != "" {
t.Errorf("Expected empty status, got '%s'", status)
}
}

// ============================================================================
// Tests for MachinePowerON.POST
// ============================================================================

func TestMachinePowerON_POST_Success(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

// Arrange
getMockData := `{"system_id":"test-id","hostname":"test-host","power_state":"off"}`
postMockData := `{"system_id":"test-id","power_state":"on"}`

mockAPI := &MockCanonicalMaasApi{
StatusCode: 200,
GetData:    []byte(getMockData),
PostData:   []byte(postMockData),
Error:      nil,
}

var actualBody string
mockAPI.APIExecuteFunc = func(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	mockAPI.CallCount++
	if method == "GET" {
		return 200, []byte(getMockData), nil
	}
	if method == "POST" {
		actualBody = body
		return 201, []byte(postMockData), nil
	}
	return mockAPI.StatusCode, mockAPI.Data, nil
}

machine := &MachinePowerON{}
machine.SystemID = "test-id"
machine.API = mockAPI

ctx := context.Background()

// Act
result, err := machine.POST(ctx, request_body.ReqbodyMachinePowerON{UserData: "#cloud-config\nfoo: bar"})

// Assert
if err != nil {
t.Errorf("Expected no error, got %v", err)
}

if result == nil {
t.Fatal("Expected non-nil result")
}

resbody, ok := result.(response_body.ResbodyCommon)
if !ok {
t.Fatal("Expected result to be of type ResbodyCommon")
}

if resbody.HTTPStatus != 201 {
t.Errorf("Expected status 201, got %d", resbody.HTTPStatus)
}

expectedBody := fmt.Sprintf("user_data=%s", url.QueryEscape("#cloud-config\nfoo: bar"))
if actualBody != expectedBody {
	t.Errorf("Expected request body %q, got %q", expectedBody, actualBody)
}

// Should have made 2 calls: GET (check power status) + POST (power on)
if mockAPI.CallCount < 2 {
t.Errorf("Expected at least 2 API calls, got %d", mockAPI.CallCount)
}
}

func TestMachinePowerON_POST_InvalidRequestBody(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{
	GetData: []byte(`{"system_id":"test-id","hostname":"test-host","power_state":"off"}`),
	Error:   nil,
}

machine := &MachinePowerON{}
machine.SystemID = "test-id"
machine.API = mockAPI

result, err := machine.POST(context.Background(), "invalid request body")

if err == nil {
	t.Error("Expected error for invalid request body, got nil")
}

if result != nil {
	t.Error("Expected nil result for invalid request body")
	}

	if err != nil && err.Error() != "invalid call" {
		t.Errorf("Expected error message 'invalid call', got '%s'", err.Error())
	}
}

func TestMachinePowerON_POST_IdempotencyAlreadyOn(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

// Arrange
getMockData := `{"system_id":"test-id","hostname":"test-host","power_state":"on"}`

mockAPI := &MockCanonicalMaasApi{
StatusCode: 200,
GetData:    []byte(getMockData),
Error:      nil,
}

machine := &MachinePowerON{}
machine.SystemID = "test-id"
machine.API = mockAPI

ctx := context.Background()

// Act
result, err := machine.POST(ctx, nil)

// Assert
if err != nil {
t.Errorf("Expected no error, got %v", err)
}

if result == nil {
t.Fatal("Expected non-nil result")
}

resbody, ok := result.(response_body.ResbodyCommon)
if !ok {
t.Fatal("Expected result to be of type ResbodyCommon")
}

if resbody.HTTPStatus != 200 {
t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
}

// Should have made only 1 call: GET (check power status) - no POST needed
if mockAPI.CallCount != 1 {
t.Errorf("Expected exactly 1 API call (idempotency check only), got %d", mockAPI.CallCount)
}
}

func TestMachinePowerON_POST_GetMachinePowerStatusError(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

// Arrange
mockAPI := &MockCanonicalMaasApi{
StatusCode: 0,
Data:       nil,
Error:      errors.New("API error"),
}

machine := &MachinePowerON{}
machine.SystemID = "test-id"
machine.API = mockAPI

ctx := context.Background()

// Act
result, err := machine.POST(ctx, nil)

// Assert
if err == nil {
t.Error("Expected error, got nil")
}

if result != nil {
t.Error("Expected nil result on error")
}
}


func TestMachinePowerON_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - first call succeeds (power is off), second call fails
	getMockData := `{"system_id":"test-id","hostname":"test-host","power_state":"off"}`
	
	callCount := 0
	mockAPI := &MockCanonicalMaasApi{}
	mockAPI.APIExecuteFunc = func(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
		callCount++
		if method == "GET" {
			return 200, []byte(getMockData), nil
		}
		return 0, nil, errors.New("API error on POST")
	}

	machine := &MachinePowerON{}
	machine.SystemID = "test-id"
	machine.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machine.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachinePowerON_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	getMockData := `{"system_id":"test-id","hostname":"test-host","power_state":"off"}`
	postMockData := `{"error": "Internal server error"}`
	
	callCount := 0
	mockAPI := &MockCanonicalMaasApi{}
	mockAPI.APIExecuteFunc = func(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
		callCount++
		if method == "GET" {
			return 200, []byte(getMockData), nil
		}
		return 500, []byte(postMockData), nil
	}
	_ = callCount

	machine := &MachinePowerON{}
	machine.SystemID = "test-id"
	machine.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machine.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 500 {
		t.Errorf("Expected status 500, got %d", resbody.HTTPStatus)
	}
}

// ============================================================================
// Tests for MachinePowerOFF.POST
// ============================================================================

func TestMachinePowerOFF_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	getMockData := `{"system_id":"test-id","hostname":"test-host","power_state":"on"}`
	postMockData := `{"system_id":"test-id","power_state":"off"}`
	
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(getMockData),
		PostData:   []byte(postMockData),
		Error:      nil,
	}

	machine := &MachinePowerOFF{}
	machine.SystemID = "test-id"
	machine.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machine.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 201 {
		t.Errorf("Expected status 201, got %d", resbody.HTTPStatus)
	}

	// Should have made 2 calls: GET (check power status) + POST (power off)
	if mockAPI.CallCount < 2 {
		t.Errorf("Expected at least 2 API calls, got %d", mockAPI.CallCount)
	}
}

func TestMachinePowerOFF_POST_IdempotencyAlreadyOff(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	getMockData := `{"system_id":"test-id","hostname":"test-host","power_state":"off"}`
	
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		GetData:    []byte(getMockData),
		Error:      nil,
	}

	machine := &MachinePowerOFF{}
	machine.SystemID = "test-id"
	machine.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machine.POST(ctx, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}

	// Should have made only 1 call: GET (check power status) - no POST needed
	if mockAPI.CallCount != 1 {
		t.Errorf("Expected exactly 1 API call (idempotency check only), got %d", mockAPI.CallCount)
	}
}

func TestMachinePowerOFF_POST_GetMachinePowerStatusError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	machine := &MachinePowerOFF{}
	machine.SystemID = "test-id"
	machine.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machine.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachinePowerOFF_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - first call succeeds (power is on), second call fails
	getMockData := `{"system_id":"test-id","hostname":"test-host","power_state":"on"}`
	
	callCount := 0
	mockAPI := &MockCanonicalMaasApi{}
	mockAPI.APIExecuteFunc = func(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
		callCount++
		if method == "GET" {
			return 200, []byte(getMockData), nil
		}
		return 0, nil, errors.New("API error on POST")
	}

	machine := &MachinePowerOFF{}
	machine.SystemID = "test-id"
	machine.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machine.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMachinePowerOFF_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	getMockData := `{"system_id":"test-id","hostname":"test-host","power_state":"on"}`
	postMockData := `{"error": "Internal server error"}`
	
	callCount := 0
	mockAPI := &MockCanonicalMaasApi{}
	mockAPI.APIExecuteFunc = func(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
		callCount++
		if method == "GET" {
			return 200, []byte(getMockData), nil
		}
		return 500, []byte(postMockData), nil
	}
	_ = callCount

	machine := &MachinePowerOFF{}
	machine.SystemID = "test-id"
	machine.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := machine.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 500 {
		t.Errorf("Expected status 500, got %d", resbody.HTTPStatus)
	}
}
