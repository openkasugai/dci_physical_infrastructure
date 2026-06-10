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

// TestTNwSwitch_TableName tests the TableName method
func TestTNwSwitch_TableName(t *testing.T) {
	sw := &TNwSwitch{}
	expected := "t_nw_switch"
	if sw.TableName() != expected {
		t.Errorf("TableName() = %v, want %v", sw.TableName(), expected)
	}
}

// TestTNwSwitch_Parse tests parsing JSON array
func TestTNwSwitch_Parse(t *testing.T) {
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
					"nw_ip_address":    "192.168.10.1",
					"nw_user":          "admin",
					"product_info":     `{}`,
					"extra_parameters": `{}`,
				},
			},
			expectErr: false,
			expected:  1,
		},
		{
			name: "Valid with jsonb object",
			input: []interface{}{
				map[string]interface{}{
					"nw_ip_address": "192.168.10.2",
					"nw_user":       "admin",
					"product_info": map[string]interface{}{
						"vendor": "Cisco",
						"model":  "Catalyst",
					},
					"extra_parameters": map[string]interface{}{
						"vlan": "100",
					},
				},
			},
			expectErr: false,
			expected:  1,
		},
		{
			name: "Valid without product_info and extra_parameters",
			input: []interface{}{
				map[string]interface{}{
					"nw_ip_address": "192.168.10.3",
					"nw_user":       "admin",
				},
			},
			expectErr: false,
			expected:  1,
		},
		{
			name:      "Invalid input type",
			input:     "not an array",
			expectErr: true,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := &TNwSwitch{}
			result, err := sw.Parse(tt.input)
			
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

// TestTNwSwitch_ParseSingle tests parsing a single JSON object
func TestTNwSwitch_ParseSingle(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		expectErr bool
		validate  func(*testing.T, *TNwSwitch)
	}{
		{
			name: "Valid complete data with string",
			input: map[string]interface{}{
				"nw_ip_address":    "192.168.10.1",
				"nw_user":          "admin",
				"product_info":     `{"vendor":"Cisco"}`,
				"extra_parameters": `{"key":"value"}`,
			},
			expectErr: false,
			validate: func(t *testing.T, sw *TNwSwitch) {
				if sw.NwIPAddress != "192.168.10.1" {
					t.Errorf("NwIPAddress = %v, want 192.168.10.1", sw.NwIPAddress)
				}
				if sw.NwUser != "admin" {
					t.Errorf("NwUser = %v, want admin", sw.NwUser)
				}
			},
		},
		{
			name: "Valid with jsonb object",
			input: map[string]interface{}{
				"nw_ip_address": "192.168.10.2",
				"nw_user":       "admin",
				"product_info": map[string]interface{}{
					"vendor": "Arista",
				},
				"extra_parameters": map[string]interface{}{
					"port": "ge-0/0/0",
				},
			},
			expectErr: false,
			validate: func(t *testing.T, sw *TNwSwitch) {
				if sw.ProductInfo == "" {
					t.Error("ProductInfo should not be empty for jsonb object")
				}
			},
		},
		{
			name: "Missing product_info",
			input: map[string]interface{}{
				"nw_ip_address":    "192.168.10.3",
				"nw_user":          "admin",
				"extra_parameters": `{}`,
			},
			expectErr: false,
		},
		{
			name: "Missing extra_parameters",
			input: map[string]interface{}{
				"nw_ip_address": "192.168.10.4",
				"nw_user":       "admin",
				"product_info":  `{}`,
			},
			expectErr: false,
		},
		{
			name: "Missing nw_ip_address",
			input: map[string]interface{}{
				"nw_user":          "admin",
				"product_info":     `{}`,
				"extra_parameters": `{}`,
			},
			expectErr: true,
		},
		{
			name: "Invalid type for nw_ip_address",
			input: map[string]interface{}{
				"nw_ip_address":    123,
				"nw_user":          "admin",
				"product_info":     `{}`,
				"extra_parameters": `{}`,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := &TNwSwitch{}
			err := sw.parseSingle(tt.input)
			
			if tt.expectErr && err == nil {
				t.Error("parseSingle() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("parseSingle() unexpected error = %v", err)
			}
			
			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, sw)
			}
		})
	}
}

// TestTableNameTNwSwitch tests the constant
func TestTableNameTNwSwitch(t *testing.T) {
	expected := "t_nw_switch"
	if TableNameTNwSwitch != expected {
		t.Errorf("TableNameTNwSwitch = %v, want %v", TableNameTNwSwitch, expected)
	}
}
