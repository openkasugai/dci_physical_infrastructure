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
	"testing"
)

// Test ResbodyCommon
func TestResbodyCommon(t *testing.T) {
	// Arrange
	httpStatus := 200
	errorMessage := "No error"
	rawJSONData := `{"status": "success"}`

	// Act
	common := ResbodyCommon{
		HTTPStatus:   httpStatus,
		ErrorMessage: errorMessage,
		RawJSONData:  rawJSONData,
	}

	// Assert
	if common.HTTPStatus != httpStatus {
		t.Errorf("Expected HTTPStatus to be %d, got %d", httpStatus, common.HTTPStatus)
	}

	if common.ErrorMessage != errorMessage {
		t.Errorf("Expected ErrorMessage to be %s, got %s", errorMessage, common.ErrorMessage)
	}

	if common.RawJSONData != rawJSONData {
		t.Errorf("Expected RawJSONData to be %s, got %s", rawJSONData, common.RawJSONData)
	}
}

// Test Vlan
func TestVlan(t *testing.T) {
	// Arrange
	vid := 100

	// Act
	vlan := Vlan{
		Vid: vid,
	}

	// Assert
	if vlan.Vid != vid {
		t.Errorf("Expected Vid to be %d, got %d", vid, vlan.Vid)
	}
}

// Test ResbodyPostFabrics
func TestResbodyPostFabrics(t *testing.T) {
	// Arrange
	fabricID := 1
	vlans := []Vlan{
		{Vid: 10},
		{Vid: 20},
	}

	// Act
	fabric := ResbodyPostFabrics{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   200,
			ErrorMessage: "",
			RawJSONData:  `{"id": 1}`,
		},
		ID:    fabricID,
		Vlans: vlans,
	}

	// Assert
	if fabric.ID != fabricID {
		t.Errorf("Expected ID to be %d, got %d", fabricID, fabric.ID)
	}

	if len(fabric.Vlans) != 2 {
		t.Errorf("Expected Vlans length to be 2, got %d", len(fabric.Vlans))
	}

	if fabric.Vlans[0].Vid != 10 {
		t.Errorf("Expected first VLAN Vid to be 10, got %d", fabric.Vlans[0].Vid)
	}
}

// Test Subnet
func TestSubnet(t *testing.T) {
	// Arrange
	cidr := "192.168.1.0/24"
	subnetID := 1

	// Act
	subnet := Subnet{
		Cidr: cidr,
		ID:   subnetID,
	}

	// Assert
	if subnet.Cidr != cidr {
		t.Errorf("Expected Cidr to be %s, got %s", cidr, subnet.Cidr)
	}

	if subnet.ID != subnetID {
		t.Errorf("Expected ID to be %d, got %d", subnetID, subnet.ID)
	}
}

// Test ResbodyGetSubnets
func TestResbodyGetSubnets(t *testing.T) {
	// Arrange
	subnets := []Subnet{
		{Cidr: "192.168.1.0/24", ID: 1},
		{Cidr: "10.0.0.0/8", ID: 2},
	}

	// Act
	response := ResbodyGetSubnets{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   200,
			ErrorMessage: "",
			RawJSONData:  `[{"cidr": "192.168.1.0/24", "id": 1}]`,
		},
		List: subnets,
	}

	// Assert
	if len(response.List) != 2 {
		t.Errorf("Expected List length to be 2, got %d", len(response.List))
	}

	if response.List[0].Cidr != "192.168.1.0/24" {
		t.Errorf("Expected first subnet CIDR to be 192.168.1.0/24, got %s", response.List[0].Cidr)
	}
}

// Test ResbodyPostSubnets
func TestResbodyPostSubnets(t *testing.T) {
	// Arrange
	subnetID := 1

	// Act
	response := ResbodyPostSubnets{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   201,
			ErrorMessage: "",
			RawJSONData:  `{"id": 1}`,
		},
		ID: subnetID,
	}

	// Assert
	if response.ID != subnetID {
		t.Errorf("Expected ID to be %d, got %d", subnetID, response.ID)
	}
}

// Test Link
func TestLink(t *testing.T) {
	// Arrange
	ipAddress := "192.168.1.100"
	subnet := Subnet{
		Cidr: "192.168.1.0/24",
		ID:   1,
	}

	// Act
	link := Link{
		IPAddress: ipAddress,
		Subnet:    subnet,
	}

	// Assert
	if link.IPAddress != ipAddress {
		t.Errorf("Expected IPAddress to be %s, got %s", ipAddress, link.IPAddress)
	}

	if link.Subnet.ID != 1 {
		t.Errorf("Expected Subnet ID to be 1, got %d", link.Subnet.ID)
	}
}

// Test Interface
func TestInterface(t *testing.T) {
	// Arrange
	interfaceID := 1
	name := "eth0"
	children := []string{"eth0.10", "eth0.20"}
	macAddress := "aa:bb:cc:dd:ee:ff"
	tags := []string{"192.168.1.100", "10.0.0.50"}
	links := []Link{
		{
			IPAddress: "192.168.1.100",
			Subnet:    Subnet{Cidr: "192.168.1.0/24", ID: 1},
		},
	}

	// Act
	iface := Interface{
		ID:         interfaceID,
		Name:       name,
		Children:   children,
		MacAddress: macAddress,
		Links:      links,
		Tags:       tags,
	}

	// Assert
	if iface.ID != interfaceID {
		t.Errorf("Expected ID to be %d, got %d", interfaceID, iface.ID)
	}

	if iface.Name != name {
		t.Errorf("Expected Name to be %s, got %s", name, iface.Name)
	}

	if len(iface.Children) != 2 {
		t.Errorf("Expected Children length to be 2, got %d", len(iface.Children))
	}

	if iface.MacAddress != macAddress {
		t.Errorf("Expected MacAddress to be %s, got %s", macAddress, iface.MacAddress)
	}

	if len(iface.Links) != 1 {
		t.Errorf("Expected Links length to be 1, got %d", len(iface.Links))
	}

	if len(iface.Tags) != 2 {
		t.Errorf("Expected Tags length to be 2, got %d", len(iface.Tags))
	}

	if iface.Tags[0] != "192.168.1.100" {
		t.Errorf("Expected first tag to be '192.168.1.100', got %s", iface.Tags[0])
	}
}

