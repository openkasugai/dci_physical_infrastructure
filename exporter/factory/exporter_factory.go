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

	"exporter_module/internal/server/implementation" // import of implement
	"exporter_module/internal/server/interfaces"     // import of interface
)

type DatabaseInstanceCreator func(klog.Logger, interfaces.API) interfaces.Database
type ServerInstanceCreator func(klog.Logger, interfaces.Ansible, interfaces.API, interfaces.Metrics, interfaces.Manager) interfaces.Server
type NetworkInstanceCreator func(klog.Logger, interfaces.Ansible, interfaces.Metrics, interfaces.Manager) interfaces.Network
type MetricsInstanceCreator func(klog.Logger) interfaces.Metrics
type AnsibleInstanceCreator func(klog.Logger) interfaces.Ansible
type APIInstanceCreator func(klog.Logger) interfaces.API
type ManagerInstanceCreator func(klog.Logger, interfaces.Ansible, interfaces.Metrics) interfaces.Manager

// generate instance of Database
var CreateDatabaseInstance DatabaseInstanceCreator = func(logger klog.Logger, api interfaces.API) interfaces.Database {
	logger.V(2).Info("start CreateDatabaseInstance")
	defer func() {
		logger.V(2).Info("end CreateDatabaseInstance")
	}()

	logger.V(2).Info("creating DatabaseImplement instance")
	return &implementation.DatabaseImplement{Logger: logger, API: api}
}

// generate instance of Server
var CreateServerInstance ServerInstanceCreator = func(logger klog.Logger, ansible interfaces.Ansible, api interfaces.API, metrics interfaces.Metrics, manager interfaces.Manager) interfaces.Server {
	logger.V(2).Info("start CreateServerInstance")
	defer func() {
		logger.V(2).Info("end CreateServerInstance")
	}()

	logger.V(2).Info("creating ServerImplement instance")
	return &implementation.ServerImplement{Logger: logger, Ansible: ansible, API: api, Metrics: metrics, Manager: manager}
}

// generate instance of Network
var CreateNetworkInstance NetworkInstanceCreator = func(logger klog.Logger, ansible interfaces.Ansible, metrics interfaces.Metrics, manager interfaces.Manager) interfaces.Network {
	logger.V(2).Info("start CreateNetworkInstance")
	defer func() {
		logger.V(2).Info("end CreateNetworkInstance")
	}()

	logger.V(2).Info("creating NetworkImplement instance")
	return &implementation.NetworkImplement{Logger: logger, Ansible: ansible, Metrics: metrics, Manager: manager}
}

// generate instance of Metrics
var CreateMetricsInstance MetricsInstanceCreator = func(logger klog.Logger) interfaces.Metrics {
	logger.V(2).Info("start CreateMetricsInstance")
	defer func() {
		logger.V(2).Info("end CreateMetricsInstance")
	}()

	logger.V(2).Info("creating MetricsImplement instance")
	return &implementation.MetricsImplement{Logger: logger}
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

	logger.V(2).Info("creating APIImplement instance")
	return &implementation.APIImplement{Logger: logger}
}

// generate instance of Manager
var CreateManagerInstance ManagerInstanceCreator = func(logger klog.Logger, ansible interfaces.Ansible, metrics interfaces.Metrics) interfaces.Manager {
	logger.V(2).Info("start CreateManagerInstance")
	defer func() {
		logger.V(2).Info("end CreateManagerInstance")
	}()

	logger.V(2).Info("creating ManagerImplement instance")
	return &implementation.ManagerImplement{Logger: logger, Ansible: ansible, Metrics: metrics}
}
