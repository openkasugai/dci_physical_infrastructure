package factory

import (
	"k8s.io/klog/v2"

	"common/models"
	proto "maas_module/api/proto" // import of gRPC protobuf
	"maas_module/internal/server/implementation/async_job"
	"maas_module/internal/server/implementation/canonical_maas" // import MaaS implement
	"maas_module/internal/server/implementation/canonical_maas/maas_api"
	"maas_module/internal/server/interfaces" // import MaaS interface
	"maas_module/internal/server/utils"
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

		config := utils.GetConfig()
		workerImage := ""
		serviceAccount := ""
		sshSecretName := ""
		tlsSecretName := ""
		tlsEnabled := false
		tlsCertPath := ""
		backoffLimit := int32(0)
		ttlSeconds := int32(600)
		activeDeadline := int64(7200)
		if config != nil {
			workerImage = config.K8sJobWorkerImage
			serviceAccount = config.K8sJobServiceAccount
			sshSecretName = config.K8sJobSSHSecretName
			tlsSecretName = config.K8sJobTLSSecretName
			tlsEnabled = config.TlsEnable
			tlsCertPath = config.TlsCertPath
			backoffLimit = config.K8sJobBackoffLimit
			ttlSeconds = config.K8sJobTTLSeconds
			activeDeadline = config.K8sJobActiveDeadlineSecs
		}
		klog.V(2).InfoS("branch: creating K8sJobManager")
		jobManager := async_job.NewK8sJobManager(logger, async_job.K8sJobStoreOptions{
			Namespace:             "",
			WorkerImage:           workerImage,
			ServiceAccountName:    serviceAccount,
			SshSecretName:         sshSecretName,
			TlsSecretName:         tlsSecretName,
			TlsEnabled:            tlsEnabled,
			TlsCertPath:           tlsCertPath,
			BackoffLimit:          backoffLimit,
			TTLSecondsAfterFinish: ttlSeconds,
			ActiveDeadlineSeconds: activeDeadline,
		})

		klog.V(2).InfoS("Canonical-MaaS controller components initialized successfully")
		return &canonical_maas.CanonicalMaasController{Logger: logger, APIFactory: api, Ansible: ansible, JobManager: jobManager}
	}

	// unsupported product information
	logger.V(2).Info("branch: unsupported product information")
	return nil
}
