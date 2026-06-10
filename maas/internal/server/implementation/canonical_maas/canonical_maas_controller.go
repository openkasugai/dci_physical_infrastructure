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

package canonical_maas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"

	proto "maas_module/api/proto" // import for gRPC protobuf
    common "common/api/proto"    // import of common protobuf
	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/interfaces" // import of MaaS interface
	"maas_module/internal/server/interfaces/maas_api"
	"maas_module/internal/server/utils" // import of utility functions
)

// SubnetLinkPair struct
type SubnetLinkPair struct {
	linkMode  string
	subnetIds []int
}

// FabricPair struct
type FabricPair struct {
	fabricID int
	vlanID   int
}

// CanonicalMaasController struct
type CanonicalMaasController struct {
	Logger     klog.Logger
	Ansible    interfaces.MaasAnsible
	APIFactory maas_api.MaasAPIFactory
	JobManager *JobManager
}

/**********************************
* private functions
***********************************/
func isIPv4(address string) bool {
	ip := net.ParseIP(address)
	return ip.To4() != nil
}

// convert to int IP address
func ipv4ToInt(ipstr string) (ipint uint32) {
	ip := net.ParseIP(ipstr)
	ipBytes := ip.To4()

	ipint |= uint32(ipBytes[0]) << 24
	ipint |= uint32(ipBytes[1]) << 16
	ipint |= uint32(ipBytes[2]) << 8
	ipint |= uint32(ipBytes[3])

	return ipint
}

// convert to string IP address
func intToIpv4(ipint uint32) (ipstr string) {
	ipBytes := make(net.IP, net.IPv4len)
	ipBytes[0] = byte(ipint >> 24)
	ipBytes[1] = byte(ipint >> 16)
	ipBytes[2] = byte(ipint >> 8)
	ipBytes[3] = byte(ipint)
	return ipBytes.String()
}

// reverse IP address range.
func (l CanonicalMaasController) reverseIPAddressRange(cidr string, addStart string, addEnd string) (ipRanges [][]string) {
	klog.V(2).InfoS("start reverseIPAddressRange", "cidr", cidr, "addStart", addStart, "addEnd", addEnd)
	defer func() {
		klog.V(2).InfoS("end reverseIPAddressRange", "ipRanges", ipRanges)
	}()

	// extract the subnet mask part
	parts := strings.Split(cidr, "/")

	// convert to int
	subnetMask, _ := strconv.Atoi(parts[1])

	// calculate mask value
	var mask uint32
	for i := 0; i < subnetMask; i++ {
		mask |= 1 << (31 - i)
	}

	// Calculate the first part when inverting the IP address range.
	if addStart != "" {
		addStartInt := ipv4ToInt(addStart)
		if 0x00000001 < (addStartInt & (^mask)) {
			revStart := intToIpv4((addStartInt & mask) + 1)
			revEnd := intToIpv4(addStartInt - 1)
			row := []string{revStart, revEnd}
			ipRanges = append(ipRanges, row)
		}
	}

	// Calculate the second part when inverting the IP address range.
	if addEnd != "" {
		addEndInt := ipv4ToInt(addEnd)
		if (addEndInt | mask) < 0xFFFFFFFE {
			revStart := intToIpv4(addEndInt + 1)
			revEnd := intToIpv4((addEndInt | (^mask)) - 1)
			row := []string{revStart, revEnd}
			ipRanges = append(ipRanges, row)
		}
	}

	return ipRanges
}

// machine register internal
func (l CanonicalMaasController) internalMachineRegister(ctx context.Context, in *proto.MachineRegisterRequest) (systemID string, err error) {
	klog.V(2).InfoS("start internalMachineRegister", "in", in)
	defer func() {
		klog.V(2).InfoS("end internalMachineRegister", "systemID", systemID, "err", err)
	}()

	// API execute
	reqBody := request_body.ReqbodyMachines{
		Architecture: "amd64",
		MACAddresses: in.GetMacAddress(),
		Hostname:     in.GetHostName(),
		Commission:   false,
		EnableSSH:    true,
		PowerType:    "ipmi",
		PowerAddress: in.GetIpmiAddress(),
		PowerUser:    in.GetIpmiUser(),
		PowerPass:    in.GetIpmiPassword(),
	}
	res, err := l.APIFactory.NewMachines().POST(ctx, reqBody)
	if err != nil {
		klog.V(2).InfoS("branch: machine POST API failed", "hostName", in.GetHostName(), "error", err)
		return
	}

	// extract system_id
	var responseBody response_body.ResbodyPostMachines
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyPostMachines); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: response type invalid", "hostName", in.GetHostName(), "error", err)
		return
	}
	systemID = responseBody.SystemID
	err = nil
	return
}

// commission internal
func (l CanonicalMaasController) internalCommission(ctx context.Context, systemID string) (err error) {
	klog.V(2).InfoS("start internalCommission", "systemID", systemID)
	defer func() {
		klog.V(2).InfoS("end internalCommission", "err", err)
	}()

	_, err = l.APIFactory.NewMachineCommission(systemID).POST(ctx, nil)
	if err != nil {
		klog.V(2).InfoS("branch: commission POST API failed", "systemID", systemID, "error", err)
		return
	}

	klog.InfoS("Commission started, polling for completion", "systemID", systemID)

	// status polling
	pollingInterval := 10 * time.Second
	err = l.pollingMachineStatus(ctx, systemID, pollingInterval, []string{"Ready", "Failed commissioning"})
	if err != nil {
		klog.V(2).InfoS("branch: commission polling failed", "systemID", systemID, "error", err)
		return
	}

	err = nil
	return
}

// machine status polling
func (l CanonicalMaasController) pollingMachineStatus(ctx context.Context, systemID string, pollingInterval time.Duration, checkStatus []string) (err error) {
	klog.V(2).InfoS("start pollingMachineStatus", "systemID", systemID, "pollingInterval", pollingInterval, "checkStatus", checkStatus)
	defer func() {
		klog.V(2).InfoS("end pollingMachineStatus", "err", err)
	}()

	var status string
	for {
		// execute machine show
		_, status, _, err = l.internalMachineShow(ctx, &proto.MachineShowRequest{
			SystemId: systemID,
		})
		if err != nil {
			klog.V(2).InfoS("branch: machine show failed during polling", "systemID", systemID, "error", err)
			return
		}

		klog.V(2).InfoS("Polling machine status", "systemID", systemID, "currentStatus", status)

		// status check
		for _, item := range checkStatus {
			if item == status {
				klog.V(2).InfoS("branch: target status reached", "systemID", systemID, "status", status)
				if status == "Failed commissioning" {
					err = &utils.SeqError{Message: "machine commission failed"}
					return
				}
				return
			}
		}

		time.Sleep(pollingInterval)
	}
}

// get Subnet list
func (l CanonicalMaasController) getSubnetList(ctx context.Context) (subnets []response_body.Subnet, err error) {
	klog.V(2).InfoS("start getSubnetList")
	defer func() {
		klog.V(2).InfoS("end getSubnetList", "subnets", subnets, "err", err)
	}()

	// get Subnet list.
	res, err := l.APIFactory.NewSubnets().GET(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: subnets GET API failed", "error", err)
		return
	}

	// extract system_id
	var responseBody response_body.ResbodyGetSubnets
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyGetSubnets); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: response type invalid", "error", err)
		return
	}
	subnets = responseBody.List
	err = nil
	return
}

// find subnet
func (l CanonicalMaasController) findSubnet(subnets []response_body.Subnet, cidr string) (subnetID *int) {
	klog.V(2).InfoS("start findSubnet", "subnets", subnets, "cidr", cidr)
	defer func() {
		klog.V(2).InfoS("end findSubnet", "subnetID", subnetID)
	}()

	// find subnet.
	for i, subnet := range subnets {
		// compare cidr.
		if cidr == subnet.Cidr {
			subnetID = &subnet.ID
			klog.V(2).InfoS("branch: subnet found", "cidr", cidr, "subnetIndex", i, "subnetID", subnet.ID)
			return
		}
	}

	subnetID = nil // Not found case
	klog.V(2).InfoS("branch: subnet not found", "cidr", cidr)
	return
}

