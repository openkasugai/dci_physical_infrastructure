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

package pg_cdi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
	"google.golang.org/grpc/codes"

	proto "cdi_module/api/proto"                // import for gRPC protobuf
    common "common/api/proto"    				 // import of common protobuf
	"common/models/extra_parameters"
	"cdi_module/internal/server/interfaces"     // import for PRIMAGY-CDI ansible interface
	"cdi_module/internal/server/utils"
)

var resultSuccess = common.ResultCode_SUCCESS
var resultAccept = common.ResultCode_ACCEPT
var resultError = common.ResultCode_ERROR

// Global variables for model mappings
var (
	modelMappings     map[string]string
	modelMappingsOnce sync.Once
)

// struct of PRIMAGY-CDI controller
type PgCDIController struct {
	Logger  klog.Logger
	Ansible interfaces.CDIAnsible
	SSHKey  string
}

func (l PgCDIController) MachineCreate(ctx context.Context, in *proto.MachineCreateRequest) (reply *proto.MachineCreateReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	machineName := in.GetMachineName()
	groupName := in.GetGroupName()
	resourceName := strings.Join(in.GetResourceList(), ",")

	defer func() {
		l.Logger.V(2).Info("end MachineCreate",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName,
			"resource_name", resourceName,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start MachineCreate",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName,
		"resource_name", resourceName,
		"input", in)

	//  To ensure idempotency, check if the machine already exists
	showReply, err := l.MachineShow(ctx, &proto.MachineShowRequest{
		ProductInfo:    in.GetProductInfo(),
		CdiInfo:        in.GetCdiInfo(),
		MachineName:    in.GetMachineName(),
		GroupName:      in.GetGroupName(),
		ExtraParameter: in.ExtraParameter,
	})
	// if exists (SUCCESS result), return success
	if err == nil && showReply.GetResult() == common.ResultCode_SUCCESS {
		l.Logger.V(2).Info("branch: machine already exists, returning success",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName)
		reply = &proto.MachineCreateReply{
			Result:       &resultAccept,
			ErrorMessage: "",
		}
		return
	}
	
	// parse extra parameter
	extraParameter, err := ParseExtraParameter(in.GetExtraParameter())
	if err != nil {
		l.Logger.V(2).Info("branch: validation failed", "error", err.Error())
		klog.Warning(err.Error())
		reply = &proto.MachineCreateReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    err.Error(),
			}),
		}
		return
	}

	// generate extra option
	extraVarsCreate, _ := json.Marshal(map[string]string{
		"cdi_user":      extraParameter.CDIUser,
		"cdi_password":  extraParameter.CDIPassword,
		"cdi_guest":     extraParameter.CDIGuest,
		"group_name":    in.GetGroupName(),
		"machine_name":  in.GetMachineName(),
		"resource_enum": strings.Join(in.GetResourceList(), ","),
	})
	extra := string(extraVarsCreate)

	// generate ansible command and execute
	errMsg, _ := l.Ansible.CmdExecute(ctx, in.GetCdiInfo().GetRemoteHost(), in.GetCdiInfo().GetRemoteUser(), l.SSHKey, "machine_create.yaml", extra)
	if errMsg != nil {
		reply = &proto.MachineCreateReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errMsg),
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName,
		"resource_name", resourceName)
	reply = &proto.MachineCreateReply{
		Result:       &resultAccept,
		ErrorMessage: "",
	}
	err = nil
	return
}

