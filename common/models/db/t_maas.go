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
	"fmt"
)

const TableNameTMaas = "t_maas"

type TMaas struct {
	MaasID            int32  `json:"id"`
	PhysicalInfraId   int32  `json:"physical_infra_id"`
	AccessURL         string `json:"access_url"`
	ApiKey            string `json:"api_key"`
	Status            int32  `json:"status"`
	ProductInfo  	  string `json:"product_info"`
	ExtraParameters	  string `json:"extra_parameters"`
}

// TableName TMaas's table name
func (*TMaas) TableName() string {
	return TableNameTMaas
}

func (t *TMaas) QueryParameter() string {
	return fmt.Sprintf("id=%d", t.MaasID)
}

func (*TMaas) Parse(json interface{}) ([]TMaas, error) {
	jsonArray, ok := json.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format: expected array of objects")
	}

	var t []TMaas
	for _, item := range jsonArray {
		jsonMap, ok := item.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid item format: expected object")
        }

		var maas TMaas
		if err := maas.parseSingle(jsonMap); err != nil {
			return nil, err
		}
		t = append(t, maas)
	}

	return t, nil
}

func (t *TMaas) parseSingle(json map[string]interface{}) error {

	if val, ok := json["id"].(float64); ok {
		t.MaasID = int32(val)
	} else {
		return fmt.Errorf("invalid type for id")
	}
	
	if val, ok := json["physical_infra_id"].(float64); ok {
		t.PhysicalInfraId = int32(val)
	} else {
		return fmt.Errorf("invalid type for physical_infra_id")
	}

	if val, ok := json["access_url"].(string); ok {
		t.AccessURL = val
	} else {
		return fmt.Errorf("invalid type for access_url")
	}

	if val, ok := json["api_key"].(string); ok {
		t.ApiKey = val
	} else {
		return fmt.Errorf("invalid type for api_key")
	}

	if val, ok := json["status"].(float64); ok {
		t.Status = int32(val)
	} else {
		return fmt.Errorf("invalid type for status")
	}

	if val, ok := json["product_info"].(string); ok {
		t.ProductInfo = val
	} else {
		return fmt.Errorf("invalid type for product_info")
	}

	if val, ok := json["extra_parameters"].(string); ok {
		t.ExtraParameters = val
	} else {
		return fmt.Errorf("invalid type for extra_parameters")
	}
	
	return nil
}