// create subnet and ip range
func (l CanonicalMaasController) createSubnetAndIPRange(ctx context.Context, macToFabric map[string]FabricPair, mac string, cidr string, addStart string, addEnd string) (subnetID int, err error) {
	klog.V(2).InfoS("start createSubnetAndIPRange", "macToFabric", macToFabric, "mac", mac, "cidr", cidr, "addStart", addStart, "addEnd", addEnd)
	defer func() {
		klog.V(2).InfoS("end createSubnetAndIPRange", "subnetID", subnetID, "err", err)
	}()

	var fabID int
	var vlanID int
	if fabricPair, result := macToFabric[mac]; !result {
		klog.V(2).InfoS("branch: fabric not found for MAC, creating new fabric", "mac", mac)
		// Create fabric
		var res response_body.Resbody
		res, err = l.APIFactory.NewFabrics().POST(ctx, nil)
		if err != nil {
			klog.V(2).InfoS("branch: fabric creation failed", "mac", mac, "error", err)
			return
		}

		// extract system_id
		var responseBody response_body.ResbodyPostFabrics
		var ok bool
		if responseBody, ok = res.(response_body.ResbodyPostFabrics); !ok {
			err = &utils.RespError{Message: "response type is invalid"}
			klog.V(2).InfoS("branch: fabric creation response type invalid", "mac", mac, "error", err)
			return
		}

		fabID = responseBody.ID
		vlanID = responseBody.Vlans[0].Vid
		macToFabric[mac] = FabricPair{fabricID: fabID, vlanID: vlanID}
		klog.V(2).InfoS("new fabric created", "mac", mac, "fabID", fabID, "vlanID", vlanID)
	} else {
		fabID = fabricPair.fabricID
		vlanID = fabricPair.vlanID
		klog.V(2).InfoS("using existing fabric", "mac", mac, "fabID", fabID, "vlanID", vlanID)
	}

	// create Subnet & IP range.
	reqBody := request_body.ReqbodySubnets{
		Cidr:     cidr,
		FabricID: fabID,
		Vid:      vlanID,
	}
	klog.V(2).InfoS("creating subnet", "cidr", cidr, "fabID", fabID, "vlanID", vlanID)
	res, err := l.APIFactory.NewSubnets().POST(ctx, reqBody)
	if err != nil {
		klog.V(2).InfoS("branch: subnet creation failed", "cidr", cidr, "error", err)
		return
	}

	// find id
	var responseBody response_body.ResbodyPostSubnets
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyPostSubnets); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: subnet creation response type invalid", "cidr", cidr, "error", err)
		return
	}
	subnetID = responseBody.ID
	klog.V(2).InfoS("subnet created successfully", "cidr", cidr, "subnetID", subnetID)

	// create IP range.
	if addStart != "" || addEnd != "" {
		// Call the process to invert the IP address range.
		ipRanges := l.reverseIPAddressRange(cidr, addStart, addEnd)

		// Repeat the process for the number of data entries in ipRanges.
		for i := 0; i < len(ipRanges); i++ {

			// create IP range.
			reqBody := request_body.ReqbodyIPRanges{
				SubnetID: subnetID,
				StartIP:  ipRanges[i][0],
				EndIP:    ipRanges[i][1],
				Type:     "dynamic",
			}
			_, err = l.APIFactory.NewIPRanges().POST(ctx, reqBody)
			if err != nil {
				return
			}
		}
	}

	return
}

// get Interface list
func (l CanonicalMaasController) getInterfaceList(ctx context.Context, systemID string) (interfaces []response_body.Interface, err error) {
	klog.V(2).InfoS("start getInterfaceList", "systemID", systemID)
	defer func() {
		klog.V(2).InfoS("end getInterfaceList", "interfaces", interfaces, "err", err)
	}()

	// get interface list.
	klog.V(2).InfoS("getting interfaces from API", "systemID", systemID)
	res, err := l.APIFactory.NewInterfaces(systemID).GET(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get interfaces API call failed", "systemID", systemID, "error", err)
		return
	}

	var responseBody response_body.ResbodyGetInterfaces
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyGetInterfaces); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: interfaces response type invalid", "systemID", systemID, "error", err)
		return
	}
	interfaces = responseBody.List
	err = nil
	return
}

// linking subnet and interface
// mac2name is an optional map from lower-cased MAC address to desired interface name (e.g. "eth0").
// When non-nil, each matched interface is renamed after linking. Pass nil to skip renaming (e.g. VMCompose).
func (l CanonicalMaasController) linkSubnetInterface(ctx context.Context, systemID string, interfaces []response_body.Interface, key2sub map[string]SubnetLinkPair, keyName string, caseSensitivity bool, mac2name map[string]string) (err error) {
	klog.V(2).InfoS("start linkSubnetInterface", "systemID", systemID, "interfaces", interfaces, "key2sub", key2sub, "keyName", keyName, "caseSensitivity", caseSensitivity)
	defer func() {
		klog.V(2).InfoS("end linkSubnetInterface", "err", err)
	}()

	for i, iface := range interfaces {
		var searchValue string
		if keyName == "MacAddress" {
			searchValue = iface.MacAddress
		} else {
			searchValue = iface.Name
		}

		var subnetPair SubnetLinkPair
		var ok bool
		if !caseSensitivity {
			searchValue = strings.ToLower(searchValue)
		}
		if subnetPair, ok = key2sub[searchValue]; !ok {
			klog.V(2).InfoS("interface not found in subnet mapping", "systemID", systemID, "interface", searchValue)
			continue
		}

		klog.V(2).InfoS("processing interface", "systemID", systemID, "interfaceIndex", i, "interfaceName", iface.Name, "interfaceID", iface.ID, "subnetCount", len(subnetPair.subnetIds))

		// op-disconnect
		klog.V(2).InfoS("disconnecting interface", "systemID", systemID, "interfaceID", iface.ID)
		_, err = l.APIFactory.NewInterfaceDisconnect(systemID, iface.ID).POST(ctx, nil)
		if err != nil {
			klog.V(2).InfoS("branch: interface disconnect failed", "systemID", systemID, "interfaceID", iface.ID, "error", err)
			return
		}

		// link subnet.
		for j, subnetID := range subnetPair.subnetIds {
			reqBody := request_body.ReqbodyIFLinkSubnet{
				Mode:     subnetPair.linkMode,
				SubnetID: subnetID,
			}
			klog.V(2).InfoS("linking interface to subnet", "systemID", systemID, "interfaceID", iface.ID, "subnetIndex", j, "subnetID", subnetID, "mode", subnetPair.linkMode)
			_, err = l.APIFactory.NewInterfaceLink(systemID, iface.ID).POST(ctx, reqBody)
			if err != nil {
				klog.V(2).InfoS("branch: interface link to subnet failed", "systemID", systemID, "interfaceID", iface.ID, "subnetID", subnetID, "error", err)
				return
			}
		}

		// rename interface if mac2name is provided
		if mac2name != nil {
			macLower := strings.ToLower(iface.MacAddress)
			if newName, ok := mac2name[macLower]; ok {
				if iface.Name != newName {
					klog.V(2).InfoS("renaming interface", "systemID", systemID, "interfaceID", iface.ID, "currentName", iface.Name, "newName", newName)
					_, err = l.APIFactory.NewInterfaceUpdate(systemID, iface.ID).PUT(ctx, request_body.ReqbodyInterfaceUpdate{Name: newName})
					if err != nil {
						klog.V(2).InfoS("branch: interface rename failed", "systemID", systemID, "interfaceID", iface.ID, "newName", newName, "error", err)
						return
					}
				} else {
					klog.V(2).InfoS("interface name already matches, skipping rename", "systemID", systemID, "interfaceID", iface.ID, "name", iface.Name)
				}
			}
		}
	}

	err = nil
	klog.V(2).InfoS("linkSubnetInterface completed successfully", "systemID", systemID)
	return
}

// get host list
func (l CanonicalMaasController) getHostList(ctx context.Context) (hosts []response_body.VMHost, err error) {
	klog.V(2).InfoS("start getHostList")
	defer func() {
		klog.V(2).InfoS("end getHostList", "hosts", hosts, "err", err)
	}()

	// get host list.
	klog.V(2).InfoS("getting VM hosts from API")
	res, err := l.APIFactory.NewVMHosts().GET(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get VM hosts API call failed", "error", err)
		return
	}

	// convert host list.
	var responseBody response_body.ResbodyGetVMHosts
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyGetVMHosts); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: VM hosts response type invalid", "error", err)
		return
	}
	hosts = responseBody.List
	err = nil
	klog.V(2).InfoS("getHostList completed successfully", "hostCount", len(hosts))
	return
}

// get host id
func (l CanonicalMaasController) getHostID(ctx context.Context, systemID string) (hostID int, err error) {
	klog.V(2).InfoS("start getHostID", "systemID", systemID)
	defer func() {
		klog.V(2).InfoS("end getHostID", "hostID", hostID, "err", err)
	}()

	// get vmhosts list.
	klog.V(2).InfoS("getting host list", "systemID", systemID)
	hosts, err := l.getHostList(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get host list failed", "systemID", systemID, "error", err)
		return
	}
	if len(hosts) == 0 {
		err = errors.New("no hosts found")
		klog.V(2).InfoS("branch: no hosts found", "systemID", systemID, "error", err)
		return
	}

	// find by system id
	klog.V(2).InfoS("searching for host by system ID", "systemID", systemID, "totalHosts", len(hosts))
	for _, host := range hosts {

		if host.Host.SystemID == systemID {
			hostID = host.ID
			err = nil
			klog.V(2).InfoS("host found", "systemID", systemID, "hostID", hostID)
			return
		}
	}

	err = errors.New("no hosts found")
	klog.V(2).InfoS("branch: host not found for system ID", "systemID", systemID, "error", err)
	return
}

