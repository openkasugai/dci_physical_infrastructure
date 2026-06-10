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

// TestTCdi_TableName tests the TableName method
func TestTCdi_TableName(t *testing.T) {
	cdi := &TCdi{}
	expected := "t_cdi"
	if cdi.TableName() != expected {
		t.Errorf("TableName() = %v, want %v", cdi.TableName(), expected)
	}
}

// TestTCdi_QueryParameter tests the QueryParameter method
func TestTCdi_QueryParameter(t *testing.T) {
	tests := []struct {
		name     string
		cdiID    string
		expected string
	}{
		{
			name:     "Standard CDI ID",
			cdiID:    "cdi-001",
			expected: "cdi_id=eq.cdi-001",
		},
		{
			name:     "Empty CDI ID",
			cdiID:    "",
			expected: "cdi_id=eq.",
		},
		{
			name:     "CDI ID with special characters",
			cdiID:    "cdi-test-123_abc",
			expected: "cdi_id=eq.cdi-test-123_abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdi := &TCdi{CdiID: tt.cdiID}
			result := cdi.QueryParameter()
			if result != tt.expected {
				t.Errorf("QueryParameter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestTCdi_Parse tests parsing JSON array to TCdi slice
func TestTCdi_Parse(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expectErr bool
		expected  int // expected length of result slice
	}{
		{
			name: "Valid single item array",
			input: []interface{}{
				map[string]interface{}{
					"cdi_id":           "cdi-001",
					"remote_host":      "192.168.1.10",
					"remote_user":      "admin",
					"group_name":       "group1",
					"product_info":     `{"vendor":"Fujitsu","product_name":"PG-CDI"}`,
					"extra_parameters": `{"param1":"value1"}`,
				},
			},
			expectErr: false,
			expected:  1,
		},
		{
			name: "Valid multiple items array",
			input: []interface{}{
				map[string]interface{}{
					"cdi_id":           "cdi-001",
					"remote_host":      "192.168.1.10",
					"remote_user":      "admin",
					"group_name":       "group1",
					"product_info":     `{"vendor":"Fujitsu"}`,
					"extra_parameters": `{}`,
				},
				map[string]interface{}{
					"cdi_id":           "cdi-002",
					"remote_host":      "192.168.1.11",
					"remote_user":      "root",
					"group_name":       "group2",
					"product_info":     `{"vendor":"Dell"}`,
					"extra_parameters": `{"param2":"value2"}`,
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
			name: "Invalid item type - not a map",
			input: []interface{}{
				"not a map",
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
			name: "Missing required field - cdi_id",
			input: []interface{}{
				map[string]interface{}{
					"remote_host":      "192.168.1.10",
					"remote_user":      "admin",
					"group_name":       "group1",
					"product_info":     `{}`,
					"extra_parameters": `{}`,
				},
			},
			expectErr: true,
			expected:  0,
		},
		{
			name: "Invalid type for cdi_id - number instead of string",
			input: []interface{}{
				map[string]interface{}{
					"cdi_id":           123,
					"remote_host":      "192.168.1.10",
					"remote_user":      "admin",
					"group_name":       "group1",
					"product_info":     `{}`,
					"extra_parameters": `{}`,
				},
			},
			expectErr: true,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdi := &TCdi{}
			result, err := cdi.Parse(tt.input)
			
			if tt.expectErr && err == nil {
				t.Error("Parse() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("Parse() result length = %v, want %v", len(result), tt.expected)
			}
			
			// Validate parsed data for successful cases
			if !tt.expectErr && len(result) > 0 {
				first := result[0]
				if first.CdiID == "" {
					t.Error("Parse() result has empty CdiID")
				}
			}
		})
	}
}

// TestTCdi_ParseSingle tests parsing a single JSON object
func TestTCdi_ParseSingle(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		expectErr bool
		validate  func(*testing.T, *TCdi)
	}{
		{
			name: "Valid complete data",
			input: map[string]interface{}{
				"cdi_id":           "cdi-001",
				"remote_host":      "192.168.1.10",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{"vendor":"Fujitsu"}`,
				"extra_parameters": `{"key":"value"}`,
			},
			expectErr: false,
			validate: func(t *testing.T, cdi *TCdi) {
				if cdi.CdiID != "cdi-001" {
					t.Errorf("CdiID = %v, want cdi-001", cdi.CdiID)
				}
				if cdi.RemoteHost != "192.168.1.10" {
					t.Errorf("RemoteHost = %v, want 192.168.1.10", cdi.RemoteHost)
				}
				if cdi.RemoteUser != "admin" {
					t.Errorf("RemoteUser = %v, want admin", cdi.RemoteUser)
				}
				if cdi.GroupName != "group1" {
					t.Errorf("GroupName = %v, want group1", cdi.GroupName)
				}
			},
		},
		{
			name: "Missing cdi_id",
			input: map[string]interface{}{
				"remote_host":      "192.168.1.10",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{}`,
				"extra_parameters": `{}`,
			},
			expectErr: true,
		},
		{
			name: "Missing remote_host",
			input: map[string]interface{}{
				"cdi_id":           "cdi-001",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{}`,
				"extra_parameters": `{}`,
			},
			expectErr: true,
		},
		{
			name: "Missing remote_user",
			input: map[string]interface{}{
				"cdi_id":           "cdi-001",
				"remote_host":      "192.168.1.10",
				"group_name":       "group1",
				"product_info":     `{}`,
				"extra_parameters": `{}`,
			},
			expectErr: true,
		},
		{
			name: "Missing group_name",
			input: map[string]interface{}{
				"cdi_id":           "cdi-001",
				"remote_host":      "192.168.1.10",
				"remote_user":      "admin",
				"product_info":     `{}`,
				"extra_parameters": `{}`,
			},
			expectErr: true,
		},
		{
			name: "Missing product_info",
			input: map[string]interface{}{
				"cdi_id":           "cdi-001",
				"remote_host":      "192.168.1.10",
				"remote_user":      "admin",
				"group_name":       "group1",
				"extra_parameters": `{}`,
			},
			expectErr: false,
		},
		{
			name: "Missing extra_parameters",
			input: map[string]interface{}{
				"cdi_id":       "cdi-001",
				"remote_host":  "192.168.1.10",
				"remote_user":  "admin",
				"group_name":   "group1",
				"product_info": `{}`,
			},
			expectErr: false,
		},
		{
			name: "Invalid type for cdi_id",
			input: map[string]interface{}{
				"cdi_id":           123,
				"remote_host":      "192.168.1.10",
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{}`,
				"extra_parameters": `{}`,
			},
			expectErr: true,
		},
		{
			name: "Invalid type for remote_host",
			input: map[string]interface{}{
				"cdi_id":           "cdi-001",
				"remote_host":      123,
				"remote_user":      "admin",
				"group_name":       "group1",
				"product_info":     `{}`,
				"extra_parameters": `{}`,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdi := &TCdi{}
			err := cdi.parseSingle(tt.input)
			
			if tt.expectErr && err == nil {
				t.Error("parseSingle() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("parseSingle() unexpected error = %v", err)
			}
			
			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, cdi)
			}
		})
	}
}

// TestTableNameTCdi tests the constant table name
func TestTableNameTCdi(t *testing.T) {
	expected := "t_cdi"
	if TableNameTCdi != expected {
		t.Errorf("TableNameTCdi = %v, want %v", TableNameTCdi, expected)
	}
}

// TestTCdi_JSONTags tests that JSON tags are correctly defined
func TestTCdi_JSONTags(t *testing.T) {
	cdi := TCdi{
		CdiID:           "test-id",
		RemoteHost:      "test-host",
		RemoteUser:      "test-user",
		GroupName:       "test-group",
		ProductInfo:     "test-product",
		ExtraParameters: "test-params",
	}
	
	// Verify struct fields are accessible
	if cdi.CdiID != "test-id" {
		t.Error("CdiID field not accessible")
	}
	if cdi.RemoteHost != "test-host" {
		t.Error("RemoteHost field not accessible")
	}
	if cdi.RemoteUser != "test-user" {
		t.Error("RemoteUser field not accessible")
	}
	if cdi.GroupName != "test-group" {
		t.Error("GroupName field not accessible")
	}
	if cdi.ProductInfo != "test-product" {
		t.Error("ProductInfo field not accessible")
	}
	if cdi.ExtraParameters != "test-params" {
		t.Error("ExtraParameters field not accessible")
	}
}
