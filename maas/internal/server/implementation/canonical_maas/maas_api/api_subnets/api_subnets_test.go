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

func TestSubnets_GET_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `[{"id":1,"cidr":"192.168.1.0/24"}]`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnets.GET(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyGetSubnets)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetSubnets")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}

	if len(resbody.List) == 0 {
		t.Error("Expected parsed subnets list, got empty list")
	}
}

func TestSubnets_GET_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnets.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestSubnets_GET_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 500,
		Data:       []byte("Internal Server Error"),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnets.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyGetSubnets)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetSubnets")
	}

	if resbody.HTTPStatus != 500 {
		t.Errorf("Expected status 500, got %d", resbody.HTTPStatus)
	}
}

func TestSubnets_GET_JSONUnmarshalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - invalid JSON
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("invalid json"),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnets.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected JSON unmarshal error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on JSON unmarshal error")
	}
}

func TestSubnets_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `{"id":1}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 201,
		Data:       []byte(mockData),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	reqBody := request_body.ReqbodySubnets{
		Cidr:     "192.168.1.0/24",
		FabricID: 1,
		Vid:      100,
	}

	ctx := context.Background()

	// Act
	result, err := subnets.POST(ctx, reqBody)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyPostSubnets)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyPostSubnets")
	}

	if resbody.HTTPStatus != 201 {
		t.Errorf("Expected status 201, got %d", resbody.HTTPStatus)
	}
}

func TestSubnets_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	subnets := &Subnets{}
	subnets.API = &MockCanonicalMaasApi{}

	// Invalid request body type
	invalidReqBody := "invalid request body"
	ctx := context.Background()

	// Act
	result, err := subnets.POST(ctx, invalidReqBody)

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

func TestSubnets_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	reqBody := request_body.ReqbodySubnets{
		Cidr:     "192.168.1.0/24",
		FabricID: 1,
		Vid:      100,
	}

	ctx := context.Background()

	// Act
	result, err := subnets.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestSubnets_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 400,
		Data:       []byte("Bad Request"),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	reqBody := request_body.ReqbodySubnets{
		Cidr:     "invalid-cidr",
		FabricID: 1,
		Vid:      100,
	}

	ctx := context.Background()

	// Act
	result, err := subnets.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even on HTTP error")
	}

	resbody, ok := result.(response_body.ResbodyPostSubnets)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyPostSubnets")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

func TestSubnets_POST_JSONUnmarshalError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange - invalid JSON response
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 201,
		Data:       []byte("invalid json"),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	reqBody := request_body.ReqbodySubnets{
		Cidr:     "192.168.1.0/24",
		FabricID: 1,
		Vid:      100,
	}

	ctx := context.Background()

	// Act
	_, err := subnets.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected JSON unmarshal error, got nil")
	}
}

// Edge case tests for full coverage
func TestSubnets_POST_EmptyFields(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 400,
		Data:       []byte("Bad Request"),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	// Empty request body
	reqBody := request_body.ReqbodySubnets{}
	ctx := context.Background()

	// Act
	_, _ = subnets.POST(ctx, reqBody)

	// Assert - should handle empty fields gracefully
	// Note: This test validates the request handling path
}

func TestSubnets_GET_ValidJSON_EmptyArray(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `[]`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnets.GET(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error for empty array, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyGetSubnets)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyGetSubnets")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}
}

// Benchmark tests
func BenchmarkSubnets_GET(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(`[{"id":1,"cidr":"192.168.1.0/24"}]`),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := subnets.GET(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSubnets_POST(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 201,
		Data:       []byte(`{"id":1}`),
		Error:      nil,
	}

	subnets := &Subnets{}
	subnets.API = mockAPI
	ctx := context.Background()

	reqBody := request_body.ReqbodySubnets{
		Cidr:     "192.168.1.0/24",
		FabricID: 1,
		Vid:      100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := subnets.POST(ctx, reqBody)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestSubnetUnreservedIPRanges_GET_Success tests successful retrieval of unreserved IP ranges
func TestSubnetUnreservedIPRanges_GET_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `[{"start":"192.168.1.100","end":"192.168.1.150","num_addresses":51}]`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	subnetRanges := &SubnetUnreservedIPRanges{
		SubnetID: 1,
	}
	subnetRanges.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnetRanges.GET(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodySubnetUnreservedIPRanges)
	if !ok {
		t.Fatal("Expected result to be of type ResbodySubnetUnreservedIPRanges")
	}

	if resbody.HTTPStatus != 200 {
		t.Errorf("Expected status 200, got %d", resbody.HTTPStatus)
	}

	if len(resbody.List) == 0 {
		t.Error("Expected parsed IP ranges list, got empty list")
	}

	if len(resbody.List) > 0 {
		if resbody.List[0].Start != "192.168.1.100" {
			t.Errorf("Expected start IP 192.168.1.100, got %s", resbody.List[0].Start)
		}
		if resbody.List[0].End != "192.168.1.150" {
			t.Errorf("Expected end IP 192.168.1.150, got %s", resbody.List[0].End)
		}
	}
}

// TestSubnetUnreservedIPRanges_GET_ApiError tests API execution error
func TestSubnetUnreservedIPRanges_GET_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("API error"),
	}

	subnetRanges := &SubnetUnreservedIPRanges{
		SubnetID: 1,
	}
	subnetRanges.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnetRanges.GET(ctx)

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

// TestSubnetUnreservedIPRanges_GET_InvalidJSON tests invalid JSON response
func TestSubnetUnreservedIPRanges_GET_InvalidJSON(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte("INVALID JSON"),
		Error:      nil,
	}

	subnetRanges := &SubnetUnreservedIPRanges{
		SubnetID: 1,
	}
	subnetRanges.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnetRanges.GET(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if result != nil {
		t.Error("Expected nil result on JSON parse error")
	}
}

// TestSubnetUnreservedIPRanges_GET_EmptyList tests empty unreserved IP ranges
func TestSubnetUnreservedIPRanges_GET_EmptyList(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `[]`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 200,
		Data:       []byte(mockData),
		Error:      nil,
	}

	subnetRanges := &SubnetUnreservedIPRanges{
		SubnetID: 1,
	}
	subnetRanges.API = mockAPI

	ctx := context.Background()

	// Act
	result, err := subnetRanges.GET(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodySubnetUnreservedIPRanges)
	if !ok {
		t.Fatal("Expected result to be of type ResbodySubnetUnreservedIPRanges")
	}

	if len(resbody.List) != 0 {
		t.Errorf("Expected empty list, got %d items", len(resbody.List))
	}
}

// TestSubnetUnreservedIPRanges_GET_Success tests successful GET for SubnetUnreservedIPRanges
