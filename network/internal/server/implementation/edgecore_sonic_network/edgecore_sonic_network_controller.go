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

package edgecore_sonic_network

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	proto "network_module/api/proto"                                    // import for gRPC protobuf
    common "common/api/proto"    										// import of common protobuf
	"network_module/internal/server/interfaces" 						// import for network ansible interface
	"network_module/internal/server/utils"                              // import for utils
)

// struct of SONiC network controller
type EdgeCoreSonicNetworkController struct {
	Logger  klog.Logger
	Ansible interfaces.NetworkAnsible
	SSHKey  string
}

func (l EdgeCoreSonicNetworkController) VlanAdd(ctx context.Context, in *proto.VlanAddRequest) (reply *proto.VlanAddReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetSwitchInfo().GetRemoteHost()
	interfaceName := in.GetInterfaceName()
	vlanID := in.GetVlanId()

	defer func() {
		l.Logger.V(2).Info("end VlanAdd",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start VlanAdd",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID,
		"input", in)

	// Generating extended options
	extra := fmt.Sprintf("interface=%s vid=%d type=%s",
		in.GetInterfaceName(),
		in.GetVlanId(),
		in.GetVlanType(),
	)

	l.Logger.V(2).Info("branch: executing ansible command",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)

	// Ansible command generation and execution
	_, errors := l.Ansible.CmdExecute(ctx,
		in.GetSwitchInfo().GetRemoteHost(),
		in.GetSwitchInfo().GetRemoteUser(),
		l.SSHKey,
		"add_vlan.yaml",
		extra)
	if errors != nil {
		l.Logger.V(2).Info("branch: ansible command execution failed",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID)
		reply = &proto.VlanAddReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errors),
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)
	reply = &proto.VlanAddReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}

func (l EdgeCoreSonicNetworkController) VlanDelete(ctx context.Context, in *proto.VlanDeleteRequest) (reply *proto.VlanDeleteReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetSwitchInfo().GetRemoteHost()
	interfaceName := in.GetInterfaceName()
	vlanID := in.GetVlanId()

	defer func() {
		l.Logger.V(2).Info("end VlanDelete",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start VlanDelete",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID,
		"input", in)

	// Generating extended options
	extra := fmt.Sprintf("interface=%s vid=%d",
		in.GetInterfaceName(),
		in.GetVlanId(),
	)

	l.Logger.V(2).Info("branch: executing ansible command",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)

	// Ansible command generation and execution
	_, errors := l.Ansible.CmdExecute(ctx,
		in.GetSwitchInfo().GetRemoteHost(),
		in.GetSwitchInfo().GetRemoteUser(),
		l.SSHKey,
		"delete_vlan.yaml",
		extra)
	if errors != nil {
		l.Logger.V(2).Info("branch: ansible command execution failed",
			"remote_host", remoteHost,
			"interface_name", interfaceName,
			"vlan_id", vlanID)
		reply = &proto.VlanDeleteReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errors),
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)
	reply = &proto.VlanDeleteReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}

func (l EdgeCoreSonicNetworkController) VswVlanAdd(ctx context.Context, in *proto.VswVlanAddRequest) (reply *proto.VswVlanAddReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetHostInfo().GetRemoteHost()
	vlanID := int32(0)
	ifName := in.GetIfName()
	if in.VlanId != nil {
		vlanID = *in.VlanId
	}

	defer func() {
		l.Logger.V(2).Info("end VswVlanAdd",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start VswVlanAdd",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName,
		"input", in)

	// Generating extended options
	extra := ""
	if in.VlanId == nil {
		l.Logger.V(2).Info("branch: vlan_id is none",
			"remote_host", remoteHost,
			"if_name", ifName)
		extra = fmt.Sprintf("vid=none if_name=%s",
			in.GetIfName(),
		)
	} else {
		l.Logger.V(2).Info("branch: vlan_id is specified",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName)
		extra = fmt.Sprintf("vid=%d if_name=%s",
			in.GetVlanId(),
			in.GetIfName(),
		)
	}

	l.Logger.V(2).Info("branch: executing ansible command",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)

	// Ansible command generation and execution
	_, errors := l.Ansible.CmdExecute(ctx,
		in.GetHostInfo().GetRemoteHost(),
		in.GetHostInfo().GetRemoteUser(),
		l.SSHKey,
		"add_vsw_vlan.yaml",
		extra)
	if errors != nil {
		l.Logger.V(2).Info("branch: ansible command execution failed",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName)
		if (errors.DetailCode == int32(proto.DetailCode_NW_COMMAND_ERROR)) {
			errors.DetailCode = int32(proto.DetailCode_VSW_VLAN_DUPLICATE)
		}
		reply = &proto.VswVlanAddReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errors),
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)
	reply = &proto.VswVlanAddReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}

func (l EdgeCoreSonicNetworkController) VswVlanDelete(ctx context.Context, in *proto.VswVlanDeleteRequest) (reply *proto.VswVlanDeleteReply, err error) {
	// Extract identification parameters
	remoteHost := in.GetHostInfo().GetRemoteHost()
	vlanID := int32(0)
	ifName := in.GetIfName()
	if in.VlanId != nil {
		vlanID = *in.VlanId
	}

	defer func() {
		l.Logger.V(2).Info("end VswVlanDelete",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName,
			"reply", reply,
			"error", err)
	}()
	l.Logger.V(2).Info("start VswVlanDelete",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName,
		"input", in)

	// Generating extended options
	extra := ""
	if in.VlanId == nil {
		l.Logger.V(2).Info("branch: vlan_id is none",
			"remote_host", remoteHost,
			"if_name", ifName)
		extra = fmt.Sprintf("vid=none if_name=%s",
			in.GetIfName(),
		)
	} else {
		l.Logger.V(2).Info("branch: vlan_id is specified",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName)
		extra = fmt.Sprintf("vid=%d if_name=%s",
			in.GetVlanId(),
			in.GetIfName(),
		)
	}

	l.Logger.V(2).Info("branch: executing ansible command",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)

	// Ansible command generation and execution
	_, errors := l.Ansible.CmdExecute(ctx,
		in.GetHostInfo().GetRemoteHost(),
		in.GetHostInfo().GetRemoteUser(),
		l.SSHKey,
		"delete_vsw_vlan.yaml",
		extra)
	if errors != nil {
		l.Logger.V(2).Info("branch: ansible command execution failed",
			"remote_host", remoteHost,
			"vlan_id", vlanID,
			"if_name", ifName)
		if (errors.DetailCode == int32(proto.DetailCode_NW_COMMAND_ERROR)) {
			errors.DetailCode = int32(proto.DetailCode_VSW_VLAN_NOTFOUND)
		}
		reply = &proto.VswVlanDeleteReply{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(errors),
		}
		return
	}

	l.Logger.V(2).Info("branch: ansible command execution successful",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)
	reply = &proto.VswVlanDeleteReply{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}