func (l PgCDIController) MachineDestroy(ctx context.Context, in *proto.MachineDestroyRequest) (reply *proto.MachineDestroyReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	machineName := in.GetMachineName()
	groupName := in.GetGroupName()

	defer func() {
		l.Logger.V(2).Info("end MachineDestroy",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start MachineDestroy",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName,
		"input", in)
	
	// parse extra parameter
	extraParameter, err := ParseExtraParameter(in.GetExtraParameter())
	if err != nil {
		l.Logger.V(2).Info("branch: validation failed", "error", err.Error())
		klog.Warning(err.Error())
		reply = &proto.MachineDestroyReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    err.Error(),
			}),
		}
		return
	}

	// Power off the machine before destruction
	errMsg := l.powerOffMachine(ctx, in.GetCdiInfo(), in.GetMachineName(), in.GetGroupName(), extraParameter)
	if errMsg != nil {
		reply = &proto.MachineDestroyReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errMsg),
		}
		return
	}
	// Wait until the machine is powered off
	status := l.pollMachineStatus(ctx, in.GetProductInfo(), in.GetCdiInfo(), in.GetMachineName(), in.GetGroupName(), in.GetExtraParameter(), 
									[]string{"ERROR", "INACTIVE POFF"}, 1*time.Second, 2*time.Minute)

	l.Logger.V(2).Info("branch: power off done",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName,
		"status", status)

	if (status == "ERROR") {	// in case of error status, force power off
		l.Logger.V(2).Info("branch: machine is in ERROR status, force power off",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName)

		// generate extra option
		extraVarsPowerOff, _ := json.Marshal(map[string]string{
			"cdi_user":     extraParameter.CDIUser,
			"cdi_password": extraParameter.CDIPassword,
			"cdi_guest":    extraParameter.CDIGuest,
			"group_name":   in.GetGroupName(),
			"machine_name": in.GetMachineName(),
			"status":       "POWER_OFF",
		})
		extra := string(extraVarsPowerOff)

		// generate ansible command and execute
		errMsg, _ = l.Ansible.CmdExecute(ctx, in.GetCdiInfo().GetRemoteHost(), in.GetCdiInfo().GetRemoteUser(), l.SSHKey, "machine_update_status.yaml", extra)
		if errMsg != nil {
			reply = &proto.MachineDestroyReply{
				Result:       common.ResultCode_ERROR.Enum(),
				ErrorMessage: utils.ErrorMessageToJSON(errMsg),
			}
			return
		}

		// Wait until the machine is powered off
		status := l.pollMachineStatus(ctx, in.GetProductInfo(), in.GetCdiInfo(), in.GetMachineName(), in.GetGroupName(), in.GetExtraParameter(),
										[]string{"INACTIVE POFF"}, 1*time.Second, 2*time.Minute)

		l.Logger.V(2).Info("branch: force power off done",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName,
			"status", status)
	}

	// generate extra option
	extraVarsDestroy, _ := json.Marshal(map[string]string{
		"cdi_user":     extraParameter.CDIUser,
		"cdi_password": extraParameter.CDIPassword,
		"cdi_guest":    extraParameter.CDIGuest,
		"group_name":   in.GetGroupName(),
		"machine_name": in.GetMachineName(),
	})
	extra := string(extraVarsDestroy)

	// generate ansible command and execute
	errMsg, _ = l.Ansible.CmdExecute(ctx, in.GetCdiInfo().GetRemoteHost(), in.GetCdiInfo().GetRemoteUser(), l.SSHKey, "machine_destroy.yaml", extra)
	if errMsg != nil {
		reply = &proto.MachineDestroyReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errMsg),
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName)
	reply = &proto.MachineDestroyReply{
		Result:       &resultAccept,
		ErrorMessage: "",
	}
	err = nil
	return
}

