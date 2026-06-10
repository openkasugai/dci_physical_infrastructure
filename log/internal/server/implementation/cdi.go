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

package implementation

import (
	"context"
	"encoding/json"
	"fmt"
	"common/models"
	"common/models/extra_parameters"
	"log_module/internal/server/interfaces" // import for interface
	"os"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

// struct of CDI
type CDIImplement struct {
	Logger  klog.Logger
	Ansible interfaces.Ansible
	Logging interfaces.Logging
}

func (l *CDIImplement) Init() (err error) {
	l.Logger.V(2).Info("start Init")
	defer func() {
		l.Logger.V(2).Info("end Init", "err", err)
	}()

	l.Logger.Info("initializing CDI module")

	// get env
	logFile := os.Getenv("CDI_LOGFILE")
	logPath := os.Getenv("CDI_LOGPATH")
	maxSize, _ := strconv.Atoi(os.Getenv("CDI_MAXSIZE"))
	maxBackups, _ := strconv.Atoi(os.Getenv("CDI_MAXBACKUPS"))
	maxAge, _ := strconv.Atoi(os.Getenv("CDI_MAXAGE"))

	l.Logger.V(2).Info("CDI configuration loaded", "logFile", logFile, "logPath", logPath, "maxSize", maxSize, "maxBackups", maxBackups, "maxAge", maxAge)

	err = l.Logging.Init(interfaces.LoggingConfig{
		LogFile:    logFile,
		LogPath:    logPath,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
	})

	if err != nil {
		l.Logger.V(2).Info("branch: logging initialization failed", "error", err.Error())
		return
	}

	l.Logger.V(2).Info("branch: logging initialized successfully")
	l.Logger.Info("CDI module initialization completed")
	return
}

func (l *CDIImplement) Finalize() {
	l.Logger.V(2).Info("start Finalize")
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()

	l.Logger.Info("finalizing CDI module")
	l.Logging.Finalize()
}

func (l *CDIImplement) Collection(targetList []interfaces.CDITargetList) {
	l.Logger.V(2).Info("start Collection", "targetListCount", len(targetList))
	defer func() {
		l.Logger.V(2).Info("end Collection")
	}()

	l.Logger.Info("starting CDI data collection", "targetCount", len(targetList))

	for i, target := range targetList {
		l.Logger.V(2).Info("processing CDI target", "index", i, "cdiHost", target.CDIHost)

		// determine product type
		product := models.ParseProductTypeFromJSON[models.CDIProductType](target.ProductInfo)
		if (product == models.PG_CDI_1_1) {	// PG-CDI v1.1
			l.Logger.V(2).Info("branch: ", "target is PG-CDI v1.1", "cdiHost", target.CDIHost)

			// parse extra parameters
			extraParams := extra_parameters.PgCDIExtraParameters{}
			if err := json.Unmarshal([]byte(target.ExtraParameters), &extraParams); err != nil {
				l.Logger.V(2).Info("branch: ExtraParameters parse failed", "cdiHost", target.CDIHost, "error", err.Error())
				l.Logger.Error(err, err.Error())
				continue
			}

			// ansible execute
			playbook := "cdi_hh.yaml"
			extrArgs := fmt.Sprintf("cdi_pass=%s director_pass=%s",
				extraParams.CDIMgrHostPassword,
				extraParams.DirectorPassword,
			)
			l.Logger.V(2).Info("executing Ansible playbook", "playbook", playbook, "cdiHost", target.CDIHost, "cdiGuest", extraParams.CDIGuest)

			output, err := l.Ansible.CmdExecute(context.Background(), extraParams.CDIGuest, extraParams.CDIMgrGuestUser, playbook, extrArgs)
			if err != nil {
				l.Logger.V(2).Info("branch: Ansible execution failed", "cdiHost", target.CDIHost, "cdiGuest", extraParams.CDIGuest, "error", err.Error())
				l.Logger.Error(err, err.Error())
				return
			}

			l.Logger.V(2).Info("branch: Ansible execution successful", "cdiHost", target.CDIHost, "cdiGuest", extraParams.CDIGuest)

			// parse result
			json, err := l.parseMsg(output)
			if err != nil {
				l.Logger.V(2).Info("branch: message parsing failed", "cdiHost", target.CDIHost, "cdiGuest", extraParams.CDIGuest, "error", err.Error())
				l.Logger.Error(err, err.Error())
				return
			}

			l.Logger.V(2).Info("branch: message parsed successfully", "cdiHost", target.CDIHost, "cdiGuest", extraParams.CDIGuest)

			// write logging
			err = l.Logging.Write(extraParams.CDIGuest, json)
			if err != nil {
				l.Logger.V(2).Info("branch: logging write failed", "cdiHost", target.CDIHost, "cdiGuest", extraParams.CDIGuest, "error", err.Error())
				l.Logger.Error(err, err.Error())
				continue
			}
		} else {
			l.Logger.V(2).Info("branch: ", "target is unsupport product", "ProductInfo", target.ProductInfo)
		}

		l.Logger.V(2).Info("branch: log written successfully", "cdiHost", target.CDIHost)
	}

	l.Logger.Info("CDI data collection completed", "targetCount", len(targetList))
}

func (l CDIImplement) parseMsg(input interface{}) (jsonOut string, err error) {
	l.Logger.V(2).Info("start parseMsg")
	defer func() {
		l.Logger.V(2).Info("end parseMsg", "err", err)
	}()

	result := make(map[string]map[string]string)

	inputLines, ok := input.([]interface{})
	if !ok {
		l.Logger.V(2).Info("branch: input is not []interface{}")
		err = fmt.Errorf("input is not []interface{}")
		return
	}

	l.Logger.V(2).Info("branch: processing input lines", "lineCount", len(inputLines))

	for i, line := range inputLines {
		lineStr, ok := line.(string)
		if !ok {
			l.Logger.V(2).Info("branch: line is not string", "index", i)
			continue
		}

		parts := strings.SplitN(lineStr, "=", 2)
		if len(parts) != 2 {
			l.Logger.V(2).Info("branch: line does not contain '='", "index", i, "line", lineStr)
			continue
		}

		keyValuePair := strings.SplitN(strings.TrimSpace(parts[0]), " ", 2)
		if len(keyValuePair) != 2 {
			l.Logger.V(2).Info("branch: invalid key-value format", "index", i, "keyPart", parts[0])
			continue
		}

		category := keyValuePair[0]
		subKey := keyValuePair[1]
		value := strings.TrimSpace(parts[1])

		if _, ok := result[category]; !ok {
			result[category] = make(map[string]string)
			l.Logger.V(2).Info("branch: created new category", "category", category)
		}

		result[category][subKey] = value
		l.Logger.V(2).Info("branch: added key-value pair", "category", category, "subKey", subKey)
	}

	l.Logger.V(2).Info("branch: parsing completed", "categoryCount", len(result))

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		l.Logger.V(2).Info("branch: JSON marshaling failed", "error", err.Error())
		return
	}

	l.Logger.V(2).Info("branch: JSON marshaling successful")
	jsonOut = string(jsonData)
	return
}