// machine show internal
func (l CanonicalMaasController) internalMachineShow(ctx context.Context, in *proto.MachineShowRequest) (jsonStr string, machineStatus string, description string, err error) {
	klog.V(2).InfoS("start internalMachineShow", "in", in)
	defer func() {
		klog.V(2).InfoS("end internalMachineShow", "jsonStr", jsonStr, "machineStatus", machineStatus, "description", description, "err", err)
	}()

	// API execute
	klog.V(2).InfoS("getting machine details from API", "systemID", in.GetSystemId())
	res, err := l.APIFactory.NewMachineSystemID(in.GetSystemId()).GET(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get machine API call failed", "systemID", in.GetSystemId(), "error", err)
		return
	}
	var responseBody response_body.ResbodyGetMachine
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyGetMachine); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: machine show response type invalid", "systemID", in.GetSystemId(), "error", err)
		return
	}

	// extract required data into returnJson
	klog.V(2).InfoS("marshaling machine data", "systemID", in.GetSystemId())
	jsonDataBytes, err := json.Marshal(responseBody.MachineForResponse)
	if err != nil {
		err = &utils.RespError{Message: err.Error()}
		klog.V(2).InfoS("branch: machine data JSON marshal failed", "systemID", in.GetSystemId(), "error", err)
		return
	}
	jsonStr = string(jsonDataBytes)
	machineStatus = responseBody.StatusName
	description = responseBody.Description

	klog.V(2).InfoS("internalMachineShow completed successfully", "systemID", in.GetSystemId(), "machineStatus", machineStatus, "description", description)
	err = nil
	return
}

// get machine access information
func (l CanonicalMaasController) getMachineAccessInfo(ctx context.Context, systemID string) (
	hostName string, bootIf string, accessAddress string, bootMacAddress string, subnetIDs []int, storage float64, status string, powerStatus string, err error) {
	klog.V(2).InfoS("start getMachineAccessInfo", "systemID", systemID)
	defer func() {
		klog.V(2).InfoS("end getMachineAccessInfo", "hostName", hostName, "bootIf", bootIf, "accessAddress", accessAddress, "bootMacAddress", bootMacAddress, "subnetIDs", subnetIDs, "storage", storage, "status", status, "powerStatus", powerStatus, "err", err)
	}()

	// API execute
	klog.V(2).InfoS("getting machine details for access info", "systemID", systemID)
	res, err := l.APIFactory.NewMachineSystemID(systemID).GET(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get machine API call failed", "systemID", systemID, "error", err)
		return
	}
	var responseBody response_body.ResbodyGetMachine
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyGetMachine); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: machine access info response type invalid", "systemID", systemID, "error", err)
		return
	}

	// get host name
	hostName = responseBody.HostName
	klog.V(2).InfoS("extracted hostname", "systemID", systemID, "hostName", hostName)

	// get ip address
	for _, link := range responseBody.BootInterface.Links {
		ip := link.IPAddress
		if isIPv4(ip) {
			accessAddress = ip
			break
		}
	}
	// If no IP found and boot interface has children (e.g., bridge), check children interfaces
	if accessAddress == "" && len(responseBody.BootInterface.Children) > 0 {
		klog.V(2).InfoS("boot interface has no IP, checking children interfaces", "systemID", systemID, "children", responseBody.BootInterface.Children)
		// Look for child interface in interface_set
		for _, childName := range responseBody.BootInterface.Children {
			for _, iface := range responseBody.InterfaceSet {
				if iface.Name == childName {
					klog.V(2).InfoS("checking child interface", "systemID", systemID, "childName", childName)
					// Found child interface, check its links
					for _, link := range iface.Links {
						ip := link.IPAddress
						if isIPv4(ip) {
							accessAddress = ip
							klog.V(2).InfoS("found IP in child interface", "systemID", systemID, "childName", childName, "ip", ip)  
							break
						}
					}
					if accessAddress != "" {
						break
					}
				}
			}
			if accessAddress != "" {
				break
			}
		}
	}

	// get boot interface name
	bootIf = responseBody.BootInterface.Name
	// If boot interface has children (e.g., bridge), use the child interface name
	if len(responseBody.BootInterface.Children) > 0 {
		bootIf = responseBody.BootInterface.Children[0]
		klog.V(2).InfoS("using child interface as boot interface", "systemID", systemID, "bootIf", bootIf, "parent", responseBody.BootInterface.Name)
	}

	// get boot MAC address
	bootMacAddress = responseBody.BootInterface.MacAddress

	// get boot interface links
	var links []response_body.Link
	for _, ifs := range responseBody.InterfaceSet {
		if bootIf == ifs.Name {
			links = ifs.Links
			break
		}
	}

	// get boot subnetIDs (skip synthetic links with ID=0 created from tags)
	for _, ln := range links {
		cidr := ln.Subnet.Cidr
		parts := strings.Split(cidr, "/")
		if isIPv4(parts[0]) && ln.Subnet.ID > 0 {
			subnetIDs = append(subnetIDs, ln.Subnet.ID)
		}
	}

	// get storage
	storage = responseBody.Storage

	// get status
	status = responseBody.StatusName

	// get power state
	powerStatus = responseBody.PowerStatus

	return
}

// isSSHReachable checks if the SSH port (22) is reachable on the given address.
func (l CanonicalMaasController) isSSHReachable(ctx context.Context, accessAddress string, timeout time.Duration) bool {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(accessAddress, "22"))
	if err != nil {
		klog.V(2).InfoS("branch: SSH reachability check failed", "accessAddress", accessAddress, "error", err)
		return false
	}
	_ = conn.Close()
	return true
}

// isCloudInitDone checks if cloud-init has completed on the given host via ansible.
func (l CanonicalMaasController) isCloudInitDone(ctx context.Context, accessAddress string) bool {
	output, err := l.Ansible.CmdExecute(ctx, accessAddress, "check_cloud_init.yaml", "")
	if err != nil {
		klog.V(2).InfoS("branch: cloud-init status check failed", "accessAddress", accessAddress, "error", err)
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "CLOUD_INIT_STATUS=") {
			parts := strings.SplitN(trimmed, "CLOUD_INIT_STATUS=", 2)
			if len(parts) == 2 {
				return strings.Contains(parts[1], "done")
			}
		}
	}
	return false
}

// getReadyStatus computes the ready_status for a machine following the defined flow:
//   - Ready:    check in-progress jobs → "Processing" or "Ready"
//   - Deployed: check jobs + SSH reachability + cloud-init completion
//   - Broken / Failed*: "Failed"
//   - Other:    "Processing"
func (l CanonicalMaasController) getReadyStatus(ctx context.Context, systemID string, statusName string, powerStatus string, accessAddress string) (readyStatus string) {
	klog.V(2).InfoS("start getReadyStatus", "systemID", systemID, "statusName", statusName, "powerStatus", powerStatus, "accessAddress", accessAddress)
	hasJob := l.JobManager != nil && l.JobManager.HasProcessingJob(systemID)
	klog.V(2).InfoS("job check completed", "systemID", systemID, "hasJob", hasJob)
	defer func() {
		klog.V(2).InfoS("end getReadyStatus", "readyStatus", readyStatus)
	}()

	switch {
	case statusName == "Ready":
		if hasJob {
			readyStatus = "Processing"
			return
		}
		readyStatus = "Ready"
		return
	case statusName == "Deployed":
		// extended check below
	case statusName == "Broken" || strings.HasPrefix(statusName, "Failed"):
		readyStatus = "Failed"
		return
	default:
		readyStatus = "Processing"
		return
	}

	// Extended check for Deployed machines
	// Skip SSH check if machine is powered off
	if powerStatus != "on" {
		klog.V(2).InfoS("branch: machine is not powered on, skipping SSH check", "systemID", systemID, "powerStatus", powerStatus)
		readyStatus = "Processing"
		return
	}
	sshReachable := false
	cloudInitDone := false
	if accessAddress != "" {
		klog.V(2).InfoS("checking SSH reachability and cloud-init status", "systemID", systemID, "accessAddress", accessAddress)
		sshReachable = l.isSSHReachable(ctx, accessAddress, 5*time.Second)
		if sshReachable {
			klog.V(2).InfoS("SSH is reachable, checking cloud-init status", "systemID", systemID)
			cloudInitDone = l.isCloudInitDone(ctx, accessAddress)
		}
	}
	klog.V(2).InfoS("SSH reachability check completed", "systemID", systemID, "sshReachable", sshReachable)
	klog.V(2).InfoS("cloud-init status check completed", "systemID", systemID, "cloudInitDone", cloudInitDone)
	if !hasJob && sshReachable && cloudInitDone {
		readyStatus = "Ready"
		return
	}
	readyStatus = "Processing"
	return
}

// get error message from error type
func (l CanonicalMaasController) getErrorMessage(err error) (errorMessage *common.ErrorMessage) {
	if err == nil {
		return nil
	}

	if e, ok := err.(*utils.SeqError); ok {
		errorMessage = e.ErrorDetail()
	} else if e, ok := err.(*utils.CancelError); ok {
		errorMessage = e.ErrorDetail()
	} else if e, ok := err.(*utils.EnvError); ok {
		errorMessage = e.ErrorDetail()
	} else if e, ok := err.(*utils.HttpError); ok {
		errorMessage = e.ErrorDetail()
	} else if e, ok := err.(*utils.RespError); ok {
		errorMessage = e.ErrorDetail()
	} else {
		errorMessage = &common.ErrorMessage{
			ErrorCode:  int32(codes.Internal),
			DetailCode: int32(proto.DetailCode_MAAS_ENVIRONMENT_ERROR),
			Message:    err.Error(),
		}
	}
	return
}

// mark machine as broken
func (l CanonicalMaasController) markBroken(ctx context.Context, err error, systemID string) {
	l.APIFactory.NewMachineMarkBroken(systemID).POST(ctx, request_body.ReqbodyMachineMarkBroken{
		Comment: err.Error(),
	})
}

/**********************************
* public functions
***********************************/