func (l PgCDIController) MachineShow(ctx context.Context, in *proto.MachineShowRequest) (reply *proto.MachineShowReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	machineName := in.GetMachineName()
	groupName := in.GetGroupName()

	defer func() {
		l.Logger.V(2).Info("end MachineShow",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start MachineShow",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName,
		"input", in)
	
	// parse extra parameter
	extraParameter, err := ParseExtraParameter(in.GetExtraParameter())
	if err != nil {
		l.Logger.V(2).Info("branch: validation failed", "error", err.Error())
		klog.Warning(err.Error())
		reply = &proto.MachineShowReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    err.Error(),
			}),
		}
		return
	}

	// generate extra option
	extraVarsShow, _ := json.Marshal(map[string]string{
		"cdi_user":     extraParameter.CDIUser,
		"cdi_password": extraParameter.CDIPassword,
		"cdi_guest":    extraParameter.CDIGuest,
		"group_name":   in.GetGroupName(),
		"machine_name": in.GetMachineName(),
	})
	extra := string(extraVarsShow)

	// generate ansible command and execute
	errMsg, data := l.Ansible.CmdExecute(ctx, in.GetCdiInfo().GetRemoteHost(), in.GetCdiInfo().GetRemoteUser(), l.SSHKey, "machine_show.yaml", extra)
	if errMsg != nil {
		reply = &proto.MachineShowReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errMsg),
		}
		return
	}

	// extract necessary data
	var extractData map[string]interface{}
	var machines []interface{}
	var ok bool
	if extractData, ok = data["data"].(map[string]interface{}); ok {
		machines, ok = extractData["machines"].([]interface{})
	}
	if !ok {
		err = errors.New("ansible output is unexpected")
		// error process
		reply = &proto.MachineShowReply{
			Result:       &resultError,
			ErrorMessage: err.Error(),
		}
		l.Logger.Error(err, "Ansible cmd is failed")
		return
	}

	// extract data object and create json string
	jsonData, err := json.Marshal(machines[0])
	if err != nil {
		l.Logger.Error(err, "Ansible output is unexpected")
		reply = &proto.MachineShowReply{
			Result:       &resultError,
			ErrorMessage: err.Error(),
		}
		return
	}
	jsonString := string(jsonData)

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName)
	reply = &proto.MachineShowReply{
		Result:       &resultSuccess,
		ErrorMessage: "",
		Data:         jsonString,
	}
	err = nil
	return
}

func (l PgCDIController) ResourceList(ctx context.Context, in *proto.ResourceListRequest) (reply *proto.ResourceListReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	groupName := in.GetGroupName()

	defer func() {
		l.Logger.V(2).Info("end ResourceList",
			"remote_host", remoteHost,
			"group_name", groupName,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start ResourceList",
		"remote_host", remoteHost,
		"group_name", groupName,
		"input", in)
	
	// parse extra parameter
	extraParameter, err := ParseExtraParameter(in.GetExtraParameter())
	if err != nil {
		l.Logger.V(2).Info("branch: validation failed", "error", err.Error())
		klog.Warning(err.Error())
		reply = &proto.ResourceListReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    err.Error(),
			}),
		}
		return
	}

	// generate extra option
	extraVarsList, _ := json.Marshal(map[string]string{
		"cdi_user":     extraParameter.CDIUser,
		"cdi_password": extraParameter.CDIPassword,
		"cdi_guest":    extraParameter.CDIGuest,
		"group_name":   in.GetGroupName(),
	})
	extra := string(extraVarsList)

	// generate ansible command and execute
	errMsg, data := l.Ansible.CmdExecute(ctx, in.GetCdiInfo().GetRemoteHost(), in.GetCdiInfo().GetRemoteUser(), l.SSHKey, "resource_list.yaml", extra)
	if errMsg != nil {
		reply = &proto.ResourceListReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errMsg),
		}
		return
	}

	// extract data object and create json string
	jsonData, err := json.Marshal(data["data"])
	if err != nil {
		l.Logger.Error(err, "Ansible output is unexpected")
		reply = &proto.ResourceListReply{
			Result:       &resultError,
			ErrorMessage: err.Error(),
		}
		return
	}
	jsonString := string(jsonData)

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"group_name", groupName)
	reply = &proto.ResourceListReply{
		Result:       &resultSuccess,
		ErrorMessage: "",
		Data:         jsonString,
	}
	err = nil
	return
}

