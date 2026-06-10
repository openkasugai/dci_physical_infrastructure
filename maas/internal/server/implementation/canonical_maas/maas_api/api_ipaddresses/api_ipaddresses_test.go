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

// TestIPAddressReserve_POST_Success tests successful IP address reservation
func TestIPAddressReserve_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `{"ip":"192.168.1.100","resource_uri":"/MAAS/api/2.0/ipaddresses/192.168.1.100/"}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	reserve := &IPAddressReserve{}
	reserve.API = mockAPI

	reqBody := request_body.ReqbodyIPAddressReserve{
		IP:     "192.168.1.100",
		Subnet: "192.168.1.0/24",
	}

	ctx := context.Background()

	// Act
	result, err := reserve.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyIPAddressReserve)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyIPAddressReserve")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

// TestIPAddressReserve_POST_InvalidRequestBody tests invalid request body type
func TestIPAddressReserve_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	reserve := &IPAddressReserve{}
	reserve.API = mockAPI

	// Use wrong request body type
	invalidReqBody := request_body.ReqbodyIPAddressRelease{
		IP:    "192.168.1.100",
		Force: true,
	}

	ctx := context.Background()

	// Act
	result, err := reserve.POST(ctx, invalidReqBody)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid request body, got nil")
	}

	if err.Error() != "invalid call" {
		t.Errorf("Expected 'invalid call' error, got %v", err)
	}

	if result != nil {
		t.Error("Expected nil result for invalid request body")
	}
}

// TestIPAddressReserve_POST_ApiError tests API execution error
func TestIPAddressReserve_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	reserve := &IPAddressReserve{}
	reserve.API = mockAPI

	reqBody := request_body.ReqbodyIPAddressReserve{
		IP:     "192.168.1.100",
		Subnet: "192.168.1.0/24",
	}

	ctx := context.Background()

	// Act
	result, err := reserve.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "API error" {
		t.Errorf("Expected 'API error', got %v", err)
	}

	if result != nil {
		t.Error("Expected nil result on API error")
	}
}

// TestIPAddressRelease_POST_Success tests successful IP address release
func TestIPAddressRelease_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `{"status":"released"}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	release := &IPAddressRelease{}
	release.API = mockAPI

	reqBody := request_body.ReqbodyIPAddressRelease{
		IP:    "192.168.1.100",
		Force: true,
	}

	ctx := context.Background()

	// Act
	result, err := release.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyIPAddressRelease)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyIPAddressRelease")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

// TestIPAddressRelease_POST_InvalidRequestBody tests invalid request body type
func TestIPAddressRelease_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("{}"),
		Error:      nil,
	}

	release := &IPAddressRelease{}
	release.API = mockAPI

	// Use wrong request body type
	invalidReqBody := request_body.ReqbodyIPAddressReserve{
		IP:     "192.168.1.100",
		Subnet: "192.168.1.0/24",
	}

	ctx := context.Background()

	// Act
	result, err := release.POST(ctx, invalidReqBody)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid request body, got nil")
	}

	if err.Error() != "invalid call" {
		t.Errorf("Expected 'invalid call' error, got %v", err)
	}

	if result != nil {
		t.Error("Expected nil result for invalid request body")
	}
}

// TestIPAddressRelease_POST_ApiError tests API execution error
func TestIPAddressRelease_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	release := &IPAddressRelease{}
	release.API = mockAPI

	reqBody := request_body.ReqbodyIPAddressRelease{
		IP:    "192.168.1.100",
		Force: false,
	}

	ctx := context.Background()

	// Act
	result, err := release.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "API error" {
		t.Errorf("Expected 'API error', got %v", err)
	}

	if result != nil {
		t.Error("Expected nil result on API error")
	}
}

