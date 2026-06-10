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

package maas_api

import (
	"context"
	"testing"

	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_fabrics"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_interfaces"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_ipaddresses"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_ipranges"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_machines"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_subnets"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_vmhosts"
	"maas_module/internal/server/test_utils"
)

// MockCanonicalMaasApi is a mock implementation of CanonicalMaasApi interface for testing
type MockCanonicalMaasApi struct{}

func (m *MockCanonicalMaasApi) APIExecute(ctx context.Context, method, endpoint, body string) (int, []byte, error) {
	return 200, []byte("{}"), nil
}

func TestMaasAPIFactoryImple_NewSubnets(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Act
	result := factory.NewSubnets()

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	subnet, ok := result.(*api_subnets.Subnets)
	if !ok {
		t.Fatal("Expected result to be of type *api_subnets.Subnets")
	}

	if subnet.API == nil {
		t.Error("Expected API to be set")
	}
}

func TestMaasAPIFactoryImple_NewVMHosts(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Act
	result := factory.NewVMHosts()

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	vmhosts, ok := result.(*api_vmhosts.VMhosts)
	if !ok {
		t.Fatal("Expected result to be of type *api_vmhosts.VMhosts")
	}

	if vmhosts.API == nil {
		t.Error("Expected API to be set")
	}
}

func TestMaasAPIFactoryImple_NewVMHostHostID(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	hostID := 123

	// Act
	result := factory.NewVMHostHostID(hostID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	vmhostHostID, ok := result.(*api_vmhosts.VMhostHostID)
	if !ok {
		t.Fatal("Expected result to be of type *api_vmhosts.VMhostHostID")
	}

	if vmhostHostID.API == nil {
		t.Error("Expected API to be set")
	}

	if vmhostHostID.HostID != hostID {
		t.Errorf("Expected HostID to be %d, got %d", hostID, vmhostHostID.HostID)
	}
}

func TestMaasAPIFactoryImple_NewVMHostCompose(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	hostID := 456

	// Act
	result := factory.NewVMHostCompose(hostID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	vmhostCompose, ok := result.(*api_vmhosts.VMhostCompose)
	if !ok {
		t.Fatal("Expected result to be of type *api_vmhosts.VMhostCompose")
	}

	if vmhostCompose.API == nil {
		t.Error("Expected API to be set")
	}

	if vmhostCompose.HostID != hostID {
		t.Errorf("Expected HostID to be %d, got %d", hostID, vmhostCompose.HostID)
	}
}

func TestMaasAPIFactoryImple_NewVMHostRefresh(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	hostID := 789

	// Act
	result := factory.NewVMHostRefresh(hostID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	vmhostRefresh, ok := result.(*api_vmhosts.VMhostRefresh)
	if !ok {
		t.Fatal("Expected result to be of type *api_vmhosts.VMhostRefresh")
	}

	if vmhostRefresh.API == nil {
		t.Error("Expected API to be set")
	}

	if vmhostRefresh.HostID != hostID {
		t.Errorf("Expected HostID to be %d, got %d", hostID, vmhostRefresh.HostID)
	}
}

func TestMaasAPIFactoryImple_NewVMHostParameters(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	hostID := 101112

	// Act
	result := factory.NewVMHostParameters(hostID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	vmhostParams, ok := result.(*api_vmhosts.VMhostParameters)
	if !ok {
		t.Fatal("Expected result to be of type *api_vmhosts.VMhostParameters")
	}

	if vmhostParams.API == nil {
		t.Error("Expected API to be set")
	}

	if vmhostParams.HostID != hostID {
		t.Errorf("Expected HostID to be %d, got %d", hostID, vmhostParams.HostID)
	}
}

func TestMaasAPIFactoryImple_NewInterfaces(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-system-id"

	// Act
	result := factory.NewInterfaces(systemID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	interfaces, ok := result.(*api_interfaces.Interfaces)
	if !ok {
		t.Fatal("Expected result to be of type *api_interfaces.Interfaces")
	}

	if interfaces.API == nil {
		t.Error("Expected API to be set")
	}

	if interfaces.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, interfaces.SystemID)
	}
}

func TestMaasAPIFactoryImple_NewInterfaceLink(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-system-id"
	interfaceID := 1

	// Act
	result := factory.NewInterfaceLink(systemID, interfaceID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	interfaceLink, ok := result.(*api_interfaces.InterfaceLinkSubnet)
	if !ok {
		t.Fatal("Expected result to be of type *api_interfaces.InterfaceLinkSubnet")
	}

	if interfaceLink.API == nil {
		t.Error("Expected API to be set")
	}

	if interfaceLink.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, interfaceLink.SystemID)
	}

	if interfaceLink.InterfaceID != interfaceID {
		t.Errorf("Expected InterfaceID to be %d, got %d", interfaceID, interfaceLink.InterfaceID)
	}
}

func TestMaasAPIFactoryImple_NewInterfaceDisconnect(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-system-id"
	interfaceID := 2

	// Act
	result := factory.NewInterfaceDisconnect(systemID, interfaceID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	interfaceDisconnect, ok := result.(*api_interfaces.InterfaceDisconnect)
	if !ok {
		t.Fatal("Expected result to be of type *api_interfaces.InterfaceDisconnect")
	}

	if interfaceDisconnect.API == nil {
		t.Error("Expected API to be set")
	}

	if interfaceDisconnect.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, interfaceDisconnect.SystemID)
	}

	if interfaceDisconnect.InterfaceID != interfaceID {
		t.Errorf("Expected InterfaceID to be %d, got %d", interfaceID, interfaceDisconnect.InterfaceID)
	}
}

func TestMaasAPIFactoryImple_NewInterfaceAddTag(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-system-id"
	interfaceID := 1

	// Act
	result := factory.NewInterfaceAddTag(systemID, interfaceID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	interfaceAddTag, ok := result.(*api_interfaces.InterfaceAddTag)
	if !ok {
		t.Fatal("Expected result to be of type *api_interfaces.InterfaceAddTag")
	}

	if interfaceAddTag.API == nil {
		t.Error("Expected API to be set")
	}

	if interfaceAddTag.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, interfaceAddTag.SystemID)
	}

	if interfaceAddTag.InterfaceID != interfaceID {
		t.Errorf("Expected InterfaceID to be %d, got %d", interfaceID, interfaceAddTag.InterfaceID)
	}
}

