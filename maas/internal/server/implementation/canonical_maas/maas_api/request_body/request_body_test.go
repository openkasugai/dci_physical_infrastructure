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

import (
	"testing"
)

// Test ReqbodyCommon
func TestReqbodyCommon(t *testing.T) {
	// Arrange & Act
	reqBody := ReqbodyCommon{}

	// Assert
	// ReqbodyCommon is just an empty struct, no specific behavior to test
	// This test ensures the struct can be instantiated
	if reqBody == (ReqbodyCommon{}) {
		// This is expected
	}
}

// Test ReqbodyIPRanges
func TestReqbodyIPRanges(t *testing.T) {
	// Arrange
	subnetID := 1
	startIP := "192.168.1.10"
	endIP := "192.168.1.20"
	ipType := "reserved"

	// Act
	reqBody := ReqbodyIPRanges{
		SubnetID: subnetID,
		StartIP:  startIP,
		EndIP:    endIP,
		Type:     ipType,
	}

	// Assert
	if reqBody.SubnetID != subnetID {
		t.Errorf("Expected SubnetID to be %d, got %d", subnetID, reqBody.SubnetID)
	}

	if reqBody.StartIP != startIP {
		t.Errorf("Expected StartIP to be %s, got %s", startIP, reqBody.StartIP)
	}

	if reqBody.EndIP != endIP {
		t.Errorf("Expected EndIP to be %s, got %s", endIP, reqBody.EndIP)
	}

	if reqBody.Type != ipType {
		t.Errorf("Expected Type to be %s, got %s", ipType, reqBody.Type)
	}
}

// Test ReqbodyMachines
func TestReqbodyMachines(t *testing.T) {
	// Arrange
	architecture := "amd64"
	macAddresses := "aa:bb:cc:dd:ee:ff"
	hostname := "test-machine"
	commission := true
	enableSSH := true
	powerType := "ipmi"
	powerAddress := "192.168.1.100"
	powerUser := "admin"
	powerPass := "password"

	// Act
	reqBody := ReqbodyMachines{
		Architecture: architecture,
		MACAddresses: macAddresses,
		Hostname:     hostname,
		Commission:   commission,
		EnableSSH:    enableSSH,
		PowerType:    powerType,
		PowerAddress: powerAddress,
		PowerUser:    powerUser,
		PowerPass:    powerPass,
	}

	// Assert
	if reqBody.Architecture != architecture {
		t.Errorf("Expected Architecture to be %s, got %s", architecture, reqBody.Architecture)
	}

	if reqBody.MACAddresses != macAddresses {
		t.Errorf("Expected MACAddresses to be %s, got %s", macAddresses, reqBody.MACAddresses)
	}

	if reqBody.Hostname != hostname {
		t.Errorf("Expected Hostname to be %s, got %s", hostname, reqBody.Hostname)
	}

	if reqBody.Commission != commission {
		t.Errorf("Expected Commission to be %v, got %v", commission, reqBody.Commission)
	}

	if reqBody.EnableSSH != enableSSH {
		t.Errorf("Expected EnableSSH to be %v, got %v", enableSSH, reqBody.EnableSSH)
	}

	if reqBody.PowerType != powerType {
		t.Errorf("Expected PowerType to be %s, got %s", powerType, reqBody.PowerType)
	}

	if reqBody.PowerAddress != powerAddress {
		t.Errorf("Expected PowerAddress to be %s, got %s", powerAddress, reqBody.PowerAddress)
	}

	if reqBody.PowerUser != powerUser {
		t.Errorf("Expected PowerUser to be %s, got %s", powerUser, reqBody.PowerUser)
	}

	if reqBody.PowerPass != powerPass {
		t.Errorf("Expected PowerPass to be %s, got %s", powerPass, reqBody.PowerPass)
	}
}