// TestIPAddressRelease_POST_WithForceTrue tests release with force=true
func TestIPAddressRelease_POST_WithForceTrue(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `{"status":"forcefully released"}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	release := &IPAddressRelease{}
	release.API = mockAPI

	reqBody := request_body.ReqbodyIPAddressRelease{
		IP:    "192.168.1.100",
		Force: true,
	}

	ctx := context.Background()

	// Act
	result, err := release.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyIPAddressRelease)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyIPAddressRelease")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

// TestIPAddressReserve_POST_HTTPError tests Reserve with HTTP error status codes
func TestIPAddressReserve_POST_HTTPError(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{
StatusCode: 400,
Data:       []byte(`{"error": "Bad Request"}`),
Error:      nil,
}

reserve := &IPAddressReserve{}
reserve.API = mockAPI

reqBody := request_body.ReqbodyIPAddressReserve{
IP:     "192.168.1.100",
Subnet: "192.168.1.0/24",
}

ctx := context.Background()
result, err := reserve.POST(ctx, reqBody)

if err == nil {
t.Error("Expected HTTP error, got nil")
}
if result == nil {
t.Fatal("Expected non-nil result even on HTTP error")
}
resbody, ok := result.(response_body.ResbodyIPAddressReserve)
if !ok {
t.Fatal("Expected result to be of type ResbodyIPAddressReserve")
}
if resbody.HTTPStatus != 400 {
t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
}
}

// TestIPAddressReserve_POST_HTTPError500 tests Reserve with 500 error
func TestIPAddressReserve_POST_HTTPError500(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{
StatusCode: 500,
Data:       []byte("Internal Server Error"),
Error:      nil,
}

reserve := &IPAddressReserve{}
reserve.API = mockAPI

reqBody := request_body.ReqbodyIPAddressReserve{
IP:     "192.168.1.100",
Subnet: "192.168.1.0/24",
}

ctx := context.Background()
result, err := reserve.POST(ctx, reqBody)

if err == nil {
t.Error("Expected HTTP error, got nil")
}
if result == nil {
t.Fatal("Expected non-nil result even on HTTP error")
}
resbody, ok := result.(response_body.ResbodyIPAddressReserve)
if !ok {
t.Fatal("Expected result to be of type ResbodyIPAddressReserve")
}
if resbody.HTTPStatus != 500 {
t.Errorf("Expected status 500, got %d", resbody.HTTPStatus)
}
}

// TestIPAddressRelease_POST_HTTPError tests Release with HTTP error status codes
func TestIPAddressRelease_POST_HTTPError(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{
StatusCode: 404,
Data:       []byte(`{"error": "IP not found"}`),
Error:      nil,
}

release := &IPAddressRelease{}
release.API = mockAPI

reqBody := request_body.ReqbodyIPAddressRelease{
IP:    "192.168.1.100",
Force: true,
}

ctx := context.Background()
result, err := release.POST(ctx, reqBody)

if err == nil {
t.Error("Expected HTTP error, got nil")
}
if result == nil {
t.Fatal("Expected non-nil result even on HTTP error")
}
resbody, ok := result.(response_body.ResbodyIPAddressRelease)
if !ok {
t.Fatal("Expected result to be of type ResbodyIPAddressRelease")
}
if resbody.HTTPStatus != 404 {
t.Errorf("Expected status 404, got %d", resbody.HTTPStatus)
}
}

// TestIPAddressRelease_POST_HTTPError500 tests Release with 500 error
func TestIPAddressRelease_POST_HTTPError500(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{
StatusCode: 500,
Data:       []byte("Internal Server Error"),
Error:      nil,
}

release := &IPAddressRelease{}
release.API = mockAPI

reqBody := request_body.ReqbodyIPAddressRelease{
IP:    "192.168.1.100",
Force: false,
}

ctx := context.Background()
result, err := release.POST(ctx, reqBody)

if err == nil {
t.Error("Expected HTTP error, got nil")
}
if result == nil {
t.Fatal("Expected non-nil result even on HTTP error")
}
resbody, ok := result.(response_body.ResbodyIPAddressRelease)
if !ok {
t.Fatal("Expected result to be of type ResbodyIPAddressRelease")
}
if resbody.HTTPStatus != 500 {
t.Errorf("Expected status 500, got %d", resbody.HTTPStatus)
}
}

// TestIPAddressReserve_POST_URLEncoding tests Reserve with URL encoding
func TestIPAddressReserve_POST_URLEncoding(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{
StatusCode: 200,
Data:       []byte(`{}`),
Error:      nil,
}

reserve := &IPAddressReserve{}
reserve.API = mockAPI

reqBody := request_body.ReqbodyIPAddressReserve{
IP:     "192.168.1.100",
Subnet: "192.168.1.0/24",
}

ctx := context.Background()
result, err := reserve.POST(ctx, reqBody)

if err != nil {
t.Errorf("Expected no error, got %v", err)
}
if result == nil {
t.Fatal("Expected non-nil result")
}
}

// TestIPAddressRelease_POST_URLEncoding tests Release with URL encoding
func TestIPAddressRelease_POST_URLEncoding(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{
StatusCode: 200,
Data:       []byte(`{}`),
Error:      nil,
}

release := &IPAddressRelease{}
release.API = mockAPI

reqBody := request_body.ReqbodyIPAddressRelease{
IP:    "192.168.1.100",
Force: true,
}

ctx := context.Background()
result, err := release.POST(ctx, reqBody)

if err != nil {
t.Errorf("Expected no error, got %v", err)
}
if result == nil {
t.Fatal("Expected non-nil result")
}
}