func TestMaasAPIFactoryImple_NewInterfaceRemoveTag(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-system-id"
	interfaceID := 1

	// Act
	result := factory.NewInterfaceRemoveTag(systemID, interfaceID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	interfaceRemoveTag, ok := result.(*api_interfaces.InterfaceRemoveTag)
	if !ok {
		t.Fatal("Expected result to be of type *api_interfaces.InterfaceRemoveTag")
	}

	if interfaceRemoveTag.API == nil {
		t.Error("Expected API to be set")
	}

	if interfaceRemoveTag.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, interfaceRemoveTag.SystemID)
	}

	if interfaceRemoveTag.InterfaceID != interfaceID {
		t.Errorf("Expected InterfaceID to be %d, got %d", interfaceID, interfaceRemoveTag.InterfaceID)
	}
}

func TestMaasAPIFactoryImple_NewInterfaceUpdate(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-system-id"
	interfaceID := 9

	result := factory.NewInterfaceUpdate(systemID, interfaceID)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	interfaceUpdate, ok := result.(*api_interfaces.InterfaceUpdate)
	if !ok {
		t.Fatal("Expected result to be of type *api_interfaces.InterfaceUpdate")
	}
	if interfaceUpdate.API == nil {
		t.Error("Expected API to be set")
	}
	if interfaceUpdate.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, interfaceUpdate.SystemID)
	}
	if interfaceUpdate.InterfaceID != interfaceID {
		t.Errorf("Expected InterfaceID to be %d, got %d", interfaceID, interfaceUpdate.InterfaceID)
	}
}

func TestMaasAPIFactoryImple_NewMachines(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Act
	result := factory.NewMachines()

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	machines, ok := result.(*api_machines.Machines)
	if !ok {
		t.Fatal("Expected result to be of type *api_machines.Machines")
	}

	if machines.API == nil {
		t.Error("Expected API to be set")
	}
}

func TestMaasAPIFactoryImple_NewMachineSystemID(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-machine-id"

	// Act
	result := factory.NewMachineSystemID(systemID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	machineSystemID, ok := result.(*api_machines.MachineSystemID)
	if !ok {
		t.Fatal("Expected result to be of type *api_machines.MachineSystemID")
	}

	if machineSystemID.API == nil {
		t.Error("Expected API to be set")
	}

	if machineSystemID.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, machineSystemID.SystemID)
	}
}

