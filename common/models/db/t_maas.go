package db

import (
	"encoding/json"
	"fmt"
)

const TableNameTMaas = "t_maas"

type TMaas struct {
	MaasID          int32  `json:"id"`
	PhysicalInfraId int32  `json:"physical_infra_id"`
	AccessURL       string `json:"access_url"`
	ApiKey          string `json:"api_key"`
	Status          int32  `json:"status"`
	ProductInfo     string `json:"product_info"`
	ExtraParameters string `json:"extra_parameters"`
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

func (t *TMaas) parseSingle(jsonMap map[string]interface{}) error {

	if val, ok := jsonMap["id"].(float64); ok {
		t.MaasID = int32(val)
	} else {
		return fmt.Errorf("invalid type for id")
	}

	if val, ok := jsonMap["physical_infra_id"].(float64); ok {
		t.PhysicalInfraId = int32(val)
	} else {
		return fmt.Errorf("invalid type for physical_infra_id")
	}

	if val, ok := jsonMap["access_url"].(string); ok {
		t.AccessURL = val
	} else {
		return fmt.Errorf("invalid type for access_url")
	}

	if val, ok := jsonMap["api_key"].(string); ok {
		t.ApiKey = val
	} else {
		return fmt.Errorf("invalid type for api_key")
	}

	if val, ok := jsonMap["status"].(float64); ok {
		t.Status = int32(val)
	} else {
		return fmt.Errorf("invalid type for status")
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

