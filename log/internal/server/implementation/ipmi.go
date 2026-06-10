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
	"errors"
	"fmt"
	"common/models"
	"log_module/internal/server/interfaces" // import for interface
	"os"
	"strconv"

	"k8s.io/klog/v2"
)

// HTTP custom error struct
type CustomError struct {
	StatusCode int
	Message    string
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("<%d> %s", e.StatusCode, e.Message)
}

// struct of IPMI
type IPMIImplement struct {
	Logger  klog.Logger
	API     interfaces.API
	Logging interfaces.Logging
}

func (l *IPMIImplement) Init() (err error) {
	l.Logger.V(2).Info("start Init")
	defer func() {
		l.Logger.V(2).Info("end Init", "err", err)
	}()

	l.Logger.Info("initializing IPMI logging system")

	// get env
	logFile := os.Getenv("IPMI_LOGFILE")
	logPath := os.Getenv("IPMI_LOGPATH")
	maxSize, _ := strconv.Atoi(os.Getenv("IPMI_MAXSIZE"))
	maxBackups, _ := strconv.Atoi(os.Getenv("IPMI_MAXBACKUPS"))
	maxAge, _ := strconv.Atoi(os.Getenv("IPMI_MAXAGE"))

	l.Logger.V(2).Info("IPMI logging configuration", "logFile", logFile, "logPath", logPath, "maxSize", maxSize, "maxBackups", maxBackups, "maxAge", maxAge)

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

	l.Logger.V(2).Info("branch: logging initialization successful")
	l.Logger.Info("IPMI logging system initialization completed")
	return
}

func (l *IPMIImplement) Finalize() {
	l.Logger.V(2).Info("start Finalize")
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()

	l.Logger.Info("finalizing IPMI logging system")
	l.Logging.Finalize()
}

func (l *IPMIImplement) Collection(targetList []interfaces.IPMITargetList) {
	l.Logger.V(2).Info("start Colloction", "targetListCount", len(targetList))
	defer func() {
		l.Logger.V(2).Info("end Colloction")
	}()

	l.Logger.Info("starting IPMI data collection", "targetCount", len(targetList))

	for i, target := range targetList {
		l.Logger.V(2).Info("processing IPMI target", "index", i, "serverID", target.ServerID, "ProductInfo", target.ProductInfo)

		// determine product type
		product := models.ParseProductTypeFromJSON[models.ServerProductType](target.ProductInfo)
		
		var api string
		if (product == models.Dell) {					// DELL
			api = "https://redfish/v1/Systems/System.Embedded.1"
			l.Logger.V(2).Info("branch: DELL server API selected", "api", api)
		} else if (product == models.Primergy) {		// PRIMERGY
			api = "https://redfish/v1/Systems/0"
			l.Logger.V(2).Info("branch: PRIMERGY server API selected", "api", api)
		} else if (product == models.Supermicro) {		// Supermicro
			api = "https://redfish/v1/Systems/1"
			l.Logger.V(2).Info("branch: ", "target is Supermicro server", "api", api)
		} else {
			l.Logger.V(2).Info("branch: ", "target is unsupport product", "ProductInfo", target.ProductInfo)
			continue
		}

		// API execute
		resp, err := l.API.APIExecuteUserAuth(context.Background(), "GET",
			target.IPMIAddress, api,
			target.IPMIUser, target.IPMIPassword, "")
		if err != nil {
			l.Logger.V(2).Info("branch: API execution failed", "serverID", target.ServerID, "error", err.Error())
			l.Logger.Error(err, err.Error())
			continue
		}

		l.Logger.V(2).Info("branch: API execution successful", "serverID", target.ServerID)

		// ProcessorSummary
		processer, err := getJsonObjectValue(resp, "ProcessorSummary")
		if err != nil {
			l.Logger.V(2).Info("branch: ProcessorSummary extraction failed", "serverID", target.ServerID, "error", err.Error())
			l.Logger.Error(err, err.Error())
			continue
		}
		processer, err = getJsonObjectValue(processer, "Status")
		if err != nil {
			l.Logger.V(2).Info("branch: ProcessorSummary Status extraction failed", "serverID", target.ServerID, "error", err.Error())
			l.Logger.Error(err, err.Error())
			continue
		}
		processerHelth, err := getJsonStringValue(processer, "Health")
		if err != nil {
			l.Logger.V(2).Info("branch: ProcessorSummary Health extraction failed", "serverID", target.ServerID, "error", err.Error())
			l.Logger.Error(err, err.Error())
			continue
		}

		l.Logger.V(2).Info("branch: processor health extracted", "serverID", target.ServerID, "health", processerHelth)

		// MemorrySummary
		memory, err := getJsonObjectValue(resp, "MemorySummary")
		if err != nil {
			l.Logger.V(2).Info("branch: MemorySummary extraction failed", "serverID", target.ServerID, "error", err.Error())
			l.Logger.Error(err, err.Error())
			continue
		}
		memory, err = getJsonObjectValue(memory, "Status")
		if err != nil {
			l.Logger.V(2).Info("branch: MemorySummary Status extraction failed", "serverID", target.ServerID, "error", err.Error())
			l.Logger.Error(err, err.Error())
			continue
		}
		memoryHelth, err := getJsonStringValue(memory, "Health")
		if err != nil {
			l.Logger.V(2).Info("branch: MemorySummary Health extraction failed", "serverID", target.ServerID, "error", err.Error())
			l.Logger.Error(err, err.Error())
			continue
		}

		l.Logger.V(2).Info("branch: memory health extracted", "serverID", target.ServerID, "health", memoryHelth)

		// write logging
		logData := fmt.Sprintf(`{ "processer": "%s", "memory": "%s" }`, processerHelth, memoryHelth)
		err = l.Logging.Write(target.ServerID, logData)
		if err != nil {
			l.Logger.V(2).Info("branch: logging write failed", "serverID", target.ServerID, "error", err.Error())
			l.Logger.Error(err, err.Error())
			continue
		}

		l.Logger.V(2).Info("branch: log written successfully", "serverID", target.ServerID)
	}

	l.Logger.Info("IPMI data collection completed", "targetCount", len(targetList))
}

func getValue(inputJson interface{}, key string) (value interface{}, err error) {
	fixedMessage := "invalid redfish response"
	obj, ok := inputJson.(map[string]interface{})
	if !ok {
		err = errors.New(fixedMessage + ": inputJson is not invalid type")
		return
	}

	// ket exist check
	v, ok := obj[key]
	if !ok {
		err = errors.New(fixedMessage + ": Key " + key + " not found in inputJson")
		return
	}
	value = v
	return
}

func getJsonObjectValue(inputJson interface{}, key string) (value map[string]interface{}, err error) {
	fixedMessage := "invalid redfish response"

	v, err := getValue(inputJson, key)
	if err != nil {
		return
	}

	// type check
	mapVal, ok := v.(map[string]interface{})
	if !ok {
		err = errors.New(fixedMessage + ": Value for key " + key + " is not a int")
		return
	}
	value = mapVal
	return
}

func getJsonStringValue(inputJson interface{}, key string) (value string, err error) {
	fixedMessage := "invalid redfish response"

	v, err := getValue(inputJson, key)
	if err != nil {
		return
	}

	// type check
	strVal, ok := v.(string)
	if !ok {
		err = errors.New(fixedMessage + ": Value for key " + key + " is not a string")
		return
	}
	value = strVal
	return
}