func (l PgCDIController) ResourceShow(ctx context.Context, in *proto.ResourceShowRequest) (reply *proto.ResourceShowReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	resourceName := in.GetResourceName()

	defer func() {
		l.Logger.V(2).Info("end ResourceShow",
			"remote_host", remoteHost,
			"resource_name", resourceName,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start ResourceShow",
		"remote_host", remoteHost,
		"resource_name", resourceName,
		"input", in)
	
	// parse extra parameter
	extraParameter, err := ParseExtraParameter(in.GetExtraParameter())
	if err != nil {
		l.Logger.V(2).Info("branch: validation failed", "error", err.Error())
		klog.Warning(err.Error())
		reply = &proto.ResourceShowReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    err.Error(),
			}),
		}
		return
	}

	// generate extra option
	extraVarsRShow, _ := json.Marshal(map[string]string{
		"cdi_user":      extraParameter.CDIUser,
		"cdi_password":  extraParameter.CDIPassword,
		"cdi_guest":     extraParameter.CDIGuest,
		"resource_name": in.GetResourceName(),
	})
	extra := string(extraVarsRShow)

	// generate ansible command and execute
	errMsg, data := l.Ansible.CmdExecute(ctx, in.GetCdiInfo().GetRemoteHost(), in.GetCdiInfo().GetRemoteUser(), l.SSHKey, "resource_show.yaml", extra)
	if errMsg != nil {
		reply = &proto.ResourceShowReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errMsg),
		}
		return
	}

	// Transform resspec_model values using model mappings
	l.Logger.V(2).Info("branch: transforming resource models", "resource_name", resourceName)
	transformedData := transformResourceModels(data, l.Logger)

	// extract data object and create json string
	jsonData, err := json.Marshal(transformedData)
	if err != nil {
		l.Logger.Error(err, "Ansible output is unexpected")
		reply = &proto.ResourceShowReply{
			Result:       &resultError,
			ErrorMessage: err.Error(),
		}
		return
	}
	jsonString := string(jsonData)

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"resource_name", resourceName)
	reply = &proto.ResourceShowReply{
		Result:       &resultSuccess,
		ErrorMessage: "",
		Data:         jsonString,
	}
	err = nil
	return
}