func TestMaasAPIFactoryImple_NewMachineRelease(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-machine-release-id"

	// Act
	result := factory.NewMachineRelease(systemID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	machineRelease, ok := result.(*api_machines.MachineRelease)
	if !ok {
		t.Fatal("Expected result to be of type *api_machines.MachineRelease")
	}

	if machineRelease.API == nil {
		t.Error("Expected API to be set")
	}

	if machineRelease.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, machineRelease.SystemID)
	}
}

func TestMaasAPIFactoryImple_NewMachineDeploy(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-machine-deploy-id"

	// Act
	result := factory.NewMachineDeploy(systemID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	machineDeploy, ok := result.(*api_machines.MachineDeploy)
	if !ok {
		t.Fatal("Expected result to be of type *api_machines.MachineDeploy")
	}

	if machineDeploy.API == nil {
		t.Error("Expected API to be set")
	}

	if machineDeploy.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, machineDeploy.SystemID)
	}
}

func TestMaasAPIFactoryImple_NewMachineCommission(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}
	systemID := "test-machine-commission-id"

	// Act
	result := factory.NewMachineCommission(systemID)

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	machineCommission, ok := result.(*api_machines.MachineCommission)
	if !ok {
		t.Fatal("Expected result to be of type *api_machines.MachineCommission")
	}

	if machineCommission.API == nil {
		t.Error("Expected API to be set")
	}

	if machineCommission.SystemID != systemID {
		t.Errorf("Expected SystemID to be %s, got %s", systemID, machineCommission.SystemID)
	}
}

func TestMaasAPIFactoryImple_NewIPRanges(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Act
	result := factory.NewIPRanges()

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	ipranges, ok := result.(*api_ipranges.IPranges)
	if !ok {
		t.Fatal("Expected result to be of type *api_ipranges.IPranges")
	}

	if ipranges.API == nil {
		t.Error("Expected API to be set")
	}
}

func TestMaasAPIFactoryImple_NewFabrics(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Act
	result := factory.NewFabrics()

	// Assert
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	fabrics, ok := result.(*api_fabrics.Fabrics)
	if !ok {
		t.Fatal("Expected result to be of type *api_fabrics.Fabrics")
	}

	if fabrics.API == nil {
		t.Error("Expected API to be set")
	}
}

// TestMaasAPIFactoryImple_WithNilAPI tests edge case with nil API
func TestMaasAPIFactoryImple_WithNilAPI(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	factory := MaasAPIFactoryImple{API: nil}

	// Act & Assert for each factory method
	result := factory.NewSubnets()
	if result == nil {
		t.Error("Expected non-nil result even with nil API")
	}
}

// Test with multiple arguments for variadic functions
func TestMaasAPIFactoryImple_WithMultipleArgs(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Test with extra arguments (should still work)
	result := factory.NewSubnets("extra", "args", 123)
	if result == nil {
		t.Error("Expected non-nil result with extra args")
	}
}

// TestMaasAPIFactoryImple_NewMachineAbort tests NewMachineAbort method
func TestMaasAPIFactoryImple_NewMachineAbort(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Act
	result := factory.NewMachineAbort("test-system-id")

	// Assert
	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Verify the correct type
	machineAbort, ok := result.(*api_machines.MachineAbort)
	if !ok {
		t.Error("Expected MachineAbort type")
	}

	if machineAbort.SystemID != "test-system-id" {
		t.Errorf("Expected system ID 'test-system-id', got '%s'", machineAbort.SystemID)
	}
}

// TestMaasAPIFactoryImple_NewMachineMarkBroken tests NewMachineMarkBroken method
func TestMaasAPIFactoryImple_NewMachineMarkBroken(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Act
	result := factory.NewMachineMarkBroken("test-system-id")

	// Assert
	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Verify the correct type
	machineMarkBroken, ok := result.(*api_machines.MachineMarkBroken)
	if !ok {
		t.Error("Expected MachineMarkBroken type")
	}

	if machineMarkBroken.SystemID != "test-system-id" {
		t.Errorf("Expected system ID 'test-system-id', got '%s'", machineMarkBroken.SystemID)
	}
}