// Test InterfaceForResponse
func TestInterfaceForResponse(t *testing.T) {
	// Arrange
	macAddress := "aa:bb:cc:dd:ee:ff"
	ipAddresses := []string{"192.168.1.100", "10.0.0.50"}
	ifName := "eth0"

	// Act
	ifaceResp := InterfaceForResponse{
		MacAddress: macAddress,
		IPAddress:  ipAddresses,
		IFName:     ifName,
	}

	// Assert
	if ifaceResp.MacAddress != macAddress {
		t.Errorf("Expected MacAddress to be %s, got %s", macAddress, ifaceResp.MacAddress)
	}

	if len(ifaceResp.IPAddress) != 2 {
		t.Errorf("Expected IPAddress length to be 2, got %d", len(ifaceResp.IPAddress))
	}

	if ifaceResp.IFName != ifName {
		t.Errorf("Expected IFName to be %s, got %s", ifName, ifaceResp.IFName)
	}
}

// Test ResbodyGetInterfaces
func TestResbodyGetInterfaces(t *testing.T) {
	// Arrange
	interfaces := []Interface{
		{
			ID:         1,
			Name:       "eth0",
			Children:   []string{},
			MacAddress: "aa:bb:cc:dd:ee:ff",
			Links:      []Link{},
		},
	}

	// Act
	response := ResbodyGetInterfaces{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   200,
			ErrorMessage: "",
			RawJSONData:  `[{"id": 1, "name": "eth0"}]`,
		},
		List: interfaces,
	}

	// Assert
	if len(response.List) != 1 {
		t.Errorf("Expected List length to be 1, got %d", len(response.List))
	}

	if response.List[0].Name != "eth0" {
		t.Errorf("Expected first interface name to be eth0, got %s", response.List[0].Name)
	}
}

// Test ResbodyPostMachines
func TestResbodyPostMachines(t *testing.T) {
	// Arrange
	systemID := "test-system-id"

	// Act
	response := ResbodyPostMachines{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   201,
			ErrorMessage: "",
			RawJSONData:  `{"system_id": "test-system-id"}`,
		},
		SystemID: systemID,
	}

	// Assert
	if response.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, response.SystemID)
	}
}

// Test ResbodyGetMachine
func TestResbodyGetMachine(t *testing.T) {
	// Arrange
	systemID := "test-system-id"
	hostname := "test-machine"
	ipAddresses := []string{"192.168.1.100", "10.0.0.50"}
	statusName := "Ready"
	interfaces := []Interface{
		{ID: 1, Name: "eth0", Children: []string{}, MacAddress: "aa:bb:cc:dd:ee:ff", Links: []Link{}},
	}
	bootInterface := Interface{
		ID: 1, Name: "eth0", Children: []string{}, MacAddress: "aa:bb:cc:dd:ee:ff", Links: []Link{},
	}

	// Act
	response := ResbodyGetMachine{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   200,
			ErrorMessage: "",
			RawJSONData:  `{"system_id": "test-system-id"}`,
		},
		SystemID:      systemID,
		HostName:      hostname,
		IPAddresses:   ipAddresses,
		StatusName:    statusName,
		InterfaceSet:  interfaces,
		BootInterface: bootInterface,
	}

	// Assert
	if response.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, response.SystemID)
	}

	if response.HostName != hostname {
		t.Errorf("Expected HostName to be %s, got %s", hostname, response.HostName)
	}

	if len(response.IPAddresses) != 2 {
		t.Errorf("Expected IPAddresses length to be 2, got %d", len(response.IPAddresses))
	}

	if response.StatusName != statusName {
		t.Errorf("Expected StatusName to be %s, got %s", statusName, response.StatusName)
	}

	if len(response.InterfaceSet) != 1 {
		t.Errorf("Expected InterfaceSet length to be 1, got %d", len(response.InterfaceSet))
	}

	if response.BootInterface.ID != 1 {
		t.Errorf("Expected BootInterface ID to be 1, got %d", response.BootInterface.ID)
	}
}

