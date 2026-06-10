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

package factory

import (
	"k8s.io/klog/v2"

	"log_module/internal/server/implementation" // import of implement
	"log_module/internal/server/interfaces"     // import of interface
)

type DatabaseInstanceCreator func(klog.Logger, interfaces.API) interfaces.Database
type IPMIInstanceCreator func(klog.Logger, interfaces.API, interfaces.Logging) interfaces.IPMI
type CDIInstanceCreator func(klog.Logger, interfaces.Ansible, interfaces.Logging) interfaces.CDI
type CDISoftInstanceCreator func(klog.Logger, interfaces.Ansible, interfaces.Logging) interfaces.CDI
type MaasServerInstanceCreator func(klog.Logger, interfaces.API, interfaces.Logging) interfaces.MaasServer
type AnsibleInstanceCreator func(klog.Logger) interfaces.Ansible
type APIInstanceCreator func(klog.Logger) interfaces.API
type LoggingInstanceCreator func(klog.Logger) interfaces.Logging

// generate instance of Database
var CreateDatabaseInstance DatabaseInstanceCreator = func(logger klog.Logger, api interfaces.API) interfaces.Database {
	logger.V(2).Info("start CreateDatabaseInstance")
	defer func() {
		logger.V(2).Info("end CreateDatabaseInstance")
	}()

	logger.V(2).Info("creating DatabaseImplement instance")
	return &implementation.DatabaseImplement{Logger: logger, API: api}
}

// generate instance of IPMI
var CreateIPMIInstance IPMIInstanceCreator = func(logger klog.Logger, api interfaces.API, logging interfaces.Logging) interfaces.IPMI {
	logger.V(2).Info("start CreateIPMIInstance")
	defer func() {
		logger.V(2).Info("end CreateIPMIInstance")
	}()

	logger.V(2).Info("creating IpmiImplement instance")
	return &implementation.IPMIImplement{Logger: logger, API: api, Logging: logging}
}

// generate instance of CDI
var CreateCDIInstance CDIInstanceCreator = func(logger klog.Logger, ansible interfaces.Ansible, logging interfaces.Logging) interfaces.CDI {
	logger.V(2).Info("start CreateCDIInstance")
	defer func() {
		logger.V(2).Info("end CreateCDIInstance")
	}()

	logger.V(2).Info("creating CdiImplement instance")
	return &implementation.CDIImplement{Logger: logger, Ansible: ansible, Logging: logging}
}

// generate instance of CDISoft
var CreateCDISoftInstance CDISoftInstanceCreator = func(logger klog.Logger, ansible interfaces.Ansible, logging interfaces.Logging) interfaces.CDI {
	logger.V(2).Info("start CreateCDISoftInstance")
	defer func() {
		logger.V(2).Info("end CreateCDISoftInstance")
	}()

	logger.V(2).Info("creating CdiSoftImplement instance")
	return &implementation.CDISoftImplement{Logger: logger, Ansible: ansible, Logging: logging}
}

// generate instance of MaasServer
var CreateMaasServerInstance MaasServerInstanceCreator = func(logger klog.Logger, api interfaces.API, logging interfaces.Logging) interfaces.MaasServer {
	logger.V(2).Info("start CreateMaasServerInstance")
	defer func() {
		logger.V(2).Info("end CreateMaasServerInstance")
	}()

	logger.V(2).Info("creating MaaSImplement instance")
	return &implementation.MaaSImplement{Logger: logger, API: api, Logging: logging}
}

// generate instance of Ansible
var CreateAnsibleInstance AnsibleInstanceCreator = func(logger klog.Logger) interfaces.Ansible {
	logger.V(2).Info("start CreateAnsibleInstance")
	defer func() {
		logger.V(2).Info("end CreateAnsibleInstance")
	}()

	logger.V(2).Info("creating AnsibleImplement instance")
	return implementation.NewAnsibleImplement(logger)
}

// generate instance of API
var CreateAPIInstance APIInstanceCreator = func(logger klog.Logger) interfaces.API {
	logger.V(2).Info("start CreateAPIInstance")
	defer func() {
		logger.V(2).Info("end CreateAPIInstance")
	}()

	logger.V(2).Info("creating ApiImplement instance")
	return &implementation.APIImplement{Logger: logger}
}

// generate instance of Logging
var CreateLoggingInstance LoggingInstanceCreator = func(logger klog.Logger) interfaces.Logging {
	logger.V(2).Info("start CreateLoggingInstance")
	defer func() {
		logger.V(2).Info("end CreateLoggingInstance")
	}()

	logger.V(2).Info("creating LoggingImplement instance")
	return &implementation.LoggingImplement{Logger: logger}
}
