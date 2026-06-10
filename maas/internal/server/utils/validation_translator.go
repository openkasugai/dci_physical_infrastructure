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

package utils

import (
	"strings"
)

// ValidationErrorMessages maps field patterns to custom error messages
var ValidationErrorMessages = map[string]string{
	// MAC address patterns
	"MacAddress":                    "MAC address must be in format XX:XX:XX:XX:XX:XX (e.g., 00:11:22:33:44:55)",
	"NetworkInformation.MacAddress": "MAC address must be in format XX:XX:XX:XX:XX:XX (e.g., 00:11:22:33:44:55)",

	// IPMI address patterns
	"IpmiAddress": "IPMI address must be a valid IPv4 address (e.g., 192.168.1.100)",

	// CIDR patterns
	"Cidr":                       "CIDR must be in format x.x.x.x/y (e.g., 192.168.1.0/24)",
	"NetworkInformation.Cidr":    "CIDR must be in format x.x.x.x/y (e.g., 192.168.1.0/24)",
	"NetworkInformationCni.Cidr": "CIDR must be in format x.x.x.x/y (e.g., 192.168.1.0/24)",

	// IP address patterns
	"AddressStart":                       "IP address must be a valid IPv4 address (e.g., 192.168.1.10)",
	"AddressEnd":                         "IP address must be a valid IPv4 address (e.g., 192.168.1.20)",
	"NetworkInformation.AddressStart":    "IP address must be a valid IPv4 address (e.g., 192.168.1.10)",
	"NetworkInformation.AddressEnd":      "IP address must be a valid IPv4 address (e.g., 192.168.1.20)",
	"NetworkInformationCni.AddressStart": "IP address must be a valid IPv4 address (e.g., 192.168.1.10)",
	"NetworkInformationCni.AddressEnd":   "IP address must be a valid IPv4 address (e.g., 192.168.1.20)",
}

// TranslateValidationError converts protobuf validation errors to user-friendly messages
func TranslateValidationError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Check if it's a regex pattern validation error
	if !strings.Contains(errStr, "regex pattern") {
		return errStr // Return original error if not a pattern validation error
	}

	// Try to match field names with custom error messages
	for field, message := range ValidationErrorMessages {
		if strings.Contains(errStr, field) {
			return message
		}
	}

	// Return original error if no pattern matches
	return errStr
}

// ValidateAndTranslateError validates a protobuf message and returns a user-friendly error message
func ValidateAndTranslateError(validator interface{ Validate() error }) string {
	if err := validator.Validate(); err != nil {
		return TranslateValidationError(err)
	}
	return ""
}
