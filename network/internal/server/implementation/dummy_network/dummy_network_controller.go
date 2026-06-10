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

package dummy_network

import (
	"context"
	proto "network_module/api/proto" // import for gRPC protobuf
    common "common/api/proto"    // import of common protobuf

	"k8s.io/klog/v2"
)

// struct of dummy network controller
type DummyNetworkController struct {
	Logger klog.Logger
}

func (l DummyNetworkController) VlanAdd(ctx context.Context, in *proto.VlanAddRequest) (reply *proto.VlanAddReply, err error) {
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

	l.Logger.V(2).Info("branch: dummy implementation - no action required",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)

	// dummy; nothing to do
	result := common.ResultCode_SUCCESS
	reply = &proto.VlanAddReply{
		Result:       &result,
		ErrorMessage: "",
	}
	err = nil
	return
}

func (l DummyNetworkController) VlanDelete(ctx context.Context, in *proto.VlanDeleteRequest) (reply *proto.VlanDeleteReply, err error) {
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

	l.Logger.V(2).Info("branch: dummy implementation - no action required",
		"remote_host", remoteHost,
		"interface_name", interfaceName,
		"vlan_id", vlanID)

	// dummy; nothing to do
	result := common.ResultCode_SUCCESS
	reply = &proto.VlanDeleteReply{
		Result:       &result,
		ErrorMessage: "",
	}
	err = nil
	return
}

func (l DummyNetworkController) VswVlanAdd(ctx context.Context, in *proto.VswVlanAddRequest) (reply *proto.VswVlanAddReply, err error) {
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

	l.Logger.V(2).Info("branch: dummy implementation - no action required",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)

	// dummy; nothing to do
	result := common.ResultCode_SUCCESS
	reply = &proto.VswVlanAddReply{
		Result:       &result,
		ErrorMessage: "",
	}
	err = nil
	return
}

func (l DummyNetworkController) VswVlanDelete(ctx context.Context, in *proto.VswVlanDeleteRequest) (reply *proto.VswVlanDeleteReply, err error) {
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

	l.Logger.V(2).Info("branch: dummy implementation - no action required",
		"remote_host", remoteHost,
		"vlan_id", vlanID,
		"if_name", ifName)

	// dummy; nothing to do
	result := common.ResultCode_SUCCESS
	reply = &proto.VswVlanDeleteReply{
		Result:       &result,
		ErrorMessage: "",
	}
	err = nil
	return
}