// Test ResbodyGetMachine UnmarshalJSON
func TestResbodyGetMachine_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name                 string
		jsonData             string
		expectError          bool
		expectedSystemID     string
		expectedHostName     string
		expectedStatusName   string
		expectedPowerStatus  string
		expectedInterfaceLen int
	}{
		{
			name: "valid machine with interfaces",
			jsonData: `{
				"system_id": "test-sys-id",
				"hostname": "test-host",
				"ip_addresses": ["192.168.1.100"],
				"status_name": "Ready",
				"power_state": "on",
				"interface_set": [
					{
						"name": "eth0",
						"mac_address": "aa:bb:cc:dd:ee:ff",
						"links": [
							{
								"ip_address": "192.168.1.100"
							}
						]
					}
				],
				"boot_interface": {
					"name": "eth0",
					"mac_address": "aa:bb:cc:dd:ee:ff",
					"links": []
				},
				"description": "Test machine",
				"storage": 1024.0
			}`,
			expectError:          false,
			expectedSystemID:     "test-sys-id",
			expectedHostName:     "test-host",
			expectedStatusName:   "Ready",
			expectedPowerStatus:  "on",
			expectedInterfaceLen: 1,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resbody ResbodyGetMachine
			err := resbody.UnmarshalJSON([]byte(tt.jsonData))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resbody.SystemID != tt.expectedSystemID {
				t.Errorf("Expected SystemID %s, got %s", tt.expectedSystemID, resbody.SystemID)
			}

			if resbody.HostName != tt.expectedHostName {
				t.Errorf("Expected HostName %s, got %s", tt.expectedHostName, resbody.HostName)
			}

			if resbody.StatusName != tt.expectedStatusName {
				t.Errorf("Expected StatusName %s, got %s", tt.expectedStatusName, resbody.StatusName)
			}

			if resbody.PowerStatus != tt.expectedPowerStatus {
				t.Errorf("Expected PowerStatus %s, got %s", tt.expectedPowerStatus, resbody.PowerStatus)
			}

			if len(resbody.MachineForResponse.InterfaceList) != tt.expectedInterfaceLen {
				t.Errorf("Expected InterfaceList length %d, got %d", tt.expectedInterfaceLen, len(resbody.MachineForResponse.InterfaceList))
			}

			// Verify MachineForResponse is properly populated
			if resbody.MachineForResponse.SystemID != tt.expectedSystemID {
				t.Errorf("Expected MachineForResponse.SystemID %s, got %s", tt.expectedSystemID, resbody.MachineForResponse.SystemID)
			}

			if resbody.MachineForResponse.HostName != tt.expectedHostName {
				t.Errorf("Expected MachineForResponse.HostName %s, got %s", tt.expectedHostName, resbody.MachineForResponse.HostName)
			}

			if resbody.MachineForResponse.PowerStatus != tt.expectedPowerStatus {
				t.Errorf("Expected MachineForResponse.PowerStatus %s, got %s", tt.expectedPowerStatus, resbody.MachineForResponse.PowerStatus)
			}
		})
	}
}

// Test Machine
func TestMachine(t *testing.T) {
	// Arrange
	systemID := "machine-system-id"
	hostname := "machine-hostname"
	statusName := "Deployed"
	interfaces := []Interface{
		{ID: 1, Name: "eth0", Children: []string{}, MacAddress: "aa:bb:cc:dd:ee:ff", Links: []Link{}},
	}

	// Act
	machine := Machine{
		SystemID:     systemID,
		HostName:     hostname,
		StatusName:   statusName,
		InterfaceSet: interfaces,
	}

	// Assert
	if machine.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, machine.SystemID)
	}

	if machine.HostName != hostname {
		t.Errorf("Expected HostName to be %s, got %s", hostname, machine.HostName)
	}

	if machine.StatusName != statusName {
		t.Errorf("Expected StatusName to be %s, got %s", statusName, machine.StatusName)
	}

	if len(machine.InterfaceSet) != 1 {
		t.Errorf("Expected InterfaceSet length to be 1, got %d", len(machine.InterfaceSet))
	}
}

// Test MachineForResponse
func TestMachineForResponse(t *testing.T) {
	// Arrange
	systemID := "machine-system-id"
	hostname := "machine-hostname"
	statusName := "Deployed"
	interfaceList := []InterfaceForResponse{
		{MacAddress: "aa:bb:cc:dd:ee:ff", IPAddress: []string{"192.168.1.100"}, IFName: "eth0"},
	}

	// Act
	machine := MachineForResponse{
		SystemID:      systemID,
		HostName:      hostname,
		StatusName:    statusName,
		InterfaceList: interfaceList,
	}

	// Assert
	if machine.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, machine.SystemID)
	}

	if machine.HostName != hostname {
		t.Errorf("Expected HostName to be %s, got %s", hostname, machine.HostName)
	}

	if machine.StatusName != statusName {
		t.Errorf("Expected StatusName to be %s, got %s", statusName, machine.StatusName)
	}

	if len(machine.InterfaceList) != 1 {
		t.Errorf("Expected InterfaceList length to be 1, got %d", len(machine.InterfaceList))
	}
}

// Test ResbodyGetMachines
func TestResbodyGetMachines(t *testing.T) {
	// Arrange
	machines := []MachineForResponse{
		{
			SystemID:      "machine-1",
			HostName:      "machine-1-hostname",
			StatusName:    "Ready",
			InterfaceList: []InterfaceForResponse{},
		},
		{
			SystemID:      "machine-2",
			HostName:      "machine-2-hostname",
			StatusName:    "Deployed",
			InterfaceList: []InterfaceForResponse{},
		},
	}

	// Act
	response := ResbodyGetMachines{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   200,
			ErrorMessage: "",
			RawJSONData:  `[{"system_id": "machine-1"}]`,
		},
		Machines: machines,
	}

	// Assert
	if len(response.Machines) != 2 {
		t.Errorf("Expected Machines length to be 2, got %d", len(response.Machines))
	}

	if response.Machines[0].SystemID != "machine-1" {
		t.Errorf("Expected first machine SystemID to be machine-1, got %s", response.Machines[0].SystemID)
	}
}

// Test Host
func TestHost(t *testing.T) {
	// Arrange
	systemID := "host-system-id"

	// Act
	host := Host{
		SystemID: systemID,
	}

	// Assert
	if host.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, host.SystemID)
	}
}

