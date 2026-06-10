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
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"

	"exporter_module/factory"               // import of factory
	"exporter_module/internal/server/utils" // import of utilities
)

// init klog for once done
var klogInitOnce sync.Once

func initKlog() {
	klogInitOnce.Do(func() {
		defer func() {
			klog.V(2).InfoS("end initKlog")
		}()
		klog.V(2).InfoS("start initKlog")

		klog.InitFlags(nil)

		// get configuration (should be initialized before calling this function)
		config := utils.GetConfig()

		klog.V(2).InfoS("branch: setting log level", "log_level", config.LogLevel)
		err := flag.Set("v", config.LogLevel) // setting log-level
		if err != nil {
			klog.V(2).InfoS("branch: failed to set log level", "error", err.Error())
			return
		}

		klog.V(2).InfoS("branch: log level set successfully")
		flag.Parse()
	})
}

// metrics endpoint registerd for once done
var metricsEndpointRegistered sync.Once

var running bool = true
var p2pRunning bool = true

// main process
func main() {
	defer func() {
		klog.V(2).InfoS("end main")
	}()
	klog.V(2).InfoS("start main")

	// Initialize and validate environment variables first
	if err := utils.InitializeConfig(); err != nil {
		klog.V(2).InfoS("branch: configuration initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on validation failure
		return
	}

	klog.V(2).InfoS("branch: configuration initialization successful")
	config := utils.GetConfig()

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

	klog.InfoS("Start dci_physical_infrastructure process", "metrics_port", config.MetricsPort, "interval", config.Interval)

	klog.V(2).InfoS("branch: creating instances")
	// get instance
	ansible := factory.CreateAnsibleInstance(klog.Background())
	api := factory.CreateAPIInstance(klog.Background())
	database := factory.CreateDatabaseInstance(klog.Background(), api)
	metrics := factory.CreateMetricsInstance(klog.Background())
	manager := factory.CreateManagerInstance(klog.Background(), ansible, metrics)
	server := factory.CreateServerInstance(klog.Background(), ansible, api, metrics, manager)
	network := factory.CreateNetworkInstance(klog.Background(), ansible, metrics, manager)

	klog.InfoS("instances created successfully")

	klog.V(2).InfoS("branch: initializing database")
	// initialize database
	err := database.Init()
	if err != nil {
		klog.V(2).InfoS("branch: database initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on initialization failure
		return
	}

	klog.InfoS("database initialization successful")

	klog.V(2).InfoS("branch: initializing server")
	// initialize Server
	err = server.Init()
	if err != nil {
		klog.V(2).InfoS("branch: server initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on initialization failure
		return
	}

	klog.InfoS("server initialization successful")

	klog.V(2).InfoS("branch: initializing network")
	// initialize Network
	err = network.Init()
	if err != nil {
		klog.V(2).InfoS("branch: network initialization failed", "error", err.Error())
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on initialization failure
		return
	}

	klog.InfoS("network initialization successful")
	klog.InfoS("Start monitoring...")

	// Create a channel to signal when the server is ready
	serverReady := make(chan bool)

	klog.V(2).InfoS("branch: starting HTTP server")
	// Start the HTTP server in a goroutine
	go func() {
		defer func() {
			klog.V(2).InfoS("end HTTP server goroutine")
		}()
		klog.V(2).InfoS("start HTTP server goroutine")

		// HTTP endpoint setting
		metricsEndpointRegistered.Do(func() {
			klog.V(2).InfoS("branch: registering metrics endpoint", "endpoint", config.MetricsEndpoint)
			http.Handle(config.MetricsEndpoint, promhttp.Handler())
		})

		serverReady <- true // Signal that the server is ready
		klog.InfoS("HTTP server starting", "port", config.MetricsPort)

		err = http.ListenAndServe(fmt.Sprintf(":%d", config.MetricsPort), nil)
		if err != nil {
			klog.V(2).InfoS("branch: HTTP server failed", "error", err.Error())
			klog.Error(err.Error())
			os.Exit(1) // Ensure exit on server failure
			return
		}

		klog.InfoS("HTTP server stopped", "port", config.MetricsPort)
	}()

	// Wait for the server to be ready
	<-serverReady
	klog.InfoS("HTTP server ready")

	klog.V(2).InfoS("branch: starting monitoring ticker", "interval", config.Interval)
	// create ticker
	ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
	defer ticker.Stop()

	// goroutine monitoring process
	go func() {
		defer func() {
			klog.V(2).InfoS("end monitoring goroutine")
		}()
		klog.V(2).InfoS("start monitoring goroutine")

		for range ticker.C {
			if !running {
				klog.V(2).InfoS("branch: monitoring stopped")
				return
			}
			klog.InfoS("ticker triggered")

			klog.V(2).InfoS("branch: selecting network switch targets")
			// select network switch target
			nwList, err := database.SelectNwSwitchTable()
			if err != nil {
				klog.V(2).InfoS("branch: network switch selection failed", "error", err.Error())
				klog.Error(err.Error())
			}

			klog.V(2).InfoS("branch: selecting server targets")
			// select server target
			serverList, err := database.SelectServerTable()
			if err != nil {
				klog.V(2).InfoS("branch: server selection failed", "error", err.Error())
				klog.Error(err.Error())
			}

			klog.V(2).InfoS("branch: starting collection processes", "network_targets", len(nwList), "server_targets", len(serverList))
			// collection network target
			go func() {
				klog.V(2).InfoS("start network collection goroutine")
				network.Collection(nwList)
				klog.V(2).InfoS("end network collection goroutine")
			}()

			// collection server target
			go func() {
				klog.V(2).InfoS("start server collection goroutine")
				server.Colloction(serverList)
				klog.V(2).InfoS("end server collection goroutine")
			}()
		}
	}()

	if config.P2PEnable {
		klog.V(2).InfoS("branch: starting p2p ticker", "interval", config.P2PInterval)
		// create ticker
		p2pTicker := time.NewTicker(time.Duration(config.P2PInterval) * time.Second)
		defer p2pTicker.Stop()

		// goroutine monitoring process
		go func() {
			defer func() {
				klog.V(2).InfoS("end p2p monitoring goroutine")
			}()
			klog.V(2).InfoS("start p2p monitoring goroutine")

			for range p2pTicker.C {
				if !p2pRunning {
					klog.V(2).InfoS("branch: p2p monitoring stopped")
					return
				}
				klog.InfoS("p2p ticker triggered")


				klog.V(2).InfoS("branch: selecting server targets")
				// select server target
				serverList, err := database.SelectServerTable()
				if err != nil {
					klog.V(2).InfoS("branch: server selection failed", "error", err.Error())
					klog.Error(err.Error())
					// continue to next tick
					continue
				}

				// p2p check
				manager.SetP2POn(serverList)
			}
		}()
	} else {
		klog.InfoS("P2P monitoring is disabled")
	}

	klog.InfoS("monitoring process started")
	for {
		time.Sleep(time.Second)
	}
}

