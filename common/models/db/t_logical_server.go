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

const TableNameTLogicalServer = "t_logical_server"

type TLogicalServer struct {
	CotsServerID         *string   `json:"cots_server_id"`
	CdiComputeServerID   *string   `json:"cdi_compute_server_id"`
	Status               int32     `json:"status"`
	OsID                 *int32    `json:"os_id"`
	HostIPAddress        *string   `json:"host_ip_address"`
	MgrIPAddress         *string   `json:"mgr_ip_address"`
	P2PEnabled           bool      `json:"p2p_enabled"`
	CdiMachineName       *string   `json:"cdi_machine_name"`
	ServerType		  	 int32     `json:"server_type"`
}

// TableName TLogicalServer's table name
func (*TLogicalServer) TableName() string {
	return TableNameTLogicalServer
}

func (t *TLogicalServer) QueryParameter() string {
	return fmt.Sprintf("status=eq.%d", t.Status)
}

func (*TLogicalServer) Parse(json interface{}) ([]TLogicalServer, error) {
	jsonArray, ok := json.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format: expected array of objects")
	}

	var t []TLogicalServer
	for _, item := range jsonArray {
		jsonMap, ok := item.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid item format: expected object")
        }

		var logServer TLogicalServer
		if err := logServer.parseSingle(jsonMap); err != nil {
			return nil, err
		}
		t = append(t, logServer)
	}

	return t, nil
}

func (t *TLogicalServer) parseSingle(json map[string]interface{}) error {

	// cots_server_id can be nil or string
	if val, ok := json["cots_server_id"]; ok && val != nil {
		if strVal, ok := val.(string); ok {
			t.CotsServerID = &strVal
		} else {
			return fmt.Errorf("invalid type for cots_server_id")
		}
	}

	// cdi_compute_server_id can be nil or string
	if val, ok := json["cdi_compute_server_id"]; ok && val != nil {
		if strVal, ok := val.(string); ok {
			t.CdiComputeServerID = &strVal
		} else {
			return fmt.Errorf("invalid type for cdi_compute_server_id")
		}
	}

	if val, ok := json["status"].(float64); ok {
		t.Status = int32(val)
	} else {
		return fmt.Errorf("invalid type for status")
	}

	// os_id can be nil or float64
	if val, ok := json["os_id"]; ok && val != nil {
		if floatVal, ok := val.(float64); ok {
			osID := int32(floatVal)
			t.OsID = &osID
		} else {
			return fmt.Errorf("invalid type for os_id")
		}
	}

	// host_ip_address can be nil or string
	if val, ok := json["host_ip_address"]; ok && val != nil {
		if strVal, ok := val.(string); ok {
			t.HostIPAddress = &strVal
		} else {
			return fmt.Errorf("invalid type for host_ip_address")
		}
	}

	// mgr_ip_address can be nil or string
	if val, ok := json["mgr_ip_address"]; ok && val != nil {
		if strVal, ok := val.(string); ok {
			t.MgrIPAddress = &strVal
		} else {
			return fmt.Errorf("invalid type for mgr_ip_address")
		}
	}

	if val, ok := json["p2p_enabled"].(bool); ok {
		t.P2PEnabled = val
	} else {
		return fmt.Errorf("invalid type for p2p_enabled")
	}
	
	// cdi_machine_name can be nil or string
	if val, ok := json["cdi_machine_name"]; ok && val != nil {
		if strVal, ok := val.(string); ok {
			t.CdiMachineName = &strVal
		} else {
			return fmt.Errorf("invalid type for cdi_machine_name")
		}
	}

	// server_type
	if val, ok := json["server_type"].(float64); ok {
		t.ServerType = int32(val)
	} else {
		return fmt.Errorf("invalid type for server_type")
	}
	
	return nil
}