// Test VMHost
func TestVMHost(t *testing.T) {
	// Arrange
	vmhostID := 1
	host := Host{SystemID: "vmhost-system-id"}

	// Act
	vmhost := VMHost{
		Host: host,
		ID:   vmhostID,
	}

	// Assert
	if vmhost.ID != vmhostID {
		t.Errorf("Expected ID to be %d, got %d", vmhostID, vmhost.ID)
	}

	if vmhost.Host.SystemID != "vmhost-system-id" {
		t.Errorf("Expected Host SystemID to be vmhost-system-id, got %s", vmhost.Host.SystemID)
	}
}

// Test ResbodyGetVMHosts
func TestResbodyGetVMHosts(t *testing.T) {
	// Arrange
	vmhosts := []VMHost{
		{
			Host: Host{SystemID: "vmhost-1"},
			ID:   1,
		},
		{
			Host: Host{SystemID: "vmhost-2"},
			ID:   2,
		},
	}

	// Act
	response := ResbodyGetVMHosts{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   200,
			ErrorMessage: "",
			RawJSONData:  `[{"id": 1}]`,
		},
		List: vmhosts,
	}

	// Assert
	if len(response.List) != 2 {
		t.Errorf("Expected List length to be 2, got %d", len(response.List))
	}

	if response.List[0].ID != 1 {
		t.Errorf("Expected first VMHost ID to be 1, got %d", response.List[0].ID)
	}
}

// Test ResbodyPostVMHost
func TestResbodyPostVMHost(t *testing.T) {
	// Arrange
	vmhostID := 1

	// Act
	response := ResbodyPostVMHost{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   201,
			ErrorMessage: "",
			RawJSONData:  `{"id": 1}`,
		},
		ID: vmhostID,
	}

	// Assert
	if response.ID != vmhostID {
		t.Errorf("Expected ID to be %d, got %d", vmhostID, response.ID)
	}
}

// Test ResbodyGetOpParameter
func TestResbodyGetOpParameter(t *testing.T) {
	// Arrange
	certificate := "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKoK/heBjcOuMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV\n-----END CERTIFICATE-----"

	// Act
	response := ResbodyGetOpParameter{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   200,
			ErrorMessage: "",
			RawJSONData:  `{"certificate": "..."}`,
		},
		Certificate: certificate,
	}

	// Assert
	if response.Certificate != certificate {
		t.Errorf("Expected Certificate to match, got different value")
	}
}

// Test ResbodyPostVMCompose
func TestResbodyPostVMCompose(t *testing.T) {
	// Arrange
	systemID := "composed-vm-system-id"

	// Act
	response := ResbodyPostVMCompose{
		ResbodyCommon: ResbodyCommon{
			HTTPStatus:   201,
			ErrorMessage: "",
			RawJSONData:  `{"system_id": "composed-vm-system-id"}`,
		},
		SystemID: systemID,
	}

	// Assert
	if response.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, response.SystemID)
	}
}

// Test standard JSON marshaling for other response types
func TestResbodyPostMachines_JSONMarshal(t *testing.T) {
	resbody := ResbodyPostMachines{
		SystemID: "test-system-id",
	}
	resbody.HTTPStatus = 201
	resbody.ErrorMessage = ""
	resbody.RawJSONData = `{"system_id":"test-system-id"}`

	data, err := json.Marshal(resbody)
	if err != nil {
		t.Errorf("Unexpected error during marshal: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty JSON data")
	}
}

// Test JSON marshaling for subnet response
func TestResbodyPostSubnets_JSONMarshal(t *testing.T) {
	resbody := ResbodyPostSubnets{
		ID: 123,
	}
	resbody.HTTPStatus = 201

	data, err := json.Marshal(resbody)
	if err != nil {
		t.Errorf("Unexpected error during marshal: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty JSON data")
	}
}

// Test interface implementation
func TestResbodyInterface(t *testing.T) {
	// Test all structs implement the interface
	tests := []struct {
		name    string
		resbody Resbody
	}{
		{"ResbodyCommon", ResbodyCommon{}},
		{"ResbodyPostFabrics", ResbodyPostFabrics{}},
		{"ResbodyGetSubnets", ResbodyGetSubnets{}},
		{"ResbodyPostSubnets", ResbodyPostSubnets{}},
		{"ResbodyGetInterfaces", ResbodyGetInterfaces{}},
		{"ResbodyPostMachines", ResbodyPostMachines{}},
		{"ResbodyGetMachine", ResbodyGetMachine{}},
		{"ResbodyGetMachines", ResbodyGetMachines{}},
		{"ResbodyGetVMHosts", ResbodyGetVMHosts{}},
		{"ResbodyPostVMHost", ResbodyPostVMHost{}},
		{"ResbodyGetOpParameter", ResbodyGetOpParameter{}},
		{"ResbodyPostVMCompose", ResbodyPostVMCompose{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just check that the type implements the interface
			_ = tt.resbody
		})
	}
}

// Edge case tests
func TestResbodyWithZeroValues(t *testing.T) {
	// Test that zero values work correctly
	common := ResbodyCommon{}
	if common.HTTPStatus != 0 {
		t.Error("Expected default HTTPStatus to be 0")
	}

	if common.ErrorMessage != "" {
		t.Error("Expected default ErrorMessage to be empty string")
	}

	machine := Machine{}
	if machine.SystemID != "" {
		t.Error("Expected default SystemID to be empty string")
	}

	if len(machine.InterfaceSet) != 0 {
		t.Error("Expected default InterfaceSet to be empty slice")
	}
}

func TestResbodyWithEmptySlices(t *testing.T) {
	// Test that empty slices work correctly
	response := ResbodyGetSubnets{
		List: []Subnet{},
	}

	if len(response.List) != 0 {
		t.Error("Expected List length to be 0")
	}

	machines := ResbodyGetMachines{
		Machines: []MachineForResponse{},
	}

	if len(machines.Machines) != 0 {
		t.Error("Expected Machines length to be 0")
	}
}

