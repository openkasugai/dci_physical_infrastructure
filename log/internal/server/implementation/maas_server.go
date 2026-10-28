package implementation

import (
	"context"
	"common/models"
	"log_module/internal/server/interfaces" // import for interface
	"os"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

// struct of MaaS
type MaaSImplement struct {
	Logger  klog.Logger
	API interfaces.API
	Logging interfaces.Logging
}

func (l MaaSImplement) Init() (err error) {
	l.Logger.V(2).Info("start Init")
	defer func() {
		l.Logger.V(2).Info("end Init", "err", err)
	}()

	l.Logger.Info("initializing MaaS module")

	// get env
	logFile := os.Getenv("MAASSERVER_LOGFILE")
	logPath := os.Getenv("MAASSERVER_LOGPATH")
	maxSize, _ := strconv.Atoi(os.Getenv("MAASSERVER_MAXSIZE"))
	maxBackups, _ := strconv.Atoi(os.Getenv("MAASSERVER_MAXBACKUPS"))
	maxAge, _ := strconv.Atoi(os.Getenv("MAASSERVER_MAXAGE"))

	l.Logger.V(2).Info("MaaS configuration loaded", "logFile", logFile, "logPath", logPath, "maxSize", maxSize, "maxBackups", maxBackups, "maxAge", maxAge)

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
	l.Logger.Info("MaaS module initialization completed")
	return
}

func (l MaaSImplement) Finalize() {
	l.Logger.V(2).Info("start Finalize")
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()

	l.Logger.Info("finalizing MaaS module")
	l.Logging.Finalize()
}

func (l MaaSImplement) Collection(targetList []interfaces.MaasServerTargetList) {
	l.Logger.V(3).Info("start Collection", "targetListCount", len(targetList))
	defer func() {
		l.Logger.V(3).Info("end Collection")
	}()

	l.Logger.V(3).Info("starting MaaS data collection", "targetCount", len(targetList))

	for i, target := range targetList {
		l.Logger.V(3).Info("processing MaaS target", "index", i, "maasURL", target.MaasAccessUrl)

		// determine product type
		product := models.ParseProductTypeFromJSON[models.MaasProductType](target.ProductInfo)
		if (product == models.Canonical) {	// Canonical MaaS
			l.Logger.V(3).Info("branch: ", "target is Canonical MaaS", "maasURL", target.MaasAccessUrl)

			// Execute API to get MaaS configuration
			result := ""
			_, err := l.API.APIExecuteOAuth(context.Background(), "GET", strings.TrimSuffix(target.MaasAccessUrl, "/"), "maas/op-get_config", target.MaasApiKey, "name=maas_name")
			if err == nil {
				result = "{\"Health\":\"OK\"}"
			} else {
				result = "{\"Health\":\"NG\"}"
			}
			l.Logger.V(3).Info("branch: MaaS API executed", "maasURL", target.MaasAccessUrl)

			// write logging
			err = l.Logging.Write(target.MaasAccessUrl, result)
			if err != nil {
					l.Logger.V(3).Info("branch: logging write failed", "maasURL", target.MaasAccessUrl, "error", err.Error())
					l.Logger.Error(err, err.Error())
				continue
			}
		} else {
			l.Logger.V(3).Info("branch: ", "target is unsupport product", "ProductInfo", target.ProductInfo)
		}

		l.Logger.V(3).Info("branch: log written successfully", "maasURL", target.MaasAccessUrl)
	}

	l.Logger.V(3).Info("MaaS data collection completed", "targetCount", len(targetList))
}
