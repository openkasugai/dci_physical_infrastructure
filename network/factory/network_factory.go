package factory

import (
	"k8s.io/klog/v2"

	"common/models"
	proto "network_module/api/proto" // import of gRPC protobuf
	"network_module/internal/server/implementation/edgecore_sonic_network" // import edgecode sonic network implement
	"network_module/internal/server/implementation/broadcom_sonic_network" // import broadcom sonic network implement
	"network_module/internal/server/implementation/dummy_network" // import dummy network implement
	"network_module/internal/server/interfaces"                   // import of network interface
	"network_module/internal/server/utils"                        // import of network utils
)

// generate instance of network controller
func CreateNetworkController(logger klog.Logger, productInfo *proto.ProductInformation) interfaces.NetworkController {
	defer func() {
		logger.V(2).Info("end CreateNetworkController")
	}()
	logger.V(2).Info("start CreateNetworkController", "logger", logger, "product_info", productInfo)

	// judge product information and create Network controller
	product := models.ParseProductTypeFromFields[models.NWProductType](productInfo.GetVendor(), productInfo.GetProductName(), productInfo.GetVersion(), productInfo.GetOs())
	if (product == models.EdgeCoreSonic) {	// EdgeCore-SONiC
		// get configuration
		config := utils.GetConfig()

		logger.V(2).Info("branch: creating EdgeCore-SONiC network ansible instance")
		networkAnsible := &edgecore_sonic_network.EdgeCoreSonicAnsible{Logger: logger.WithName("EdgeCoreSonicNetworkController"), AnsibleSubDir: "edgecore_sonic_network"} // instance of network Ansible

		logger.V(2).Info("branch: creating EdgeCore-SONiC network controller instance")
		return &edgecore_sonic_network.EdgeCoreSonicNetworkController{Logger: logger.WithName("EdgeCoreSonicNetworkController"), Ansible: networkAnsible, SSHKey: config.SSHKey}
	} else if (product == models.BroadcomSonic) {	// Broadcom-SONiC
		// get configuration
		config := utils.GetConfig()

		broadcomLogger := logger.WithName("BroadcomSonicNetworkController")
		logger.V(2).Info("branch: creating Broadcom-SONiC network ansible instance")
		broadcomAnsible := &broadcom_sonic_network.BroadcomSonicAnsible{
			EdgeCoreSonicAnsible: edgecore_sonic_network.EdgeCoreSonicAnsible{
				Logger:        broadcomLogger,
				AnsibleSubDir: "broadcom_sonic_network",
			},
		}

		logger.V(2).Info("branch: creating Broadcom-SONiC network controller instance")
		return &broadcom_sonic_network.BroadcomSonicNetworkController{
			EdgeCoreSonicNetworkController: edgecore_sonic_network.EdgeCoreSonicNetworkController{
				Logger:  broadcomLogger,
				Ansible: broadcomAnsible,
				SSHKey:  config.SSHKey,
			},
		}
	} else if (product == models.Dummy) {	// dummy
		logger.V(2).Info("branch: creating dummy network controller instance")
		return &dummy_network.DummyNetworkController{Logger: logger}
	}

	// unsupported product information
	logger.V(2).Info("branch: unsupported product information")
	return nil
}