func TestResbodyWithNilSlices(t *testing.T) {
	// Test that nil slices work correctly
	response := ResbodyGetSubnets{
		List: nil,
	}

	if response.List != nil {
		t.Error("Expected List to be nil")
	}

	machines := ResbodyGetMachines{
		Machines: nil,
	}

	if machines.Machines != nil {
		t.Error("Expected Machines to be nil")
	}
}

// Error case tests
func TestResbodyWithErrors(t *testing.T) {
	// Test error scenarios
	errorResponse := ResbodyCommon{
		HTTPStatus:   400,
		ErrorMessage: "Bad Request",
		RawJSONData:  `{"error": "Invalid parameter"}`,
	}

	if errorResponse.HTTPStatus != 400 {
		t.Errorf("Expected HTTPStatus to be 400, got %d", errorResponse.HTTPStatus)
	}

	if errorResponse.ErrorMessage != "Bad Request" {
		t.Errorf("Expected ErrorMessage to be 'Bad Request', got %s", errorResponse.ErrorMessage)
	}
}

// Benchmarks
func BenchmarkResbodyCommonCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		common := ResbodyCommon{
			HTTPStatus:   200,
			ErrorMessage: "",
			RawJSONData:  `{"status": "success"}`,
		}
		_ = common
	}
}

func BenchmarkMachineCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		machine := Machine{
			SystemID:     "test-system-id",
			HostName:     "test-machine",
			StatusName:   "Ready",
			InterfaceSet: []Interface{},
		}
		_ = machine
	}
}

func BenchmarkInterfaceCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		iface := Interface{
			ID:         1,
			Name:       "eth0",
			Children:   []string{},
			MacAddress: "aa:bb:cc:dd:ee:ff",
			Links:      []Link{},
		}
		_ = iface
	}
}

// Test ResbodyGetSubnets UnmarshalJSON
func TestResbodyGetSubnets_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectLen   int
	}{
		{
			name:        "valid subnet list",
			jsonData:    `[{"id":1,"cidr":"192.168.1.0/24"},{"id":2,"cidr":"192.168.2.0/24"}]`,
			expectError: false,
			expectLen:   2,
		},
		{
			name:        "empty subnet list",
			jsonData:    `[]`,
			expectError: false,
			expectLen:   0,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
			expectLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resbody ResbodyGetSubnets
			err := resbody.UnmarshalJSON([]byte(tt.jsonData))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(resbody.List) != tt.expectLen {
				t.Errorf("Expected length %d, got %d", tt.expectLen, len(resbody.List))
			}
		})
	}
}

// Test ResbodyGetInterfaces UnmarshalJSON
func TestResbodyGetInterfaces_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectLen   int
	}{
		{
			name:        "valid interface list",
			jsonData:    `[{"id":1,"name":"eth0","mac_address":"aa:bb:cc:dd:ee:ff","children":[],"links":[]}]`,
			expectError: false,
			expectLen:   1,
		},
		{
			name:        "empty interface list",
			jsonData:    `[]`,
			expectError: false,
			expectLen:   0,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
			expectLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resbody ResbodyGetInterfaces
			err := resbody.UnmarshalJSON([]byte(tt.jsonData))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(resbody.List) != tt.expectLen {
				t.Errorf("Expected length %d, got %d", tt.expectLen, len(resbody.List))
			}
		})
	}
}

// Test ResbodyGetMachines UnmarshalJSON
func TestResbodyGetMachines_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectLen   int
	}{
		{
			name: "valid machine list with interfaces",
			jsonData: `[
				{
					"system_id":"test-1",
					"hostname":"machine1",
					"status_name":"Ready",
					"interface_set":[
						{
							"name": "eth0",
							"mac_address": "aa:bb:cc:dd:ee:ff",
							"links": [
								{
									"ip_address": "192.168.1.100"
								}
							]
						}
					]
				}
			]`,
			expectError: false,
			expectLen:   1,
		},
		{
			name:        "empty machine list",
			jsonData:    `[]`,
			expectError: false,
			expectLen:   0,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
			expectLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resbody ResbodyGetMachines
			err := resbody.UnmarshalJSON([]byte(tt.jsonData))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(resbody.Machines) != tt.expectLen {
				t.Errorf("Expected length %d, got %d", tt.expectLen, len(resbody.Machines))
			}

			// Additional validation for non-empty results
			if tt.expectLen > 0 {
				// Check that machines are properly converted to MachineForResponse
				machine := resbody.Machines[0]
				if machine.SystemID != "test-1" {
					t.Errorf("Expected SystemID 'test-1', got '%s'", machine.SystemID)
				}
				if machine.HostName != "machine1" {
					t.Errorf("Expected HostName 'machine1', got '%s'", machine.HostName)
				}
				if machine.StatusName != "Ready" {
					t.Errorf("Expected StatusName 'Ready', got '%s'", machine.StatusName)
				}
				if len(machine.InterfaceList) != 1 {
					t.Errorf("Expected InterfaceList length 1, got %d", len(machine.InterfaceList))
				} else {
					// Check interface conversion
					iface := machine.InterfaceList[0]
					if iface.MacAddress != "aa:bb:cc:dd:ee:ff" {
						t.Errorf("Expected MacAddress 'aa:bb:cc:dd:ee:ff', got '%s'", iface.MacAddress)
					}
					if iface.IFName != "eth0" {
						t.Errorf("Expected IFName 'eth0', got '%s'", iface.IFName)
					}
					if len(iface.IPAddress) != 1 {
						t.Errorf("Expected IPAddress length 1, got %d", len(iface.IPAddress))
					} else if iface.IPAddress[0] != "192.168.1.100" {
						t.Errorf("Expected IPAddress '192.168.1.100', got '%s'", iface.IPAddress[0])
					}
				}
			}
		})
	}
}

