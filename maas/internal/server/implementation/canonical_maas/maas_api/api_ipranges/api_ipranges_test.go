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
	"testing"

	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/test_utils"
)

// Mock implementation of CanonicalMaasApi interface
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

func TestIPranges_POST_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `{"status": "success"}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 201,
		Data:       []byte(mockData),
		Error:      nil,
	}

	ipranges := &IPranges{}
	ipranges.API = mockAPI

	ctx := context.Background()
	reqBody := request_body.ReqbodyIPRanges{
		SubnetID: 1,
		StartIP:  "192.168.1.10",
		EndIP:    "192.168.1.20",
		Type:     "reserved",
	}

	// Act
	result, err := ipranges.POST(ctx, reqBody)

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

func TestIPranges_POST_InvalidRequestBody(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	ipranges := &IPranges{}

	ctx := context.Background()
	invalidReqBody := request_body.ReqbodyCommon{} // Wrong type

	// Act
	result, err := ipranges.POST(ctx, invalidReqBody)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid request body type, got nil")
	}

	if err.Error() != "invalid call" {
		t.Errorf("Expected 'invalid call' error, got %v", err)
	}

	if result != nil {
		t.Error("Expected result to be nil")
	}
}

func TestIPranges_POST_ApiError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	expectedError := errors.New("API execution failed")
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      expectedError,
	}

	ipranges := &IPranges{}
	ipranges.API = mockAPI

	ctx := context.Background()
	reqBody := request_body.ReqbodyIPRanges{
		SubnetID: 1,
		StartIP:  "192.168.1.10",
		EndIP:    "192.168.1.20",
		Type:     "reserved",
	}

	// Act
	result, err := ipranges.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err != expectedError {
		t.Errorf("Expected error to be %v, got %v", expectedError, err)
	}

	if result != nil {
		t.Error("Expected result to be nil")
	}
}

func TestIPranges_POST_HTTPError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockData := `{"error": "Bad Request"}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 400,
		Data:       []byte(mockData),
		Error:      nil,
	}

	ipranges := &IPranges{}
	ipranges.API = mockAPI

	ctx := context.Background()
	reqBody := request_body.ReqbodyIPRanges{
		SubnetID: 1,
		StartIP:  "192.168.1.10",
		EndIP:    "192.168.1.20",
		Type:     "reserved",
	}

	// Act
	result, err := ipranges.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected HTTP error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resbody, ok := result.(response_body.ResbodyCommon)
	if !ok {
		t.Fatal("Expected result to be of type ResbodyCommon")
	}

	if resbody.HTTPStatus != 400 {
		t.Errorf("Expected status 400, got %d", resbody.HTTPStatus)
	}
}

func TestIPranges_POST_WithDifferentTypes(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test with different IP range types
	types := []string{"reserved", "dynamic", "static"}

	for _, rangeType := range types {
		t.Run("Type_"+rangeType, func(t *testing.T) {
			// Arrange
			mockData := `{"status": "success"}`
			mockAPI := &MockCanonicalMaasApi{
				StatusCode: 201,
				Data:       []byte(mockData),
				Error:      nil,
			}

			ipranges := &IPranges{}
			ipranges.API = mockAPI

			ctx := context.Background()
			reqBody := request_body.ReqbodyIPRanges{
				SubnetID: 1,
				StartIP:  "192.168.1.10",
				EndIP:    "192.168.1.20",
				Type:     rangeType,
			}

			// Act
			result, err := ipranges.POST(ctx, reqBody)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for type %s, got %v", rangeType, err)
			}

			if result == nil {
				t.Fatalf("Expected non-nil result for type %s", rangeType)
			}
		})
	}
}

func TestIPranges_POST_WithDifferentSubnetIDs(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test with different subnet IDs
	subnetIDs := []int{1, 10, 100, 999}

	for _, subnetID := range subnetIDs {
		t.Run("SubnetID_"+string(rune(subnetID)), func(t *testing.T) {
			// Arrange
			mockData := `{"status": "success"}`
			mockAPI := &MockCanonicalMaasApi{
				StatusCode: 201,
				Data:       []byte(mockData),
				Error:      nil,
			}

			ipranges := &IPranges{}
			ipranges.API = mockAPI

			ctx := context.Background()
			reqBody := request_body.ReqbodyIPRanges{
				SubnetID: subnetID,
				StartIP:  "192.168.1.10",
				EndIP:    "192.168.1.20",
				Type:     "reserved",
			}

			// Act
			result, err := ipranges.POST(ctx, reqBody)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for subnet ID %d, got %v", subnetID, err)
			}

			if result == nil {
				t.Fatalf("Expected non-nil result for subnet ID %d", subnetID)
			}
		})
	}
}

