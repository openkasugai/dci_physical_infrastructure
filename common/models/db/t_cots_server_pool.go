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

const TableNameTCotsServerPool = "t_cots_server_pool"

type TCotsServerPool struct {
	ServerID          string  `json:"server_id"`
	IpmiAddress       string  `json:"ipmi_address"`
	IpmiUser          string  `json:"ipmi_user"`
	IpmiPassword      string  `json:"ipmi_password"`
	ProductInfo  	  string  `json:"product_info"`
	ExtraParameters	  string  `json:"extra_parameters"`
}

// TableName TCotsServerPool's table name
func (*TCotsServerPool) TableName() string {
	return TableNameTCotsServerPool
}

func (t *TCotsServerPool) QueryParameter() string {
	return fmt.Sprintf("server_id=eq.%s", t.ServerID)
}

func (*TCotsServerPool) Parse(json interface{}) ([]TCotsServerPool, error) {
	jsonArray, ok := json.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format: expected array of objects")
	}

	var t []TCotsServerPool
	for _, item := range jsonArray {
		jsonMap, ok := item.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid item format: expected object")
        }

		var cots TCotsServerPool
		if err := cots.parseSingle(jsonMap); err != nil {
			return nil, err
		}
		t = append(t, cots)
	}

	return t, nil
}

func (t *TCotsServerPool) parseSingle(jsonMap map[string]interface{}) error {

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