// MachineRegister is a method to register a machine in the MaaS system.
func (l CanonicalMaasController) MachineRegister(ctx context.Context, in *proto.MachineRegisterRequest) (reply *proto.MachineRegisterResponse, err error) {
	klog.V(2).InfoS("start MachineRegister", "in", in)
	defer func() {
		klog.V(2).InfoS("end MachineRegister", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.MachineRegisterResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	// create machine.
	systemID, err := l.internalMachineRegister(ctx, in)
	if err != nil {
		klog.V(2).InfoS("branch: machine registration failed", "hostName", in.GetHostName(), "error", err)
		errorProcess(err)
		return
	}

	// register job
	if l.JobManager != nil {
		l.JobManager.Register(systemID, JobTypeMachineRegister)
	}

	// goroutine start
	go func() {
		if l.JobManager != nil {
			defer l.JobManager.Deregister(systemID, JobTypeMachineRegister)
		}

		// Create independent context for async processing to avoid request context cancellation
		asyncCtx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		// commission.
		err = l.internalCommission(asyncCtx, systemID)
		if err != nil {
			klog.V(2).InfoS("branch: commission failed", "systemID", systemID, "error", err)
			l.markBroken(asyncCtx, err, systemID)
			return
		}

		// get subnet list
		subnets, err := l.getSubnetList(asyncCtx)
		if err != nil {
			klog.V(2).InfoS("branch: get subnet list failed", "systemID", systemID, "error", err)
			l.markBroken(asyncCtx, err, systemID)
			return
		}

		// keep subnetID and mac address associated
		macToSubnet := make(map[string]SubnetLinkPair)

		// keep fabric_id and mac address associated
		macToFabric := make(map[string]FabricPair)

		// get machine access information
		_, _, _, bootMac, bootSubnets, _, _, _, err := l.getMachineAccessInfo(asyncCtx, systemID)
		if err != nil {
			klog.V(2).InfoS("branch: get machine access info failed", "systemID", systemID, "error", err)
			l.markBroken(asyncCtx, err, systemID)
			return
		}
		macToSubnet[strings.ToLower(bootMac)] = SubnetLinkPair{subnetIds: bootSubnets, linkMode: "AUTO"}

		// loop for network infomations
		mac2name := make(map[string]string)
		seenMacs := make(map[string]bool)
		ethIdx := 0
		for i := 0; i < len(in.GetNetworkInformation()); i++ {
			klog.V(2).InfoS("Processing network information", "systemID", systemID, "index", i, "macAddress", in.GetNetworkInformation()[i].GetMacAddress())

			inCidr := in.GetNetworkInformation()[i].GetCidr()
			inAddrStart := in.GetNetworkInformation()[i].GetAddressStart()
			inAddrEnd := in.GetNetworkInformation()[i].GetAddressEnd()
			macAddr := in.GetNetworkInformation()[i].GetMacAddress()

			// find subnet
			matchID := l.findSubnet(subnets, inCidr)

			var subnetID int
			if matchID == nil {
				klog.V(2).InfoS("branch: subnet not found, creating new subnet", "systemID", systemID, "cidr", inCidr)
				// create subnet and ip range
				id, err := l.createSubnetAndIPRange(asyncCtx, macToFabric, macAddr, inCidr, inAddrStart, inAddrEnd)
				if err != nil {
					klog.V(2).InfoS("branch: create subnet failed", "systemID", systemID, "cidr", inCidr, "error", err)
					l.markBroken(asyncCtx, err, systemID)
					return
				}
				subnetID = id
				subnets = append(subnets, response_body.Subnet {
					Cidr: inCidr,
					ID: id,
				})
			} else {
				klog.V(2).InfoS("branch: subnet found", "systemID", systemID, "cidr", inCidr, "subnetID", *matchID)
				subnetID = *matchID
			}

			// save subnetID and mac address associated
			existingPair := macToSubnet[strings.ToLower(macAddr)]
			isDuplicate := false
			for _, existingSubnetID := range existingPair.subnetIds {
				if existingSubnetID == subnetID {
					isDuplicate = true
					klog.V(2).InfoS("subnet already linked to MAC, skipping duplicate", 
						"macAddress", macAddr, "subnetID", subnetID)
					break
				}
			}
			if !isDuplicate {
				macToSubnet[strings.ToLower(macAddr)] = SubnetLinkPair{
					subnetIds: append(existingPair.subnetIds, subnetID), 
					linkMode: "AUTO",
				}
			}

			// build mac2name map: NetworkInformation order -> eth{i}
			macLower := strings.ToLower(macAddr)
			if !seenMacs[macLower] {
				mac2name[macLower] = fmt.Sprintf("eth%d", ethIdx)
				seenMacs[macLower] = true
				ethIdx++
			}
		}

		// get interface list.
		interfaces, err := l.getInterfaceList(asyncCtx, systemID)
		if err != nil {
			klog.V(2).InfoS("branch: get interface list failed", "systemID", systemID, "error", err)
			l.markBroken(asyncCtx, err, systemID)
			return
		}

		// link interface to subnet (and rename per mac2name)
		err = l.linkSubnetInterface(asyncCtx, systemID, interfaces, macToSubnet, "MacAddress", false, mac2name)
		if err != nil {
			klog.V(2).InfoS("branch: link subnet interface failed", "systemID", systemID, "error", err)
			l.markBroken(asyncCtx, err, systemID)
			return
		}
	}()

	reply = &proto.MachineRegisterResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
		SystemId:     systemID,
	}
	err = nil
	return
}

// MachineDelete is a method to delete a machine from the MaaS system.
func (l CanonicalMaasController) MachineDelete(ctx context.Context, in *proto.MachineDeleteRequest) (reply *proto.MachineDeleteResponse, err error) {
	klog.V(2).InfoS("start MachineDelete", "in", in)
	defer func() {
		klog.V(2).InfoS("end MachineDelete", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.MachineDeleteResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	klog.V(2).InfoS("deleting machine via API", "systemID", in.GetSystemId())
	_, err = l.APIFactory.NewMachineSystemID(in.GetSystemId()).DELETE(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: machine deletion failed", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}
	reply = &proto.MachineDeleteResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}

	return
}

// OsDeploy is a method to deploy an operating system on a machine in the MaaS system.
func (l CanonicalMaasController) OsDeploy(ctx context.Context, in *proto.OsDeployRequest) (reply *proto.OsDeployResponse, err error) {
	klog.V(2).InfoS("start OsDeploy", "in", in)
	defer func() {
		klog.V(2).InfoS("end OsDeploy", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.OsDeployResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	reqbody := request_body.ReqbodyMachineDeploy{
		BridgeAll:    in.VmFlag.Value,
		Distribution: in.GetOs().GetDistribution(),
		Version:      in.GetOs().GetVersion(),
		UserData:     in.GetUserData(),
	}
	klog.V(2).InfoS("starting machine deployment", "systemID", in.GetSystemId(), "distribution", reqbody.Distribution, "version", reqbody.Version)
	_, err = l.APIFactory.NewMachineDeploy(in.GetSystemId()).POST(ctx, reqbody)
	if err != nil {
		klog.V(2).InfoS("branch: machine deployment failed", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}

	reply = &proto.OsDeployResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}

	return
}

// OsRelease is a method to release an operating system from a machine in the MaaS system.
func (l CanonicalMaasController) OsRelease(ctx context.Context, in *proto.OsReleaseRequest) (reply *proto.OsReleaseResponse, err error) {
	klog.V(2).InfoS("start OsRelease", "in", in)
	defer func() {
		klog.V(2).InfoS("end OsRelease", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.OsReleaseResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	// find vm-host
	klog.V(2).InfoS("checking for VM host", "systemID", in.GetSystemId())
	hostID, err := l.getHostID(ctx, in.GetSystemId())
	if err == nil { // if exits, delete vm-host
		klog.V(2).InfoS("deleting VM host", "systemID", in.GetSystemId(), "hostID", hostID)
		_, err = l.APIFactory.NewVMHostHostID(hostID).DELETE(ctx)
		if err != nil {
			klog.V(2).InfoS("branch: VM host deletion failed", "systemID", in.GetSystemId(), "hostID", hostID, "error", err)
			errorProcess(err)
			return
		}
		klog.V(2).InfoS("VM host deleted successfully", "systemID", in.GetSystemId(), "hostID", hostID)
	} else {
		klog.V(2).InfoS("no VM host found for machine", "systemID", in.GetSystemId())
	}

	// get machine access information
	klog.V(2).InfoS("getting machine access information", "systemID", in.GetSystemId())
	_, _, accessAddress, _, _, _, status, powerStatus, err := l.getMachineAccessInfo(ctx, in.SystemId)
	if err != nil {
		klog.V(2).InfoS("branch: get machine access info failed", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}

	// Unregister subscription if machine is deployed and powered on
	if status == "Deployed" && powerStatus == "on" {
		klog.V(2).InfoS("unregistering subscription", "systemID", in.GetSystemId(), "accessAddress", accessAddress, "status", status, "powerStatus", powerStatus)
		_, err = l.Ansible.CmdExecute(ctx, accessAddress, "unregister_subscription.yaml", "")
		if err != nil {
			klog.V(2).InfoS("branch: unregister subscription failed", "systemID", in.GetSystemId(), "error", err)
			errorProcess(err)
			return
		}
	} else {
		klog.V(2).InfoS("skip unregister_subscription: not Deployed or not powered on", "systemID", in.GetSystemId(), "status", status, "powerStatus", powerStatus)
	}

	klog.V(2).InfoS("releasing machine", "systemID", in.GetSystemId())
	reqBody := request_body.ReqbodyMachineRelease{
		Erase: true,
		QuickErase: true,
		SecureErase: true,
	}
	_, err = l.APIFactory.NewMachineRelease(in.GetSystemId()).POST(ctx, reqBody)
	if err != nil {
		klog.V(2).InfoS("branch: machine release failed", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}

	// Release IPs tracked by interface tags after machine release
	klog.V(2).InfoS("checking interfaces for IP tags to release", "systemID", in.GetSystemId())
	interfacesRes, err := l.APIFactory.NewInterfaces(in.GetSystemId()).GET(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get interfaces failed", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}

	interfaces, ok := interfacesRes.(response_body.ResbodyGetInterfaces)
	if !ok {
		err = errors.New("invalid response type for interfaces")
		klog.V(2).InfoS("branch: interfaces response type invalid", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}

	// Iterate through all interfaces: release IPs tracked in tags, then remove those tags
	for _, iface := range interfaces.List {
		for _, tag := range iface.IPWithPrefixTags() {
			ip := strings.SplitN(tag, "/", 2)[0]

			// Release the IP address in MAAS
			klog.V(2).InfoS("releasing IP from tag", "systemID", in.GetSystemId(), "interfaceID", iface.ID, "ip", ip)
			_, err = l.APIFactory.NewIPAddressRelease().POST(ctx, request_body.ReqbodyIPAddressRelease{
				IP:    ip,
				Force: true,
			})
			if err != nil {
				klog.V(2).InfoS("branch: IP address release from tag failed", "systemID", in.GetSystemId(), "interfaceID", iface.ID, "ip", ip, "error", err)
				errorProcess(err)
				return
			}
			klog.V(2).InfoS("IP released successfully from tag", "systemID", in.GetSystemId(), "interfaceID", iface.ID, "ip", ip)

			// Remove the IP/prefix tag from the interface to prevent stale tags on re-use
			klog.V(2).InfoS("removing IP tag from interface", "systemID", in.GetSystemId(), "interfaceID", iface.ID, "tag", tag)
			_, err = l.APIFactory.NewInterfaceRemoveTag(in.GetSystemId(), iface.ID).POST(ctx, request_body.ReqbodyInterfaceTag{
				Tag: tag,
			})
			if err != nil {
				klog.V(2).InfoS("branch: interface remove tag failed", "systemID", in.GetSystemId(), "interfaceID", iface.ID, "tag", tag, "error", err)
				errorProcess(err)
				return
			}
			klog.V(2).InfoS("IP tag removed successfully from interface", "systemID", in.GetSystemId(), "interfaceID", iface.ID, "tag", tag)
		}
	}

	reply = &proto.OsReleaseResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}

	return
}

// VMCompose is a method to compose a virtual machine in the MaaS system.
func (l CanonicalMaasController) VMCompose(ctx context.Context, in *proto.VmComposeRequest) (reply *proto.VmComposeResponse, err error) {
	klog.V(2).InfoS("start VMCompose", "in", in)
	defer func() {
		klog.V(2).InfoS("end VMCompose", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.VmComposeResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	// get machine access information
	hostName, _, accessAddress, _, _, storage, _, _, err := l.getMachineAccessInfo(ctx, in.SystemId)
	if err != nil {
		klog.V(2).InfoS("branch: get machine access info failed", "hostSystemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}

	// culculate vm disk size: storage[MB] * diskPercent / 100 / 1024 = storage[GiB]
	config := utils.GetConfig()
	diskPercent := config.VmHostDisk
	vmDiskSizeGiB := int(storage * float64(diskPercent) / 100.0 / 1024.0)

	// get vmhosts list.
	hostID, err := l.getHostID(ctx, in.GetSystemId())
	if err != nil { // if not exist, register vmhosts
		klog.V(2).InfoS("branch: VM host not found, creating new VM host", "hostSystemID", in.GetSystemId())

		lxdPort := os.Getenv("LXD_PORT")

		// setup vm host
		extra := fmt.Sprintf("host_disk_size=%d host_name=%s maas_url=%s maas_api_key=%s lxd_port=%s",
			vmDiskSizeGiB, hostName, strings.ReplaceAll(in.GetMaasInfo().GetAccessUrl(), "/api/2.0/", ""),
			in.GetMaasInfo().GetApiKey(), lxdPort,
		)
		_, err = l.Ansible.CmdExecute(ctx, accessAddress, "setup_lxd.yaml", extra)
		if err != nil {
			klog.V(2).InfoS("branch: LXD setup failed", "hostSystemID", in.GetSystemId(), "error", err)
			errorProcess(err)
			return
		}

		// register vm host
		reqBody := request_body.ReqbodyVMhosts{
			PowerAddress: fmt.Sprintf("%s:%s", accessAddress, lxdPort),
			Type:         "lxd",
		}
		var res request_body.Reqbody
		res, err = l.APIFactory.NewVMHosts().POST(ctx, reqBody)
		if err != nil {
			klog.V(2).InfoS("branch: VM host registration failed", "hostSystemID", in.GetSystemId(), "error", err)
			errorProcess(err)
			return
		}

		var responseBody response_body.ResbodyPostVMHost
		var ok bool
		if responseBody, ok = res.(response_body.ResbodyPostVMHost); !ok {
			err = &utils.RespError{Message: "response type is invalid"}
			klog.V(2).InfoS("branch: response type invalid", "hostSystemID", in.GetSystemId(), "error", err)
			errorProcess(err)
			return
		}
		hostID = responseBody.ID

		// get certificate key
		res, err = l.APIFactory.NewVMHostParameters(hostID).POST(ctx, nil)
		if err != nil {
			klog.V(2).InfoS("branch: get VM host parameters failed", "hostID", hostID, "error", err)
			errorProcess(err)
			return
		}
		var responseBodyParams response_body.ResbodyGetOpParameter
		if responseBodyParams, ok = res.(response_body.ResbodyGetOpParameter); !ok {
			err = &utils.RespError{Message: "response type is invalid"}
			klog.V(2).InfoS("branch: parameter response type invalid", "hostID", hostID, "error", err)
			errorProcess(err)
			return
		}

		// register certificate
		extra = fmt.Sprintf("certificate='%s'", responseBodyParams.Certificate)
		_, err = l.Ansible.CmdExecute(ctx, accessAddress, "register_lxd_certificate.yaml", extra)
		if err != nil {
			klog.V(2).InfoS("branch: certificate registration failed", "hostID", hostID, "error", err)
			errorProcess(err)
			return
		}
	} else {
		klog.V(2).InfoS("branch: VM host found", "hostSystemID", in.GetSystemId(), "hostID", hostID)
	}
	klog.V(2).InfoS("VM host setup completed", "hostSystemID", in.GetSystemId(), "hostID", hostID)

	// refresh vm host
	_, err = l.APIFactory.NewVMHostRefresh(hostID).POST(ctx, nil)
	if err != nil {
		klog.V(2).InfoS("branch: VM host refresh failed", "hostID", hostID, "error", err)
		errorProcess(err)
		return
	}
	time.Sleep(5 * time.Second) // delay for waiting complete

	// get subnet list
	subnets, err := l.getSubnetList(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get subnet list failed", "hostName", in.GetHostName(), "error", err)
		errorProcess(err)
		return
	}
	klog.V(2).InfoS("Retrieved subnet list for VM compose", "hostName", in.GetHostName(), "subnetCount", len(subnets))

	// keep subnetID and if name associated
	nameToSubnet := make(map[string]SubnetLinkPair)

	// loop for network infomations
	var interfaces []string
	for i := 0; i < len(in.GetNetworkInformation()); i++ {
		inName := in.GetNetworkInformation()[i].GetIfName()
		inBridge := in.GetNetworkInformation()[i].GetBridgeName()
		inCidr := in.GetNetworkInformation()[i].GetCidr()
		inAddrStart := in.GetNetworkInformation()[i].GetAddressStart()
		inAddrEnd := in.GetNetworkInformation()[i].GetAddressEnd()

		// find subnet
		matchID := l.findSubnet(subnets, inCidr)
		var subnetID int
		if matchID == nil {

			klog.V(2).InfoS("branch: subnet not found, creating new subnet", "hostName", in.GetHostName(), "ifName", inName)
			// create subnet and ip range
			var id int
			id, err = l.createSubnetAndIPRange(ctx, map[string]FabricPair{}, "", inCidr, inAddrStart, inAddrEnd)
			if err != nil {
				klog.V(2).InfoS("branch: create subnet failed", "hostName", in.GetHostName(), "ifName", inName, "cidr", inCidr, "error", err)
				errorProcess(err)
				return
			}
			subnetID = id
			subnets = append(subnets, response_body.Subnet {
				Cidr: inCidr,
				ID: id,
			})
		} else {
			subnetID = *matchID
			klog.V(2).InfoS("branch: subnet found for interface", "hostName", in.GetHostName(), "ifName", inName, "cidr", inCidr, "subnetID", subnetID)
		}

		// save subnetID and if name associated
		nameToSubnet[inName] = SubnetLinkPair{subnetIds: append(nameToSubnet[inName].subnetIds, subnetID), linkMode: "AUTO"}

		ifName := url.QueryEscape(inName)
		bridgeName := url.QueryEscape(inBridge)
		interfaces = append(interfaces, fmt.Sprintf("%s:name=%s", ifName, bridgeName))
	}
	// sort by interface name for work around maas's bug
	sort.Strings(interfaces)

	// compose vmhosts
	reqBody := request_body.ReqbodyVMhostCompose{
		HostName:   in.GetHostName(),
		Cores:      int(in.GetCpuCore()),
		Memory:     int(in.GetMemory()),
		Storage:    int(in.GetDiskSize()),
		Interfaces: strings.Join(interfaces, ";"),
	}
	klog.V(2).InfoS("starting VM compose", "hostName", in.GetHostName(), "hostID", hostID, "cores", reqBody.Cores, "memory", reqBody.Memory, "storage", reqBody.Storage)
	res, err := l.APIFactory.NewVMHostCompose(hostID).POST(ctx, reqBody)
	if err != nil {
		klog.V(2).InfoS("branch: VM compose failed", "hostName", in.GetHostName(), "hostID", hostID, "error", err)
		errorProcess(err)
		return
	}

	// extract system_id
	var responseBody response_body.ResbodyPostVMCompose
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyPostVMCompose); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: VM compose response type invalid", "hostName", in.GetHostName(), "error", err)
		errorProcess(err)
		return
	}
	systemID := responseBody.SystemID

	// register job
	if l.JobManager != nil {
		l.JobManager.Register(systemID, JobTypeVMCompose)
	}

	// goroutine start
	klog.V(2).InfoS("starting VM compose async processing", "hostName", in.GetHostName(), "systemID", systemID)
	go func() {
		if l.JobManager != nil {
			defer l.JobManager.Deregister(systemID, JobTypeVMCompose)
		}

		// Create independent context for async processing to avoid request context cancellation
		asyncCtx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		// status polling
		pollingInterval := 10 * time.Second
		klog.V(2).InfoS("starting machine status polling", "systemID", systemID, "interval", pollingInterval)
		e := l.pollingMachineStatus(asyncCtx, systemID, pollingInterval, []string{"Ready", "Failed commissioning"})
		if e != nil {
			klog.V(2).InfoS("branch: machine status polling failed", "systemID", systemID, "error", e)
			l.markBroken(asyncCtx, e, systemID)
			return
		}

		// get interface list.
		klog.V(2).InfoS("getting interface list", "systemID", systemID)
		interfaceList, e := l.getInterfaceList(asyncCtx, systemID)
		if e != nil {
			klog.V(2).InfoS("branch: get interface list failed", "systemID", systemID, "error", e)
			l.markBroken(asyncCtx, e, systemID)
			return
		}

		// link interface to subnet (rename not required for VMCompose)
		klog.V(2).InfoS("linking interfaces to subnets", "systemID", systemID)
		e = l.linkSubnetInterface(asyncCtx, systemID, interfaceList, nameToSubnet, "Name", true, nil)
		if e != nil {
			klog.V(2).InfoS("branch: link subnet interface failed", "systemID", systemID, "error", e)
			l.markBroken(asyncCtx, e, systemID)
			return
		}
	}()

	reply = &proto.VmComposeResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
		SystemId:     systemID,
	}
	return
}

// VMDelete is a method to delete a virtual machine in the MaaS system.
func (l CanonicalMaasController) VMDelete(ctx context.Context, in *proto.VmDeleteRequest) (reply *proto.VmDeleteResponse, err error) {
	klog.V(2).InfoS("start VMDelete", "in", in)
	defer func() {
		klog.V(2).InfoS("end VMDelete", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.VmDeleteResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	klog.V(2).InfoS("deleting machine", "systemID", in.GetSystemId())
	_, err = l.APIFactory.NewMachineSystemID(in.GetSystemId()).DELETE(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: machine deletion failed", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}
	reply = &proto.VmDeleteResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}

	return
}

// MachineList is a method to list all machines in the MaaS system.
func (l CanonicalMaasController) MachineList(ctx context.Context, in *proto.MachineListRequest) (reply *proto.MachineListResponse, err error) {
	klog.V(2).InfoS("start MachineList", "in", in)
	defer func() {
		klog.V(2).InfoS("end MachineList", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.MachineListResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	klog.V(2).InfoS("getting machine list from MAAS API")
	res, err := l.APIFactory.NewMachines().GET(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get machines API call failed", "error", err)
		errorProcess(err)
		return
	}

	var responseBody response_body.ResbodyGetMachines
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyGetMachines); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: machine list response type invalid", "error", err)
		errorProcess(err)
		return
	}

	// create a variable for the return value
	returnJson   := make(map[string]interface{})

	// compute ready_status for each machine
	for i := range responseBody.Machines {
		machine := &responseBody.Machines[i]
		machine.ReadyStatus = l.getReadyStatus(ctx, machine.SystemID, machine.StatusName, machine.PowerStatus, machine.AccessAddress)
	}

	returnJson["machines"] = responseBody.Machines

	// extract required data into returnJson
	klog.V(2).InfoS("marshaling machine list data", "machineCount", len(responseBody.Machines))
	jsonDataBytes, err := json.Marshal(returnJson)
	if err != nil {
		err = &utils.RespError{Message: err.Error()}
		klog.V(2).InfoS("branch: machine list JSON marshal failed", "error", err)
		errorProcess(err)
		return
	}
	jsonData := string(jsonDataBytes)

	// succsess
	klog.InfoS("Machine list operation completed successfully", "machineCount", len(responseBody.Machines))
	reply = &proto.MachineListResponse{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
		Data:         jsonData,
	}

	return
}

// MachineShow is a method to show the details of a machine in the MaaS system.
func (l CanonicalMaasController) MachineShow(ctx context.Context, in *proto.MachineShowRequest) (reply *proto.MachineShowResponse, err error) {
	klog.V(2).InfoS("start MachineShow", "in", in)
	defer func() {
		klog.V(2).InfoS("end MachineShow", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.MachineShowResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	klog.V(2).InfoS("getting machine details", "systemID", in.GetSystemId())
	res, err := l.APIFactory.NewMachineSystemID(in.GetSystemId()).GET(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get machine API call failed", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}
	var responseBody response_body.ResbodyGetMachine
	var ok bool
	if responseBody, ok = res.(response_body.ResbodyGetMachine); !ok {
		err = &utils.RespError{Message: "response type is invalid"}
		klog.V(2).InfoS("branch: machine show response type invalid", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}

	// compute access address and ready_status
	responseBody.MachineForResponse.ReadyStatus = l.getReadyStatus(ctx, in.GetSystemId(), responseBody.StatusName, responseBody.PowerStatus, responseBody.AccessAddress)

	jsonDataBytes, err := json.Marshal(responseBody.MachineForResponse)
	if err != nil {
		err = &utils.RespError{Message: err.Error()}
		klog.V(2).InfoS("branch: machine data JSON marshal failed", "systemID", in.GetSystemId(), "error", err)
		errorProcess(err)
		return
	}

	klog.V(2).InfoS("MachineShow completed successfully", "systemID", in.GetSystemId())
	reply = &proto.MachineShowResponse{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
		Data:         string(jsonDataBytes),
	}

	return
}

// Cancel is a method to cancel a process in the MaaS system.
func (l CanonicalMaasController) Cancel(ctx context.Context, in *proto.CancelRequest) (reply *proto.CancelResponse, err error) {
	klog.V(2).InfoS("start Cancel", "in", in)
	defer func() {
		klog.V(2).InfoS("end Cancel", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.CancelResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	systemID := in.GetSystemId()

	// Get current machine state before attempting abort
	klog.V(2).InfoS("checking machine state before cancellation", "systemID", systemID)
	_, machineStatus, _, err := l.internalMachineShow(ctx, &proto.MachineShowRequest{SystemId: systemID}) // check if machine exists
	if err != nil {
		errorProcess(err)
		return
	}

	// Validate machine state - only allow abort for commissioning or deploying states
	allowedStates := []string{"Commissioning", "Deploying", "Testing"}
	isAbortable := false
	for _, allowedState := range allowedStates {
		if machineStatus == allowedState {
			isAbortable = true
			break
		}
	}
	if !isAbortable {
		err = &utils.CancelError{}
		l.Logger.Error(err, err.Error())
		errorProcess(err)
		return
	}

	// Execute abort operation
	_, err = l.APIFactory.NewMachineAbort(systemID).POST(ctx, nil)
	if err != nil {
		errorProcess(err)
		return
	}

	reply = &proto.CancelResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}

// PowerON is a method to PowerON a process in the MaaS system.
func (l CanonicalMaasController) PowerON(ctx context.Context, in *proto.PowerOnRequest) (reply *proto.PowerOnResponse, err error) {
	klog.V(2).InfoS("start PowerOn", "in", in)
	defer func() {
		klog.V(2).InfoS("end PowerOn", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.PowerOnResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	systemID := in.GetSystemId()
	reqBody := request_body.ReqbodyMachinePowerON{UserData: in.GetUserData()}

	// Execute power-on operation
	_, err = l.APIFactory.NewMachinePowerON(systemID).POST(ctx, reqBody)
	if err != nil {
		errorProcess(err)
		return
	}

	reply = &proto.PowerOnResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}

// PowerOFF is a method to PowerOFF a process in the MaaS system.
func (l CanonicalMaasController) PowerOFF(ctx context.Context, in *proto.PowerOffRequest) (reply *proto.PowerOffResponse, err error) {
	klog.V(2).InfoS("start PowerOFF", "in", in)
	defer func() {
		klog.V(2).InfoS("end PowerOFF", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.PowerOffResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	systemID := in.GetSystemId()

	// Execute power-off operation
	_, err = l.APIFactory.NewMachinePowerOFF(systemID).POST(ctx, nil)
	if err != nil {
		errorProcess(err)
		return
	}

	reply = &proto.PowerOffResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}

// NetworkUpdate is a method to update network configuration for the Server.
func (l CanonicalMaasController) NetworkUpdate(ctx context.Context, in *proto.NetworkUpdateRequest) (reply *proto.NetworkUpdateResponse, err error) {
	klog.V(2).InfoS("start NetworkUpdate", "in", in)
	defer func() {
		klog.V(2).InfoS("end NetworkUpdate", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.NetworkUpdateResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	systemID := in.GetSystemId()

	// Get interface list (synchronous - outside goroutine)
	klog.V(2).InfoS("getting interface list", "systemID", systemID)
	interfaces, err := l.getInterfaceList(ctx, systemID)
	if err != nil {
		klog.V(2).InfoS("branch: get interface list failed", "systemID", systemID, "error", err)
		errorProcess(err)
		return
	}

	// Get subnet list (synchronous - outside goroutine)
	klog.V(2).InfoS("getting subnet list", "systemID", systemID)
	subnets, err := l.getSubnetList(ctx)
	if err != nil {
		klog.V(2).InfoS("branch: get subnet list failed", "systemID", systemID, "error", err)
		errorProcess(err)
		return
	}

	// Get machine access information (synchronous - outside goroutine)
	klog.V(2).InfoS("getting machine access information", "systemID", systemID)
	_, _, accessAddress, _, _, _, _, _, err := l.getMachineAccessInfo(ctx, systemID)
	if err != nil {
		klog.V(2).InfoS("branch: get machine access info failed", "systemID", systemID, "error", err)
		errorProcess(err)
		return
	}

	// Start synchronous network update processing
	klog.V(2).InfoS("starting network update processing", "systemID", systemID)

	// Group network information by MAC address, preserving first-appearance order.
	// This allows multiple CIDRs to be assigned to the same interface in one Ansible call.
	type macNetInfoGroup struct {
		iface    *response_body.Interface
		netInfos []*proto.NetworkInformation
	}
	macOrder := make([]string, 0)
	macGroups := make(map[string]*macNetInfoGroup)

	for _, netInfo := range in.GetNetworkInformation() {
		mac := strings.ToLower(netInfo.GetMacAddress())
		if _, exists := macGroups[mac]; !exists {
			var targetInterface *response_body.Interface
			for idx := range interfaces {
				if strings.ToLower(interfaces[idx].MacAddress) == mac {
					targetInterface = &interfaces[idx]
					break
				}
			}
			if targetInterface == nil {
				err = errors.New("interface not found for MAC address: " + netInfo.GetMacAddress())
				klog.V(2).InfoS("branch: interface not found", "systemID", systemID, "macAddress", netInfo.GetMacAddress(), "error", err)
				errorProcess(err)
				return
			}
			macGroups[mac] = &macNetInfoGroup{iface: targetInterface}
			macOrder = append(macOrder, mac)
		}
		macGroups[mac].netInfos = append(macGroups[mac].netInfos, netInfo)
	}

	// Process each MAC group: reserve all new IPs, apply via Ansible once, then update tags.
	for _, mac := range macOrder {
		group := macGroups[mac]
		iface := group.iface
		klog.V(2).InfoS("processing MAC group", "systemID", systemID, "mac", mac, "interfaceName", iface.Name, "cidrCount", len(group.netInfos))

		// Step 1: Collect current IPs tracked in tags (IP/prefix format)
		currentIPs := iface.TaggedIPs()
		klog.V(2).InfoS("current IPs from tags", "systemID", systemID, "interfaceName", iface.Name, "currentIPs", currentIPs)

		// Steps 2-5: For each CIDR, find subnet, get unreserved IP, and reserve it
		var newIPs []string
		var newIPsWithPrefix []string
		for _, netInfo := range group.netInfos {
			// Step 2: Find subnet ID by CIDR
			subnetIDPtr := l.findSubnet(subnets, netInfo.GetCidr())
			if subnetIDPtr == nil {
				err = errors.New("subnet not found for CIDR: " + netInfo.GetCidr())
				klog.V(2).InfoS("branch: subnet not found", "systemID", systemID, "cidr", netInfo.GetCidr(), "error", err)
				errorProcess(err)
				return
			}
			subnetID := *subnetIDPtr
			klog.V(2).InfoS("subnet found", "systemID", systemID, "cidr", netInfo.GetCidr(), "subnetID", subnetID)

			// Extract mask length from CIDR
			parts := strings.Split(netInfo.GetCidr(), "/")
			if len(parts) != 2 {
				err = errors.New("invalid CIDR format: " + netInfo.GetCidr())
				klog.V(2).InfoS("branch: invalid CIDR format", "systemID", systemID, "cidr", netInfo.GetCidr(), "error", err)
				errorProcess(err)
				return
			}
			maskLength := parts[1]

			// Step 4: Query unreserved IP ranges from the subnet
			klog.V(2).InfoS("querying unreserved IP ranges", "systemID", systemID, "subnetID", subnetID)
			var res interface{}
			res, err = l.APIFactory.NewSubnetUnreservedIPRanges(subnetID).GET(ctx)
			if err != nil {
				klog.V(2).InfoS("branch: get unreserved IP ranges failed", "systemID", systemID, "subnetID", subnetID, "error", err)
				errorProcess(err)
				return
			}
			var unreservedRanges response_body.ResbodySubnetUnreservedIPRanges
			var ok bool
			if unreservedRanges, ok = res.(response_body.ResbodySubnetUnreservedIPRanges); !ok {
				err = &utils.RespError{Message: "response type is invalid"}
				klog.V(2).InfoS("branch: unreserved IP ranges response type invalid", "systemID", systemID, "error", err)
				errorProcess(err)
				return
			}
			if len(unreservedRanges.List) == 0 {
				err = errors.New("no unreserved IP ranges available in subnet")
				klog.V(2).InfoS("branch: no unreserved IP ranges", "systemID", systemID, "subnetID", subnetID, "error", err)
				errorProcess(err)
				return
			}
			newIP := unreservedRanges.List[0].Start
			klog.V(2).InfoS("new IP determined", "systemID", systemID, "newIP", newIP)

			// Step 5: Reserve the new IP address in MAAS
			klog.V(2).InfoS("reserving new IP", "systemID", systemID, "newIP", newIP, "subnet", netInfo.GetCidr())
			_, err = l.APIFactory.NewIPAddressReserve().POST(ctx, request_body.ReqbodyIPAddressReserve{
				IP:     newIP,
				Subnet: netInfo.GetCidr(),
			})
			if err != nil {
				klog.V(2).InfoS("branch: IP address reserve failed", "systemID", systemID, "newIP", newIP, "error", err)
				errorProcess(err)
				return
			}
			klog.V(2).InfoS("new IP reserved successfully", "systemID", systemID, "newIP", newIP)

			newIPs = append(newIPs, newIP)
			newIPsWithPrefix = append(newIPsWithPrefix, fmt.Sprintf("%s/%s", newIP, maskLength))
		}

		// Step 6: Set all static IPs on the machine using a single Ansible call
		ipWithPrefixList := strings.Join(newIPsWithPrefix, ",")
		klog.V(2).InfoS("setting static IPs via Ansible", "systemID", systemID, "interfaceName", iface.Name, "ipWithPrefixList", ipWithPrefixList)
		extra := fmt.Sprintf("interface_name=%s ip_with_prefix_list=%s", iface.Name, ipWithPrefixList)
		_, err = l.Ansible.CmdExecute(ctx, accessAddress, "set_static_ip.yaml", extra)
		if err != nil {
			klog.V(2).InfoS("branch: Ansible set static IPs failed", "systemID", systemID, "error", err)
			errorProcess(err)
			return
		}
		klog.V(2).InfoS("static IPs set successfully via Ansible", "systemID", systemID, "ipWithPrefixList", ipWithPrefixList)

		// Step 7: Release old IPs that are no longer assigned
		newIPSet := make(map[string]bool, len(newIPs))
		for _, ip := range newIPs {
			newIPSet[ip] = true
		}
		for _, oldIP := range currentIPs {
			if !newIPSet[oldIP] {
				klog.V(2).InfoS("releasing old IP", "systemID", systemID, "oldIP", oldIP)
				_, err = l.APIFactory.NewIPAddressRelease().POST(ctx, request_body.ReqbodyIPAddressRelease{
					IP:    oldIP,
					Force: true,
				})
				if err != nil {
					klog.V(2).InfoS("branch: IP address release failed", "systemID", systemID, "oldIP", oldIP, "error", err)
					errorProcess(err)
					return
				}
				klog.V(2).InfoS("old IP released successfully", "systemID", systemID, "oldIP", oldIP)
			}
		}

		// Step 8: Remove all existing IP/prefix tags, then add the new ones
		for _, tag := range iface.IPWithPrefixTags() {
			klog.V(2).InfoS("removing existing IP tag", "systemID", systemID, "interfaceID", iface.ID, "tag", tag)
			_, err = l.APIFactory.NewInterfaceRemoveTag(systemID, iface.ID).POST(ctx, request_body.ReqbodyInterfaceTag{
				Tag: tag,
			})
			if err != nil {
				klog.V(2).InfoS("branch: interface remove tag failed", "systemID", systemID, "interfaceID", iface.ID, "tag", tag, "error", err)
				errorProcess(err)
				return
			}
			klog.V(2).InfoS("IP tag removed successfully", "systemID", systemID, "interfaceID", iface.ID, "tag", tag)
		}
		for _, ipWithPrefix := range newIPsWithPrefix {
			klog.V(2).InfoS("adding new IP tag", "systemID", systemID, "interfaceID", iface.ID, "ipWithPrefix", ipWithPrefix)
			_, err = l.APIFactory.NewInterfaceAddTag(systemID, iface.ID).POST(ctx, request_body.ReqbodyInterfaceTag{
				Tag: ipWithPrefix,
			})
			if err != nil {
				klog.V(2).InfoS("branch: interface add tag failed", "systemID", systemID, "interfaceID", iface.ID, "ipWithPrefix", ipWithPrefix, "error", err)
				errorProcess(err)
				return
			}
			klog.V(2).InfoS("new IP tag added successfully", "systemID", systemID, "interfaceID", iface.ID, "ipWithPrefix", ipWithPrefix)
		}

		klog.V(2).InfoS("MAC group processing completed", "systemID", systemID, "mac", mac, "newIPs", newIPs)
	}

	// If user_data is provided, re-execute cloud-init on the machine
	if in.GetUserData() != "" {
		klog.V(2).InfoS("user_data provided, re-running cloud-init", "systemID", systemID, "accessAddress", accessAddress)
		extraVarsBytes, _ := json.Marshal(map[string]string{"user_data": in.GetUserData()})
		_, err = l.Ansible.CmdExecute(ctx, accessAddress, "run_cloud_init.yaml", string(extraVarsBytes))
		if err != nil {
			klog.V(2).InfoS("branch: cloud-init re-run failed", "systemID", systemID, "error", err)
			errorProcess(err)
			return
		}
		klog.V(2).InfoS("cloud-init re-run completed successfully", "systemID", systemID)
	}

	klog.InfoS("NetworkUpdate processing completed successfully", "systemID", systemID)

	// Return ACCEPT: cloud-init is only started on the target (asynchronous)
	reply = &proto.NetworkUpdateResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}

// KubeadmReset is a method to reset of kubeadm for the Server.
func (l CanonicalMaasController) KubeadmReset(ctx context.Context, in *proto.KubeadmResetRequest) (reply *proto.KubeadmResetResponse, err error) {
	klog.V(2).InfoS("start KubeadmReset", "in", in)
	defer func() {
		klog.V(2).InfoS("end KubeadmReset", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.KubeadmResetResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	systemID := in.GetSystemId()

	// get machine access information
	klog.V(2).InfoS("getting machine access information", "systemID", systemID)
	_, _, accessAddress, _, _, _, _, _, err := l.getMachineAccessInfo(ctx, systemID)
	if err != nil {
		klog.V(2).InfoS("branch: get machine access info failed", "systemID", systemID, "error", err)
		errorProcess(err)
		return
	}

	// execute kubeadm reset via ansible
	klog.V(2).InfoS("executing kubeadm reset", "systemID", systemID, "accessAddress", accessAddress)
	_, err = l.Ansible.CmdExecute(ctx, accessAddress, "kubeadm_reset.yaml", "")
	if err != nil {
		klog.V(2).InfoS("branch: kubeadm reset failed", "systemID", systemID, "error", err)
		errorProcess(err)
		return
	}

	reply = &proto.KubeadmResetResponse{
		Result:       common.ResultCode_SUCCESS.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}

// KubeadmJoin is a method to join to k8s-cluster for the Server.
func (l CanonicalMaasController) KubeadmJoin(ctx context.Context, in *proto.KubeadmJoinRequest) (reply *proto.KubeadmJoinResponse, err error) {
	klog.V(2).InfoS("start KubeadmJoin", "in", in)
	defer func() {
		klog.V(2).InfoS("end KubeadmJoin", "reply", reply, "err", err)
	}()

	errorProcess := func(errorDetail error) {
		reply = &proto.KubeadmJoinResponse{
			Result:       common.ResultCode_ERROR.Enum(),
			ErrorMessage: utils.ErrorMessageToJSON(l.getErrorMessage(errorDetail)),
		}
		err = errorDetail
	}

	systemID := in.GetSystemId()

	// get worker machine access information
	klog.V(2).InfoS("getting worker machine access information", "systemID", systemID)
	_, _, workerAccessAddress, _, _, _, _, _, err := l.getMachineAccessInfo(ctx, systemID)
	if err != nil {
		klog.V(2).InfoS("branch: get worker machine access info failed", "systemID", systemID, "error", err)
		errorProcess(err)
		return
	}

	// register job
	if l.JobManager != nil {
		l.JobManager.Register(systemID, JobTypeKubeadmJoin)
	}

	// execute kubeadm join process asynchronously
	go func() {
		if l.JobManager != nil {
			defer l.JobManager.Deregister(systemID, JobTypeKubeadmJoin)
		}
		
		// Create independent context for async processing to avoid request context cancellation
		asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		cpSystemIDs := in.GetCpSystemId()
		klog.V(2).InfoS("starting async kubeadm join process", "systemID", systemID, "cpSystemIDs", cpSystemIDs, "contextStatus", asyncCtx.Err())

		// loop for control plane system IDs
		joinCommand := ""
		for _, cpSystemID := range cpSystemIDs {

			// get control plane machine access information
			klog.V(2).InfoS("getting control plane machine access information", "cpSystemID", cpSystemID)
			_, _, cpAccessAddress, _, _, _, _, _, err := l.getMachineAccessInfo(asyncCtx, cpSystemID)
			if err != nil {
				klog.V(2).InfoS("branch: get control plane machine access info failed", "cpSystemID", cpSystemID, "error", err)
				continue // try next control plane
			}

			// execute kubeadm token create on control plane to get join command
			klog.V(2).InfoS("generating join token on control plane", "cpSystemID", cpSystemID, "cpAccessAddress", cpAccessAddress, "contextStatus", asyncCtx.Err())
			tokenOutput, err := l.Ansible.CmdExecute(asyncCtx, cpAccessAddress, "kubeadm_token_create.yaml", "")
			if err != nil {
				klog.V(2).InfoS("branch: kubeadm token create failed", "cpSystemID", cpSystemID, "error", err)
				continue // try next control plane
			}
			
			// parse join command from ansible output
			// Ansible debug output wraps the message value in JSON quotes:
			//   ok: [host] => {
			//       "msg": "KUBEADM_JOIN_COMMAND=kubeadm join ..."
			//   }
			// After SplitN on "KUBEADM_JOIN_COMMAND=", parts[1] ends with `"` (JSON closing quote).
			// Use TrimSuffix to remove exactly that trailing `"` without touching the command value.
			tokenStr := string(tokenOutput)
			lines := strings.Split(tokenStr, "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				// First try to find the marked output
				if strings.Contains(trimmed, "KUBEADM_JOIN_COMMAND=") {
					parts := strings.SplitN(trimmed, "KUBEADM_JOIN_COMMAND=", 2)
					if len(parts) == 2 {
						// Strip the trailing `"` from Ansible's JSON debug output format
						candidate := strings.TrimSuffix(strings.TrimSpace(parts[1]), `"`)
						if strings.HasPrefix(candidate, "kubeadm join") {
							joinCommand = candidate
							break
						}
					}
				}
				// Fallback: look for direct kubeadm join command
				if joinCommand == "" && strings.HasPrefix(trimmed, "kubeadm join") {
					joinCommand = trimmed
				}
			}
			if joinCommand == "" {
				klog.V(2).InfoS("branch: join command not found in output", "error", err, "output", tokenStr)
				continue // try next control plane
			} else {
				// join command found, break the loop
				break
			}
		}

		if joinCommand == "" {
			klog.V(2).InfoS("branch: failed to extract join command from any control plane")
			err := errors.New("failed to extract join command from control plane output")
			l.markBroken(asyncCtx, err, systemID)
			return
		}

		klog.V(2).InfoS("extracted join command", "joinCommand", joinCommand)

		// prepare extra-vars for ansible with join command
		// Use JSON format to safely handle special characters in the join command
		extraVarsBytes, _ := json.Marshal(map[string]string{"join_command": joinCommand})
		extra := string(extraVarsBytes)

		// execute kubeadm join on worker node via ansible
		klog.V(2).InfoS("executing kubeadm join on worker", "systemID", systemID, "workerAccessAddress", workerAccessAddress, "contextStatus", asyncCtx.Err())
		_, err = l.Ansible.CmdExecute(asyncCtx, workerAccessAddress, "kubeadm_join.yaml", extra)
		if err != nil {
			klog.V(2).InfoS("branch: kubeadm join failed", "systemID", systemID, "error", err)
			l.markBroken(asyncCtx, err, systemID)
			return
		}

		klog.V(2).InfoS("kubeadm join completed successfully", "systemID", systemID)
	}()

	// return ACCEPT immediately
	reply = &proto.KubeadmJoinResponse{
		Result:       common.ResultCode_ACCEPT.Enum(),
		ErrorMessage: "",
	}
	err = nil
	return
}
