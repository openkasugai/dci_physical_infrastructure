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
	"testing"

	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/test_utils"
)

// MockCanonicalMaasApi is a mock implementation for testing
type MockCanonicalMaasApi struct {
	StatusCode int
	Data       []byte
	Error      error
}

func (m *MockCanonicalMaasApi) APIExecute(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	if m.Error != nil {
		return 0, nil, m.Error
	}
	return m.StatusCode, m.Data, nil
}

func TestInterfaces_GET_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockData := `[{"id":1,"name":"eth0","mac_address":"aa:bb:cc:dd:ee:ff","links":[]}]`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	interfaces := &Interfaces{
		SystemID: systemID,
	}
	interfaces.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := interfaces.GET(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyGetInterfaces)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetInterfaces")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}

	if len(resbody.List) == 0 {
		t.Error("Expected parsed interface list, got empty list")
	}
}

func TestInterfaces_GET_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	interfaces := &Interfaces{
		SystemID: systemID,
	}
	interfaces.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := interfaces.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestInterfaces_GET_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 404,
		Data:       []byte("Not Found"),
		Error:      nil,
	}

	interfaces := &Interfaces{
		SystemID: systemID,
	}
	interfaces.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := interfaces.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyGetInterfaces)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetInterfaces")
	}

	if resbody.HTTPStatus != 404 {
		t.Errorf("Expected status 404, got %d", resbody.HTTPStatus)
	}
}

