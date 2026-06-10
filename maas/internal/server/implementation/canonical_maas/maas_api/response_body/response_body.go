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

package response_body

import (
	"encoding/json"
	"net"
	"strings"
)

// extractBootIP returns the first IPv4 address reachable via the boot interface (or its children).
func extractBootIP(bootInterface Interface, interfaceSet []Interface) string {
	isIPv4 := func(s string) bool {
		ip := net.ParseIP(s)
		return ip != nil && ip.To4() != nil
	}
	for _, link := range bootInterface.Links {
		if isIPv4(link.IPAddress) {
			return link.IPAddress
		}
	}
	for _, childName := range bootInterface.Children {
		for _, iface := range interfaceSet {
			if iface.Name == childName {
				for _, link := range iface.Links {
					if isIPv4(link.IPAddress) {
						return link.IPAddress
					}
				}
			}
		}
	}
	return ""
}

// Resbody is the interface for all response body types.
type Resbody interface{}

// ResbodyCommon is a common structure for all response bodies.
type ResbodyCommon struct {
	HTTPStatus   int
	ErrorMessage string
	RawJSONData  string
}

/*
 * fabric
 */

// Vlan represents a VLAN in a fabric.
type Vlan struct {
	Vid int `json:"vid"`
}

// ResbodyPostFabrics represents the response body for creating a fabric.
type ResbodyPostFabrics struct {
	ResbodyCommon `json:"-"`
	ID            int    `json:"id"`
	Vlans         []Vlan `json:"vlans"`
}

/*
 * subnets
 */

// Subnet represents a subnet in Canonical MAAS.
type Subnet struct {
	Cidr string `json:"cidr"`
	ID   int    `json:"id"`
}

// ResbodyGetSubnets represents the response body for retrieving subnets.
type ResbodyGetSubnets struct {
	ResbodyCommon `json:"-"`
	List          []Subnet `json:"-"`
}

func (r *ResbodyGetSubnets) UnmarshalJSON(data []byte) error {
	var tmp []Subnet
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	r.List = tmp
	return nil
}

// ResbodyPostSubnets represents the response body for creating a subnet.
type ResbodyPostSubnets struct {
	ResbodyCommon `json:"-"`
	ID            int `json:"id"`
}

// UnreservedIPRange represents an unreserved IP range in a subnet.
type UnreservedIPRange struct {
	Start        string `json:"start"`
	End          string `json:"end"`
	NumAddresses int    `json:"num_addresses"`
}

// ResbodySubnetUnreservedIPRanges represents the response body for retrieving unreserved IP ranges.
type ResbodySubnetUnreservedIPRanges struct {
	ResbodyCommon `json:"-"`
	List          []UnreservedIPRange `json:"-"`
}

func (r *ResbodySubnetUnreservedIPRanges) UnmarshalJSON(data []byte) error {
	var tmp []UnreservedIPRange
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	r.List = tmp
	return nil
}

// ResbodyIPAddressReserve represents the response body for reserving an IP address.
type ResbodyIPAddressReserve struct {
	ResbodyCommon `json:"-"`
}

// ResbodyIPAddressRelease represents the response body for releasing an IP address.
type ResbodyIPAddressRelease struct {
	ResbodyCommon `json:"-"`
}

/*
 * interface
 */

// Link represents a link in an interface.
type Link struct {
	IPAddress string `json:"ip_address"`
	Subnet    Subnet `json:"subnet"`
}

// Interface represents a network interface in Canonical MAAS.
type Interface struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Children   []string `json:"children"`
	MacAddress string   `json:"mac_address"`
	Links      []Link   `json:"links"`
	Tags       []string `json:"tags"`
}

// isIPWithPrefix reports whether s is in "IPv4/prefix" format (e.g., "192.168.1.1/24").
func isIPWithPrefix(s string) bool {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return false
	}
	ip := net.ParseIP(parts[0])
	return ip != nil && ip.To4() != nil
}

// TaggedIPs returns the IP part of all tags that are in "IPv4/prefix" format.
// Tags written by NetworkUpdate use this format (e.g., "192.168.1.100/24").
func (i Interface) TaggedIPs() []string {
	var ips []string
	for _, tag := range i.Tags {
		if isIPWithPrefix(tag) {
			ips = append(ips, strings.SplitN(tag, "/", 2)[0])
		}
	}
	return ips
}

// IPWithPrefixTags returns the full tag strings that are in "IPv4/prefix" format.
// Used to enumerate the exact tag values that need to be removed via the MAAS API.
func (i Interface) IPWithPrefixTags() []string {
	var tags []string
	for _, tag := range i.Tags {
		if isIPWithPrefix(tag) {
			tags = append(tags, tag)
		}
	}
	return tags
}

