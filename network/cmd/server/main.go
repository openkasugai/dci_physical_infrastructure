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
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"

	proto "network_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"

	"network_module/factory"                    // import of network factory
	"network_module/internal/server/utils"      // import of network utils
)

// struct of gRPC-server for Network-Service
type networkServer struct {
	proto.UnimplementedNetworkServer
}

// generate instance of gRPC-server for Network-service
func newNetworkServer() *networkServer {
	return &networkServer{}
}

// Network-service
func (s *networkServer) VlanAdd(ctx context.Context, in *proto.VlanAddRequest) (reply *proto.VlanAddReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetSwitchInfo().GetRemoteHost()
	interfaceName := in.InterfaceName
	vlanID := in.VlanId

	defer func() {
		klog.V(2).InfoS("end VlanAdd",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID,
			"reply", reply,
			"error", err)
	}()
	klog.V(2).InfoS("start VlanAdd",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID,
		"input", in)

	klog.InfoS("gRPC interface VlanAdd accepted",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)

	// validation
	var validErr error
	validErr = in.Validate()

	// int32 required check
	if validErr != nil && in.VlanId == nil {
		validErr = errors.New("invalid VlanAddRequest.VlanID: value is required")
	}

	if validErr != nil {
		klog.V(2).InfoS("branch: validation failed",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID,
			"error", validErr.Error())
		klog.Warning(validErr.Error())
		reply = &proto.VlanAddReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    validErr.Error(),
			}),
		}
		klog.InfoS("gRPC interface VlanAdd response sent",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID,
			"result", "ERROR")
		return
	}

	klog.V(2).InfoS("branch: validation successful",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)
	
	// generate instanse of network controller
	nwController := factory.CreateNetworkController(klog.Background(), in.GetProductInfo())
	if nwController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.VlanAddReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
				Message:    eMsg,
			}),
		}
		return
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = nwController.VlanAdd(ctx, in)

	klog.InfoS("gRPC interface VlanAdd response sent",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID,
		"result", reply.GetResult())
	return
}

func (s *networkServer) VlanDelete(ctx context.Context, in *proto.VlanDeleteRequest) (reply *proto.VlanDeleteReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetSwitchInfo().GetRemoteHost()
	interfaceName := in.InterfaceName
	vlanID := in.VlanId

	defer func() {
		klog.V(2).InfoS("end VlanDelete",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID,
			"reply", reply,
			"error", err)
	}()
	klog.V(2).InfoS("start VlanDelete",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID,
		"input", in)

	klog.InfoS("gRPC interface VlanDelete accepted",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)

	// validation
	var validErr error
	validErr = in.Validate()

	// int32 required check
	if validErr != nil && in.VlanId == nil {
		validErr = errors.New("invalid VlanDeleteRequest.VlanID: value is required")
	}

	if validErr != nil {
		klog.V(2).InfoS("branch: validation failed",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID,
			"error", validErr.Error())
		klog.Warning(validErr.Error())
		reply = &proto.VlanDeleteReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    validErr.Error(),
			}),
		}
		klog.InfoS("gRPC interface VlanDelete response sent",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID,
			"result", "ERROR")
		return
	}

	klog.V(2).InfoS("branch: validation successful",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)
	
	// generate instanse of network controller
	nwController := factory.CreateNetworkController(klog.Background(), in.GetProductInfo())
	if nwController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.VlanDeleteReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
				Message:    eMsg,
			}),
		}
		return
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = nwController.VlanDelete(ctx, in)

	klog.InfoS("gRPC interface VlanDelete response sent",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID,
		"result", reply.GetResult())
	return
}

func (s *networkServer) VswVlanAdd(ctx context.Context, in *proto.VswVlanAddRequest) (reply *proto.VswVlanAddReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetHostInfo().GetRemoteHost()
	vlanID := in.VlanId
	ifName := in.GetIfName()

	defer func() {
		klog.V(2).InfoS("end VswVlanAdd",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName,
			"reply", reply,
			"error", err)
	}()
	klog.V(2).InfoS("start VswVlanAdd",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName,
		"input", in)

	klog.InfoS("gRPC interface VswVlanAdd accepted",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)

	// validation
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName,
			"error", e.Error())
		klog.Warning(e.Error())
		reply = &proto.VswVlanAddReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    e.Error(),
			}),
		}
		klog.InfoS("gRPC interface VswVlanAdd response sent",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName,
			"result", "ERROR")
		return
	}

	klog.V(2).InfoS("branch: validation successful",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)

	// generate instanse of network controller
	nwController := factory.CreateNetworkController(klog.Background(), in.GetProductInfo())
	if nwController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.VswVlanAddReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
				Message:    eMsg,
			}),
		}
		return
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = nwController.VswVlanAdd(ctx, in)

	klog.InfoS("gRPC interface VswVlanAdd response sent",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName,
		"result", reply.GetResult())
	return
}