// TestMaasAPIFactoryImple_NewVMHostHostID_WithDifferentArgs tests NewVMHostHostID with different arguments
func TestMaasAPIFactoryImple_NewVMHostHostID_WithDifferentArgs(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	// Test with string host ID - should return nil because it expects int
	result1 := factory.NewVMHostHostID("test-host-1")
	if result1 != nil {
		t.Error("Expected nil result with string host ID")
	}

	// Test with integer host ID - should succeed
	result2 := factory.NewVMHostHostID(123)
	if result2 == nil {
		t.Error("Expected non-nil result with int host ID")
	}

	// Test with no arguments - should return nil because args are required
	result3 := factory.NewVMHostHostID()
	if result3 != nil {
		t.Error("Expected nil result with no args")
	}

	// Test with multiple arguments - should use first int arg
	result4 := factory.NewVMHostHostID(456, "host2", "extra")
	if result4 == nil {
		t.Error("Expected non-nil result with multiple args starting with int")
	}
}

// Benchmark tests for performance
func BenchmarkMaasAPIFactoryImple_NewMachines(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	for i := 0; i < b.N; i++ {
		result := factory.NewMachines()
		if result == nil {
			b.Fatal("Expected non-nil result")
		}
	}
}

func BenchmarkMaasAPIFactoryImple_NewSubnets(b *testing.B) {
	mockAPI := &MockCanonicalMaasApi{}
	factory := MaasAPIFactoryImple{API: mockAPI}

	for i := 0; i < b.N; i++ {
		result := factory.NewSubnets()
		if result == nil {
			b.Fatal("Expected non-nil result")
		}
	}
}

// TestMaasAPIFactoryImple_NewIPAddressReserve tests NewIPAddressReserve factory method
func TestMaasAPIFactoryImple_NewIPAddressReserve(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{}
factory := MaasAPIFactoryImple{API: mockAPI}

result := factory.NewIPAddressReserve()

if result == nil {
t.Fatal("Expected non-nil result")
}

// Type assertion to verify correct type
_, ok := result.(*api_ipaddresses.IPAddressReserve)
if !ok {
t.Fatal("Expected result to be of type *api_ipaddresses.IPAddressReserve")
}
}

// TestMaasAPIFactoryImple_NewIPAddressRelease tests NewIPAddressRelease factory method
func TestMaasAPIFactoryImple_NewIPAddressRelease(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{}
factory := MaasAPIFactoryImple{API: mockAPI}

result := factory.NewIPAddressRelease()

if result == nil {
t.Fatal("Expected non-nil result")
}

// Type assertion to verify correct type
_, ok := result.(*api_ipaddresses.IPAddressRelease)
if !ok {
t.Fatal("Expected result to be of type *api_ipaddresses.IPAddressRelease")
}
}

// TestMaasAPIFactoryImple_NewSubnetUnreservedIPRanges tests NewSubnetUnreservedIPRanges factory method
func TestMaasAPIFactoryImple_NewSubnetUnreservedIPRanges(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{}
factory := MaasAPIFactoryImple{API: mockAPI}
subnetID := 123

result := factory.NewSubnetUnreservedIPRanges(subnetID)

if result == nil {
t.Fatal("Expected non-nil result")
}

// Type assertion and SubnetID verification
subnetUnreserved, ok := result.(*api_subnets.SubnetUnreservedIPRanges)
if !ok {
t.Fatal("Expected result to be of type *api_subnets.SubnetUnreservedIPRanges")
}

if subnetUnreserved.SubnetID != subnetID {
t.Errorf("Expected SubnetID to be %d, got %d", subnetID, subnetUnreserved.SubnetID)
}
}

// TestMaasAPIFactoryImple_NewSubnetUnreservedIPRanges_InsufficientArgs tests with insufficient arguments
func TestMaasAPIFactoryImple_NewSubnetUnreservedIPRanges_InsufficientArgs(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{}
factory := MaasAPIFactoryImple{API: mockAPI}

result := factory.NewSubnetUnreservedIPRanges() // No arguments

if result != nil {
t.Error("Expected nil result for insufficient arguments")
}
}

// TestMaasAPIFactoryImple_NewSubnetUnreservedIPRanges_InvalidArgType tests with invalid argument type
func TestMaasAPIFactoryImple_NewSubnetUnreservedIPRanges_InvalidArgType(t *testing.T) {
cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
defer cleanup()

mockAPI := &MockCanonicalMaasApi{}
factory := MaasAPIFactoryImple{API: mockAPI}

result := factory.NewSubnetUnreservedIPRanges("invalid-type") // String instead of int

if result != nil {
t.Error("Expected nil result for invalid argument type")
}
}
