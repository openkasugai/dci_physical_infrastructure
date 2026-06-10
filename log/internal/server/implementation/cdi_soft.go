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

	"k8s.io/klog/v2"
)

// struct of CDISoft
type CDISoftImplement struct {
	Logger  klog.Logger
	Ansible interfaces.Ansible
	Logging interfaces.Logging
}

func (l CDISoftImplement) Init() (err error) {
	l.Logger.V(2).Info("start Init")
	defer func() {
		l.Logger.V(2).Info("end Init", "err", err)
	}()

	l.Logger.Info("initializing CDISoft module")

	// get env
	logFile := os.Getenv("CDISOFT_LOGFILE")
	logPath := os.Getenv("CDISOFT_LOGPATH")
	maxSize, _ := strconv.Atoi(os.Getenv("CDISOFT_MAXSIZE"))
	maxBackups, _ := strconv.Atoi(os.Getenv("CDISOFT_MAXBACKUPS"))
	maxAge, _ := strconv.Atoi(os.Getenv("CDISOFT_MAXAGE"))

	l.Logger.V(2).Info("CDISoft configuration loaded", "logFile", logFile, "logPath", logPath, "maxSize", maxSize, "maxBackups", maxBackups, "maxAge", maxAge)

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
	l.Logger.Info("CDISoft module initialization completed")
	return
}

func (l CDISoftImplement) Finalize() {
	l.Logger.V(2).Info("start Finalize")
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()

	l.Logger.Info("finalizing CDISoft module")
	l.Logging.Finalize()
}

func (l CDISoftImplement) Collection(targetList []interfaces.CDITargetList) {
	l.Logger.V(2).Info("start Collection", "targetListCount", len(targetList))
	defer func() {
		l.Logger.V(2).Info("end Collection")
	}()

	l.Logger.Info("starting CDISoft data collection", "targetCount", len(targetList))

	for i, target := range targetList {
		l.Logger.V(2).Info("processing CDISoft target", "index", i, "cdiHost", target.CDIHost)

		// determine product type
		product := models.ParseProductTypeFromJSON[models.CDIProductType](target.ProductInfo)
		if (product == models.PG_CDI_1_0 || product == models.PG_CDI_1_1) {	// PG-CDI
			l.Logger.V(2).Info("branch: ", "target is PG-CDI v1.0/v1.1", "cdiHost", target.CDIHost)

			// parse extra parameters
			extraParams := extra_parameters.PgCDIExtraParameters{}
			if err := json.Unmarshal([]byte(target.ExtraParameters), &extraParams); err != nil {
				l.Logger.V(2).Info("branch: ExtraParameters parse failed", "cdiHost", target.CDIHost, "error", err.Error())
				l.Logger.Error(err, err.Error())
				continue
			}

			// ansible execute
			playbook := "cdi_spec_list.yaml"
			extrArgs := fmt.Sprintf("cdi_user=%s cdi_password=%s cdi_guest=%s",
				extraParams.CDIMgrGuestUser,
				extraParams.CDIMgrGuestPassword,
				extraParams.CDIGuest,
			)
			l.Logger.V(2).Info("executing Ansible playbook", "playbook", playbook, "cdiHost", target.CDIHost, "cdiGuest", extraParams.CDIGuest)

			result := ""
			_, err := l.Ansible.CmdExecute(context.Background(), extraParams.CDIGuest, extraParams.CDIMgrGuestUser, playbook, extrArgs)
			if err == nil {
				result = "{\"Health\":\"OK\"}"
			} else {
				result = "{\"Health\":\"NG\"}"
			}

			l.Logger.V(2).Info("branch: Ansible execution successful", "cdiHost", target.CDIHost, "cdiGuest", extraParams.CDIGuest)

			// write logging
			err = l.Logging.Write(extraParams.CDIGuest, result)
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

	l.Logger.Info("CDISoft data collection completed", "targetCount", len(targetList))
}