// Test ResbodyGetVMHosts UnmarshalJSON
func TestResbodyGetVMHosts_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectLen   int
	}{
		{
			name:        "valid vmhost list",
			jsonData:    `[{"id":1,"host":{"system_id":"vmhost-1"}}]`,
			expectError: false,
			expectLen:   1,
		},
		{
			name:        "empty vmhost list",
			jsonData:    `[]`,
			expectError: false,
			expectLen:   0,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
			expectLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resbody ResbodyGetVMHosts
			err := resbody.UnmarshalJSON([]byte(tt.jsonData))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(resbody.List) != tt.expectLen {
				t.Errorf("Expected length %d, got %d", tt.expectLen, len(resbody.List))
			}
		})
	}
}

// TestUnreservedIPRange tests UnreservedIPRange struct
func TestUnreservedIPRange(t *testing.T) {
start := "192.168.1.100"
end := "192.168.1.200"
numAddresses := 101

ipRange := UnreservedIPRange{
Start:        start,
End:          end,
NumAddresses: numAddresses,
}

if ipRange.Start != start {
t.Errorf("Expected Start to be %s, got %s", start, ipRange.Start)
}

if ipRange.End != end {
t.Errorf("Expected End to be %s, got %s", end, ipRange.End)
}

if ipRange.NumAddresses != numAddresses {
t.Errorf("Expected NumAddresses to be %d, got %d", numAddresses, ipRange.NumAddresses)
}
}

// TestResbodySubnetUnreservedIPRanges tests ResbodySubnetUnreservedIPRanges struct
func TestResbodySubnetUnreservedIPRanges(t *testing.T) {
ranges := []UnreservedIPRange{
{Start: "192.168.1.100", End: "192.168.1.150", NumAddresses: 51},
{Start: "192.168.1.200", End: "192.168.1.250", NumAddresses: 51},
}

resbody := ResbodySubnetUnreservedIPRanges{
List:          ranges,
ResbodyCommon: ResbodyCommon{HTTPStatus: 200},
}

if len(resbody.List) != 2 {
t.Errorf("Expected List length to be 2, got %d", len(resbody.List))
}

if resbody.List[0].Start != "192.168.1.100" {
t.Errorf("Expected first range Start to be 192.168.1.100, got %s", resbody.List[0].Start)
}

if resbody.HTTPStatus != 200 {
t.Errorf("Expected HTTPStatus to be 200, got %d", resbody.HTTPStatus)
}
}

// TestResbodySubnetUnreservedIPRanges_UnmarshalJSON_Success tests successful JSON unmarshaling
func TestResbodySubnetUnreservedIPRanges_UnmarshalJSON_Success(t *testing.T) {
jsonData := []byte(`[
{"start":"192.168.1.100","end":"192.168.1.150","num_addresses":51},
{"start":"192.168.1.200","end":"192.168.1.250","num_addresses":51}
]`)

var resbody ResbodySubnetUnreservedIPRanges
err := resbody.UnmarshalJSON(jsonData)

if err != nil {
t.Errorf("Expected no error, got %v", err)
}

if len(resbody.List) != 2 {
t.Errorf("Expected List length to be 2, got %d", len(resbody.List))
}

if resbody.List[0].Start != "192.168.1.100" {
t.Errorf("Expected first range Start to be 192.168.1.100, got %s", resbody.List[0].Start)
}

if resbody.List[1].NumAddresses != 51 {
t.Errorf("Expected second range NumAddresses to be 51, got %d", resbody.List[1].NumAddresses)
}
}

// TestResbodySubnetUnreservedIPRanges_UnmarshalJSON_InvalidJSON tests invalid JSON
func TestResbodySubnetUnreservedIPRanges_UnmarshalJSON_InvalidJSON(t *testing.T) {
jsonData := []byte(`invalid json`)

var resbody ResbodySubnetUnreservedIPRanges
err := resbody.UnmarshalJSON(jsonData)

if err == nil {
t.Error("Expected error for invalid JSON, got nil")
}
}

// TestResbodySubnetUnreservedIPRanges_UnmarshalJSON_EmptyArray tests empty array
func TestResbodySubnetUnreservedIPRanges_UnmarshalJSON_EmptyArray(t *testing.T) {
jsonData := []byte(`[]`)

var resbody ResbodySubnetUnreservedIPRanges
err := resbody.UnmarshalJSON(jsonData)

if err != nil {
t.Errorf("Expected no error for empty array, got %v", err)
}

if len(resbody.List) != 0 {
t.Errorf("Expected empty List, got length %d", len(resbody.List))
}
}

// TestResbodyIPAddressReserve tests ResbodyIPAddressReserve struct
func TestResbodyIPAddressReserve(t *testing.T) {
resbody := ResbodyIPAddressReserve{
ResbodyCommon: ResbodyCommon{
HTTPStatus:   200,
ErrorMessage: "",
RawJSONData:  `{"ip":"192.168.1.100"}`,
},
}

if resbody.HTTPStatus != 200 {
t.Errorf("Expected HTTPStatus to be 200, got %d", resbody.HTTPStatus)
}

if resbody.ErrorMessage != "" {
t.Errorf("Expected ErrorMessage to be empty, got %s", resbody.ErrorMessage)
}
}

