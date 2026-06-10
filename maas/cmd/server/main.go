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

	proto "maas_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"

	"maas_module/factory"                    // import of Maas factory
	"maas_module/internal/server/interfaces" // import of interface definitions
	"maas_module/internal/server/utils"      // import of utility functions
)

// struct of gRPC-server for Maas-Service
type maasServer struct {
	proto.UnimplementedMaasServer
	testController interfaces.MaasController // Optional: used only in tests to inject mock controller
}

// generate instanse of gRPC-server for MaaS-service
func newMaasServer() *maasServer {
	return &maasServer{}
}

// generate instanse of gRPC-server for MaaS-service with test controller (for testing only)
func newMaasServerWithController(controller interfaces.MaasController) *maasServer {
	return &maasServer{testController: controller}
}

// MaaS-service
func (s *maasServer) MachineRegister(ctx context.Context, in *proto.MachineRegisterRequest) (reply *proto.MachineRegisterResponse, err error) {
	klog.InfoS("MachineRegister request received")
	klog.V(2).InfoS("start MachineRegister", "in", in)
	defer func() {
		klog.InfoS("MachineRegister response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end MachineRegister", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		// Translate validation error to user-friendly message
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.MachineRegisterResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.MachineRegisterResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = maasController.MachineRegister(ctx, in)
	return
}

func (s *maasServer) MachineDelete(ctx context.Context, in *proto.MachineDeleteRequest) (reply *proto.MachineDeleteResponse, err error) {
	klog.InfoS("MachineDelete request received")
	klog.V(2).InfoS("start MachineDelete", "in", in)
	defer func() {
		klog.InfoS("MachineDelete response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end MachineDelete", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.MachineDeleteResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.MachineDeleteResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")
	
	// process of back-end
	reply, _ = maasController.MachineDelete(ctx, in)
	return
}

func (s *maasServer) OsDeploy(ctx context.Context, in *proto.OsDeployRequest) (reply *proto.OsDeployResponse, err error) {
	klog.InfoS("OsDeploy request received")
	klog.V(2).InfoS("start OsDeploy", "in", in)
	defer func() {
		klog.InfoS("OsDeploy response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end OsDeploy", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.OsDeployResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.OsDeployResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")
	
	// process of back-end
	reply, _ = maasController.OsDeploy(ctx, in)
	return
}

func (s *maasServer) OsRelease(ctx context.Context, in *proto.OsReleaseRequest) (reply *proto.OsReleaseResponse, err error) {
	klog.InfoS("OsRelease request received")
	klog.V(2).InfoS("start OsRelease", "in", in)
	defer func() {
		klog.InfoS("OsRelease response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end OsRelease", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.OsReleaseResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.OsReleaseResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")
	
	// process of back-end
	reply, _ = maasController.OsRelease(ctx, in)
	return
}

func (s *maasServer) VmCompose(ctx context.Context, in *proto.VmComposeRequest) (reply *proto.VmComposeResponse, err error) {
	klog.InfoS("VmCompose request received")
	klog.V(2).InfoS("start VmCompose", "in", in)
	defer func() {
		klog.InfoS("VmCompose response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end VmCompose", "reply", reply, "err", err)
	}()

	var validErr error
	validErr = in.Validate()

	// int32 required check
	if validErr == nil {
		klog.V(2).InfoS("branch: initial validation passed, checking int32 fields")
		if in.CpuCore == nil {
			validErr = errors.New("invalid VmComposeRequest.CpuCore: value is required")
		} else if in.Memory == nil {
			validErr = errors.New("invalid VmComposeRequest.Memory: value is required")
		} else if in.DiskSize == nil {
			validErr = errors.New("invalid VmComposeRequest.DiskSize: value is required")
		}
	} else {
		klog.V(2).InfoS("branch: initial validation failed", "error", validErr.Error())
	}

	if validErr != nil {
		klog.V(2).InfoS("branch: final validation failed", "error", validErr.Error())
		friendlyError := utils.TranslateValidationError(validErr)
		klog.Warning(friendlyError)
		reply = &proto.VmComposeResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.VmComposeResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")
	
	// process of back-end
	reply, _ = maasController.VMCompose(ctx, in)
	return
}

func (s *maasServer) VmDelete(ctx context.Context, in *proto.VmDeleteRequest) (reply *proto.VmDeleteResponse, err error) {
	klog.InfoS("VmDelete request received")
	klog.V(2).InfoS("start VmDelete", "in", in)
	defer func() {
		klog.InfoS("VmDelete response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end VmDelete", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.VmDeleteResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.VmDeleteResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")
	
	// process of back-end
	reply, _ = maasController.VMDelete(ctx, in)
	return
}

func (s *maasServer) MachineList(ctx context.Context, in *proto.MachineListRequest) (reply *proto.MachineListResponse, err error) {
	klog.InfoS("MachineList request received")
	klog.V(2).InfoS("start MachineList", "in", in)
	defer func() {
		klog.InfoS("MachineList response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end MachineList", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.MachineListResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.MachineListResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")
	
	// process of back-end
	reply, _ = maasController.MachineList(ctx, in)
	return
}

func (s *maasServer) MachineShow(ctx context.Context, in *proto.MachineShowRequest) (reply *proto.MachineShowResponse, err error) {
	klog.InfoS("MachineShow request received")
	klog.V(2).InfoS("start MachineShow", "in", in)
	defer func() {
		klog.InfoS("MachineShow response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end MachineShow", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.MachineShowResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.MachineShowResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")
	
	// process of back-end
	reply, _ = maasController.MachineShow(ctx, in)
	return
}

func (s *maasServer) Cancel(ctx context.Context, in *proto.CancelRequest) (reply *proto.CancelResponse, err error) {
	klog.InfoS("Cancel request received")
	klog.V(2).InfoS("start Cancel", "in", in)
	defer func() {
		klog.InfoS("Cancel response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end Cancel", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.CancelResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.CancelResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")
	
	// process of back-end
	reply, _ = maasController.Cancel(ctx, in)
	return
}

func (s *maasServer) PowerOn(ctx context.Context, in *proto.PowerOnRequest) (reply *proto.PowerOnResponse, err error) {
	klog.InfoS("PowerOn request received")
	klog.V(2).InfoS("start PowerOn", "in", in)
	defer func() {
		klog.InfoS("PowerOn response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end PowerOn", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		// Translate validation error to user-friendly message
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.PowerOnResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.PowerOnResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = maasController.PowerON(ctx, in)
	return
}

func (s *maasServer) PowerOff(ctx context.Context, in *proto.PowerOffRequest) (reply *proto.PowerOffResponse, err error) {
	klog.InfoS("PowerOff request received")
	klog.V(2).InfoS("start PowerOff", "in", in)
	defer func() {
		klog.InfoS("PowerOff response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end PowerOff", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		// Translate validation error to user-friendly message
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.PowerOffResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.PowerOffResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = maasController.PowerOFF(ctx, in)
	return
}

func (s *maasServer) NetworkUpdate(ctx context.Context, in *proto.NetworkUpdateRequest) (reply *proto.NetworkUpdateResponse, err error) {
	klog.InfoS("NetworkUpdate request received")
	klog.V(2).InfoS("start NetworkUpdate", "in", in)
	defer func() {
		klog.InfoS("NetworkUpdate response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end NetworkUpdate", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		// Translate validation error to user-friendly message
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.NetworkUpdateResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.NetworkUpdateResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = maasController.NetworkUpdate(ctx, in)
	return
}

func (s *maasServer) KubeadmReset(ctx context.Context, in *proto.KubeadmResetRequest) (reply *proto.KubeadmResetResponse, err error) {
	klog.InfoS("KubeadmReset request received")
	klog.V(2).InfoS("start KubeadmReset", "in", in)
	defer func() {
		klog.InfoS("KubeadmReset response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end KubeadmReset", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		// Translate validation error to user-friendly message
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.KubeadmResetResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.KubeadmResetResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = maasController.KubeadmReset(ctx, in)
	return
}

func (s *maasServer) KubeadmJoin(ctx context.Context, in *proto.KubeadmJoinRequest) (reply *proto.KubeadmJoinResponse, err error) {
	klog.InfoS("KubeadmJoin request received")
	klog.V(2).InfoS("start KubeadmJoin", "in", in)
	defer func() {
		klog.InfoS("KubeadmJoin response sent", "result", reply.GetResult())
		klog.V(2).InfoS("end KubeadmJoin", "reply", reply, "err", err)
	}()

	// validation with custom error messages
	if e := in.Validate(); e != nil {
		klog.V(2).InfoS("branch: validation failed", "error", e.Error())
		// Translate validation error to user-friendly message
		friendlyError := utils.TranslateValidationError(e)
		klog.Warning(friendlyError)
		reply = &proto.KubeadmJoinResponse{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    friendlyError,
			}),
		}
		return
	}

	klog.V(2).InfoS("branch: validation successful")
	
	// generate instanse of maas controller
	var maasController interfaces.MaasController
	
	if s.testController != nil {
		// Use test controller if provided (for testing)
		klog.V(2).InfoS("branch: using test controller")
		maasController = s.testController
	} else {
		// Create real controller (production path)
		maasController = factory.CreateMaasController(klog.Background(), in.GetProductInfo(), in.GetMaasInfo())
		if maasController == nil {
			eMsg := "unsupport product: vendor["+in.GetProductInfo().GetVendor()+"], product_name["+in.GetProductInfo().GetProductName()+"], version["+in.GetProductInfo().GetVersion()+"], os["+in.GetProductInfo().GetOs()+"]"
			klog.Warning(eMsg)
			reply = &proto.KubeadmJoinResponse{
				Result: common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
					ErrorCode:  int32(codes.InvalidArgument),
					DetailCode: int32(proto.DetailCode_UNSUPPORT_PRODUCT),
					Message:    eMsg,
				}),
			}
			return
		}
	}
	klog.V(2).InfoS("branch: create controller instanse successful, processing backend request")

	// process of back-end
	reply, _ = maasController.KubeadmJoin(ctx, in)
	return
}

// gRPC server wrapper for testing
var serveWrapper = func(s *grpc.Server, lis net.Listener) error {
	return s.Serve(lis)
}

// start gRPC server
var testListener net.Listener // For test only
var isTest = false
var skipOsExit = false // For testing: skip os.Exit calls
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
	proto.RegisterMaasServer(s, newMaasServer())

	// Start receiving requests
	klog.InfoS("Listening gRPC server", "listen port", port, "tls_enabled", config.TlsEnable)
	if err := serveWrapper(s, lis); err != nil {
		klog.Error(err.Error())
		if !skipOsExit {
			os.Exit(1) // Ensure exit on grpc server start failure
		}
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

	config := utils.GetConfig()

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

	klog.InfoS("Start dci_physical_infrastructure process", "port", config.ServerPort)

	// run gRPC server
	run(config.ServerPort)

	klog.InfoS("End dci_physical_infrastructure process")
}
