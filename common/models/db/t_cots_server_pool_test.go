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

// TestTCotsServerPool_TableName tests the TableName method
func TestTCotsServerPool_TableName(t *testing.T) {
	pool := &TCotsServerPool{}
	expected := "t_cots_server_pool"
	if pool.TableName() != expected {
		t.Errorf("TableName() = %v, want %v", pool.TableName(), expected)
	}
}

// TestTCotsServerPool_QueryParameter tests the QueryParameter method
func TestTCotsServerPool_QueryParameter(t *testing.T) {
	tests := []struct {
		name     string
		serverID string
		expected string
	}{
		{
			name:     "Standard Server ID",
			serverID: "cots-001",
			expected: "server_id=eq.cots-001",
		},
		{
			name:     "Empty Server ID",
			serverID: "",
			expected: "server_id=eq.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &TCotsServerPool{ServerID: tt.serverID}
			result := pool.QueryParameter()
			if result != tt.expected {
				t.Errorf("QueryParameter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestTCotsServerPool_Parse tests parsing JSON array
func TestTCotsServerPool_Parse(t *testing.T) {
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
					"server_id":        "cots-001",
					"ipmi_address":     "192.168.2.100",
					"ipmi_user":        "root",
					"ipmi_password":    "admin123",
					"product_info":     `{}`,
					"extra_parameters": `{}`,
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
			pool := &TCotsServerPool{}
			result, err := pool.Parse(tt.input)
			
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

// TestTableNameTCotsServerPool tests the constant
func TestTableNameTCotsServerPool(t *testing.T) {
	expected := "t_cots_server_pool"
	if TableNameTCotsServerPool != expected {
		t.Errorf("TableNameTCotsServerPool = %v, want %v", TableNameTCotsServerPool, expected)
	}
}