func TestInterfaces_GET_JSONUnmarshalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	invalidJSON := `{"invalid": json}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(invalidJSON),
		Error:      nil,
	}

	interfaces := &Interfaces{
		SystemID: systemID,
	}
	interfaces.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := interfaces.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected JSON unmarshal error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on JSON unmarshal error")
	}
}

func TestInterfaceLinkSubnet_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 1
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	interfaceLink := &InterfaceLinkSubnet{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceLink.API = mockAPI

	reqBody := request_body.ReqbodyIFLinkSubnet{
		Mode:     "STATIC",
		SubnetID: 1,
	}

	ctx := context.Background()

	// Act
	result, err := interfaceLink.POST(ctx, reqBody)

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

func TestInterfaceLinkSubnet_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	interfaceLink := &InterfaceLinkSubnet{}
	interfaceLink.API = &MockCanonicalMaasApi{}

	// Invalid request body type
	invalidReqBody := "invalid request body"
	ctx := context.Background()

	// Act
	result, err := interfaceLink.POST(ctx, invalidReqBody)

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

func TestInterfaceLinkSubnet_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 1
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	interfaceLink := &InterfaceLinkSubnet{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceLink.API = mockAPI

	reqBody := request_body.ReqbodyIFLinkSubnet{
		Mode:     "STATIC",
		SubnetID: 1,
	}

	ctx := context.Background()

	// Act
	result, err := interfaceLink.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestInterfaceDisconnect_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 2
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	interfaceDisconnect := &InterfaceDisconnect{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceDisconnect.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := interfaceDisconnect.POST(ctx, nil)

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

func TestInterfaceDisconnect_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 2
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	interfaceDisconnect := &InterfaceDisconnect{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceDisconnect.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := interfaceDisconnect.POST(ctx, nil)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestInterfaceDisconnect_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 2
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 400,
		Data:       []byte("Bad Request"),
		Error:      nil,
	}

	interfaceDisconnect := &InterfaceDisconnect{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceDisconnect.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := interfaceDisconnect.POST(ctx, nil)

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

func TestInterfaceAddTag_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 1
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	interfaceAddTag := &InterfaceAddTag{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceAddTag.API = mockAPI

	reqBody := request_body.ReqbodyInterfaceTag{
		Tag: "192.168.1.100",
	}

	ctx := context.Background()

	// Act
	result, err := interfaceAddTag.POST(ctx, reqBody)

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

func TestInterfaceAddTag_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	interfaceAddTag := &InterfaceAddTag{}
	interfaceAddTag.API = &MockCanonicalMaasApi{}

	// Invalid request body type
	invalidReqBody := "invalid request body"
	ctx := context.Background()

	// Act
	result, err := interfaceAddTag.POST(ctx, invalidReqBody)

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

func TestInterfaceAddTag_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 1
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	interfaceAddTag := &InterfaceAddTag{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceAddTag.API = mockAPI

	reqBody := request_body.ReqbodyInterfaceTag{
		Tag: "192.168.1.100",
	}

	ctx := context.Background()

	// Act
	result, err := interfaceAddTag.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestInterfaceAddTag_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 1
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 500,
		Data:       []byte("Internal Server Error"),
		Error:      nil,
	}

	interfaceAddTag := &InterfaceAddTag{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceAddTag.API = mockAPI

	reqBody := request_body.ReqbodyInterfaceTag{
		Tag: "192.168.1.100",
	}

	ctx := context.Background()

	// Act
	result, err := interfaceAddTag.POST(ctx, reqBody)

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

func TestInterfaceRemoveTag_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 1
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	interfaceRemoveTag := &InterfaceRemoveTag{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceRemoveTag.API = mockAPI

	reqBody := request_body.ReqbodyInterfaceTag{
		Tag: "192.168.1.100",
	}

	ctx := context.Background()

	// Act
	result, err := interfaceRemoveTag.POST(ctx, reqBody)

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

func TestInterfaceRemoveTag_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	interfaceRemoveTag := &InterfaceRemoveTag{}
	interfaceRemoveTag.API = &MockCanonicalMaasApi{}

	// Invalid request body type
	invalidReqBody := 123
	ctx := context.Background()

	// Act
	result, err := interfaceRemoveTag.POST(ctx, invalidReqBody)

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

func TestInterfaceRemoveTag_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 1
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	interfaceRemoveTag := &InterfaceRemoveTag{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceRemoveTag.API = mockAPI

	reqBody := request_body.ReqbodyInterfaceTag{
		Tag: "192.168.1.100",
	}

	ctx := context.Background()

	// Act
	result, err := interfaceRemoveTag.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestInterfaceRemoveTag_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 1
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 404,
		Data:       []byte("Not Found"),
		Error:      nil,
	}

	interfaceRemoveTag := &InterfaceRemoveTag{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceRemoveTag.API = mockAPI

	reqBody := request_body.ReqbodyInterfaceTag{
		Tag: "192.168.1.100",
	}

	ctx := context.Background()

	// Act
	result, err := interfaceRemoveTag.POST(ctx, reqBody)

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

	if resbody.HTTPStatus != 404 {
		t.Errorf("Expected status 404, got %d", resbody.HTTPStatus)
	}
}

func TestInterfaceUpdate_PUT_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	systemID := "test-system-id"
	interfaceID := 7
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	interfaceUpdate := &InterfaceUpdate{
		Interfaces: Interfaces{SystemID: systemID},
		InterfaceID: interfaceID,
	}
	interfaceUpdate.API = mockAPI

	result, err := interfaceUpdate.PUT(context.Background(), request_body.ReqbodyInterfaceUpdate{Name: "eth0"})
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

func TestInterfaceUpdate_PUT_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	interfaceUpdate := &InterfaceUpdate{}
	interfaceUpdate.API = &MockCanonicalMaasApi{}

	result, err := interfaceUpdate.PUT(context.Background(), "invalid body")
	if err == nil {
		t.Error("Expected error for invalid request body, got nil")
	}
	if result != nil {
		t.Error("Expected nil result for invalid request body")
	}
	if err.Error() != "invalid call" {
		t.Errorf("Expected error message 'invalid call', got '%s'", err.Error())
	}
}

func TestInterfaceUpdate_PUT_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	interfaceUpdate := &InterfaceUpdate{
		Interfaces: Interfaces{SystemID: "test-system-id"},
		InterfaceID: 7,
	}
	interfaceUpdate.API = &MockCanonicalMaasApi{Error: errors.New("API error")}

	result, err := interfaceUpdate.PUT(context.Background(), request_body.ReqbodyInterfaceUpdate{Name: "eth0"})
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if result != nil {
		t.Error("Expected nil result on API error")
	}
}

func TestInterfaceUpdate_PUT_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	interfaceUpdate := &InterfaceUpdate{
		Interfaces: Interfaces{SystemID: "test-system-id"},
		InterfaceID: 7,
	}
	interfaceUpdate.API = &MockCanonicalMaasApi{
		StatusCode: 404,
		Data:       []byte("Not Found"),
		Error:      nil,
	}

	result, err := interfaceUpdate.PUT(context.Background(), request_body.ReqbodyInterfaceUpdate{Name: "eth0"})
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
	if resbody.HTTPStatus != 404 {
		t.Errorf("Expected status 404, got %d", resbody.HTTPStatus)
	}
}

// Edge case tests for full coverage
func TestInterfaces_GET_EmptySystemID(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("[]"),
		Error:      nil,
	}

	interfaces := &Interfaces{
		SystemID: "", // Empty system ID
	}
	interfaces.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := interfaces.GET(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error even with empty system ID, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestInterfaceLinkSubnet_POST_ZeroInterfaceID(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	systemID := "test-system-id"
	interfaceID := 0 // Zero interface ID
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	interfaceLink := &InterfaceLinkSubnet{
		Interfaces: Interfaces{
			SystemID: systemID,
		},
		InterfaceID: interfaceID,
	}
	interfaceLink.API = mockAPI

	reqBody := request_body.ReqbodyIFLinkSubnet{
		Mode:     "AUTO",
		SubnetID: 0,
	}

	ctx := context.Background()

	// Act
	result, err := interfaceLink.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error even with zero interface ID, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// Benchmark tests
func BenchmarkInterfaces_GET(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("[]"),
		Error:      nil,
	}

	interfaces := &Interfaces{
		SystemID: "test-system-id",
	}
	interfaces.API = mockAPI
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := interfaces.GET(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInterfaceLinkSubnet_POST(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	interfaceLink := &InterfaceLinkSubnet{
		Interfaces: Interfaces{
			SystemID: "test-system-id",
		},
		InterfaceID: 1,
	}
	interfaceLink.API = mockAPI
	ctx := context.Background()

	reqBody := request_body.ReqbodyIFLinkSubnet{
		Mode:     "STATIC",
		SubnetID: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := interfaceLink.POST(ctx, reqBody)
		if err != nil {
			b.Fatal(err)
		}
	}
}
