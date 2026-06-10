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

const TableNameTCdi = "t_cdi"

type TCdi struct {
	CdiID             string `json:"cdi_id"`
	RemoteHost        string `json:"remote_host"`
	RemoteUser        string `json:"remote_user"`
	GroupName         string `json:"group_name"`
	ProductInfo  	  string `json:"product_info"`
	ExtraParameters	  string `json:"extra_parameters"`
}

// TableName TCdi's table name
func (*TCdi) TableName() string {
	return TableNameTCdi
}

func (t *TCdi) QueryParameter() string {
	return fmt.Sprintf("cdi_id=eq.%s", t.CdiID)
}

func (*TCdi) Parse(json interface{}) ([]TCdi, error) {
	jsonArray, ok := json.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format: expected array of objects")
	}

	var t []TCdi
	for _, item := range jsonArray {
		jsonMap, ok := item.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid item format: expected object")
        }

		var cdi TCdi
		if err := cdi.parseSingle(jsonMap); err != nil {
			return nil, err
		}
		t = append(t, cdi)
	}

	return t, nil
}

func (t *TCdi) parseSingle(jsonMap map[string]interface{}) error {

	if val, ok := jsonMap["cdi_id"].(string); ok {
		t.CdiID = val
	} else {
		return fmt.Errorf("invalid type for cdi_id")
	}

	if val, ok := jsonMap["remote_host"].(string); ok {
		t.RemoteHost = val
	} else {
		return fmt.Errorf("invalid type for remote_host")
	}

	if val, ok := jsonMap["remote_user"].(string); ok {
		t.RemoteUser = val
	} else {
		return fmt.Errorf("invalid type for remote_user")
	}

	if val, ok := jsonMap["group_name"].(string); ok {
		t.GroupName = val
	} else {
		return fmt.Errorf("invalid type for group_name")
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
