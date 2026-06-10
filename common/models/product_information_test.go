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

package models

import (
	"os"
	"sync"
	"testing"
)

// TestParseProductTypeFromJSON_NWProduct tests parsing NW product from JSON
func TestParseProductTypeFromJSON_NWProduct(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		jsonStr  string
		expected NWProductType
	}{
		{
			name:     "EdgeCore SONiC product",
			jsonStr:  `{"vendor":"EdgeCore","product_name":"AS7326-56X","version":"1.0","os":"SONiC"}`,
			expected: EdgeCoreSonic,
		},
		{
			name:     "Broadcom SONiC product",
			jsonStr:  `{"vendor":"Broadcom","product_name":"BCM56960","version":"1.0","os":"SONiC"}`,
			expected: BroadcomSonic,
		},
		{
			name:     "Dummy product",
			jsonStr:  `{"vendor":"Dummy","product_name":"DummySwitch","version":"1.0","os":"Linux"}`,
			expected: Dummy,
		},
		{
			name:     "Unknown product",
			jsonStr:  `{"vendor":"Unknown","product_name":"Unknown","version":"1.0","os":"Unknown"}`,
			expected: NWProductTypeNone,
		},
		{
			name:     "Invalid JSON",
			jsonStr:  `{invalid json}`,
			expected: NWProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseProductTypeFromJSON[NWProductType](tt.jsonStr)
			if result != tt.expected {
				t.Errorf("ParseProductTypeFromJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseProductTypeFromJSON_ServerProduct tests parsing Server product from JSON
func TestParseProductTypeFromJSON_ServerProduct(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		jsonStr  string
		expected ServerProductType
	}{
		{
			name:     "Dell server",
			jsonStr:  `{"vendor":"Dell","product_name":"PowerEdge","version":"1.0","os":"Linux"}`,
			expected: Dell,
		},
		{
			name:     "Primergy server",
			jsonStr:  `{"vendor":"Fujitsu","product_name":"PRIMERGY","version":"1.0","os":"Linux"}`,
			expected: Primergy,
		},
		{
			name:     "Supermicro server",
			jsonStr:  `{"vendor":"Supermicro","product_name":"SuperServer","version":"1.0","os":"Linux"}`,
			expected: Supermicro,
		},
		{
			name:     "Unknown server",
			jsonStr:  `{"vendor":"Unknown","product_name":"Unknown","version":"1.0","os":"Linux"}`,
			expected: ServerProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseProductTypeFromJSON[ServerProductType](tt.jsonStr)
			if result != tt.expected {
				t.Errorf("ParseProductTypeFromJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseProductTypeFromJSON_CDIProduct tests parsing CDI product from JSON
func TestParseProductTypeFromJSON_CDIProduct(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		jsonStr  string
		expected CDIProductType
	}{
		{
			name:     "PG-CDI 1.1",
			jsonStr:  `{"vendor":"Fujitsu","product_name":"PG-CDI","version":"1.1","os":"Linux"}`,
			expected: PG_CDI_1_1,
		},
		{
			name:     "PG-CDI 1.0",
			jsonStr:  `{"vendor":"Fujitsu","product_name":"PG-CDI","version":"1.0","os":"Linux"}`,
			expected: PG_CDI_1_0,
		},
		{
			name:     "Unknown CDI",
			jsonStr:  `{"vendor":"Unknown","product_name":"Unknown","version":"1.0","os":"Linux"}`,
			expected: CDIProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseProductTypeFromJSON[CDIProductType](tt.jsonStr)
			if result != tt.expected {
				t.Errorf("ParseProductTypeFromJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseProductTypeFromJSON_MaasProduct tests parsing MaaS product from JSON
func TestParseProductTypeFromJSON_MaasProduct(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		jsonStr  string
		expected MaasProductType
	}{
		{
			name:     "Canonical MaaS",
			jsonStr:  `{"vendor":"Canonical","product_name":"MAAS","version":"3.0","os":"Ubuntu"}`,
			expected: Canonical,
		},
		{
			name:     "Unknown MaaS",
			jsonStr:  `{"vendor":"Unknown","product_name":"Unknown","version":"1.0","os":"Linux"}`,
			expected: MaasProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseProductTypeFromJSON[MaasProductType](tt.jsonStr)
			if result != tt.expected {
				t.Errorf("ParseProductTypeFromJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseProductTypeFromFields tests parsing with individual fields
func TestParseProductTypeFromFields(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		vendor   string
		prodName string
		version  string
		os       string
		expected NWProductType
	}{
		{
			name:     "EdgeCore with all fields",
			vendor:   "EdgeCore",
			prodName: "AS7326-56X",
			version:  "1.0",
			os:       "SONiC",
			expected: EdgeCoreSonic,
		},
		{
			name:     "No match with vendor only - requires exact match",
			vendor:   "EdgeCore",
			prodName: "",
			version:  "",
			os:       "",
			expected: NWProductTypeNone, // Changed: Partial matching doesn't work without wildcard mapping
		},
		{
			name:     "No match",
			vendor:   "NonExistent",
			prodName: "NonExistent",
			version:  "1.0",
			os:       "Linux",
			expected: NWProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseProductTypeFromFields[NWProductType](tt.vendor, tt.prodName, tt.version, tt.os)
			if result != tt.expected {
				t.Errorf("ParseProductTypeFromFields() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestLoadMappings_NoEnvironmentVariable tests when no mappings are provided
func TestLoadMappings_NoEnvironmentVariable(t *testing.T) {
	// Cleanup and reset
	cleanupTestMappings()

	// Ensure PRODUCT_MAPPINGS is not set
	os.Unsetenv("PRODUCT_MAPPINGS")

	// Reset the sync.Once
	mappingsOnce = sync.Once{}

	result := loadMappings()
	if result != nil {
		t.Errorf("loadMappings() with no env var should return nil, got %v", result)
	}
}

// TestLoadMappings_InvalidJSON tests when invalid JSON is provided
func TestLoadMappings_InvalidJSON(t *testing.T) {
	// Cleanup and reset
	cleanupTestMappings()

	// Set invalid JSON
	os.Setenv("PRODUCT_MAPPINGS", "{invalid json}")
	defer os.Unsetenv("PRODUCT_MAPPINGS")

	// Reset the sync.Once
	mappingsOnce = sync.Once{}

	result := loadMappings()
	if result != nil {
		t.Errorf("loadMappings() with invalid JSON should return nil, got %v", result)
	}
}

// TestParseNWProductType tests NW product type parsing
func TestParseNWProductType(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		vendor   string
		prodName string
		version  string
		os       string
		expected NWProductType
	}{
		{
			name:     "EdgeCore match",
			vendor:   "EdgeCore",
			prodName: "AS7326-56X",
			version:  "1.0",
			os:       "SONiC",
			expected: EdgeCoreSonic,
		},
		{
			name:     "Broadcom match",
			vendor:   "Broadcom",
			prodName: "BCM56960",
			version:  "1.0",
			os:       "SONiC",
			expected: BroadcomSonic,
		},
		{
			name:     "Dummy match",
			vendor:   "Dummy",
			prodName: "DummySwitch",
			version:  "1.0",
			os:       "Linux",
			expected: Dummy,
		},
		{
			name:     "No match",
			vendor:   "Unknown",
			prodName: "Unknown",
			version:  "1.0",
			os:       "Unknown",
			expected: NWProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNWProductType(tt.vendor, tt.prodName, tt.version, tt.os)
			if result != tt.expected {
				t.Errorf("parseNWProductType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseServerProductType tests Server product type parsing
func TestParseServerProductType(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		vendor   string
		prodName string
		version  string
		os       string
		expected ServerProductType
	}{
		{
			name:     "Dell match",
			vendor:   "Dell",
			prodName: "PowerEdge",
			version:  "1.0",
			os:       "Linux",
			expected: Dell,
		},
		{
			name:     "Primergy match",
			vendor:   "Fujitsu",
			prodName: "PRIMERGY",
			version:  "1.0",
			os:       "Linux",
			expected: Primergy,
		},
		{
			name:     "Supermicro match",
			vendor:   "Supermicro",
			prodName: "SuperServer",
			version:  "1.0",
			os:       "Linux",
			expected: Supermicro,
		},
		{
			name:     "No match",
			vendor:   "Unknown",
			prodName: "Unknown",
			version:  "1.0",
			os:       "Linux",
			expected: ServerProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseServerProductType(tt.vendor, tt.prodName, tt.version, tt.os)
			if result != tt.expected {
				t.Errorf("parseServerProductType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseCDIProductType tests CDI product type parsing
func TestParseCDIProductType(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		vendor   string
		prodName string
		version  string
		os       string
		expected CDIProductType
	}{
		{
			name:     "PG-CDI 1.1 match",
			vendor:   "Fujitsu",
			prodName: "PG-CDI",
			version:  "1.1",
			os:       "Linux",
			expected: PG_CDI_1_1,
		},
		{
			name:     "PG-CDI 1.0 match",
			vendor:   "Fujitsu",
			prodName: "PG-CDI",
			version:  "1.0",
			os:       "Linux",
			expected: PG_CDI_1_0,
		},
		{
			name:     "No match",
			vendor:   "Unknown",
			prodName: "Unknown",
			version:  "1.0",
			os:       "Linux",
			expected: CDIProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCDIProductType(tt.vendor, tt.prodName, tt.version, tt.os)
			if result != tt.expected {
				t.Errorf("parseCDIProductType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseMaasProductType tests MaaS product type parsing
func TestParseMaasProductType(t *testing.T) {
	// Setup
	setupTestMappings(t)
	defer cleanupTestMappings()

	tests := []struct {
		name     string
		vendor   string
		prodName string
		version  string
		os       string
		expected MaasProductType
	}{
		{
			name:     "Canonical match",
			vendor:   "Canonical",
			prodName: "MAAS",
			version:  "3.0",
			os:       "Ubuntu",
			expected: Canonical,
		},
		{
			name:     "No match",
			vendor:   "Unknown",
			prodName: "Unknown",
			version:  "1.0",
			os:       "Linux",
			expected: MaasProductTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMaasProductType(tt.vendor, tt.prodName, tt.version, tt.os)
			if result != tt.expected {
				t.Errorf("parseMaasProductType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseProductTypeFromFields_WildcardMatching tests wildcard matching in product mappings
func TestParseProductTypeFromFields_WildcardMatching(t *testing.T) {
	// Setup mappings with wildcard (empty string) fields
	cleanupTestMappings()
	
	mappingsJSON := `{
		"nw_products": [
			{"vendor": "EdgeCore", "product_name": "", "version": "", "os": "", "type": "EdgeCoreSonic"}
		],
		"server_products": [
			{"vendor": "Dell", "product_name": "", "version": "", "os": "", "type": "Dell"}
		],
		"cdi_products": [],
		"maas_products": []
	}`
	
	os.Setenv("PRODUCT_MAPPINGS", mappingsJSON)
	defer cleanupTestMappings()
	
	// Reset sync.Once
	mappingsOnce = sync.Once{}

	tests := []struct {
		name     string
		vendor   string
		prodName string
		version  string
		os       string
		expected NWProductType
	}{
		{
			name:     "Wildcard match - vendor only specified",
			vendor:   "EdgeCore",
			prodName: "AnyProduct",
			version:  "AnyVersion",
			os:       "AnyOS",
			expected: EdgeCoreSonic,
		},
		{
			name:     "Wildcard match - minimal fields",
			vendor:   "EdgeCore",
			prodName: "",
			version:  "",
			os:       "",
			expected: EdgeCoreSonic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseProductTypeFromFields[NWProductType](tt.vendor, tt.prodName, tt.version, tt.os)
			if result != tt.expected {
				t.Errorf("ParseProductTypeFromFields() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseProductTypeFromJSON_EmptyString tests parsing with empty string
func TestParseProductTypeFromJSON_EmptyString(t *testing.T) {
	result := ParseProductTypeFromJSON[NWProductType]("")
	if result != NWProductTypeNone {
		t.Errorf("ParseProductTypeFromJSON() with empty string = %v, want %v", result, NWProductTypeNone)
	}
}

// setupTestMappings sets up test environment with product mappings
func setupTestMappings(t *testing.T) {
	t.Helper()
	
	mappingsJSON := `{
		"nw_products": [
			{"vendor": "EdgeCore", "product_name": "AS7326-56X", "version": "1.0", "os": "SONiC", "type": "EdgeCoreSonic"},
			{"vendor": "Broadcom", "product_name": "BCM56960", "version": "1.0", "os": "SONiC", "type": "BroadcomSonic"},
			{"vendor": "Dummy", "product_name": "DummySwitch", "version": "1.0", "os": "Linux", "type": "Dummy"}
		],
		"server_products": [
			{"vendor": "Dell", "product_name": "PowerEdge", "version": "1.0", "os": "Linux", "type": "Dell"},
			{"vendor": "Fujitsu", "product_name": "PRIMERGY", "version": "1.0", "os": "Linux", "type": "Primergy"},
			{"vendor": "Supermicro", "product_name": "SuperServer", "version": "1.0", "os": "Linux", "type": "Supermicro"}
		],
		"cdi_products": [
			{"vendor": "Fujitsu", "product_name": "PG-CDI", "version": "1.1", "os": "Linux", "type": "PG_CDI_1_1"},
			{"vendor": "Fujitsu", "product_name": "PG-CDI", "version": "1.0", "os": "Linux", "type": "PG_CDI_1_0"}
		],
		"maas_products": [
			{"vendor": "Canonical", "product_name": "MAAS", "version": "3.0", "os": "Ubuntu", "type": "Canonical"}
		]
	}`
	
	os.Setenv("PRODUCT_MAPPINGS", mappingsJSON)
	
	// Reset sync.Once to allow reloading
	mappingsOnce = sync.Once{}
}

// cleanupTestMappings cleans up test environment
func cleanupTestMappings() {
	os.Unsetenv("PRODUCT_MAPPINGS")
	mappingsOnce = sync.Once{}
	mappings = nil
}