// Test ReqbodyMachineDeploy
func TestReqbodyMachineDeploy(t *testing.T) {
	// Arrange
	bridgeAll := true
	distribution := "ubuntu"
	version := "20.04"
	userData := "#!/bin/bash\necho 'Hello World'"

	// Act
	reqBody := ReqbodyMachineDeploy{
		BridgeAll:    bridgeAll,
		Distribution: distribution,
		Version:      version,
		UserData:     userData,
	}

	// Assert
	if reqBody.BridgeAll != bridgeAll {
		t.Errorf("Expected BridgeAll to be %v, got %v", bridgeAll, reqBody.BridgeAll)
	}

	if reqBody.Distribution != distribution {
		t.Errorf("Expected Distribution to be %s, got %s", distribution, reqBody.Distribution)
	}

	if reqBody.Version != version {
		t.Errorf("Expected Version to be %s, got %s", version, reqBody.Version)
	}

	if reqBody.UserData != userData {
		t.Errorf("Expected UserData to be %s, got %s", userData, reqBody.UserData)
	}
}

// Test ReqbodyIFLinkSubnet
func TestReqbodyIFLinkSubnet(t *testing.T) {
	// Arrange
	mode := "STATIC"
	subnetID := 1

	// Act
	reqBody := ReqbodyIFLinkSubnet{
		Mode:     mode,
		SubnetID: subnetID,
	}

	// Assert
	if reqBody.Mode != mode {
		t.Errorf("Expected Mode to be %s, got %s", mode, reqBody.Mode)
	}

	if reqBody.SubnetID != subnetID {
		t.Errorf("Expected SubnetID to be %d, got %d", subnetID, reqBody.SubnetID)
	}
}

// Test ReqbodyVMhosts
func TestReqbodyVMhosts(t *testing.T) {
	// Arrange
	powerAddress := "192.168.1.200"
	vmType := "lxd"

	// Act
	reqBody := ReqbodyVMhosts{
		PowerAddress: powerAddress,
		Type:         vmType,
	}

	// Assert
	if reqBody.PowerAddress != powerAddress {
		t.Errorf("Expected PowerAddress to be %s, got %s", powerAddress, reqBody.PowerAddress)
	}

	if reqBody.Type != vmType {
		t.Errorf("Expected Type to be %s, got %s", vmType, reqBody.Type)
	}
}

// Test ReqbodyVMhostCompose
func TestReqbodyVMhostCompose(t *testing.T) {
	// Arrange
	cores := 4
	hostName := "test-vm"
	memory := 8192
	storage := 100
	interfaces := "eth0:br0"

	// Act
	reqBody := ReqbodyVMhostCompose{
		Cores:      cores,
		HostName:   hostName,
		Memory:     memory,
		Storage:    storage,
		Interfaces: interfaces,
	}

	// Assert
	if reqBody.Cores != cores {
		t.Errorf("Expected Cores to be %d, got %d", cores, reqBody.Cores)
	}

	if reqBody.HostName != hostName {
		t.Errorf("Expected HostName to be %s, got %s", hostName, reqBody.HostName)
	}

	if reqBody.Memory != memory {
		t.Errorf("Expected Memory to be %d, got %d", memory, reqBody.Memory)
	}

	if reqBody.Storage != storage {
		t.Errorf("Expected Storage to be %d, got %d", storage, reqBody.Storage)
	}

	if reqBody.Interfaces != interfaces {
		t.Errorf("Expected Interfaces to be %s, got %s", interfaces, reqBody.Interfaces)
	}
}

func TestReqbodyInterfaceUpdate(t *testing.T) {
	name := "eth0"

	reqBody := ReqbodyInterfaceUpdate{
		Name: name,
	}

	if reqBody.Name != name {
		t.Errorf("Expected Name to be %s, got %s", name, reqBody.Name)
	}
}

