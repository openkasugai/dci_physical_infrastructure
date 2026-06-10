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
	"flag"
	"fmt"
	"net"
	"os"
	"sync"

	proto "cdi_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"

	"cdi_module/factory"                    // import of CDI factory
	"cdi_module/internal/server/utils"      // import of CDI utils
)

// struct of gRPC-server for CDI-Service
type cdiServer struct {
	proto.UnimplementedCdiServer
}

// generate instance of gRPC-server for CDI-service
func newCDIServer() *cdiServer {
	return &cdiServer{}
}

// CDI-service
func (s *cdiServer) MachineCreate(ctx context.Context, in *proto.MachineCreateRequest) (reply *proto.MachineCreateReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	machineName := in.GetMachineName()
	groupName := in.GetGroupName()

	klog.V(2).InfoS("start MachineCreate", "in", in)
	defer func() {
		klog.V(2).InfoS("end MachineCreate", "reply", reply, "err", err)
	}()

	klog.InfoS("gRPC interface accepted MachineCreate",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName)
	defer func() {
		klog.InfoS("gRPC interface response MachineCreate",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName,
			"result", reply.GetResult())
	}()

	// validation
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		klog.Warning(e.Error())
		reply = &proto.MachineCreateReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    e.Error(),
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")

	// generate instanse of network controller
	cdiController := factory.CreateCDIController(klog.Background(), in.GetProductInfo())
	if cdiController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.MachineCreateReply{
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
	reply, _ = cdiController.MachineCreate(ctx, in)
	return
}

func (s *cdiServer) MachineDestroy(ctx context.Context, in *proto.MachineDestroyRequest) (reply *proto.MachineDestroyReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	machineName := in.GetMachineName()
	groupName := in.GetGroupName()

	klog.V(2).InfoS("start MachineDestroy", "in", in)
	defer func() {
		klog.V(2).InfoS("end MachineDestroy", "reply", reply, "err", err)
	}()

	klog.InfoS("gRPC interface accepted MachineDestroy",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName)
	defer func() {
		klog.InfoS("gRPC interface response MachineDestroy",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName,
			"result", reply.GetResult())
	}()

	// validation
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		klog.Warning(e.Error())
		reply = &proto.MachineDestroyReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    e.Error(),
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")

	// generate instanse of network controller
	cdiController := factory.CreateCDIController(klog.Background(), in.GetProductInfo())
	if cdiController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.MachineDestroyReply{
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
	reply, _ = cdiController.MachineDestroy(ctx, in)
	return
}

func (s *cdiServer) MachineShow(ctx context.Context, in *proto.MachineShowRequest) (reply *proto.MachineShowReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	machineName := in.GetMachineName()
	groupName := in.GetGroupName()

	klog.V(2).InfoS("start MachineShow", "in", in)
	defer func() {
		klog.V(2).InfoS("end MachineShow", "reply", reply, "err", err)
	}()

	klog.InfoS("gRPC interface accepted MachineShow",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName)
	defer func() {
		klog.InfoS("gRPC interface response MachineShow",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName,
			"result", reply.GetResult())
	}()

	// validation
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		klog.Warning(e.Error())
		reply = &proto.MachineShowReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    e.Error(),
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")

	// generate instanse of network controller
	cdiController := factory.CreateCDIController(klog.Background(), in.GetProductInfo())
	if cdiController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.MachineShowReply{
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
	reply, _ = cdiController.MachineShow(ctx, in)
	return
}

func (s *cdiServer) ResourceList(ctx context.Context, in *proto.ResourceListRequest) (reply *proto.ResourceListReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	groupName := in.GetGroupName()

	klog.V(2).InfoS("start ResourceList", "in", in)
	defer func() {
		klog.V(2).InfoS("end ResourceList", "reply", reply, "err", err)
	}()

	klog.InfoS("gRPC interface accepted ResourceList",
		"remote_host", remoteHost,
		"group_name", groupName)
	defer func() {
		klog.InfoS("gRPC interface response ResourceList",
			"remote_host", remoteHost,
			"group_name", groupName,
			"result", reply.GetResult())
	}()

	// validation
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		klog.Warning(e.Error())
		reply = &proto.ResourceListReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    e.Error(),
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")

	// generate instanse of network controller
	cdiController := factory.CreateCDIController(klog.Background(), in.GetProductInfo())
	if cdiController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.ResourceListReply{
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
	reply, _ = cdiController.ResourceList(ctx, in)
	return
}

func (s *cdiServer) ResourceShow(ctx context.Context, in *proto.ResourceShowRequest) (reply *proto.ResourceShowReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	resourceName := in.GetResourceName()

	klog.V(2).InfoS("start ResourceShow", "in", in)
	defer func() {
		klog.V(2).InfoS("end ResourceShow", "reply", reply, "err", err)
	}()

	klog.InfoS("gRPC interface accepted ResourceShow",
		"remote_host", remoteHost,
		"resource_name", resourceName)
	defer func() {
		klog.InfoS("gRPC interface response ResourceShow",
			"remote_host", remoteHost,
			"resource_name", resourceName,
			"result", reply.GetResult())
	}()

	// validation
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		klog.Warning(e.Error())
		reply = &proto.ResourceShowReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    e.Error(),
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")

	// generate instanse of network controller
	cdiController := factory.CreateCDIController(klog.Background(), in.GetProductInfo())
	if cdiController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.ResourceShowReply{
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
	reply, _ = cdiController.ResourceShow(ctx, in)
	return
}

func (s *cdiServer) CardScaling(ctx context.Context, in *proto.CardScalingRequest) (reply *proto.CardScalingReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()

	klog.V(2).InfoS("start CardScaling", "in", in)
	defer func() {
		klog.V(2).InfoS("end CardScaling", "reply", reply, "err", err)
	}()

	klog.InfoS("gRPC interface accepted CardScaling",
		"remote_host", remoteHost)
	defer func() {
		klog.InfoS("gRPC interface response CardScaling",
			"remote_host", remoteHost,
			"result", reply.GetResult())
	}()

	// validation
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		klog.Warning(e.Error())
		reply = &proto.CardScalingReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    e.Error(),
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")

	// generate instanse of network controller
	cdiController := factory.CreateCDIController(klog.Background(), in.GetProductInfo())
	if cdiController == nil {
		eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
		klog.Warning(eMsg)
		reply = &proto.CardScalingReply{
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
	reply, _ = cdiController.CardScaling(ctx, in)
	return
}

// gRPC server wrapper for testing
var serveWrapper = func(s *grpc.Server, lis net.Listener) error {
	return s.Serve(lis)
}

// start gRPC server
var testListener net.Listener = nil // For test only
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

	// Test only, save listener on global
	if isTest {
		testListener = lis
	}

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
	proto.RegisterCdiServer(s, newCDIServer())

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
	// get LOG_LEVEL from env
	level := os.Getenv("LOG_LEVEL")

	klogInitOnce.Do(func() {
		klog.InitFlags(nil)
		err := flag.Set("v", level) // setting log-level
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

	klog.InfoS("Start dci_physical_infrastructure process")

	// Get configuration
	config := utils.GetConfig()

	// run gRPC server with parsed port
	run(config.CDIServerPort)

	klog.InfoS("End dci_physical_infrastructure process")
}
