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

const TableNameTOsInfo = "t_os_info"

type TOsInfo struct {
	ID             int32   `json:"id"`
	LoginUser      string  `json:"login_user"`
}

// TableName TOsInfo's table name
func (*TOsInfo) TableName() string {
	return TableNameTOsInfo
}

func (t *TOsInfo) QueryParameter() string {
	return fmt.Sprintf("id=eq.%d", t.ID)
}

func (*TOsInfo) Parse(json interface{}) ([]TOsInfo, error) {
	jsonArray, ok := json.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format: expected array of objects")
	}

	var t []TOsInfo
	for _, item := range jsonArray {
		jsonMap, ok := item.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid item format: expected object")
        }

		var os TOsInfo
		if err := os.parseSingle(jsonMap); err != nil {
			return nil, err
		}
		t = append(t, os)
	}

	return t, nil
}

func (t *TOsInfo) parseSingle(json map[string]interface{}) error {

	if val, ok := json["id"].(float64); ok {
		t.ID = int32(val)
	} else {
		return fmt.Errorf("invalid type for id")
	}

	if val, ok := json["login_user"].(string); ok {
		t.LoginUser = val
	} else {
		return fmt.Errorf("invalid type for login_user")
	}

	return nil
}
