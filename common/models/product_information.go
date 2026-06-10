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
    "encoding/json"
    "os"
    "sync"
)

type ProductInformation struct {
	Vendor       string `json:"vendor"`
	ProductName  string `json:"product_name"`
	Version      string `json:"version"`
	OS           string `json:"os"`
}

type NWProductType int
const (
    NWProductTypeNone NWProductType = iota
    EdgeCoreSonic
    BroadcomSonic
    Dummy
)

type ServerProductType int
const (
    ServerProductTypeNone ServerProductType = iota
    Dell
    Primergy
	Supermicro
)

type CDIProductType int
const (
    CDIProductTypeNone CDIProductType = iota
    PG_CDI_1_0
    PG_CDI_1_1
)

type MaasProductType int
const (
    MaasProductTypeNone MaasProductType = iota
    Canonical
)

type ProductMapping struct {
    Vendor  string `json:"vendor"`
    Name    string `json:"product_name"`
    Version string `json:"version,omitempty"`
    OS      string `json:"os,omitempty"`
    Type    string `json:"type"`
}

type ProductMappings struct {
    NWProducts     []ProductMapping `json:"nw_products"`
    ServerProducts []ProductMapping `json:"server_products"`
    CDIProducts    []ProductMapping `json:"cdi_products"`
    MaasProducts   []ProductMapping `json:"maas_products"`
}

var (
    mappings     *ProductMappings
    mappingsOnce sync.Once
)

func loadMappings() *ProductMappings {
    mappingsOnce.Do(func() {
        jsonData := os.Getenv("PRODUCT_MAPPINGS")
        if jsonData == "" {
            // No mappings provided
            mappings = nil
            return
        }

        var m ProductMappings
        if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
            // Failed to parse mappings
            mappings = nil
            return
        }
        mappings = &m
    })
    return mappings
}

func ParseProductTypeFromJSON[T any](productInfoJson string) T {
	var zero T
    var productInfo ProductInformation
    if err := json.Unmarshal([]byte(productInfoJson), &productInfo); err != nil {
        return zero
    }

    return ParseProductTypeFromFields[T](productInfo.Vendor, productInfo.ProductName, productInfo.Version, productInfo.OS)
}

func ParseProductTypeFromFields[T any](vendor, name, version, os string) T {
    var result any

    // Use type assertion to determine the concrete type
    var zero T
    switch any(zero).(type) {
    case NWProductType:
        result = parseNWProductType(vendor, name, version, os)
    case ServerProductType:
        result = parseServerProductType(vendor, name, version, os)
    case CDIProductType:
        result = parseCDIProductType(vendor, name, version, os)
    case MaasProductType:
        result = parseMaasProductType(vendor, name, version, os)
    default:
        return zero
    }

    return result.(T)
}

func parseNWProductType(vendor string, name string, version string, os string) NWProductType {
    m := loadMappings()
    if m == nil {
        return NWProductTypeNone
    }
    for _, p := range m.NWProducts {
        if p.Vendor == vendor && 
          (p.Name == "" || p.Name == name) && 
          (p.Version == "" || p.Version == version) && 
          (p.OS == "" || p.OS == os) {
            switch p.Type {
            case "EdgeCoreSonic":
                return EdgeCoreSonic
            case "BroadcomSonic":
                return BroadcomSonic
            case "Dummy":
                return Dummy
            }
        }
    }
    return NWProductTypeNone
}

func parseServerProductType(vendor string, name string, version string, os string) ServerProductType {
    m := loadMappings()
    if m == nil {
        return ServerProductTypeNone
    }
    for _, p := range m.ServerProducts {
        if p.Vendor == vendor && 
          (p.Name == "" || p.Name == name) && 
          (p.Version == "" || p.Version == version) && 
          (p.OS == "" || p.OS == os) {
            switch p.Type {
            case "Dell":
                return Dell
            case "Primergy":
                return Primergy
            case "Supermicro":
                return Supermicro
            }
        }
    }
    return ServerProductTypeNone
}

func parseCDIProductType(vendor string, name string, version string, os string) CDIProductType {
    m := loadMappings()
    if m == nil {
        return CDIProductTypeNone
    }
    for _, p := range m.CDIProducts {
        if p.Vendor == vendor && 
          (p.Name == "" || p.Name == name) && 
          (p.Version == "" || p.Version == version) && 
          (p.OS == "" || p.OS == os) {
            switch p.Type {
            case "PG_CDI_1_1":
                return PG_CDI_1_1
            case "PG_CDI_1_0":
                return PG_CDI_1_0
            }
        }
    }
    return CDIProductTypeNone
}

func parseMaasProductType(vendor string, name string, version string, os string) MaasProductType {
    m := loadMappings()
    if m == nil {
        return MaasProductTypeNone
    }
    for _, p := range m.MaasProducts {
        if p.Vendor == vendor && 
          (p.Name == "" || p.Name == name) && 
          (p.Version == "" || p.Version == version) && 
          (p.OS == "" || p.OS == os) {
            switch p.Type {
            case "Canonical":
                return Canonical
            }
        }
    }
    return MaasProductTypeNone
}
