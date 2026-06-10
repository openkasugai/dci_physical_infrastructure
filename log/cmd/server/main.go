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

package main

import (
	"flag"
	"os"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"log_module/factory"               // import of factory
	"log_module/internal/server/utils" // import of utilities
)

// init klog for once done
var klogInitOnce sync.Once

func initKlog() {
	klog.V(2).InfoS("start initKlog")
	defer func() {
		klog.V(2).InfoS("end initKlog")
	}()

	klogInitOnce.Do(func() {
		klog.V(2).InfoS("branch: initializing klog for the first time")
		klog.InitFlags(nil)

		// get configuration (should be initialized before calling this function)
		config := utils.GetConfig()
		klog.V(2).InfoS("configuration retrieved", "logLevel", config.LogLevel)

		err := flag.Set("v", config.LogLevel) // setting log-level
		if err != nil {
			klog.V(2).InfoS("branch: flag.Set failed", "error", err.Error())
			return
		}
		klog.V(2).InfoS("branch: log level set successfully")
		flag.Parse()
	})
}

var running bool = true

// main process
func main() {
	klog.V(2).InfoS("start main")
	defer func() {
		klog.V(2).InfoS("end main")
	}()

	klog.InfoS("starting log module main process")

	// Initialize and validate environment variables first
	err := utils.InitializeConfig()
	if err != nil {
		klog.V(2).InfoS("branch: InitializeConfig failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on validation failure
		return
	}
	klog.V(2).InfoS("branch: configuration initialized successfully")

	config := utils.GetConfig()
	klog.V(2).InfoS("configuration loaded", "logLevel", config.LogLevel, "interval", config.Interval)

	// setup klog
	initKlog()
	defer klog.Flush() // flash of log when ending

	klog.V(0).InfoS("LOG LEVEL 0")
	klog.V(1).InfoS("LOG LEVEL 1")
	klog.V(2).InfoS("LOG LEVEL 2")
	klog.V(3).InfoS("LOG LEVEL 3")
	klog.V(4).InfoS("LOG LEVEL 4")
	klog.V(5).InfoS("LOG LEVEL 5")
	klog.V(6).InfoS("LOG LEVEL 6")
	klog.V(7).InfoS("LOG LEVEL 7")
	klog.V(8).InfoS("LOG LEVEL 8")
	klog.V(9).InfoS("LOG LEVEL 9")

	klog.InfoS("starting DCI physical infrastructure process")

	// get instanse
	klog.V(2).InfoS("creating instances")
	ansible := factory.CreateAnsibleInstance(klog.Background())
	api := factory.CreateAPIInstance(klog.Background())
	database := factory.CreateDatabaseInstance(klog.Background(), api)
	ipmiLogging := factory.CreateLoggingInstance(klog.Background())
	cdiLogging := factory.CreateLoggingInstance(klog.Background())
	cdiSoftLogging := factory.CreateLoggingInstance(klog.Background())
	maasServerLogging := factory.CreateLoggingInstance(klog.Background())
	ipmi := factory.CreateIPMIInstance(klog.Background(), api, ipmiLogging)
	cdi := factory.CreateCDIInstance(klog.Background(), ansible, cdiLogging)
	cdiSoft := factory.CreateCDISoftInstance(klog.Background(), ansible, cdiSoftLogging)
	maasServer := factory.CreateMaasServerInstance(klog.Background(), api, maasServerLogging)
	klog.V(2).InfoS("instances created successfully")

	// initialize database
	klog.InfoS("initializing database")
	err = database.Init()
	if err != nil {
		klog.V(2).InfoS("branch: database initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on initialization failure
		return
	}
	klog.V(2).InfoS("branch: database initialized successfully")

	// initialize IPMI
	klog.InfoS("initializing IPMI")
	err = ipmi.Init()
	if err != nil {
		klog.V(2).InfoS("branch: IPMI initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on initialization failure
		return
	}
	klog.V(2).InfoS("branch: IPMI initialized successfully")

	// initialize CDI
	klog.InfoS("initializing CDI")
	err = cdi.Init()
	if err != nil {
		klog.V(2).InfoS("branch: CDI initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on initialization failure
		return
	}
	klog.V(2).InfoS("branch: CDI initialized successfully")

	// initialize CDISoft
	klog.InfoS("initializing CDISoft")
	err = cdiSoft.Init()
	if err != nil {
		klog.V(2).InfoS("branch: CDISoft initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on initialization failure
		return
	}
	klog.V(2).InfoS("branch: CDISoft initialized successfully")

	// initialize MaasServer
	klog.InfoS("initializing MaasServer")
	err = maasServer.Init()
	if err != nil {
		klog.V(2).InfoS("branch: MaasServer initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on initialization failure
		return
	}
	klog.V(2).InfoS("branch: MaasServer initialized successfully")

	klog.InfoS("starting monitoring process", "interval", config.Interval)

	// create ticker
	ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
	defer ticker.Stop()
	klog.V(2).InfoS("ticker created", "intervalSeconds", config.Interval)

	// goroutine monitoring process
	go func() {
		klog.V(2).InfoS("monitoring goroutine started")
		for range ticker.C {
			if !running {
				klog.V(2).InfoS("branch: monitoring stopped")
				return
			}

			klog.V(2).InfoS("monitoring cycle started")

			// select cdi target
			cdiList, err := database.SelectCDITable()
			if err != nil {
				klog.V(2).InfoS("branch: SelectCdiTable failed", "error", err.Error())
				klog.Error(err.Error())
			} else {
				klog.V(2).InfoS("branch: CDI targets selected", "count", len(cdiList))
			}

			// select ipmi target
			ipmiList, err := database.SelectServerTable()
			if err != nil {
				klog.V(2).InfoS("branch: SelectServerTable failed", "error", err.Error())
				klog.Error(err.Error())
			} else {
				klog.V(2).InfoS("branch: IPMI targets selected", "count", len(ipmiList))
			}

			// select maas server target
			maasServerList, err := database.SelectMaasTable()
			if err != nil {
				klog.V(2).InfoS("branch: SelectMaasTable failed", "error", err.Error())
				klog.Error(err.Error())
			} else {
				klog.V(2).InfoS("branch: MaasServer targets selected", "count", len(maasServerList))
			}

			// collection cdi target
			go func() {
				klog.V(2).InfoS("CDI collection goroutine started")
				cdi.Collection(cdiList)
			}()

			// collection ipmi target
			go func() {
				klog.V(2).InfoS("IPMI collection goroutine started")
				ipmi.Collection(ipmiList)
			}()

			// collection cdi soft target
			go func() {
				klog.V(2).InfoS("CDISoft collection goroutine started")
				cdiSoft.Collection(cdiList)
			}()

			// collection maas server target
			go func() {
				klog.V(2).InfoS("MaasServer collection goroutine started")
				maasServer.Collection(maasServerList)
			}()

			klog.V(2).InfoS("monitoring cycle completed")
		}
	}()

	klog.InfoS("entering main loop")
	for {
		time.Sleep(time.Second)
	}
}
