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

// TestTLogicalServer_TableName tests the TableName method
func TestTLogicalServer_TableName(t *testing.T) {
	server := &TLogicalServer{}
	expected := "t_logical_server"
	if server.TableName() != expected {
		t.Errorf("TableName() = %v, want %v", server.TableName(), expected)
	}
}

// TestTLogicalServer_QueryParameter tests the QueryParameter method
func TestTLogicalServer_QueryParameter(t *testing.T) {
	server := &TLogicalServer{
		Status: 1,
	}

	result := server.QueryParameter()
	expected := "status=eq.1"

	if result != expected {
		t.Errorf("QueryParameter() = %v, want %v", result, expected)
	}
}

// TestTLogicalServer_Parse tests parsing JSON array
func TestTLogicalServer_Parse(t *testing.T) {
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
					"cots_server_id":         "server-123",
					"cdi_compute_server_id":  "cdi-456",
					"status":                 float64(1),
					"os_id":                  float64(10),
					"host_ip_address":        "192.168.1.100",
					"mgr_ip_address":         "10.0.0.100",
					"p2p_enabled":            true,
					"cdi_machine_name":       "machine-001",
					"server_type":            float64(1),
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
		{
			name:      "Empty array",
			input:     []interface{}{},
			expectErr: false,
			expected:  0,
		},
		{
			name: "Valid multiple items with different server types",
			input: []interface{}{
				map[string]interface{}{
					"cots_server_id":         "server-001",
					"status":                 float64(1),
					"p2p_enabled":            false,
					"server_type":            float64(1), // COTS
				},
				map[string]interface{}{
					"cdi_compute_server_id":  "cdi-002",
					"status":                 float64(2),
					"p2p_enabled":            true,
					"server_type":            float64(2), // VM
				},
			},
			expectErr: false,
			expected:  2,
		},
		{
			name: "Missing server_type field",
			input: []interface{}{
				map[string]interface{}{
					"status":      float64(1),
					"p2p_enabled": true,
				},
			},
			expectErr: true,
			expected:  0,
		},
		{
			name: "Invalid server_type type (string instead of number)",
			input: []interface{}{
				map[string]interface{}{
					"status":      float64(1),
					"p2p_enabled": true,
					"server_type": "invalid",
				},
			},
			expectErr: true,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &TLogicalServer{}
			result, err := server.Parse(tt.input)
			
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

// TestTableNameTLogicalServer tests the constant
func TestTableNameTLogicalServer(t *testing.T) {
	expected := "t_logical_server"
	if TableNameTLogicalServer != expected {
		t.Errorf("TableNameTLogicalServer = %v, want %v", TableNameTLogicalServer, expected)
	}
}

// TestTLogicalServer_ServerType_BoundaryValues tests ServerType field with boundary values
func TestTLogicalServer_ServerType_BoundaryValues(t *testing.T) {
	tests := []struct {
		name       string
		serverType float64
		expectErr  bool
		expected   int32
	}{
		{
			name:       "ServerType 0",
			serverType: float64(0),
			expectErr:  false,
			expected:   0,
		},
		{
			name:       "ServerType 1 (COTS)",
			serverType: float64(1),
			expectErr:  false,
			expected:   1,
		},
		{
			name:       "ServerType 2 (VM)",
			serverType: float64(2),
			expectErr:  false,
			expected:   2,
		},
		{
			name:       "ServerType negative",
			serverType: float64(-1),
			expectErr:  false,
			expected:   -1,
		},
		{
			name:       "ServerType large value",
			serverType: float64(999),
			expectErr:  false,
			expected:   999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &TLogicalServer{}
			input := []interface{}{
				map[string]interface{}{
					"status":      float64(1),
					"p2p_enabled": true,
					"server_type": tt.serverType,
				},
			}

			result, err := server.Parse(input)

			if tt.expectErr && err == nil {
				t.Error("Parse() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
			if !tt.expectErr && len(result) > 0 {
				if result[0].ServerType != tt.expected {
					t.Errorf("ServerType = %v, want %v", result[0].ServerType, tt.expected)
				}
			}
		})
	}
}

// TestTLogicalServer_ParseSingle_ServerTypeErrors tests parseSingle error handling for server_type
func TestTLogicalServer_ParseSingle_ServerTypeErrors(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		expectErr bool
		errMsg    string
	}{
		{
			name: "server_type as string",
			input: map[string]interface{}{
				"status":      float64(1),
				"p2p_enabled": true,
				"server_type": "invalid_type",
			},
			expectErr: true,
			errMsg:    "invalid type for server_type",
		},
		{
			name: "server_type as bool",
			input: map[string]interface{}{
				"status":      float64(1),
				"p2p_enabled": true,
				"server_type": true,
			},
			expectErr: true,
			errMsg:    "invalid type for server_type",
		},
		{
			name: "server_type missing",
			input: map[string]interface{}{
				"status":      float64(1),
				"p2p_enabled": true,
			},
			expectErr: true,
			errMsg:    "invalid type for server_type",
		},
		{
			name: "server_type as nil",
			input: map[string]interface{}{
				"status":      float64(1),
				"p2p_enabled": true,
				"server_type": nil,
			},
			expectErr: true,
			errMsg:    "invalid type for server_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &TLogicalServer{}
			err := server.parseSingle(tt.input)

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

// TestTLogicalServer_ParseSingle_OptionalFieldErrors tests parseSingle error handling for optional fields
func TestTLogicalServer_ParseSingle_OptionalFieldErrors(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		expectErr bool
		errMsg    string
	}{
		{
			name: "cots_server_id wrong type (number)",
			input: map[string]interface{}{
				"cots_server_id": float64(123),
				"status":         float64(1),
				"p2p_enabled":    true,
				"server_type":    float64(1),
			},
			expectErr: true,
			errMsg:    "invalid type for cots_server_id",
		},
		{
			name: "cdi_compute_server_id wrong type (bool)",
			input: map[string]interface{}{
				"cdi_compute_server_id": true,
				"status":                float64(1),
				"p2p_enabled":           true,
				"server_type":           float64(1),
			},
			expectErr: true,
			errMsg:    "invalid type for cdi_compute_server_id",
		},
		{
			name: "os_id wrong type (string)",
			input: map[string]interface{}{
				"os_id":       "invalid",
				"status":      float64(1),
				"p2p_enabled": true,
				"server_type": float64(1),
			},
			expectErr: true,
			errMsg:    "invalid type for os_id",
		},
		{
			name: "host_ip_address wrong type (number)",
			input: map[string]interface{}{
				"host_ip_address": float64(192),
				"status":          float64(1),
				"p2p_enabled":     true,
				"server_type":     float64(1),
			},
			expectErr: true,
			errMsg:    "invalid type for host_ip_address",
		},
		{
			name: "mgr_ip_address wrong type (number)",
			input: map[string]interface{}{
				"mgr_ip_address": float64(10),
				"status":         float64(1),
				"p2p_enabled":    true,
				"server_type":    float64(1),
			},
			expectErr: true,
			errMsg:    "invalid type for mgr_ip_address",
		},
		{
			name: "cdi_machine_name wrong type (number)",
			input: map[string]interface{}{
				"cdi_machine_name": float64(999),
				"status":           float64(1),
				"p2p_enabled":      true,
				"server_type":      float64(1),
			},
			expectErr: true,
			errMsg:    "invalid type for cdi_machine_name",
		},
		{
			name: "status wrong type (string)",
			input: map[string]interface{}{
				"status":      "invalid",
				"p2p_enabled": true,
				"server_type": float64(1),
			},
			expectErr: true,
			errMsg:    "invalid type for status",
		},
		{
			name: "p2p_enabled wrong type (string)",
			input: map[string]interface{}{
				"status":      float64(1),
				"p2p_enabled": "invalid",
				"server_type": float64(1),
			},
			expectErr: true,
			errMsg:    "invalid type for p2p_enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &TLogicalServer{}
			err := server.parseSingle(tt.input)

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

// TestTLogicalServer_ParseSingle_AllOptionalFieldsNil tests parseSingle with all optional fields as nil
func TestTLogicalServer_ParseSingle_AllOptionalFieldsNil(t *testing.T) {
	input := map[string]interface{}{
		"cots_server_id":        nil,
		"cdi_compute_server_id": nil,
		"os_id":                 nil,
		"host_ip_address":       nil,
		"mgr_ip_address":        nil,
		"cdi_machine_name":      nil,
		"status":                float64(1),
		"p2p_enabled":           false,
		"server_type":           float64(1),
	}

	server := &TLogicalServer{}
	err := server.parseSingle(input)

	if err != nil {
		t.Errorf("parseSingle() unexpected error = %v", err)
	}
	if server.CotsServerID != nil {
		t.Errorf("CotsServerID should be nil")
	}
	if server.CdiComputeServerID != nil {
		t.Errorf("CdiComputeServerID should be nil")
	}
	if server.OsID != nil {
		t.Errorf("OsID should be nil")
	}
	if server.HostIPAddress != nil {
		t.Errorf("HostIPAddress should be nil")
	}
	if server.MgrIPAddress != nil {
		t.Errorf("MgrIPAddress should be nil")
	}
	if server.CdiMachineName != nil {
		t.Errorf("CdiMachineName should be nil")
	}
}

// TestTLogicalServer_Parse_ItemWithError tests Parse when parseSingle returns error
func TestTLogicalServer_Parse_ItemWithError(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"status":      float64(1),
			"p2p_enabled": true,
			"server_type": float64(1),
		},
		map[string]interface{}{
			"status":      "invalid", // This will cause error
			"p2p_enabled": true,
			"server_type": float64(1),
		},
	}

	server := &TLogicalServer{}
	result, err := server.Parse(input)

	if err == nil {
		t.Error("Parse() expected error, got nil")
	}
	if result != nil {
		t.Errorf("Parse() should return nil on error, got %v", result)
	}
}

// TestTLogicalServer_Parse_InvalidItemType tests Parse when array contains non-map items
func TestTLogicalServer_Parse_InvalidItemType(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "Array item is string",
			input: []interface{}{"invalid item"},
		},
		{
			name:  "Array item is number",
			input: []interface{}{float64(123)},
		},
		{
			name:  "Array item is bool",
			input: []interface{}{true},
		},
		{
			name: "Array item is array",
			input: []interface{}{
				[]interface{}{"nested"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &TLogicalServer{}
			result, err := server.Parse(tt.input)

			if err == nil {
				t.Error("Parse() expected error, got nil")
			}
			if err != nil && err.Error() != "invalid item format: expected object" {
				t.Errorf("Parse() error message = %v, want 'invalid item format: expected object'", err.Error())
			}
			if result != nil {
				t.Errorf("Parse() should return nil on error, got %v", result)
			}
		})
	}
}