// TestResbodyIPAddressRelease tests ResbodyIPAddressRelease struct
func TestResbodyIPAddressRelease(t *testing.T) {
resbody := ResbodyIPAddressRelease{
ResbodyCommon: ResbodyCommon{
HTTPStatus:   200,
ErrorMessage: "",
RawJSONData:  `{"success":true}`,
},
}

if resbody.HTTPStatus != 200 {
t.Errorf("Expected HTTPStatus to be 200, got %d", resbody.HTTPStatus)
}

if resbody.ErrorMessage != "" {
t.Errorf("Expected ErrorMessage to be empty, got %s", resbody.ErrorMessage)
}
}
// =============================================================
// extractBootIP
// =============================================================

func TestExtractBootIP(t *testing.T) {
	tests := []struct {
		name          string
		bootInterface Interface
		interfaceSet  []Interface
		want          string
	}{
		{
			name: "DirectLinkOnBootInterface_ReturnsIP",
			bootInterface: Interface{
				Name: "eth0",
				Links: []Link{
					{IPAddress: "192.168.1.10", Subnet: Subnet{ID: 1, Cidr: "192.168.1.0/24"}},
				},
			},
			interfaceSet: nil,
			want:         "192.168.1.10",
		},
		{
			name: "IPv6OnlyDirectLink_ReturnsEmpty",
			bootInterface: Interface{
				Name: "eth0",
				Links: []Link{
					{IPAddress: "2001:db8::1"},
				},
			},
			interfaceSet: nil,
			want:         "",
		},
		{
			name: "NoLinksOnBootInterface_ReturnsEmpty",
			bootInterface: Interface{
				Name:  "eth0",
				Links: []Link{},
			},
			interfaceSet: nil,
			want:         "",
		},
		{
			name: "BridgeWithChildHavingIP_ReturnsChildIP",
			bootInterface: Interface{
				Name:     "br0",
				Links:    []Link{},
				Children: []string{"eth0"},
			},
			interfaceSet: []Interface{
				{
					Name: "eth0",
					Links: []Link{
						{IPAddress: "10.0.0.5", Subnet: Subnet{ID: 2, Cidr: "10.0.0.0/24"}},
					},
				},
			},
			want: "10.0.0.5",
		},
		{
			name: "BridgeWithChildHavingOnlyIPv6_ReturnsEmpty",
			bootInterface: Interface{
				Name:     "br0",
				Links:    []Link{},
				Children: []string{"eth0"},
			},
			interfaceSet: []Interface{
				{
					Name: "eth0",
					Links: []Link{
						{IPAddress: "2001:db8::2"},
					},
				},
			},
			want: "",
		},
		{
			name: "BridgeChildNotFoundInInterfaceSet_ReturnsEmpty",
			bootInterface: Interface{
				Name:     "br0",
				Links:    []Link{},
				Children: []string{"eth99"},
			},
			interfaceSet: []Interface{
				{Name: "eth0", Links: []Link{{IPAddress: "192.168.1.1"}}},
			},
			want: "",
		},
		{
			name: "MultipleChildrenSecondHasIP_ReturnsSecondChildIP",
			bootInterface: Interface{
				Name:     "br0",
				Links:    []Link{},
				Children: []string{"eth0", "eth1"},
			},
			interfaceSet: []Interface{
				{Name: "eth0", Links: []Link{}},
				{Name: "eth1", Links: []Link{
					{IPAddress: "172.16.0.1"},
				}},
			},
			want: "172.16.0.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractBootIP(tc.bootInterface, tc.interfaceSet)
			if got != tc.want {
				t.Errorf("extractBootIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

// =============================================================
// isIPWithPrefix
// =============================================================

func TestIsIPWithPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"192.168.1.1/24", true},
		{"10.0.0.1/8", true},
		{"255.255.255.255/32", true},
		{"0.0.0.0/0", true},
		// IPv6 — must be rejected (only IPv4 is accepted)
		{"2001:db8::1/64", false},
		// No prefix
		{"192.168.1.1", false},
		// Empty
		{"", false},
		// Only slash
		{"/24", false},
		// Non-IP host part
		{"not-an-ip/24", false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := isIPWithPrefix(tc.input)
			if got != tc.want {
				t.Errorf("isIPWithPrefix(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// =============================================================
// Interface.TaggedIPs
// =============================================================

func TestInterface_TaggedIPs(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want []string
	}{
		{
			name: "NoTags_ReturnsNil",
			tags: []string{},
			want: nil,
		},
		{
			name: "OnlyIPWithPrefixTags_ReturnsIPs",
			tags: []string{"192.168.1.100/24", "10.0.0.50/8"},
			want: []string{"192.168.1.100", "10.0.0.50"},
		},
		{
			name: "MixedTags_ReturnsOnlyIPWithPrefixIPs",
			tags: []string{"some-label", "192.168.1.100/24", "not-an-ip", "10.0.0.50/16"},
			want: []string{"192.168.1.100", "10.0.0.50"},
		},
		{
			name: "PlainIPTags_ReturnsNil",
			tags: []string{"192.168.1.100", "10.0.0.50"},
			want: nil,
		},
		{
			name: "IPv6PrefixTag_Ignored",
			tags: []string{"2001:db8::1/64"},
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iface := Interface{Tags: tc.tags}
			got := iface.TaggedIPs()
			if len(got) != len(tc.want) {
				t.Fatalf("TaggedIPs() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("TaggedIPs()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// =============================================================
// Interface.IPWithPrefixTags
// =============================================================

func TestInterface_IPWithPrefixTags(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want []string
	}{
		{
			name: "NoTags_ReturnsNil",
			tags: []string{},
			want: nil,
		},
		{
			name: "OnlyIPWithPrefixTags_ReturnsFullTags",
			tags: []string{"192.168.1.100/24", "10.0.0.50/8"},
			want: []string{"192.168.1.100/24", "10.0.0.50/8"},
		},
		{
			name: "MixedTags_ReturnsOnlyIPWithPrefixTags",
			tags: []string{"some-label", "192.168.1.100/24", "not-an-ip"},
			want: []string{"192.168.1.100/24"},
		},
		{
			name: "PlainIPTags_ReturnsNil",
			tags: []string{"192.168.1.100"},
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iface := Interface{Tags: tc.tags}
			got := iface.IPWithPrefixTags()
			if len(got) != len(tc.want) {
				t.Fatalf("IPWithPrefixTags() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("IPWithPrefixTags()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// =============================================================
// Interface.UnmarshalJSON
// =============================================================

func TestInterface_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		wantErr       bool
		wantLinkCount int
		// per-link assertions (index → expected values)
		wantLinks []struct {
			ip         string
			subnetCidr string
			subnetID   int
		}
		// when true, Links must remain unchanged (same count as original JSON links)
		linksUnchanged bool
	}{
		{
			name: "InvalidJSON_ReturnsError",
			jsonData: `{invalid}`,
			wantErr:  true,
		},
		{
			name: "NoIPPrefixTags_LinksUnchanged",
			jsonData: `{
				"id": 1,
				"name": "eth0",
				"mac_address": "aa:bb:cc:dd:ee:ff",
				"tags": ["some-label", "192.168.1.1"],
				"links": [
					{"ip_address": "192.168.1.10", "subnet": {"id": 5, "cidr": "192.168.1.0/24"}}
				]
			}`,
			wantErr:       false,
			wantLinkCount: 1,
			wantLinks: []struct {
				ip         string
				subnetCidr string
				subnetID   int
			}{
				{"192.168.1.10", "192.168.1.0/24", 5},
			},
		},
		{
			name: "IPPrefixTagMatchesExistingSubnet_LinkUsesMatchedSubnet",
			jsonData: `{
				"id": 2,
				"name": "eth1",
				"mac_address": "00:11:22:33:44:55",
				"tags": ["192.168.30.100/24"],
				"links": [
					{"ip_address": "192.168.30.10", "subnet": {"id": 10, "cidr": "192.168.30.0/24"}}
				]
			}`,
			wantErr:       false,
			wantLinkCount: 1,
			wantLinks: []struct {
				ip         string
				subnetCidr string
				subnetID   int
			}{
				// IP comes from tag, Subnet from matched original Link
				{"192.168.30.100", "192.168.30.0/24", 10},
			},
		},
		{
			name: "IPPrefixTagNoMatchingSubnet_SyntheticLinkWithNetworkCIDR",
			jsonData: `{
				"id": 3,
				"name": "eth2",
				"mac_address": "ff:ee:dd:cc:bb:aa",
				"tags": ["10.0.0.99/16"],
				"links": [
					{"ip_address": "192.168.1.1", "subnet": {"id": 1, "cidr": "192.168.1.0/24"}}
				]
			}`,
			wantErr:       false,
			wantLinkCount: 1,
			wantLinks: []struct {
				ip         string
				subnetCidr string
				subnetID   int
			}{
				// No original link is in 10.0.0.0/16 → synthetic link
				// net.ParseCIDR("10.0.0.99/16") normalises to "10.0.0.0/16"
				{"10.0.0.99", "10.0.0.0/16", 0},
			},
		},
		{
			name: "MultipleIPPrefixTags_LinksRebuiltForEach",
			jsonData: `{
				"id": 4,
				"name": "eth3",
				"mac_address": "11:22:33:44:55:66",
				"tags": ["192.168.1.100/24", "10.0.0.50/8"],
				"links": [
					{"ip_address": "192.168.1.1", "subnet": {"id": 1, "cidr": "192.168.1.0/24"}},
					{"ip_address": "10.0.0.1",   "subnet": {"id": 2, "cidr": "10.0.0.0/8"}}
				]
			}`,
			wantErr:       false,
			wantLinkCount: 2,
			wantLinks: []struct {
				ip         string
				subnetCidr string
				subnetID   int
			}{
				{"192.168.1.100", "192.168.1.0/24", 1},
				{"10.0.0.50", "10.0.0.0/8", 2},
			},
		},
		{
			name: "EmptyLinks_SyntheticLinksCreated",
			jsonData: `{
				"id": 5,
				"name": "eth4",
				"mac_address": "aa:bb:cc:dd:ee:01",
				"tags": ["172.16.0.100/12"],
				"links": []
			}`,
			wantErr:       false,
			wantLinkCount: 1,
			wantLinks: []struct {
				ip         string
				subnetCidr string
				subnetID   int
			}{
				// net.ParseCIDR("172.16.0.100/12") normalises to "172.16.0.0/12"
				{"172.16.0.100", "172.16.0.0/12", 0},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var iface Interface
			err := json.Unmarshal([]byte(tc.jsonData), &iface)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(iface.Links) != tc.wantLinkCount {
				t.Fatalf("len(Links) = %d, want %d (Links=%v)", len(iface.Links), tc.wantLinkCount, iface.Links)
			}
			for i, wl := range tc.wantLinks {
				got := iface.Links[i]
				if got.IPAddress != wl.ip {
					t.Errorf("Links[%d].IPAddress = %q, want %q", i, got.IPAddress, wl.ip)
				}
				if got.Subnet.Cidr != wl.subnetCidr {
					t.Errorf("Links[%d].Subnet.Cidr = %q, want %q", i, got.Subnet.Cidr, wl.subnetCidr)
				}
				if got.Subnet.ID != wl.subnetID {
					t.Errorf("Links[%d].Subnet.ID = %d, want %d", i, got.Subnet.ID, wl.subnetID)
				}
			}
		})
	}
}