func TestIPranges_POST_WithSpecialCharacters(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test with IP addresses that might need URL encoding
	testCases := []struct {
		name    string
		startIP string
		endIP   string
	}{
		{"Standard_IPv4", "192.168.1.10", "192.168.1.20"},
		{"Edge_IPv4", "0.0.0.1", "255.255.255.254"},
		{"Single_IP", "192.168.1.100", "192.168.1.100"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockData := `{"status": "success"}`
			mockAPI := &MockCanonicalMaasApi{
				StatusCode: 201,
				Data:       []byte(mockData),
				Error:      nil,
			}

			ipranges := &IPranges{}
			ipranges.API = mockAPI

			ctx := context.Background()
			reqBody := request_body.ReqbodyIPRanges{
				SubnetID: 1,
				StartIP:  tc.startIP,
				EndIP:    tc.endIP,
				Type:     "reserved",
			}

			// Act
			result, err := ipranges.POST(ctx, reqBody)

			// Assert
			if err != nil {
				t.Errorf("Expected no error for %s, got %v", tc.name, err)
			}

			if result == nil {
				t.Fatalf("Expected non-nil result for %s", tc.name)
			}
		})
	}
}

func TestIPranges_POST_EmptyFields(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test with empty string fields
	testCases := []struct {
		name    string
		reqBody request_body.ReqbodyIPRanges
	}{
		{
			"Empty_StartIP",
			request_body.ReqbodyIPRanges{
				SubnetID: 1,
				StartIP:  "",
				EndIP:    "192.168.1.20",
				Type:     "reserved",
			},
		},
		{
			"Empty_EndIP",
			request_body.ReqbodyIPRanges{
				SubnetID: 1,
				StartIP:  "192.168.1.10",
				EndIP:    "",
				Type:     "reserved",
			},
		},
		{
			"Empty_Type",
			request_body.ReqbodyIPRanges{
				SubnetID: 1,
				StartIP:  "192.168.1.10",
				EndIP:    "192.168.1.20",
				Type:     "",
			},
		},
		{
			"Zero_SubnetID",
			request_body.ReqbodyIPRanges{
				SubnetID: 0,
				StartIP:  "192.168.1.10",
				EndIP:    "192.168.1.20",
				Type:     "reserved",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockData := `{"status": "success"}`
			mockAPI := &MockCanonicalMaasApi{
				StatusCode: 201,
				Data:       []byte(mockData),
				Error:      nil,
			}

			ipranges := &IPranges{}
			ipranges.API = mockAPI

			ctx := context.Background()

			// Act
			result, err := ipranges.POST(ctx, tc.reqBody)

			// Assert - Should still work even with empty fields
			if err != nil {
				t.Errorf("Expected no error for %s, got %v", tc.name, err)
			}

			if result == nil {
				t.Fatalf("Expected non-nil result for %s", tc.name)
			}
		})
	}
}

func TestIPranges_POST_ContextCancellation(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      context.Canceled,
	}

	ipranges := &IPranges{}
	ipranges.API = mockAPI

	reqBody := request_body.ReqbodyIPRanges{
		SubnetID: 1,
		StartIP:  "192.168.1.10",
		EndIP:    "192.168.1.20",
		Type:     "reserved",
	}

	// Act
	result, err := ipranges.POST(ctx, reqBody)

	// Assert
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if result != nil {
		t.Error("Expected result to be nil")
	}
}

// Benchmark tests
func BenchmarkIPranges_POST_Success(b *testing.B) {
	mockData := `{"status": "success"}`
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 201,
		Data:       []byte(mockData),
		Error:      nil,
	}

	ipranges := &IPranges{}
	ipranges.API = mockAPI

	ctx := context.Background()
	reqBody := request_body.ReqbodyIPRanges{
		SubnetID: 1,
		StartIP:  "192.168.1.10",
		EndIP:    "192.168.1.20",
		Type:     "reserved",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ipranges.POST(ctx, reqBody)
	}
}

func BenchmarkIPranges_POST_InvalidBody(b *testing.B) {
	ipranges := &IPranges{}

	ctx := context.Background()
	invalidReqBody := request_body.ReqbodyCommon{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ipranges.POST(ctx, invalidReqBody)
	}
}

func BenchmarkIPranges_POST_Error(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{
		StatusCode: 0,
		Data:       nil,
		Error:      errors.New("benchmark error"),
	}

	ipranges := &IPranges{}
	ipranges.API = mockAPI

	ctx := context.Background()
	reqBody := request_body.ReqbodyIPRanges{
		SubnetID: 1,
		StartIP:  "192.168.1.10",
		EndIP:    "192.168.1.20",
		Type:     "reserved",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ipranges.POST(ctx, reqBody)
	}
}