func (l PgCDIController) CardScaling(ctx context.Context, in *proto.CardScalingRequest) (reply *proto.CardScalingReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetCdiInfo().GetRemoteHost()
	machineName := in.GetMachineName()
	groupName := in.GetGroupName()

	defer func() {
		l.Logger.V(2).Info("end CardScaling",
			"remote_host", remoteHost,
			"machine_name", machineName,
			"group_name", groupName,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start CardScaling",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName,
		"input", in)
	
	// parse extra parameter
	extraParameter, err := ParseExtraParameter(in.GetExtraParameter())
	if err != nil {
		l.Logger.V(2).Info("branch: validation failed", "error", err.Error())
		klog.Warning(err.Error())
		reply = &proto.CardScalingReply{
			Result: common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(&common.ErrorMessage{
				ErrorCode:  int32(codes.InvalidArgument),
				DetailCode: int32(proto.DetailCode_IF_PARAMETER_INVALID),
				Message:    err.Error(),
			}),
		}
		return
	}

	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		l.Logger.V(2).Info("goroutine context status",
			"contextStatus", fmt.Sprintf("asyncCtx.Err=%v", asyncCtx.Err()))

		//  To ensure idempotency, check if the machine already exists
		currentResources, err := l.getMachineResources(asyncCtx, in.GetProductInfo(), in.GetCdiInfo(), in.GetMachineName(), in.GetGroupName(), in.GetExtraParameter())
		if err != nil {
			l.Logger.Error(err, "Ansible cmd is failed")
			return
		}

		// Create resource set for quick lookup
        resourceSet := make(map[string]bool)
        for _, res := range currentResources {
            resourceSet[res] = true
        }

        // Build add and remove lists
        var resourcesToAdd []string
        var resourcesToRemove []string

		// loop for each resource modification request
		for _, item := range in.GetResourceModifyRequests() {
			resource := item.GetResourceName()
			operation := item.GetOp()

            if operation == "add" {
                if resourceSet[resource] {
                    l.Logger.V(2).Info("branch: resource already exists, skipping addition",
                        "remote_host", remoteHost,
                        "machine_name", machineName,
                        "group_name", groupName,
                        "resource", resource)
                    continue
                }
                resourcesToAdd = append(resourcesToAdd, resource)
            } else if operation == "remove" {
                if !resourceSet[resource] {
                    l.Logger.V(2).Info("branch: resource does not exist, skipping removal",
                        "remote_host", remoteHost,
                        "machine_name", machineName,
                        "group_name", groupName,
                        "resource", resource)
                    continue
                }
                resourcesToRemove = append(resourcesToRemove, resource)
            }
		}

        // Execute add operation if there are resources to add
        if len(resourcesToAdd) > 0 {
            l.Logger.V(2).Info("branch: executing add operation",
                "remote_host", remoteHost,
                "machine_name", machineName,
                "group_name", groupName,
                "resources", resourcesToAdd)

            extraVarsAdd, _ := json.Marshal(map[string]string{
                "cdi_user":      extraParameter.CDIUser,
                "cdi_password":  extraParameter.CDIPassword,
                "cdi_guest":     extraParameter.CDIGuest,
                "group_name":    in.GetGroupName(),
                "machine_name":  in.GetMachineName(),
                "resource_enum": strings.Join(resourcesToAdd, ","),
                "operation":     "add",
            })
            extra := string(extraVarsAdd)

            errMsg, _ := l.Ansible.CmdExecute(asyncCtx, in.GetCdiInfo().GetRemoteHost(), in.GetCdiInfo().GetRemoteUser(), l.SSHKey, "machine_modify.yaml", extra)
            if errMsg != nil {
                l.Logger.Error(errors.New(utils.ErrorMessageToJSON(errMsg)), "Ansible cmd is failed")
                return
            }

            // machine modify is fire-and-forget; poll until machine reaches stable state
            status := l.pollMachineStatus(asyncCtx, in.GetProductInfo(), in.GetCdiInfo(), in.GetMachineName(), in.GetGroupName(), in.GetExtraParameter(),
                []string{"INACTIVE POFF", "ERROR"}, 1*time.Second, 2*time.Minute)
            l.Logger.V(2).Info("branch: polling after add operation done",
                "remote_host", remoteHost,
                "machine_name", machineName,
                "group_name", groupName,
                "status", status)
            if status != "INACTIVE POFF" {
                l.Logger.Error(errors.New("machine did not reach expected status after add operation"),
                    "polling failed",
                    "remote_host", remoteHost,
                    "machine_name", machineName,
                    "status", status)
                return
            }
        }

        // Execute remove operation if there are resources to remove
        if len(resourcesToRemove) > 0 {
            l.Logger.V(2).Info("branch: executing remove operation",
                "remote_host", remoteHost,
                "machine_name", machineName,
                "group_name", groupName,
                "resources", resourcesToRemove)

            extraVarsRemove, _ := json.Marshal(map[string]string{
                "cdi_user":      extraParameter.CDIUser,
                "cdi_password":  extraParameter.CDIPassword,
                "cdi_guest":     extraParameter.CDIGuest,
                "group_name":    in.GetGroupName(),
                "machine_name":  in.GetMachineName(),
                "resource_enum": strings.Join(resourcesToRemove, ","),
                "operation":     "remove",
            })
            extra := string(extraVarsRemove)

            errMsg, _ := l.Ansible.CmdExecute(asyncCtx, in.GetCdiInfo().GetRemoteHost(), in.GetCdiInfo().GetRemoteUser(), l.SSHKey, "machine_modify.yaml", extra)
            if errMsg != nil {
                l.Logger.Error(errors.New(utils.ErrorMessageToJSON(errMsg)), "Ansible cmd is failed")
                return
            }

            // machine modify is fire-and-forget; poll until machine reaches stable state
            status := l.pollMachineStatus(asyncCtx, in.GetProductInfo(), in.GetCdiInfo(), in.GetMachineName(), in.GetGroupName(), in.GetExtraParameter(),
                []string{"INACTIVE POFF", "ERROR"}, 1*time.Second, 2*time.Minute)
            l.Logger.V(2).Info("branch: polling after remove operation done",
                "remote_host", remoteHost,
                "machine_name", machineName,
                "group_name", groupName,
                "status", status)
            if status != "INACTIVE POFF" {
                l.Logger.Error(errors.New("machine did not reach expected status after remove operation"),
                    "polling failed",
                    "remote_host", remoteHost,
                    "machine_name", machineName,
                    "status", status)
                return
            }
        }

        l.Logger.V(2).Info("branch: card scaling completed successfully",
            "remote_host", remoteHost,
            "machine_name", machineName,
            "group_name", groupName,
            "added_resources", resourcesToAdd,
            "removed_resources", resourcesToRemove)
	}()

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"machine_name", machineName,
		"group_name", groupName)
	reply = &proto.CardScalingReply{
		Result:       &resultAccept,
		ErrorMessage: "",
	}
	err = nil
	return
}

// private method

