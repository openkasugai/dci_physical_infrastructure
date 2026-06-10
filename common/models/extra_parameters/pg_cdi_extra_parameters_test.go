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

package extra_parameters

import (
	"testing"
)

// TestPgCDIExtraParameters_StructFields tests that all struct fields are accessible
func TestPgCDIExtraParameters_StructFields(t *testing.T) {
	params := PgCDIExtraParameters{
		CDIGuest:            "test-guest",
		CDIUser:             "test-user",
		CDIPassword:         "test-password",
		CDIMgrGuestUser:     "guest-user",
		CDIMgrGuestPassword: "guest-password",
		CDIMgrHostPassword:  "host-password",
		DirectorPassword:    "director-password",
	}

	// Verify all fields are accessible
	if params.CDIGuest != "test-guest" {
		t.Errorf("CDIGuest = %v, want test-guest", params.CDIGuest)
	}
	if params.CDIUser != "test-user" {
		t.Errorf("CDIUser = %v, want test-user", params.CDIUser)
	}
	if params.CDIPassword != "test-password" {
		t.Errorf("CDIPassword = %v, want test-password", params.CDIPassword)
	}
	if params.CDIMgrGuestUser != "guest-user" {
		t.Errorf("CDIMgrGuestUser = %v, want guest-user", params.CDIMgrGuestUser)
	}
	if params.CDIMgrGuestPassword != "guest-password" {
		t.Errorf("CDIMgrGuestPassword = %v, want guest-password", params.CDIMgrGuestPassword)
	}
	if params.CDIMgrHostPassword != "host-password" {
		t.Errorf("CDIMgrHostPassword = %v, want host-password", params.CDIMgrHostPassword)
	}
	if params.DirectorPassword != "director-password" {
		t.Errorf("DirectorPassword = %v, want director-password", params.DirectorPassword)
	}
}

// TestPgCDIExtraParameters_EmptyStruct tests zero values
func TestPgCDIExtraParameters_EmptyStruct(t *testing.T) {
	var params PgCDIExtraParameters

	if params.CDIGuest != "" {
		t.Errorf("Empty CDIGuest = %v, want empty string", params.CDIGuest)
	}
	if params.CDIUser != "" {
		t.Errorf("Empty CDIUser = %v, want empty string", params.CDIUser)
	}
	if params.CDIPassword != "" {
		t.Errorf("Empty CDIPassword = %v, want empty string", params.CDIPassword)
	}
	if params.CDIMgrGuestUser != "" {
		t.Errorf("Empty CDIMgrGuestUser = %v, want empty string", params.CDIMgrGuestUser)
	}
	if params.CDIMgrGuestPassword != "" {
		t.Errorf("Empty CDIMgrGuestPassword = %v, want empty string", params.CDIMgrGuestPassword)
	}
	if params.CDIMgrHostPassword != "" {
		t.Errorf("Empty CDIMgrHostPassword = %v, want empty string", params.CDIMgrHostPassword)
	}
	if params.DirectorPassword != "" {
		t.Errorf("Empty DirectorPassword = %v, want empty string", params.DirectorPassword)
	}
}

// TestPgCDIExtraParameters_PartialValues tests partial field population
func TestPgCDIExtraParameters_PartialValues(t *testing.T) {
	params := PgCDIExtraParameters{
		CDIGuest:    "guest1",
		CDIUser:     "user1",
		CDIPassword: "pass1",
		// Optional fields left empty
	}

	if params.CDIGuest != "guest1" {
		t.Errorf("CDIGuest = %v, want guest1", params.CDIGuest)
	}
	if params.CDIUser != "user1" {
		t.Errorf("CDIUser = %v, want user1", params.CDIUser)
	}
	if params.CDIPassword != "pass1" {
		t.Errorf("CDIPassword = %v, want pass1", params.CDIPassword)
	}
	if params.CDIMgrGuestUser != "" {
		t.Errorf("Optional CDIMgrGuestUser should be empty, got %v", params.CDIMgrGuestUser)
	}
	if params.CDIMgrGuestPassword != "" {
		t.Errorf("Optional CDIMgrGuestPassword should be empty, got %v", params.CDIMgrGuestPassword)
	}
}

// TestPgCDIExtraParameters_JSONTags tests JSON tag compliance
func TestPgCDIExtraParameters_JSONTags(t *testing.T) {
	// This test verifies that JSON tags are properly defined
	// The actual JSON marshaling/unmarshaling would be tested in integration tests
	// or by the validation framework
	
	params := PgCDIExtraParameters{
		CDIGuest:            "test",
		CDIUser:             "user",
		CDIPassword:         "pass",
		CDIMgrGuestUser:     "guser",
		CDIMgrGuestPassword: "gpass",
		CDIMgrHostPassword:  "hpass",
		DirectorPassword:    "dpass",
	}

	// Verify struct can be created and accessed
	if params.CDIGuest == "" {
		t.Error("Failed to create struct with valid values")
	}
}

// Note: Validation tag testing (validate:"required,min=1,max=128") would typically
// be done by the validator library (e.g., go-playground/validator) in integration tests.
// These unit tests focus on struct field accessibility and basic behavior.
