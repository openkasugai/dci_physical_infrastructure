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

package request_body

// Reqbody is an interface for request bodies in Canonical MAAS API.
type Reqbody interface{}

// ReqbodyCommon is a common struct for request bodies in Canonical MAAS API.
type ReqbodyCommon struct{}

// ReqbodyIPRanges represents the request body for creating or updating IP ranges in Canonical MAAS.
type ReqbodyIPRanges struct {
	ReqbodyCommon
	SubnetID int
	StartIP  string
	EndIP    string
	Type     string
}

// ReqbodyMachines represents the request body for creating or updating machines in Canonical MAAS.
type ReqbodyMachines struct {
	ReqbodyCommon
	Architecture string
	MACAddresses string
	Hostname     string
	Commission   bool
	EnableSSH    bool
	PowerType    string
	PowerAddress string
	PowerUser    string
	PowerPass    string
	Description  string
}

// ReqbodyMachineDeploy represents the request body for deploying a machine in Canonical MAAS.
type ReqbodyMachineDeploy struct {
	ReqbodyCommon
	BridgeAll    bool
	Distribution string
	Version      string
	UserData     string
}

// ReqbodyMachinePowerON represents the request body for powering on a machine in Canonical MAAS.
type ReqbodyMachinePowerON struct {
	ReqbodyCommon
	UserData string
}

// ReqbodyMachineRelease represents the request body for releasing a machine in Canonical MAAS.
type ReqbodyMachineRelease struct {
	ReqbodyCommon
	Erase    	bool
	QuickErase 	bool
	SecureErase	bool
}

// ReqbodyIFLinkSubnet represents the request body for linking a subnet to an interface in Canonical MAAS.
type ReqbodyIFLinkSubnet struct {
	ReqbodyCommon
	Mode     string
	SubnetID int
}

// ReqbodyVMhosts represents the request body for creating or updating VM hosts in Canonical MAAS.
type ReqbodyVMhosts struct {
	ReqbodyCommon
	PowerAddress string
	Type         string
}

// ReqbodyVMhostCompose represents the request body for composing a VM host in Canonical MAAS.
type ReqbodyVMhostCompose struct {
	ReqbodyCommon
	Cores      int
	HostName   string
	Memory     int
	Storage    int
	Interfaces string //ifName string, bridgeName stringを保持
}

// ReqbodySubnets represents the request body for creating or updating subnets in Canonical MAAS.
type ReqbodySubnets struct {
	ReqbodyCommon
	Cidr     string
	FabricID int
	Vid      int
}

// ReqbodyMachineMarkBroken represents the request body for marking a machine as broken in Canonical MAAS.
type ReqbodyMachineMarkBroken struct {
	ReqbodyCommon
	Comment string // Optional comment for marking machine as broken
}

// ReqbodyMachineUpdate represents the request body for updating a machine in Canonical MAAS.
type ReqbodyMachineUpdate struct {
	ReqbodyCommon
	Description string
}

// ReqbodyIPAddressReserve represents the request body for reserving an IP address in Canonical MAAS.
type ReqbodyIPAddressReserve struct {
	ReqbodyCommon
	IP     string // IP address to reserve
	Subnet string // Subnet in CIDR format (e.g., "192.168.30.0/24")
}

// ReqbodyIPAddressRelease represents the request body for releasing an IP address in Canonical MAAS.
type ReqbodyIPAddressRelease struct {
	ReqbodyCommon
	IP    string // IP address to release
	Force bool   // Force release even if in use
}

// ReqbodyInterfaceTag represents the request body for adding or removing a tag from an interface in Canonical MAAS.
type ReqbodyInterfaceTag struct {
	ReqbodyCommon
	Tag string // Tag to add or remove
}

// ReqbodyInterfaceUpdate represents the request body for updating an interface in Canonical MAAS.
type ReqbodyInterfaceUpdate struct {
	ReqbodyCommon
	Name string // New name for the interface
}
