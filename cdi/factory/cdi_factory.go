package factory

import (
	"k8s.io/klog/v2"

	"common/models"
	proto "cdi_module/api/proto" // import for gRPC protobuf
	"cdi_module/internal/server/implementation/pg_cdi" // import CDI implement
	"cdi_module/internal/server/interfaces"            // import of CDI interface
	"cdi_module/internal/server/utils"                 // import CDI utils
)

// testCreateCDIControllerFunc is a function variable for testing purposes
var testCreateCDIControllerFunc func(logger klog.Logger, productInfo *proto.ProductInformation) interfaces.CDIController

// SetTestCreateCDIControllerFunc sets the test function for CreateCDIController
func SetTestCreateCDIControllerFunc(f func(logger klog.Logger, productInfo *proto.ProductInformation) interfaces.CDIController) {
	testCreateCDIControllerFunc = f
}

// generate instance of CDI controller
func CreateCDIController(logger klog.Logger, productInfo *proto.ProductInformation) interfaces.CDIController {
	// Use test function if set (for testing purposes)
	if testCreateCDIControllerFunc != nil {
		return testCreateCDIControllerFunc(logger, productInfo)
	}
	defer func() {
		logger.V(2).Info("end CreateCDIController")
	}()
	logger.V(2).Info("start CreateCDIController", "logger", logger, "productInfo", productInfo)

	// get configuration
	config := utils.GetConfig()

	// judge product information and create CDI controller
	product := models.ParseProductTypeFromFields[models.CDIProductType](productInfo.GetVendor(), productInfo.GetProductName(), productInfo.GetVersion(), productInfo.GetOs())
	if (product == models.PG_CDI_1_0 || product == models.PG_CDI_1_1 || product == models.PG_CDI_2_0) {	// PG-CDI

		logger.V(2).Info("branch: creating PG-CDI ansible instance")
		cdiAnsible := &pg_cdi.PgCDIAnsibleImple{Logger: logger} // instance of CDI Ansible

		logger.V(2).Info("branch: creating PG-CDI controller instance")
		return &pg_cdi.PgCDIController{Logger: logger, Ansible: cdiAnsible, SSHKey: config.SSHKey}
	}
	
	// unsupported product information
	logger.V(2).Info("branch: unsupported product information")
	return nil
}
