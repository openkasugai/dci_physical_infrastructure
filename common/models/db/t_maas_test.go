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

package db

import (
	"testing"
)

// TestTMaas_TableName tests the TableName method
func TestTMaas_TableName(t *testing.T) {
	maas := &TMaas{}
	expected := "t_maas"
	if maas.TableName() != expected {
		t.Errorf("TableName() = %v, want %v", maas.TableName(), expected)
	}
}

// TestTMaas_QueryParameter tests the QueryParameter method
func TestTMaas_QueryParameter(t *testing.T) {
	tests := []struct {
		name     string
		maasID   int32
		expected string
	}{
		{
			name:     "MaasID 1",
			maasID:   1,
			expected: "id=1",
		},
		{
			name:     "MaasID 0",
			maasID:   0,
			expected: "id=0",
		},
		{
			name:     "MaasID negative",
			maasID:   -1,
			expected: "id=-1",
		},
		{
			name:     "MaasID large value",
			maasID:   999999,
			expected: "id=999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maas := &TMaas{
				MaasID: tt.maasID,
			}

			result := maas.QueryParameter()

			if result != tt.expected {
				t.Errorf("QueryParameter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestTMaas_Parse tests parsing JSON array
func TestTMaas_Parse(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expectErr bool
		expected  int
	}{
		{
			name: "Valid single item",
			input: []interface{}{
				map[string]interface{}{
					"id":           float64(1),
					"physical_infra_id": float64(100),
					"access_url":        "https://maas.example.com",
					"api_key":           "test-api-key-123",
					"status":            float64(1),
					"product_info":      `{"vendor":"Canonical","product":"MaaS","version":"3.0"}`,
					"extra_parameters":  `{"param1":"value1"}`,
				},
			},
			expectErr: false,
			expected:  1,
		},
		{
			name: "Valid multiple items",
			input: []interface{}{
				map[string]interface{}{
					"id":           float64(1),
					"physical_infra_id": float64(100),
					"access_url":        "https://maas1.example.com",
					"api_key":           "key-1",
					"status":            float64(1),
					"product_info":      `{}`,
					"extra_parameters":  `{}`,
				},
				map[string]interface{}{
					"id":           float64(2),
					"physical_infra_id": float64(200),
					"access_url":        "https://maas2.example.com",
					"api_key":           "key-2",
					"status":            float64(2),
					"product_info":      `{"vendor":"Test"}`,
					"extra_parameters":  `{"test":"data"}`,
				},
			},
			expectErr: false,
			expected:  2,
		},
		{
			name:      "Invalid input type - not an array",
			input:     "not an array",
			expectErr: true,
			expected:  0,
		},
		{
			name:      "Invalid input type - map instead of array",
			input:     map[string]interface{}{"id": float64(1)},
			expectErr: true,
			expected:  0,
		},
		{
			name: "Invalid item format - not an object",
			input: []interface{}{
				"invalid item",
			},
			expectErr: true,
			expected:  0,
		},
		{
			name:      "Empty array",
			input:     []interface{}{},
			expectErr: false,
			expected:  0,
		},
		{
			name: "Missing required field - maas_id",
			input: []interface{}{
				map[string]interface{}{
					"physical_infra_id": float64(100),
					"access_url":        "https://maas.example.com",
					"api_key":           "test-key",
					"status":            float64(1),
					"product_info":      `{}`,
					"extra_parameters":  `{}`,
				},
			},
			expectErr: true,
			expected:  0,
		},
		{
			name: "Missing required field - access_url",
			input: []interface{}{
				map[string]interface{}{
					"id":           float64(1),
					"physical_infra_id": float64(100),
					"api_key":           "test-key",
					"status":            float64(1),
					"product_info":      `{}`,
					"extra_parameters":  `{}`,
				},
			},
			expectErr: true,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maas := &TMaas{}
			result, err := maas.Parse(tt.input)

			if tt.expectErr && err == nil {
				t.Error("Parse() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("Parse() result length = %v, want %v", len(result), tt.expected)
			}
		})
	}
}

// TestTMaas_ParseSingle_FieldTypes tests parseSingle with various field type errors
func TestTMaas_ParseSingle_FieldTypes(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid all fields",
			input: map[string]interface{}{
				"id":           float64(1),
				"physical_infra_id": float64(100),
				"access_url":        "https://maas.example.com",
				"api_key":           "api-key-123",
				"status":            float64(1),
				"product_info":      `{"vendor":"Canonical"}`,
				"extra_parameters":  `{"key":"value"}`,
			},
			expectErr: false,
		},
		{
			name: "maas_id wrong type (string)",
			input: map[string]interface{}{
				"id":           "invalid",
				"physical_infra_id": float64(100),
				"access_url":        "https://maas.example.com",
				"api_key":           "key",
				"status":            float64(1),
				"product_info":      `{}`,
				"extra_parameters":  `{}`,
			},
			expectErr: true,
			errMsg:    "invalid type for id",
		},
		{
			name: "physical_infra_id wrong type (string)",
			input: map[string]interface{}{
				"id":           float64(1),
				"physical_infra_id": "invalid",
				"access_url":        "https://maas.example.com",
				"api_key":           "key",
				"status":            float64(1),
				"product_info":      `{}`,
				"extra_parameters":  `{}`,
			},
			expectErr: true,
			errMsg:    "invalid type for physical_infra_id",
		},
		{
			name: "access_url wrong type (number)",
			input: map[string]interface{}{
				"id":           float64(1),
				"physical_infra_id": float64(100),
				"access_url":        float64(123),
				"api_key":           "key",
				"status":            float64(1),
				"product_info":      `{}`,
				"extra_parameters":  `{}`,
			},
			expectErr: true,
			errMsg:    "invalid type for access_url",
		},
		{
			name: "api_key wrong type (bool)",
			input: map[string]interface{}{
				"id":           float64(1),
				"physical_infra_id": float64(100),
				"access_url":        "https://maas.example.com",
				"api_key":           true,
				"status":            float64(1),
				"product_info":      `{}`,
				"extra_parameters":  `{}`,
			},
			expectErr: true,
			errMsg:    "invalid type for api_key",
		},
		{
			name: "status wrong type (string)",
			input: map[string]interface{}{
				"id":           float64(1),
				"physical_infra_id": float64(100),
				"access_url":        "https://maas.example.com",
				"api_key":           "key",
				"status":            "invalid",
				"product_info":      `{}`,
				"extra_parameters":  `{}`,
			},
			expectErr: true,
			errMsg:    "invalid type for status",
		},
		{
			name: "product_info wrong type (number)",
			input: map[string]interface{}{
				"id":           float64(1),
				"physical_infra_id": float64(100),
				"access_url":        "https://maas.example.com",
				"api_key":           "key",
				"status":            float64(1),
				"product_info":      float64(123),
				"extra_parameters":  `{}`,
			},
			expectErr: true,
			errMsg:    "invalid type for product_info",
		},
		{
			name: "extra_parameters wrong type (bool)",
			input: map[string]interface{}{
				"id":           float64(1),
				"physical_infra_id": float64(100),
				"access_url":        "https://maas.example.com",
				"api_key":           "key",
				"status":            float64(1),
				"product_info":      `{}`,
				"extra_parameters":  false,
			},
			expectErr: true,
			errMsg:    "invalid type for extra_parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maas := &TMaas{}
			err := maas.parseSingle(tt.input)

			if tt.expectErr && err == nil {
				t.Error("parseSingle() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("parseSingle() unexpected error = %v", err)
			}
			if tt.expectErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("parseSingle() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestTMaas_ParseSingle_FieldValues tests parseSingle with correct field value assignment
func TestTMaas_ParseSingle_FieldValues(t *testing.T) {
	input := map[string]interface{}{
		"id":           float64(42),
		"physical_infra_id": float64(999),
		"access_url":        "https://test-maas.local",
		"api_key":           "secret-key-xyz",
		"status":            float64(2),
		"product_info":      `{"vendor":"Canonical","product":"MaaS","version":"3.1"}`,
		"extra_parameters":  `{"setting1":"value1","setting2":"value2"}`,
	}

	maas := &TMaas{}
	err := maas.parseSingle(input)

	if err != nil {
		t.Fatalf("parseSingle() unexpected error = %v", err)
	}

	if maas.MaasID != 42 {
		t.Errorf("MaasID = %v, want %v", maas.MaasID, 42)
	}
	if maas.PhysicalInfraId != 999 {
		t.Errorf("PhysicalInfraId = %v, want %v", maas.PhysicalInfraId, 999)
	}
	if maas.AccessURL != "https://test-maas.local" {
		t.Errorf("AccessURL = %v, want %v", maas.AccessURL, "https://test-maas.local")
	}
	if maas.ApiKey != "secret-key-xyz" {
		t.Errorf("ApiKey = %v, want %v", maas.ApiKey, "secret-key-xyz")
	}
	if maas.Status != 2 {
		t.Errorf("Status = %v, want %v", maas.Status, 2)
	}
	if maas.ProductInfo != `{"vendor":"Canonical","product":"MaaS","version":"3.1"}` {
		t.Errorf("ProductInfo = %v, want %v", maas.ProductInfo, `{"vendor":"Canonical","product":"MaaS","version":"3.1"}`)
	}
	if maas.ExtraParameters != `{"setting1":"value1","setting2":"value2"}` {
		t.Errorf("ExtraParameters = %v, want %v", maas.ExtraParameters, `{"setting1":"value1","setting2":"value2"}`)
	}
}

// TestTMaas_ParseSingle_BoundaryValues tests parseSingle with boundary values
func TestTMaas_ParseSingle_BoundaryValues(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		expectErr bool
	}{
		{
			name: "Zero values",
			input: map[string]interface{}{
				"id":           float64(0),
				"physical_infra_id": float64(0),
				"access_url":        "",
				"api_key":           "",
				"status":            float64(0),
				"product_info":      "",
				"extra_parameters":  "",
			},
			expectErr: false,
		},
		{
			name: "Negative values for IDs",
			input: map[string]interface{}{
				"id":           float64(-1),
				"physical_infra_id": float64(-999),
				"access_url":        "https://maas.example.com",
				"api_key":           "key",
				"status":            float64(-1),
				"product_info":      `{}`,
				"extra_parameters":  `{}`,
			},
			expectErr: false,
		},
		{
			name: "Large values",
			input: map[string]interface{}{
				"id":           float64(2147483647), // max int32
				"physical_infra_id": float64(2147483647),
				"access_url":        "https://very-long-url.example.com/with/many/segments",
				"api_key":           "very-long-api-key-" + string(make([]byte, 100)),
				"status":            float64(2147483647),
				"product_info":      `{"key":"value"}`,
				"extra_parameters":  `{"key":"value"}`,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maas := &TMaas{}
			err := maas.parseSingle(tt.input)

			if tt.expectErr && err == nil {
				t.Error("parseSingle() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("parseSingle() unexpected error = %v", err)
			}
		})
	}
}

// TestTableNameTMaas tests the constant
func TestTableNameTMaas(t *testing.T) {
	expected := "t_maas"
	if TableNameTMaas != expected {
		t.Errorf("TableNameTMaas = %v, want %v", TableNameTMaas, expected)
	}
}
