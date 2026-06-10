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

const TableNameTNwSwitch = "t_nw_switch"

type TNwSwitch struct {
	NwIPAddress  	string `json:"nw_ip_address"`
	NwUser       	string `json:"nw_user"`
	ProductInfo  	string `json:"product_info"`
	ExtraParameters	string `json:"extra_parameters"`
}

// TableName TNwSwitch's table name
func (*TNwSwitch) TableName() string {
	return TableNameTNwSwitch
}

func (*TNwSwitch) Parse(json interface{}) ([]TNwSwitch, error) {
	jsonArray, ok := json.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format: expected array of objects")
	}

	var t []TNwSwitch
	for _, item := range jsonArray {
		jsonMap, ok := item.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid item format: expected object")
        }

		var nwSwitch TNwSwitch
		if err := nwSwitch.parseSingle(jsonMap); err != nil {
			return nil, err
		}
		t = append(t, nwSwitch)
	}

	return t, nil
}

func (t *TNwSwitch) parseSingle(jsonMap map[string]interface{}) error {

	if val, ok := jsonMap["nw_ip_address"].(string); ok {
		t.NwIPAddress = val
	} else {
		return fmt.Errorf("invalid type for nw_ip_address")
	}

	if val, ok := jsonMap["nw_user"].(string); ok {
		t.NwUser = val
	} else {
		return fmt.Errorf("invalid type for nw_user")
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
