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

package pg_cdi

import (
	"testing"
)

// TestParseExtraParameter_ValidJSON tests parsing valid JSON
func TestParseExtraParameter_ValidJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		expectErr bool
		wantUser  string
		wantPass  string
		wantGuest string
	}{
		{
			name: "Valid complete parameters",
			jsonStr: `{
				"cdi_user": "admin",
				"cdi_password": "secret123",
				"cdi_guest": "guest1"
			}`,
			expectErr: false,
			wantUser:  "admin",
			wantPass:  "secret123",
			wantGuest: "guest1",
		},
		{
			name: "Valid with special characters",
			jsonStr: `{
				"cdi_user": "user@domain.com",
				"cdi_password": "P@ssw0rd!#$",
				"cdi_guest": "guest_123"
			}`,
			expectErr: false,
			wantUser:  "user@domain.com",
			wantPass:  "P@ssw0rd!#$",
			wantGuest: "guest_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseExtraParameter(tt.jsonStr)

			if tt.expectErr {
				if err == nil {
					t.Error("ParseExtraParameter() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseExtraParameter() unexpected error = %v", err)
				return
			}

			if result.CDIUser != tt.wantUser {
				t.Errorf("CDIUser = %v, want %v", result.CDIUser, tt.wantUser)
			}
			if result.CDIPassword != tt.wantPass {
				t.Errorf("CDIPassword = %v, want %v", result.CDIPassword, tt.wantPass)
			}
			if result.CDIGuest != tt.wantGuest {
				t.Errorf("CDIGuest = %v, want %v", result.CDIGuest, tt.wantGuest)
			}
		})
	}
}

// TestParseExtraParameter_InvalidJSON tests error handling for invalid JSON
func TestParseExtraParameter_InvalidJSON(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
	}{
		{
			name:    "Empty string",
			jsonStr: "",
		},
		{
			name:    "Invalid JSON syntax",
			jsonStr: `{invalid json}`,
		},
		{
			name:    "Not JSON",
			jsonStr: "not a json string",
		},
		{
			name:    "Unclosed bracket",
			jsonStr: `{"cdi_user": "admin"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseExtraParameter(tt.jsonStr)

			if err == nil {
				t.Error("ParseExtraParameter() expected error for invalid JSON, got nil")
			}
			if result != nil {
				t.Errorf("ParseExtraParameter() expected nil result, got %v", result)
			}
		})
	}
}

// TestParseExtraParameter_ValidationFailure tests validation errors
func TestParseExtraParameter_ValidationFailure(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
	}{
		{
			name: "Missing cdi_user",
			jsonStr: `{
				"cdi_password": "secret",
				"cdi_guest": "guest"
			}`,
		},
		{
			name: "Missing cdi_password",
			jsonStr: `{
				"cdi_user": "admin",
				"cdi_guest": "guest"
			}`,
		},
		{
			name: "Missing cdi_guest",
			jsonStr: `{
				"cdi_user": "admin",
				"cdi_password": "secret"
			}`,
		},
		{
			name: "Empty cdi_user",
			jsonStr: `{
				"cdi_user": "",
				"cdi_password": "secret",
				"cdi_guest": "guest"
			}`,
		},
		{
			name: "Empty cdi_password",
			jsonStr: `{
				"cdi_user": "admin",
				"cdi_password": "",
				"cdi_guest": "guest"
			}`,
		},
		{
			name: "Empty cdi_guest",
			jsonStr: `{
				"cdi_user": "admin",
				"cdi_password": "secret",
				"cdi_guest": ""
			}`,
		},
		{
			name:    "Empty JSON object",
			jsonStr: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseExtraParameter(tt.jsonStr)

			if err == nil {
				t.Error("ParseExtraParameter() expected validation error, got nil")
			}
			if result != nil {
				t.Errorf("ParseExtraParameter() expected nil result, got %v", result)
			}
		})
	}
}

// TestParseExtraParameter_ValidatorInit tests validator initialization
func TestParseExtraParameter_ValidatorInit(t *testing.T) {
	if validate == nil {
		t.Error("Validator should be initialized in init() function")
	}
}

// TestParseExtraParameter_TypeMismatch tests type mismatch in JSON
func TestParseExtraParameter_TypeMismatch(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
	}{
		{
			name: "Numeric cdi_user",
			jsonStr: `{
				"cdi_user": 123,
				"cdi_password": "secret",
				"cdi_guest": "guest"
			}`,
		},
		{
			name: "Boolean cdi_password",
			jsonStr: `{
				"cdi_user": "admin",
				"cdi_password": true,
				"cdi_guest": "guest"
			}`,
		},
		{
			name: "Array cdi_guest",
			jsonStr: `{
				"cdi_user": "admin",
				"cdi_password": "secret",
				"cdi_guest": []
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseExtraParameter(tt.jsonStr)

			if err == nil {
				t.Error("ParseExtraParameter() expected type mismatch error, got nil")
			}
			if result != nil {
				t.Errorf("ParseExtraParameter() expected nil result, got %v", result)
			}
		})
	}
}
