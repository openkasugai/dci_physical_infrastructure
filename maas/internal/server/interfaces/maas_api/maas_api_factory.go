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

// MaasAPIFactory defines the interface for creating instances of various Canonical MAAS API components.
type MaasAPIFactory interface {
	NewSubnets(args ...interface{}) BasisMaasAPI
	NewVMHosts(args ...interface{}) BasisMaasAPI
	NewVMHostHostID(args ...interface{}) BasisMaasAPI
	NewVMHostCompose(args ...interface{}) BasisMaasAPI
	NewVMHostRefresh(args ...interface{}) BasisMaasAPI
	NewVMHostParameters(args ...interface{}) BasisMaasAPI
	NewInterfaces(args ...interface{}) BasisMaasAPI
	NewInterfaceLink(args ...interface{}) BasisMaasAPI
	NewInterfaceDisconnect(args ...interface{}) BasisMaasAPI
	NewInterfaceAddTag(args ...interface{}) BasisMaasAPI
	NewInterfaceRemoveTag(args ...interface{}) BasisMaasAPI
	NewInterfaceUpdate(args ...interface{}) BasisMaasAPI
	NewMachines(args ...interface{}) BasisMaasAPI
	NewMachineSystemID(args ...interface{}) BasisMaasAPI
	NewMachineRelease(args ...interface{}) BasisMaasAPI
	NewMachineDeploy(args ...interface{}) BasisMaasAPI
	NewMachineCommission(args ...interface{}) BasisMaasAPI
	NewMachineAbort(args ...interface{}) BasisMaasAPI
	NewMachinePowerON(args ...interface{}) BasisMaasAPI
	NewMachinePowerOFF(args ...interface{}) BasisMaasAPI
	NewMachineMarkBroken(args ...interface{}) BasisMaasAPI
	NewIPRanges(args ...interface{}) BasisMaasAPI
	NewFabrics(args ...interface{}) BasisMaasAPI
	NewIPAddressReserve(args ...interface{}) BasisMaasAPI
	NewIPAddressRelease(args ...interface{}) BasisMaasAPI
	NewSubnetUnreservedIPRanges(args ...interface{}) BasisMaasAPI
}