func (s *networkServer) VswVlanDelete(ctx context.Context, in *proto.VswVlanDeleteRequest) (reply *proto.VswVlanDeleteReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetHostInfo().GetRemoteHost()
	vlanID := in.VlanId
	ifName := in.GetIfName()

	defer func() {
		klog.V(2).InfoS("end VswVlanDelete",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName,
			"reply", reply,
			"error", err)
	}()
	klog.V(2).InfoS("start VswVlanDelete",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName,
		"input", in)

	klog.InfoS("gRPC interface VswVlanDelete accepted",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)

	// validation
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName,
			"error", e.Error())
		klog.Warning(e.Error())
		reply = &proto.VswVlanDeleteReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    e.Error(),
			}),
		}
		klog.InfoS("gRPC interface VswVlanDelete response sent",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName,
			"result", "ERROR")
		return
	}

	klog.V(2).InfoS("branch: validation successful",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)
	
	// generate instanse of network controller
	nwController := factory.CreateNetworkController(klog.Background(), in.GetProductInfo())
	if nwController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.VswVlanDeleteReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
				Message:    eMsg,
			}),
		}
		return
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = nwController.VswVlanDelete(ctx, in)

	klog.InfoS("gRPC interface VswVlanDelete response sent",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName,
		"result", reply.GetResult())
	return
}

// gRPC server wrapper for testing
var serveWrapper = func(s *grpc.Server, lis net.Listener) error {
	return s.Serve(lis)
}

// start gRPC server
var testListener net.Listener // For test only
var isTest = false

func run(port int) {
	klog.InfoS("Starting gRPC server", "listen port", port)

	// listen on port for receive
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		klog.Error(err.Error())
		if !skipOsExit {
			os.Exit(1) // Ensure exit on grpc server start failure
		}
		return
	}
	// only test case, copy net.Listener on global
	if isTest {
		testListener = lis
	}
	_ = testListener // for lint pass: var testListener is unused (unused)

	// Get configuration
	config := utils.GetConfig()

	var s *grpc.Server

	// Configure gRPC server with or without TLS based on TLS_ENABLE
	if config.TlsEnable {
		// Load TLS certificates
		tlsCertFile := config.TlsCertPath + "/tls.crt"
		tlsKeyFile := config.TlsCertPath + "/tls.key"
		cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
		if err != nil {
			klog.ErrorS(err, "Failed to load TLS certificates",
				"cert_path", tlsCertFile,
				"key_path", tlsKeyFile)
			if !skipOsExit {
				os.Exit(1)
			}
			return
		}

		// Load CA certificate for client authentication
		caCertFile := config.TlsCertPath + "/ca.crt"
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			klog.ErrorS(err, "Failed to load CA certificate", "ca_path", caCertFile)
			if !skipOsExit {
				os.Exit(1)
			}
			return
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			klog.ErrorS(nil, "Failed to append CA certificate to pool")
			if !skipOsExit {
				os.Exit(1)
			}
			return
		}

		// Configure mTLS (require and verify client certificate)
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caCertPool,
		}

		creds := credentials.NewTLS(tlsConfig)
		s = grpc.NewServer(grpc.Creds(creds))
		klog.InfoS("Starting gRPC server with mTLS", "listen port", port)
	} else {
		s = grpc.NewServer()
		klog.InfoS("Starting gRPC server without TLS", "listen port", port)
	}

	// Register server
	proto.RegisterNetworkServer(s, newNetworkServer())

	// Start receiving requests
	klog.InfoS("Listening gRPC server", "listen port", port, "tls_enabled", config.TlsEnable)
	if err := serveWrapper(s, lis); err != nil {
		klog.Error(err.Error())
		os.Exit(1) // Ensure exit on grpc server start failure
		return
	}
}

// init klog for once done
var klogInitOnce sync.Once

func initKlog() {
	klogInitOnce.Do(func() {
		klog.InitFlags(nil)

		// get configuration (should be initialized before calling this function)
		config := utils.GetConfig()

		err := flag.Set("v", config.LogLevel) // setting log-level
		if err != nil {
			return
		}
		flag.Parse()
	})
}

// main process
var skipOsExit bool // For testing purposes

func main() {
	// Initialize and validate environment variables first
	if err := utils.InitializeConfig(); err != nil {
		klog.Error(err.Error())
		if !skipOsExit {
			os.Exit(1) // Ensure exit on validation failure
		}
		return
	}

	// Get configuration
	config := utils.GetConfig()

	// setup klog with parsed log level
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

	klog.InfoS("Start dci_physical_infrastructure process")

	// run gRPC server with parsed port
	run(config.NWServerPort)

	klog.InfoS("End dci_physical_infrastructure process")
}
