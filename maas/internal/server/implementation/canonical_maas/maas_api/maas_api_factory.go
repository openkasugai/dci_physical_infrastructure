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
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_fabrics"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_interfaces"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_ipaddresses"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_ipranges"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_machines"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_subnets"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/api_vmhosts"
	"maas_module/internal/server/interfaces"
	"maas_module/internal/server/interfaces/maas_api"

	"k8s.io/klog/v2"
)

// MaasAPIFactoryImple implements the maas_api.MaasApiFactory interface.
type MaasAPIFactoryImple struct {
	API    interfaces.MaasAPI
	Logger klog.Logger
}

// NewSubnets creates a new instance of Subnets with the provided arguments.
func (l MaasAPIFactoryImple) NewSubnets(args ...interface{}) maas_api.BasisMaasAPI {
	klog.V(2).InfoS("start NewSubnets", "args", args)
	result := &api_subnets.Subnets{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
	}
	klog.V(2).InfoS("end NewSubnets", "result", "Subnets instance created")
	return result
}

// NewVMHosts creates a new instance of VMhosts with the provided arguments.
func (l MaasAPIFactoryImple) NewVMHosts(args ...interface{}) maas_api.BasisMaasAPI {
	klog.V(2).InfoS("start NewVMHosts", "args", args)
	result := &api_vmhosts.VMhosts{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
	}
	klog.V(2).InfoS("end NewVMHosts", "result", "VMhosts instance created")
	return result
}

// NewVMHostHostID creates a new instance of VMhostHostID with the provided host ID.
func (l MaasAPIFactoryImple) NewVMHostHostID(args ...interface{}) maas_api.BasisMaasAPI {
	klog.V(2).InfoS("start NewVMHostHostID", "args", args)
	if len(args) == 0 {
		klog.V(2).InfoS("branch: insufficient arguments provided")
		return nil
	}

	hostID, ok := args[0].(int)
	if !ok {
		klog.V(2).InfoS("branch: invalid hostID type", "type", args[0])
		return nil
	}

	result := &api_vmhosts.VMhostHostID{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
		HostID: hostID,
	}
	klog.V(2).InfoS("end NewVMHostHostID", "hostID", hostID, "result", "VMhostHostID instance created")
	return result
}

// NewVMHostCompose creates a new instance of VMhostCompose with the provided host ID.
func (l MaasAPIFactoryImple) NewVMHostCompose(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_vmhosts.VMhostCompose{
		VMhostHostID: api_vmhosts.VMhostHostID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			HostID: args[0].(int),
		},
	}
}

// NewVMHostRefresh creates a new instance of VMhostRefresh with the provided host ID.
func (l MaasAPIFactoryImple) NewVMHostRefresh(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_vmhosts.VMhostRefresh{
		VMhostHostID: api_vmhosts.VMhostHostID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			HostID: args[0].(int),
		},
	}
}

// NewVMHostParameters creates a new instance of VMhostParameters with the provided host ID.
func (l MaasAPIFactoryImple) NewVMHostParameters(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_vmhosts.VMhostParameters{
		VMhostHostID: api_vmhosts.VMhostHostID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			HostID: args[0].(int),
		},
	}
}

// NewInterfaces creates a new instance of Interfaces with the provided system ID.
func (l MaasAPIFactoryImple) NewInterfaces(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_interfaces.Interfaces{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
		SystemID: args[0].(string),
	}
}

