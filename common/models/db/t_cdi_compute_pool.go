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
	"encoding/json"
	"fmt"
)

const TableNameTCdiComputePool = "t_cdi_compute_pool"

type TCdiComputePool struct {
	ServerID          string  `json:"server_id"`
	IpmiAddress       string  `json:"ipmi_address"`
	IpmiUser          string  `json:"ipmi_user"`
	IpmiPassword      string  `json:"ipmi_password"`
	CdiID             string  `json:"cdi_id"`
	ProductInfo  	  string  `json:"product_info"`
	ExtraParameters	  string  `json:"extra_parameters"`
}

// TableName TCdiComputePool's table name
func (*TCdiComputePool) TableName() string {
	return TableNameTCdiComputePool
}

func (t *TCdiComputePool) QueryParameter() string {
	return fmt.Sprintf("server_id=eq.%s", t.ServerID)
}

func (*TCdiComputePool) Parse(json interface{}) ([]TCdiComputePool, error) {
	jsonArray, ok := json.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format: expected array of objects")
	}

	var t []TCdiComputePool
	for _, item := range jsonArray {
		jsonMap, ok := item.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid item format: expected object")
        }

		var cdiCompute TCdiComputePool
		if err := cdiCompute.parseSingle(jsonMap); err != nil {
			return nil, err
		}
		t = append(t, cdiCompute)
	}

	return t, nil
}

func (t *TCdiComputePool) parseSingle(jsonMap map[string]interface{}) error {

	if val, ok := jsonMap["server_id"].(string); ok {
		t.ServerID = val
	} else {
		return fmt.Errorf("invalid type for server_id")
	}

	if val, ok := jsonMap["ipmi_address"].(string); ok {
		t.IpmiAddress = val
	} else {
		return fmt.Errorf("invalid type for ipmi_address")
	}

	if val, ok := jsonMap["ipmi_user"].(string); ok {
		t.IpmiUser = val
	} else {
		return fmt.Errorf("invalid type for ipmi_user")
	}

	if val, ok := jsonMap["ipmi_password"].(string); ok {
		t.IpmiPassword = val
	} else {
		return fmt.Errorf("invalid type for ipmi_password")
	}

	if val, ok := jsonMap["cdi_id"].(string); ok {
		t.CdiID = val
	} else {
		return fmt.Errorf("invalid type for cdi_id")
	}

	// product_info is jsonb, can be nil, string, or object
	if val, ok := jsonMap["product_info"]; ok && val != nil {
		switch v := val.(type) {
		case string:
			t.ProductInfo = v
		case map[string]interface{}, []interface{}:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal product_info: %w", err)
			}
			t.ProductInfo = string(jsonBytes)
		default:
			return fmt.Errorf("invalid type for product_info")
		}
	}

	// extra_parameters is jsonb, can be nil, string, or object
	if val, ok := jsonMap["extra_parameters"]; ok && val != nil {
		switch v := val.(type) {
		case string:
			t.ExtraParameters = v
		case map[string]interface{}, []interface{}:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal extra_parameters: %w", err)
			}
			t.ExtraParameters = string(jsonBytes)
		default:
			return fmt.Errorf("invalid type for extra_parameters")
		}
	}

	return nil
}
