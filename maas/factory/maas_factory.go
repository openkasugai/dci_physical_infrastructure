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

	"common/models"
	proto "maas_module/api/proto" // import of gRPC protobuf
	"maas_module/internal/server/implementation/canonical_maas" // import MaaS implement
	"maas_module/internal/server/implementation/canonical_maas/maas_api"
	"maas_module/internal/server/interfaces" // import MaaS interface
)

// CreateMaasController creates an instance of the Canonical MaaS controller.
func CreateMaasController(logger klog.Logger, productInfo *proto.ProductInformation, maasInfo *proto.MaasInformation) interfaces.MaasController {
	klog.V(2).InfoS("start CreateMaasController", "logger", logger, "productInfo", productInfo)
	defer func() {
		klog.V(2).InfoS("end CreateMaasController")
	}()

	// judge product information and create Maas controller
	product := models.ParseProductTypeFromFields[models.MaasProductType](productInfo.GetVendor(), productInfo.GetProductName(), productInfo.GetVersion(), productInfo.GetOs())
	if (product == models.Canonical) {	// Canonical MaaS
		klog.V(2).InfoS("branch: creating command executor")
		executor := &canonical_maas.CmdExecutor{}

		klog.V(2).InfoS("branch: creating Canonical-MaaS API factory")
		api := &maas_api.MaasAPIFactoryImple{API: canonical_maas.NewCanonicalMaasAPIImple(logger, maasInfo.GetAccessUrl(), maasInfo.GetApiKey()), Logger: logger} // instanse of MaaS API factory

		klog.V(2).InfoS("branch: creating Canonical-MaaS Ansible instance")
		ansible := &canonical_maas.CanonicalMaasAnsibleImple{Logger: logger, Executor: executor} // instanse of MaaS Ansible

		klog.InfoS("Canonical-MaaS controller components initialized successfully")
		return &canonical_maas.CanonicalMaasController{Logger: logger, APIFactory: api, Ansible: ansible, JobManager: canonical_maas.NewInMemoryJobManager()}
	}

	// unsupported product information
	logger.V(2).Info("branch: unsupported product information")
	return nil
}