// UnmarshalJSON implements custom JSON unmarshaling for Interface.
// If the interface has tags in "IPv4/prefix" format (written by NetworkUpdate),
// Links is rebuilt from those tags so all callers reading Links see the current
// effective IP addresses rather than stale MAAS-tracked values.
func (i *Interface) UnmarshalJSON(data []byte) error {
	type Alias Interface
	var tmp Alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*i = Interface(tmp)

	// Collect tags in "IP/prefix" format
	type tagEntry struct {
		ip     string
		subnet string // full CIDR from tag, e.g. "192.168.1.100/24"
	}
	var tagEntries []tagEntry
	for _, tag := range i.Tags {
		if isIPWithPrefix(tag) {
			parts := strings.SplitN(tag, "/", 2)
			tagEntries = append(tagEntries, tagEntry{ip: parts[0], subnet: tag})
		}
	}
	if len(tagEntries) == 0 {
		return nil
	}

	// Rebuild Links from tagged IPs, overwriting the original Links entirely.
	// For each tagged IP, find the matching subnet from the original Links via CIDR
	// containment. If no match is found, create a synthetic Link with the tag CIDR.
	newLinks := make([]Link, 0, len(tagEntries))
	for _, te := range tagEntries {
		tip := net.ParseIP(te.ip)
		var matchedSubnet Subnet
		for _, link := range i.Links {
			if link.Subnet.Cidr == "" {
				continue
			}
			_, snet, err := net.ParseCIDR(link.Subnet.Cidr)
			if err != nil {
				continue
			}
			if snet.Contains(tip) {
				matchedSubnet = link.Subnet
				break
			}
		}
		if matchedSubnet.Cidr == "" {
			// No matching subnet found; derive the network CIDR by zeroing host bits
			// (net.ParseCIDR normalises "192.168.1.100/24" → "192.168.1.0/24").
			if _, snet, parseErr := net.ParseCIDR(te.subnet); parseErr == nil {
				matchedSubnet = Subnet{Cidr: snet.String()}
			} else {
				matchedSubnet = Subnet{Cidr: te.subnet}
			}
		}
		newLinks = append(newLinks, Link{
			IPAddress: te.ip,
			Subnet:    matchedSubnet,
		})
	}
	i.Links = newLinks
	return nil
}
type InterfaceForResponse struct {	// for grpc response
	MacAddress string   `json:"mac_address"`
	IPAddress []string 	`json:"ip_addresses"`
	IFName string 		`json:"if_name"`
}

// ResbodyGetInterfaces represents the response body for retrieving interfaces.
type ResbodyGetInterfaces struct {
	ResbodyCommon `json:"-"`
	List          []Interface `json:"-"`
}

func (r *ResbodyGetInterfaces) UnmarshalJSON(data []byte) error {
	var tmp []Interface
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	r.List = tmp
	return nil
}

/*
 * machines
 */

// ResbodyPostMachines represents the response body for creating a machine.
type ResbodyPostMachines struct {
	ResbodyCommon `json:"-"`
	SystemID      string `json:"system_id"`
}

// ResbodyGetMachine represents the response body for retrieving a specific machine.
type ResbodyGetMachine struct {
	ResbodyCommon      `json:"-"`
	MachineForResponse MachineForResponse `json:"-"`
	AccessAddress      string      `json:"-"` // not exposed in JSON; used internally for ready_status check
	SystemID           string      `json:"system_id"`
	HostName           string      `json:"hostname"`
	IPAddresses        []string    `json:"ip_addresses"`
	StatusName         string      `json:"status_name"`
	InterfaceSet       []Interface `json:"interface_set"`
	BootInterface      Interface   `json:"boot_interface"`
	Description        string      `json:"description"`
	PowerStatus        string      `json:"power_state"`
	Storage            float64     `json:"storage"`
}