// Helper function: Power off machine
func (l PgCDIController) powerOffMachine(ctx context.Context, cdiInfo *proto.CdiInformation, machineName string, groupName string, extraParameter *extra_parameters.PgCDIExtraParameters) (errMsg *common.ErrorMessage) {
	defer func() {
		l.Logger.V(2).Info("end powerOffMachine",
			"errMsg", errMsg)
	}()
	l.Logger.V(2).Info("start powerOffMachine",
		"cdiInfo", cdiInfo,
		"machineName", machineName,
		"groupName", groupName,
		"extraParameter", extraParameter)

	// Generate extra option
	extraVarsPower, _ := json.Marshal(map[string]string{
		"cdi_user":     extraParameter.CDIUser,
		"cdi_password": extraParameter.CDIPassword,
		"cdi_guest":    extraParameter.CDIGuest,
		"machine_name": machineName,
		"power":        "off",
		"group_name":   groupName,
	})
	extra := string(extraVarsPower)

	// Execute power off command
	errMsg, _ = l.Ansible.CmdExecute(ctx,
		cdiInfo.GetRemoteHost(),
		cdiInfo.GetRemoteUser(),
		l.SSHKey,
		"machine_power.yaml",
		extra)
	if errMsg != nil {
		return
	}

	errMsg = nil
	return
}

// Helper function: Get machine status
func (l PgCDIController) getMachineStatus(ctx context.Context, 
	productInfo *proto.ProductInformation, cdiInfo *proto.CdiInformation, 
	machineName string, groupName string, extraParameter string) (status string, err error) {
	defer func() {
		l.Logger.V(2).Info("end getMachineStatus",
			"status", status,
			"err", err)
	}()
	l.Logger.V(2).Info("start getMachineStatus",
		"productInfo", productInfo,
		"cdiInfo", cdiInfo,
		"groupName", groupName,
		"machineName", machineName,
		"extraParameter", extraParameter)

	// Call MachineShow to get current status
	showRequest := &proto.MachineShowRequest{
		ProductInfo: productInfo,
		CdiInfo:     cdiInfo,
		MachineName: machineName,
		GroupName:   groupName,
		ExtraParameter: &extraParameter,
	}
	reply, err := l.MachineShow(ctx, showRequest)
	if reply.GetResult() != common.ResultCode_SUCCESS {
		err = errors.New(reply.GetErrorMessage())
		return
	}

	// Parse JSON data to extract status
	var machineData map[string]interface{}
	err = json.Unmarshal([]byte(reply.GetData()), &machineData)
	if err != nil {
		l.Logger.Error(err, err.Error())
		return
	}
	status, ok := machineData["mach_status_detail"].(string)
	if !ok {
		err = errors.New("ansible output is unexpected")
		l.Logger.Error(err, err.Error())
		return
	}

	err = nil
	return
}

// Helper function: Poll machine status
func (l PgCDIController) pollMachineStatus(ctx context.Context, 
	productInfo *proto.ProductInformation, cdiInfo *proto.CdiInformation, 
	machineName string, groupName string, extraParameter string, 
	targetStatus []string, interval, timeout time.Duration) (matchStatus string) {
	l.Logger.V(2).Info("start pollMachineStatus",
		"productInfo", productInfo,
		"cdiInfo", cdiInfo,
		"groupName", groupName,
		"machineName", machineName,
		"extraParameter", extraParameter,
		"targetStatus", targetStatus,
		"interval", interval,
		"timeout", timeout)

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer func() {
		cancel()
		l.Logger.V(2).Info("end pollMachineStatus")
	}()

	// Polling loop
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Timeout reached
			l.Logger.V(2).Info("branch: failed to get machine status during polling")
			return
		case <-ticker.C:
			status, err := l.getMachineStatus(ctx, productInfo, cdiInfo, machineName, groupName, extraParameter)
			if err != nil {
				l.Logger.V(2).Info("branch: failed to get machine status during polling", "error", err)
				continue
			}

			l.Logger.V(2).Info("branch: polling machine status",
				"current_status", status,
				"target_status", targetStatus)

			for _, item := range targetStatus {
				if strings.Contains(status, item) {
					matchStatus = item
					l.Logger.V(2).Info("branch: target status reached",
						"status", status)
					return
				}
			}
		}
	}
}