// NewInterfaceLink creates a new instance of InterfaceLink with the provided system ID and interface ID.
func (l MaasAPIFactoryImple) NewInterfaceLink(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_interfaces.InterfaceLinkSubnet{
		Interfaces: api_interfaces.Interfaces{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
		InterfaceID: args[1].(int),
	}
}

// NewInterfaceDisconnect creates a new instance of InterfaceDisconnect with the provided system ID and interface ID.
func (l MaasAPIFactoryImple) NewInterfaceDisconnect(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_interfaces.InterfaceDisconnect{
		Interfaces: api_interfaces.Interfaces{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
		InterfaceID: args[1].(int),
	}
}

// NewInterfaceAddTag creates a new instance of InterfaceAddTag with the provided system ID and interface ID.
func (l MaasAPIFactoryImple) NewInterfaceAddTag(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_interfaces.InterfaceAddTag{
		Interfaces: api_interfaces.Interfaces{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
		InterfaceID: args[1].(int),
	}
}

// NewInterfaceRemoveTag creates a new instance of InterfaceRemoveTag with the provided system ID and interface ID.
func (l MaasAPIFactoryImple) NewInterfaceRemoveTag(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_interfaces.InterfaceRemoveTag{
		Interfaces: api_interfaces.Interfaces{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
		InterfaceID: args[1].(int),
	}
}

// NewInterfaceUpdate creates a new instance of InterfaceUpdate with the provided system ID and interface ID.
func (l MaasAPIFactoryImple) NewInterfaceUpdate(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_interfaces.InterfaceUpdate{
		Interfaces: api_interfaces.Interfaces{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
		InterfaceID: args[1].(int),
	}
}

// NewMachines creates a new instance of Machines with the provided arguments.
func (l MaasAPIFactoryImple) NewMachines(args ...interface{}) maas_api.BasisMaasAPI {
	klog.V(2).InfoS("start NewMachines", "args", args)
	result := &api_machines.Machines{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
	}
	klog.V(2).InfoS("end NewMachines", "result", "Machines instance created")
	return result
}

// NewMachineSystemID creates a new instance of MachineSystemID with the provided system ID.
func (l MaasAPIFactoryImple) NewMachineSystemID(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_machines.MachineSystemID{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
		SystemID: args[0].(string),
	}
}

// NewMachineRelease creates a new instance of MachineRelease with the provided system ID.
func (l MaasAPIFactoryImple) NewMachineRelease(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_machines.MachineRelease{
		MachineSystemID: api_machines.MachineSystemID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
	}
}

// NewMachineDeploy creates a new instance of MachineDeploy with the provided system ID.
func (l MaasAPIFactoryImple) NewMachineDeploy(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_machines.MachineDeploy{
		MachineSystemID: api_machines.MachineSystemID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
	}
}

// NewMachineCommission creates a new instance of MachineCommission with the provided system ID.
func (l MaasAPIFactoryImple) NewMachineCommission(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_machines.MachineCommission{
		MachineSystemID: api_machines.MachineSystemID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
	}
}

// NewMachineAbort creates a new instance of MachineAbort with the provided system ID.
func (l MaasAPIFactoryImple) NewMachineAbort(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_machines.MachineAbort{
		MachineSystemID: api_machines.MachineSystemID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
	}
}

// NewMachinePowerON creates a new instance of MachinePowerON with the provided system ID.
func (l MaasAPIFactoryImple) NewMachinePowerON(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_machines.MachinePowerON{
		MachineSystemID: api_machines.MachineSystemID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
	}
}

// NewMachinePowerOFF creates a new instance of MachinePowerOFF with the provided system ID.
func (l MaasAPIFactoryImple) NewMachinePowerOFF(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_machines.MachinePowerOFF{
		MachineSystemID: api_machines.MachineSystemID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
	}
}

// NewMachineMarkBroken creates a new instance of MachineMarkBroken with the provided system ID.
func (l MaasAPIFactoryImple) NewMachineMarkBroken(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_machines.MachineMarkBroken{
		MachineSystemID: api_machines.MachineSystemID{
			AbstractMaas: maas_api.AbstractMaas{
				API:    l.API,
				Logger: l.Logger,
			},
			SystemID: args[0].(string),
		},
	}
}

// NewIPRanges creates a new instance of IPRanges with the provided arguments.
func (l MaasAPIFactoryImple) NewIPRanges(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_ipranges.IPranges{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
	}
}

// NewFabrics creates a new instance of Fabrics with the provided arguments.
func (l MaasAPIFactoryImple) NewFabrics(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_fabrics.Fabrics{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
	}
}

// NewIPAddressReserve creates a new instance of IPAddressReserve with the provided arguments.
func (l MaasAPIFactoryImple) NewIPAddressReserve(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_ipaddresses.IPAddressReserve{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
	}
}

// NewIPAddressRelease creates a new instance of IPAddressRelease with the provided arguments.
func (l MaasAPIFactoryImple) NewIPAddressRelease(args ...interface{}) maas_api.BasisMaasAPI {
	return &api_ipaddresses.IPAddressRelease{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
	}
}

// NewSubnetUnreservedIPRanges creates a new instance of SubnetUnreservedIPRanges with the provided subnet ID.
func (l MaasAPIFactoryImple) NewSubnetUnreservedIPRanges(args ...interface{}) maas_api.BasisMaasAPI {
	if len(args) == 0 {
		klog.V(2).InfoS("branch: insufficient arguments provided")
		return nil
	}

	subnetID, ok := args[0].(int)
	if !ok {
		klog.V(2).InfoS("branch: invalid subnetID type", "type", args[0])
		return nil
	}

	return &api_subnets.SubnetUnreservedIPRanges{
		AbstractMaas: maas_api.AbstractMaas{
			API:    l.API,
			Logger: l.Logger,
		},
		SubnetID: subnetID,
	}
}