func (r *ResbodyGetMachine) UnmarshalJSON(data []byte) error {
    type tempResbodyGetMachine struct {
        SystemID      string      `json:"system_id"`
        HostName      string      `json:"hostname"`
        IPAddresses   []string    `json:"ip_addresses"`
        StatusName    string      `json:"status_name"`
        InterfaceSet  []Interface `json:"interface_set"`
        BootInterface Interface   `json:"boot_interface"`
        Description   string      `json:"description"`
		PowerStatus   string      `json:"power_state"`
		Storage   	  float64     `json:"storage"`
    }
    
    var tmp tempResbodyGetMachine
    if err := json.Unmarshal(data, &tmp); err != nil {
        return err
    }

	var ifList []InterfaceForResponse
	for _, iface := range tmp.InterfaceSet {
		var ipAddrs []string
		for _, link := range iface.Links {
			ipAddrs = append(ipAddrs, link.IPAddress)
		}
		ifaceResp := InterfaceForResponse{
			MacAddress: iface.MacAddress,
			IPAddress:  ipAddrs,
			IFName:     iface.Name,
		}
		ifList = append(ifList, ifaceResp)
	}

	r.MachineForResponse = MachineForResponse{
		HostName:      tmp.HostName,
		SystemID:      tmp.SystemID,
		StatusName:    tmp.StatusName,
		PowerStatus:   tmp.PowerStatus,
		InterfaceList: ifList,
	}
	r.AccessAddress = extractBootIP(tmp.BootInterface, tmp.InterfaceSet)
	r.SystemID = tmp.SystemID
	r.HostName = tmp.HostName
	r.IPAddresses = tmp.IPAddresses
	r.StatusName = tmp.StatusName
	r.InterfaceSet = tmp.InterfaceSet
	r.BootInterface = tmp.BootInterface
	r.Description = tmp.Description
	r.PowerStatus = tmp.PowerStatus
	r.Storage = tmp.Storage

	return nil
}

// Machine represents a machine in Canonical MAAS.
type Machine struct {
	SystemID      string      `json:"system_id"`
	HostName      string      `json:"hostname"`
	StatusName    string      `json:"status_name"`
	PowerStatus   string      `json:"power_state"`
	InterfaceSet  []Interface `json:"interface_set"`
	BootInterface Interface   `json:"boot_interface"`
}
type MachineForResponse struct {	// for grpc response
	HostName      string                 `json:"server_id"`
	SystemID      string                 `json:"system_id"`
	StatusName    string                 `json:"status_name"`
	PowerStatus   string                 `json:"power_state"`
	InterfaceList []InterfaceForResponse `json:"interface_list"`
	ReadyStatus   string                 `json:"ready_status"`
	AccessAddress string                 `json:"-"` // not exposed in JSON; used internally for ready_status check
}

// ResbodyGetMachines represents the response body for retrieving a list of machines.
type ResbodyGetMachines struct {
	ResbodyCommon `json:"-"`
	Machines      []MachineForResponse `json:"-"`
}

func (r *ResbodyGetMachines) UnmarshalJSON(data []byte) error {
	var tmp []Machine
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	// convert Machine to MachineForResponse
	var respList []MachineForResponse
	for _, m := range tmp {
		var ifList []InterfaceForResponse
		for _, iface := range m.InterfaceSet {
			var ipAddrs []string
			for _, link := range iface.Links {
				ipAddrs = append(ipAddrs, link.IPAddress)
			}
			ifaceResp := InterfaceForResponse{
				MacAddress: iface.MacAddress,
				IPAddress:  ipAddrs,
				IFName:     iface.Name,
			}
			ifList = append(ifList, ifaceResp)
		}
		mResp := MachineForResponse{
			HostName:      m.HostName,
			SystemID:      m.SystemID,
			StatusName:    m.StatusName,
			PowerStatus:   m.PowerStatus,
			InterfaceList: ifList,
			AccessAddress: extractBootIP(m.BootInterface, m.InterfaceSet),
		}
		respList = append(respList, mResp)
	}

	r.Machines = respList
	return nil
}

/*
 * vm host
 */

// Host represents a host in Canonical MAAS.
type Host struct {
	SystemID string `json:"system_id"`
}

// VMHost represents a VM host in Canonical MAAS.
type VMHost struct {
	Host Host `json:"host"`
	ID   int  `json:"id"`
}

// ResbodyGetVMHosts represents the response body for retrieving VM hosts.
type ResbodyGetVMHosts struct {
	ResbodyCommon `json:"-"`
	List          []VMHost `json:"-"`
}

func (r *ResbodyGetVMHosts) UnmarshalJSON(data []byte) error {
	var tmp []VMHost
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	r.List = tmp
	return nil
}

// ResbodyPostVMHost represents the response body for creating a VM host.
type ResbodyPostVMHost struct {
	ResbodyCommon `json:"-"`
	ID            int `json:"id"`
}

// ResbodyGetOpParameter represents the response body for retrieving operation parameters.
type ResbodyGetOpParameter struct {
	ResbodyCommon `json:"-"`
	Certificate   string `json:"certificate"`
}

// ResbodyPostVMCompose represents the response body for composing a VM host.
type ResbodyPostVMCompose struct {
	ResbodyCommon `json:"-"`
	SystemID      string `json:"system_id"`
}