// Test ReqbodySubnets
func TestReqbodySubnets(t *testing.T) {
	// Arrange
	cidr := "192.168.1.0/24"
	fabricID := 1
	vid := 100

	// Act
	reqBody := ReqbodySubnets{
		Cidr:     cidr,
		FabricID: fabricID,
		Vid:      vid,
	}

	// Assert
	if reqBody.Cidr != cidr {
		t.Errorf("Expected Cidr to be %s, got %s", cidr, reqBody.Cidr)
	}

	if reqBody.FabricID != fabricID {
		t.Errorf("Expected FabricID to be %d, got %d", fabricID, reqBody.FabricID)
	}

	if reqBody.Vid != vid {
		t.Errorf("Expected Vid to be %d, got %d", vid, reqBody.Vid)
	}
}

// Test interface implementation
func TestReqbodyInterface(t *testing.T) {
	// Test all structs implement the interface
	tests := []struct {
		name    string
		reqbody Reqbody
	}{
		{"ReqbodyCommon", ReqbodyCommon{}},
		{"ReqbodyIPRanges", ReqbodyIPRanges{}},
		{"ReqbodyMachines", ReqbodyMachines{}},
		{"ReqbodyMachineDeploy", ReqbodyMachineDeploy{}},
		{"ReqbodyIFLinkSubnet", ReqbodyIFLinkSubnet{}},
		{"ReqbodyVMhosts", ReqbodyVMhosts{}},
		{"ReqbodyVMhostCompose", ReqbodyVMhostCompose{}},
		{"ReqbodySubnets", ReqbodySubnets{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just check that the type implements the interface
			_ = tt.reqbody
		})
	}
}

// Edge case tests
func TestReqbodyWithZeroValues(t *testing.T) {
	// Test that zero values work correctly
	machines := ReqbodyMachines{}
	if machines.Commission != false {
		t.Error("Expected default Commission to be false")
	}

	if machines.EnableSSH != false {
		t.Error("Expected default EnableSSH to be false")
	}

	vmhost := ReqbodyVMhostCompose{}
	if vmhost.Cores != 0 {
		t.Error("Expected default Cores to be 0")
	}

	if vmhost.Memory != 0 {
		t.Error("Expected default Memory to be 0")
	}
}

func TestReqbodyWithEmptyStrings(t *testing.T) {
	// Test that empty strings work correctly
	machines := ReqbodyMachines{
		Architecture: "",
		MACAddresses: "",
		Hostname:     "",
	}

	if machines.Architecture != "" {
		t.Error("Expected empty Architecture to remain empty")
	}

	if machines.MACAddresses != "" {
		t.Error("Expected empty MACAddresses to remain empty")
	}

	if machines.Hostname != "" {
		t.Error("Expected empty Hostname to remain empty")
	}
}

// TestReqbodyIPAddressReserve tests ReqbodyIPAddressReserve struct
func TestReqbodyIPAddressReserve(t *testing.T) {
ip := "192.168.1.100"
subnet := "192.168.1.0/24"

reqBody := ReqbodyIPAddressReserve{
IP:     ip,
Subnet: subnet,
}

if reqBody.IP != ip {
t.Errorf("Expected IP to be %s, got %s", ip, reqBody.IP)
}

if reqBody.Subnet != subnet {
t.Errorf("Expected Subnet to be %s, got %s", subnet, reqBody.Subnet)
}
}

// TestReqbodyIPAddressRelease tests ReqbodyIPAddressRelease struct
func TestReqbodyIPAddressRelease(t *testing.T) {
ip := "192.168.1.100"
force := true

reqBody := ReqbodyIPAddressRelease{
IP:    ip,
Force: force,
}

if reqBody.IP != ip {
t.Errorf("Expected IP to be %s, got %s", ip, reqBody.IP)
}

if reqBody.Force != force {
t.Errorf("Expected Force to be %v, got %v", force, reqBody.Force)
}
}
// TestReqbodyInterfaceTag tests ReqbodyInterfaceTag struct
func TestReqbodyInterfaceTag(t *testing.T) {
	tag := "192.168.1.100"

	reqBody := ReqbodyInterfaceTag{
		Tag: tag,
	}

	if reqBody.Tag != tag {
		t.Errorf("Expected Tag to be %s, got %s", tag, reqBody.Tag)
	}
}