// Helper function: Get machine resources
func (l PgCDIController) getMachineResources(ctx context.Context, 
	productInfo *proto.ProductInformation, cdiInfo *proto.CdiInformation, 
	machineName string, groupName string, extraParameter string) (resources []string, err error) {
	defer func() {
		l.Logger.V(2).Info("end getMachineResources",
			"resources", resources,
			"err", err)
	}()
	l.Logger.V(2).Info("start getMachineResources",
		"productInfo", productInfo,
		"cdiInfo", cdiInfo,
		"groupName", groupName,
		"machineName", machineName,
		"extraParameter", extraParameter)

	// Call MachineShow to get current status
	showRequest := &proto.MachineShowRequest{
		ProductInfo: productInfo,
		CdiInfo:     cdiInfo,
		MachineName: machineName,
		GroupName:   groupName,
		ExtraParameter: &extraParameter,
	}
	reply, err := l.MachineShow(ctx, showRequest)
	if reply.GetResult() != common.ResultCode_SUCCESS {
		err = errors.New(reply.GetErrorMessage())
		return
	}

	// Parse JSON data to extract status
	var machineData map[string]interface{}
	err = json.Unmarshal([]byte(reply.GetData()), &machineData)
	if err != nil {
		l.Logger.Error(err, err.Error())
		return
	}
	resources, err = extractResourceNames(machineData["resources"])
	if err != nil {
		l.Logger.Error(err, err.Error())
		return
	}

	err = nil
	return
}

// extractResourceNames extracts res_name list from machine data
func extractResourceNames(resources interface{}) ([]string, error) {

    resourceList, ok := resources.([]interface{})
    if !ok {
        return nil, errors.New("ansible output is unexpected: resources is not an array")
    }

    resourceNames := make([]string, 0, len(resourceList))
    for _, res := range resourceList {
        resMap, ok := res.(map[string]interface{})
        if !ok {
            return nil, errors.New("ansible output is unexpected: resource item is not a map")
        }
        
        resName, ok := resMap["res_name"].(string)
        if !ok {
            return nil, errors.New("ansible output is unexpected: res_name is not a string")
        }
        resourceNames = append(resourceNames, resName)
    }

    return resourceNames, nil
}

// loadModelMappings loads PG-CDI model to hardware name mappings from environment variable
func loadModelMappings() map[string]string {
	modelMappingsOnce.Do(func() {
		jsonData := os.Getenv("PG_CDI_MODEL_MAPPINGS")
		if jsonData == "" {
			modelMappings = make(map[string]string)
			return
		}
		
		var mappings map[string]string
		if err := json.Unmarshal([]byte(jsonData), &mappings); err != nil {
			modelMappings = make(map[string]string)
			return
		}
		modelMappings = mappings
	})
	return modelMappings
}

// transformResourceModels transforms resspec_model values in resource data using model mappings
func transformResourceModels(data map[string]interface{}, logger klog.Logger) map[string]interface{} {
	mappings := loadModelMappings()
	if len(mappings) == 0 {
		logger.V(2).Info("branch: no model mappings available, skipping transformation")
		return data
	}

	// Navigate to data.resspecs array (data is already the inner data object from CmdExecute result)
	resspecs, ok := data["resspecs"].([]interface{})
	if !ok {
		logger.V(2).Info("branch: data.resspecs not found or not an array")
		return data
	}

	// Transform each resspec_model
	for i, resspec := range resspecs {
		resspecMap, ok := resspec.(map[string]interface{})
		if !ok {
			logger.V(2).Info("branch: resspec item is not a map", "index", i)
			continue
		}

		currentModel, ok := resspecMap["resspec_model"].(string)
		if !ok {
			logger.V(2).Info("branch: resspec_model not found or not a string", "index", i)
			continue
		}

		// Look up mapping (exact match, case-sensitive)
		if hardwareName, found := mappings[currentModel]; found {
			logger.V(2).Info("branch: model mapping found", 
				"index", i, 
				"original_model", currentModel, 
				"hardware_name", hardwareName)
			resspecMap["resspec_model"] = hardwareName
		} else {
			logger.V(2).Info("branch: model mapping not found, keeping original value", 
				"index", i, 
				"model", currentModel)
		}
	}

	return data
}
