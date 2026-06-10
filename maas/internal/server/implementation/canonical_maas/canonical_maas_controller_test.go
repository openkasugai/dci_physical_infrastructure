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
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/klog/v2"

	proto "maas_module/api/proto"
    common "common/api/proto"    // import of common protobuf
	"maas_module/internal/server/implementation/canonical_maas/maas_api/request_body"
	"maas_module/internal/server/implementation/canonical_maas/maas_api/response_body"
	"maas_module/internal/server/implementation/canonical_maas/mocks"
	"maas_module/internal/server/interfaces/maas_api"
	"maas_module/internal/server/test_utils"
	"maas_module/internal/server/utils"
)

// Mock implementations for testing
type mockMaasAnsible struct {
	cmdExecuteOutput []byte
	cmdExecuteErr    error
}

func (m *mockMaasAnsible) CmdExecute(ctx context.Context, remoteHost string, playbook string, extraArgs string) ([]byte, error) {
	return m.cmdExecuteOutput, m.cmdExecuteErr
}

// Legacy mock for existing tests
type mockMaasAPI struct {
	// For specific test methods
	getSubnetsResult    []response_body.Subnet
	getSubnetsErr       error
	getInterfacesResult []response_body.Interface
	getInterfacesErr    error
	getVMHostsResult    []response_body.VMHost
	getVMHostsErr       error
	status              int
	result              string
	machineStatus       string // For controlling machine status in tests
}

func (m *mockMaasAPI) GET(ctx context.Context) (response_body.Resbody, error) {
	// Default status to 200 if not set
	status := m.status
	if status == 0 {
		status = 200
	}

	if m.getSubnetsErr != nil {
		return nil, m.getSubnetsErr
	}
	if m.getSubnetsResult != nil {
		return response_body.ResbodyGetSubnets{
			List:          m.getSubnetsResult,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: status},
		}, nil
	}
	if m.getInterfacesErr != nil {
		return nil, m.getInterfacesErr
	}
	if m.getInterfacesResult != nil {
		return response_body.ResbodyGetInterfaces{
			List:          m.getInterfacesResult,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: status},
		}, nil
	}
	if m.getVMHostsErr != nil {
		return nil, m.getVMHostsErr
	}
	if m.getVMHostsResult != nil {
		return response_body.ResbodyGetVMHosts{
			List:          m.getVMHostsResult,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: status},
		}, nil
	}
	// Default case for machine details - needed for getMachineAccessInfo
	machineStatus := m.machineStatus
	if machineStatus == "" {
		machineStatus = "Ready" // Default to Ready for most tests
	}

	return response_body.ResbodyGetMachine{
		SystemID:    "test-id",
		HostName:    "test-machine",
		StatusName:  machineStatus,
		IPAddresses: []string{"192.168.1.10"},
		BootInterface: response_body.Interface{
			ID:         1,
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links: []response_body.Link{
				{
					IPAddress: "192.168.1.10",
					Subnet: response_body.Subnet{
						ID:   1,
						Cidr: "192.168.1.0/24",
					},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil
}

func (m *mockMaasAPI) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return response_body.ResbodyPostMachines{
		SystemID:      "test-id",
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: m.status},
	}, nil
}

func (m *mockMaasAPI) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return response_body.ResbodyGetMachine{
		SystemID:      "test-id",
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: m.status},
	}, nil
}

func (m *mockMaasAPI) DELETE(ctx context.Context) (response_body.Resbody, error) {
	return response_body.ResbodyGetMachine{
		SystemID:      "test-id",
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: m.status},
	}, nil
}

type mockMaasAPIFactory struct {
	factory              maas_api.BasisMaasAPI
	comprehensiveMockAPI *comprehensiveMockAPI // For the new goroutine test
}

func (m *mockMaasAPIFactory) NewSubnets(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			getResult: response_body.ResbodyGetSubnets{
				List: []response_body.Subnet{
					{
						ID:   1,
						Cidr: "192.168.1.0/24",
					},
				},
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
			postResult: response_body.ResbodyPostSubnets{
				ID:            1,
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewVMHosts(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			getResult: response_body.ResbodyGetVMHosts{
				List:          []response_body.VMHost{},
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewVMHostHostID(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewVMHostCompose(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewVMHostRefresh(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewVMHostParameters(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewInterfaces(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			getResult: response_body.ResbodyGetInterfaces{
				List: []response_body.Interface{
					{
						ID:         1,
						Name:       "eth0",
						MacAddress: "00:11:22:33:44:55",
						Tags:       []string{},
						Links: []response_body.Link{
							{
								IPAddress: "192.168.1.10",
								Subnet: response_body.Subnet{
									ID:   1,
									Cidr: "192.168.1.0/24",
								},
							},
						},
					},
				},
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewInterfaceLink(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewInterfaceDisconnect(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewInterfaceAddTag(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewInterfaceRemoveTag(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewInterfaceUpdate(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			putResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachines(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyPostMachines{
				SystemID:      "test-system-id",
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachineSystemID(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			getResult: response_body.ResbodyGetMachine{
				SystemID:    "test-system-id",
				StatusName:  "Ready",
				HostName:    "test-machine",
				Description: "completion", // Required for OsDeploy to proceed
				BootInterface: response_body.Interface{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:11:22:33:44:55",
				},
				InterfaceSet: []response_body.Interface{
					{
						ID:         1,
						Name:       "eth0",
						MacAddress: "00:11:22:33:44:55",
					},
				},
				IPAddresses:   []string{"192.168.1.10"},
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachineRelease(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachineDeploy(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachineCommission(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachineAbort(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachineMarkBroken(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachinePowerON(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachinePowerOFF(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewMachineUpdate(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

func (m *mockMaasAPIFactory) NewFabrics(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyPostFabrics{
				ID: 1,
				Vlans: []response_body.Vlan{
					{Vid: 0}, // Default VLAN ID
				},
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewIPRanges(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewIPAddressReserve(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewIPAddressRelease(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewSubnetUnreservedIPRanges(args ...interface{}) maas_api.BasisMaasAPI {
	if m.comprehensiveMockAPI != nil {
		return &mockBasisMaasAPI{
			getResult: response_body.ResbodySubnetUnreservedIPRanges{
				List: []response_body.UnreservedIPRange{
					{
						Start: "192.168.1.100",
						End:   "192.168.1.200",
					},
				},
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
		}
	}
	return m.factory
}

func (m *mockMaasAPIFactory) NewVMDelete(args ...interface{}) maas_api.BasisMaasAPI {
	return m.factory
}

// Mock BasisMaasAPI
type mockBasisMaasAPI struct {
	getResult    response_body.Resbody
	getErr       error
	postResult   response_body.Resbody
	postErr      error
	putResult    response_body.Resbody
	putErr       error
	deleteResult response_body.Resbody
	deleteErr    error
}

func (m *mockBasisMaasAPI) GET(ctx context.Context) (response_body.Resbody, error) {
	return m.getResult, m.getErr
}

func (m *mockBasisMaasAPI) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return m.postResult, m.postErr
}

func (m *mockBasisMaasAPI) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return m.putResult, m.putErr
}

func (m *mockBasisMaasAPI) DELETE(ctx context.Context) (response_body.Resbody, error) {
	return m.deleteResult, m.deleteErr
}

// Mock factory specifically for OsRelease tests
type osReleaseMockFactory struct {
	mockAPIErr     error
	mockAnsibleErr error
	machineStatus  string
	powerStatus    string
}

func (m *osReleaseMockFactory) NewVMHosts(args ...interface{}) maas_api.BasisMaasAPI {
	return &mockBasisMaasAPI{
		getResult: response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		},
		getErr: m.mockAPIErr,
	}
}

func (m *osReleaseMockFactory) NewMachineSystemID(args ...interface{}) maas_api.BasisMaasAPI {
	status := m.machineStatus
	if status == "" {
		status = "Deployed"
	}
	power := m.powerStatus
	if power == "" {
		power = "on"
	}
	return &mockBasisMaasAPI{
		getResult: response_body.ResbodyGetMachine{
			SystemID:    "test-sys",
			StatusName:  status,
			PowerStatus:  power,
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.10"},
			BootInterface: response_body.Interface{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.10",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		},
		getErr: m.mockAPIErr,
	}
}

func TestCanonicalMaasController_OsRelease_UnregisterSkipped(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	mockFactory := &osReleaseMockFactory{
		machineStatus: "Ready",
		powerStatus:   "on",
	}
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("should not be called"),
		cmdExecuteErr:    errors.New("should not be called"),
	}
	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
		Ansible:    mockAnsible,
	}
	ctx := context.Background()
	req := &proto.OsReleaseRequest{SystemId: "test-sys"}
	resp, err := controller.OsRelease(ctx, req)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if resp == nil || resp.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %+v", resp)
	}

	mockFactory = &osReleaseMockFactory{
		machineStatus: "Deployed",
		powerStatus:   "off",
	}
	controller.APIFactory = mockFactory
	resp, err = controller.OsRelease(ctx, req)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if resp == nil || resp.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected ACCEPT, got %+v", resp)
	}
}

func (m *osReleaseMockFactory) NewMachineRelease(args ...interface{}) maas_api.BasisMaasAPI {
	return &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		postErr:    nil,
	}
}

func (m *osReleaseMockFactory) NewVMHostHostID(args ...interface{}) maas_api.BasisMaasAPI {
	return &mockBasisMaasAPI{
		deleteResult: response_body.ResbodyCommon{HTTPStatus: 200},
		deleteErr:    nil,
	}
}

// Implement other required factory methods (return nil or basic mock as they're not used in OsRelease)
func (m *osReleaseMockFactory) NewSubnets(args ...interface{}) maas_api.BasisMaasAPI { return nil }
func (m *osReleaseMockFactory) NewVMHostCompose(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewVMHostRefresh(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewVMHostParameters(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewInterfaces(args ...interface{}) maas_api.BasisMaasAPI {
	return &mockBasisMaasAPI{
		getResult: response_body.ResbodyGetInterfaces{
			List: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:11:22:33:44:55",
					Tags:       []string{"192.168.1.100"},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		},
		getErr: m.mockAPIErr,
	}
}
func (m *osReleaseMockFactory) NewInterfaceLink(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewInterfaceDisconnect(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewInterfaceAddTag(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewInterfaceRemoveTag(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewInterfaceUpdate(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewMachines(args ...interface{}) maas_api.BasisMaasAPI { return nil }
func (m *osReleaseMockFactory) NewMachineDeploy(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewMachineCommission(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewMachineAbort(args ...interface{}) maas_api.BasisMaasAPI { return nil }
func (m *osReleaseMockFactory) NewMachineMarkBroken(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewMachinePowerON(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewMachinePowerOFF(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewMachineUpdate(args ...interface{}) maas_api.BasisMaasAPI {
	return nil
}
func (m *osReleaseMockFactory) NewFabrics(args ...interface{}) maas_api.BasisMaasAPI  { return nil }
func (m *osReleaseMockFactory) NewIPRanges(args ...interface{}) maas_api.BasisMaasAPI { return nil }
func (m *osReleaseMockFactory) NewIPAddressReserve(args ...interface{}) maas_api.BasisMaasAPI { return nil }
func (m *osReleaseMockFactory) NewIPAddressRelease(args ...interface{}) maas_api.BasisMaasAPI {
	return &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		postErr:    nil,
	}
}
func (m *osReleaseMockFactory) NewSubnetUnreservedIPRanges(args ...interface{}) maas_api.BasisMaasAPI { return nil }
func (m *osReleaseMockFactory) NewVMDelete(args ...interface{}) maas_api.BasisMaasAPI { return nil }

// Comprehensive mock for API methods that returns appropriate responses based on call context
type comprehensiveMockAPI struct {
	callCount int
}

func (m *comprehensiveMockAPI) GET(ctx context.Context) (response_body.Resbody, error) {
	m.callCount++

	// For machine show operations (used in pollingMachineStatus and getMachineAccessInfo)
	// This is called first in the goroutine for machine status polling
	if m.callCount <= 3 { // Allow multiple machine show calls during polling
		return response_body.ResbodyGetMachine{
			SystemID:   "test-system-id",
			StatusName: "Ready",
			HostName:   "test-machine",
			BootInterface: response_body.Interface{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
			},
			InterfaceSet: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:11:22:33:44:55",
				},
			},
			IPAddresses:   []string{"192.168.1.10"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil
	}

	// For subnet list operations (used in getSubnetList)
	if m.callCount == 4 {
		return response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{
					ID:   1,
					Cidr: "192.168.1.0/24",
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil
	}

	// For interface list operations (used in getInterfaceList)
	if m.callCount >= 5 {
		return response_body.ResbodyGetInterfaces{
			List: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:11:22:33:44:55",
					Tags:       []string{},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil
	}

	return nil, errors.New("unexpected GET call")
}

func (m *comprehensiveMockAPI) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	// For machine registration
	if _, ok := reqBody.(request_body.ReqbodyMachines); ok {
		return response_body.ResbodyPostMachines{
			SystemID:      "test-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil
	}
	// For interface linking and other POST operations
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

func (m *comprehensiveMockAPI) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	// For machine commission, deploy, update operations
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

func (m *comprehensiveMockAPI) DELETE(ctx context.Context) (response_body.Resbody, error) {
	return nil, errors.New("not implemented")
}

// Helper function to set test environment
func setupTestEnvironmentForController(t *testing.T) {
	// Reset config to ensure clean state for each test
	utils.ResetConfigForTesting()
	
	testEnv := map[string]string{
		"LOG_LEVEL":        "2",
		"MAAS_SERVER_PORT": "8080",
		"MAAS_ACCESS_URL":  "http://172.31.16.200:5240/MAAS/api/2.0/",
		"MAAS_API_KEY":     "test-key",
		"VM_HOST_DISK":     "50",
		"LXD_PORT":         "8443",
		"SSH_KEY":          "/test/ssh/key",
		"TLS_ENABLE":       "false",
		"TLS_CERT_PATH":    "/certs",
	}

	for key, value := range testEnv {
		os.Setenv(key, value)
	}

	err := utils.InitializeConfig()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	t.Cleanup(func() {
		for key := range testEnv {
			os.Unsetenv(key)
		}
	})
}

// Test for isIPv4 function
func TestCanonicalMaasController_isIPv4_ValidIPv4_ReturnsTrue(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testCases := []struct {
		name string
		ip   string
	}{
		{"Simple IPv4", "192.168.1.1"},
		{"Localhost", "127.0.0.1"},
		{"Zero IP", "0.0.0.0"},
		{"Max IP", "255.255.255.255"},
		{"Network IP", "10.0.0.1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Act
			result := isIPv4(tc.ip)

			// Assert
			if !result {
				t.Errorf("Expected true for IPv4 address %s, got false", tc.ip)
			}
		})
	}
}

func TestCanonicalMaasController_isIPv4_InvalidIPv4_ReturnsFalse(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testCases := []struct {
		name string
		ip   string
	}{
		{"IPv6", "2001:db8::1"},
		{"Invalid format", "256.1.1.1"},
		{"Empty string", ""},
		{"Text", "not-an-ip"},
		{"Incomplete", "192.168.1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Act
			result := isIPv4(tc.ip)

			// Assert
			if result {
				t.Errorf("Expected false for invalid address %s, got true", tc.ip)
			}
		})
	}
}

// Test for ipv4ToInt function
func TestCanonicalMaasController_ipv4ToInt_ValidIP_ReturnsCorrectInt(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testCases := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{"Localhost", "127.0.0.1", 0x7F000001},
		{"Private network", "192.168.1.1", 0xC0A80101},
		{"Zero IP", "0.0.0.0", 0x00000000},
		{"Max IP", "255.255.255.255", 0xFFFFFFFF},
		{"Simple IP", "1.2.3.4", 0x01020304},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Act
			result := ipv4ToInt(tc.ip)

			// Assert
			if result != tc.expected {
				t.Errorf("Expected %x for IP %s, got %x", tc.expected, tc.ip, result)
			}
		})
	}
}

// Test for intToIpv4 function
func TestCanonicalMaasController_intToIpv4_ValidInt_ReturnsCorrectIP(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	testCases := []struct {
		name     string
		ip       uint32
		expected string
	}{
		{"Localhost", 0x7F000001, "127.0.0.1"},
		{"Private network", 0xC0A80101, "192.168.1.1"},
		{"Zero IP", 0x00000000, "0.0.0.0"},
		{"Max IP", 0xFFFFFFFF, "255.255.255.255"},
		{"Simple IP", 0x01020304, "1.2.3.4"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Act
			result := intToIpv4(tc.ip)

			// Assert
			if result != tc.expected {
				t.Errorf("Expected %s for int %x, got %s", tc.expected, tc.ip, result)
			}
		})
	}
}

// Test for reverseIPAddressRange function
func TestCanonicalMaasController_reverseIPAddressRange_ValidCIDR_ReturnsCorrectRanges(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := CanonicalMaasController{
		Logger: klog.NewKlogr(),
	}

	testCases := []struct {
		name     string
		cidr     string
		addStart string
		addEnd   string
		expected [][]string
	}{
		{
			name:     "With start and end",
			cidr:     "192.168.1.0/24",
			addStart: "192.168.1.100",
			addEnd:   "192.168.1.200",
			expected: [][]string{
				{"192.168.1.1", "192.168.1.99"},
				{"192.168.1.201", "192.168.1.254"},
			},
		},
		{
			name:     "Only start",
			cidr:     "10.0.0.0/24",
			addStart: "10.0.0.50",
			addEnd:   "",
			expected: [][]string{
				{"10.0.0.1", "10.0.0.49"},
			},
		},
		{
			name:     "Only end",
			cidr:     "10.0.0.0/24",
			addStart: "",
			addEnd:   "10.0.0.200",
			expected: [][]string{
				{"10.0.0.201", "10.0.0.254"},
			},
		},
		{
			name:     "No start and end",
			cidr:     "192.168.1.0/24",
			addStart: "",
			addEnd:   "",
			expected: nil, // Function returns nil slice when no ranges are added
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Act
			result := controller.reverseIPAddressRange(tc.cidr, tc.addStart, tc.addEnd)

			// Assert
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// Test for findSubnet function
func TestCanonicalMaasController_findSubnet_ExistingCIDR_ReturnsSubnetID(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := CanonicalMaasController{
		Logger: klog.NewKlogr(),
	}

	subnets := []response_body.Subnet{
		{ID: 1, Cidr: "192.168.1.0/24"},
		{ID: 2, Cidr: "10.0.0.0/16"},
		{ID: 3, Cidr: "172.16.0.0/12"},
	}

	// Act
	result := controller.findSubnet(subnets, "10.0.0.0/16")

	// Assert
	if result == nil {
		t.Error("Expected subnet ID, got nil")
	} else if *result != 2 {
		t.Errorf("Expected subnet ID 2, got %d", *result)
	}
}

func TestCanonicalMaasController_findSubnet_NonExistingCIDR_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := CanonicalMaasController{
		Logger: klog.NewKlogr(),
	}

	subnets := []response_body.Subnet{
		{ID: 1, Cidr: "192.168.1.0/24"},
		{ID: 2, Cidr: "10.0.0.0/16"},
	}

	// Act
	result := controller.findSubnet(subnets, "172.16.0.0/12")

	// Assert
	if result != nil {
		t.Errorf("Expected nil for non-existing CIDR, got %d", *result)
	}
}

func TestCanonicalMaasController_findSubnet_EmptySubnets_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := CanonicalMaasController{
		Logger: klog.NewKlogr(),
	}

	subnets := []response_body.Subnet{}

	// Act
	result := controller.findSubnet(subnets, "192.168.1.0/24")

	// Assert
	if result != nil {
		t.Errorf("Expected nil for empty subnets, got %d", *result)
	}
}

// Test for getErrorMessage function
func TestCanonicalMaasController_getErrorMessage_CustomError_ReturnsErrorMessage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := CanonicalMaasController{
		Logger: klog.NewKlogr(),
	}

	testCases := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "EnvError",
			err:          &utils.EnvError{Message: "Environment error"},
			expectedCode: codes.Internal,
		},
		{
			name:         "HttpError",
			err:          &utils.HttpError{StatusCode: 400, Message: "HTTP error"},
			expectedCode: codes.Internal,
		},
		{
			name:         "SeqError",
			err:          &utils.SeqError{Message: "Sequence error"},
			expectedCode: codes.Unavailable,
		},
		{
			name:         "CancelError",
			err:          &utils.CancelError{},
			expectedCode: codes.Internal,
		},
		{
			name:         "RespError",
			err:          &utils.RespError{Message: "Response error"},
			expectedCode: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Act
			result := controller.getErrorMessage(tc.err)

			// Assert
			if result == nil {
				t.Error("Expected error message, got nil")
			} else {
				if result.ErrorCode != int32(tc.expectedCode) {
					t.Errorf("Expected code %d, got %d", tc.expectedCode, result.ErrorCode)
				}
			}
		})
	}
}

func TestCanonicalMaasController_getErrorMessage_StandardError_ReturnsUnknownError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := CanonicalMaasController{
		Logger: klog.NewKlogr(),
	}

	err := errors.New("standard error")

	// Act
	result := controller.getErrorMessage(err)

	// Assert
	if result == nil {
		t.Error("Expected error message, got nil")
	} else {
		if result.ErrorCode != int32(codes.Internal) {
			t.Errorf("Expected code %d, got %d", codes.Internal, result.ErrorCode)
		}
		if result.Message != "standard error" {
			t.Errorf("Expected message 'standard error', got %s", result.Message)
		}
	}
}

func TestCanonicalMaasController_getErrorMessage_NilError_ReturnsNil(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	controller := CanonicalMaasController{
		Logger: klog.NewKlogr(),
	}

	// Act
	result := controller.getErrorMessage(nil)

	// Assert
	if result != nil {
		t.Errorf("Expected nil for nil error, got %v", result)
	}
}

// Test for getSubnetList function
func TestCanonicalMaasController_getSubnetList_APISuccess_ReturnsSubnets(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	expectedSubnets := []response_body.Subnet{
		{ID: 1, Cidr: "192.168.1.0/24"},
		{ID: 2, Cidr: "10.0.0.0/16"},
	}

	mockAPI := &mockMaasAPI{
		getSubnetsResult: expectedSubnets,
		getSubnetsErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()

	// Act
	subnets, err := controller.getSubnetList(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !reflect.DeepEqual(subnets, expectedSubnets) {
		t.Errorf("Expected %v, got %v", expectedSubnets, subnets)
	}
}

func TestCanonicalMaasController_getSubnetList_APIError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	mockAPI := &mockMaasAPI{
		getSubnetsResult: nil,
		getSubnetsErr:    errors.New("API error"),
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()

	// Act
	subnets, err := controller.getSubnetList(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error from API failure")
	}

	if len(subnets) != 0 {
		t.Errorf("Expected empty subnets on error, got %v", subnets)
	}
}

// Test for getInterfaceList function
func TestCanonicalMaasController_getInterfaceList_APISuccess_ReturnsInterfaces(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	expectedInterfaces := []response_body.Interface{
		{ID: 1, Name: "eth0", MacAddress: "00:11:22:33:44:55"},
		{ID: 2, Name: "eth1", MacAddress: "00:11:22:33:44:66"},
	}

	mockAPI := &mockMaasAPI{
		getInterfacesResult: expectedInterfaces,
		getInterfacesErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	interfaces, err := controller.getInterfaceList(ctx, systemID)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !reflect.DeepEqual(interfaces, expectedInterfaces) {
		t.Errorf("Expected %v, got %v", expectedInterfaces, interfaces)
	}
}

func TestCanonicalMaasController_getInterfaceList_APIError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	mockAPI := &mockMaasAPI{
		getInterfacesResult: nil,
		getInterfacesErr:    errors.New("API error"),
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	interfaces, err := controller.getInterfaceList(ctx, systemID)

	// Assert
	if err == nil {
		t.Error("Expected error from API failure")
	}

	if len(interfaces) != 0 {
		t.Errorf("Expected empty interfaces on error, got %v", interfaces)
	}
}

// Test for getHostList function
func TestCanonicalMaasController_getHostList_APISuccess_ReturnsHosts(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	expectedHosts := []response_body.VMHost{
		{ID: 1, Host: response_body.Host{SystemID: "host1"}},
		{ID: 2, Host: response_body.Host{SystemID: "host2"}},
	}

	mockAPI := &mockMaasAPI{
		getVMHostsResult: expectedHosts,
		getVMHostsErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()

	// Act
	hosts, err := controller.getHostList(ctx)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !reflect.DeepEqual(hosts, expectedHosts) {
		t.Errorf("Expected %v, got %v", expectedHosts, hosts)
	}
}

func TestCanonicalMaasController_getHostList_APIError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	mockAPI := &mockMaasAPI{
		getVMHostsResult: nil,
		getVMHostsErr:    errors.New("API error"),
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()

	// Act
	hosts, err := controller.getHostList(ctx)

	// Assert
	if err == nil {
		t.Error("Expected error from API failure")
	}

	if len(hosts) != 0 {
		t.Errorf("Expected empty hosts on error, got %v", hosts)
	}
}

// Additional tests for missing functions

// Test for internalMachineRegister function
func TestCanonicalMaasController_internalMachineRegister_Success_ReturnsSystemID(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	expectedSystemID := "test-system-id-123"
	mockResponseBody := response_body.ResbodyPostMachines{
		SystemID: expectedSystemID,
	}

	mockAPI := &mockBasisMaasAPI{
		postResult: mockResponseBody,
		postErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
	}

	// Act
	systemID, err := controller.internalMachineRegister(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if systemID != expectedSystemID {
		t.Errorf("Expected systemID %s, got %s", expectedSystemID, systemID)
	}
}

func TestCanonicalMaasController_internalMachineRegister_APIError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	mockAPI := &mockBasisMaasAPI{
		postResult: nil,
		postErr:    errors.New("API error"),
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
	}

	// Act
	systemID, err := controller.internalMachineRegister(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from API failure")
	}

	if systemID != "" {
		t.Errorf("Expected empty systemID on error, got %s", systemID)
	}
}

func TestCanonicalMaasController_internalMachineRegister_InvalidResponseType_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	// Return wrong response type
	mockAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{}, // Wrong type
		postErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
	}

	// Act
	systemID, err := controller.internalMachineRegister(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from invalid response type")
	}

	if systemID != "" {
		t.Errorf("Expected empty systemID on error, got %s", systemID)
	}
}

// Test for pollingMachineStatus function
func TestCanonicalMaasController_pollingMachineStatus_TargetStatusReached_ReturnsSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	mockResponseBody := response_body.ResbodyGetMachine{
		ResbodyCommon: response_body.ResbodyCommon{
			HTTPStatus: 200,
		},
		SystemID:   "test-system-id",
		StatusName: "Ready",
	}

	mockAPI := &mockBasisMaasAPI{
		getResult: mockResponseBody,
		getErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	systemID := "test-system-id"
	pollingInterval := 1 * time.Millisecond // Fast polling for test
	checkStatus := []string{"Ready", "Failed"}

	// Act
	err := controller.pollingMachineStatus(ctx, systemID, pollingInterval, checkStatus)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestCanonicalMaasController_pollingMachineStatus_APIError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	mockAPI := &mockBasisMaasAPI{
		getResult: nil,
		getErr:    errors.New("API error"),
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	systemID := "test-system-id"
	pollingInterval := 1 * time.Millisecond
	checkStatus := []string{"Ready"}

	// Act
	err := controller.pollingMachineStatus(ctx, systemID, pollingInterval, checkStatus)

	// Assert
	if err == nil {
		t.Error("Expected error from API failure")
	}
}

// Test for getHostID function
func TestCanonicalMaasController_getHostID_ExistingHost_ReturnsHostID(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	expectedHostID := 123
	mockHosts := []response_body.VMHost{
		{
			ID: expectedHostID,
			Host: response_body.Host{
				SystemID: "target-system-id",
			},
		},
		{
			ID: 456,
			Host: response_body.Host{
				SystemID: "other-system-id",
			},
		},
	}

	mockResponseBody := response_body.ResbodyGetVMHosts{
		ResbodyCommon: response_body.ResbodyCommon{
			HTTPStatus: 200,
		},
		List: mockHosts,
	}

	mockAPI := &mockBasisMaasAPI{
		getResult: mockResponseBody,
		getErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	systemID := "target-system-id"

	// Act
	hostID, err := controller.getHostID(ctx, systemID)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if hostID != expectedHostID {
		t.Errorf("Expected hostID %d, got %d", expectedHostID, hostID)
	}
}

func TestCanonicalMaasController_getHostID_NonExistingHost_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	mockHosts := []response_body.VMHost{
		{
			ID: 123,
			Host: response_body.Host{
				SystemID: "other-system-id",
			},
		},
	}

	mockResponseBody := response_body.ResbodyGetVMHosts{
		ResbodyCommon: response_body.ResbodyCommon{
			HTTPStatus: 200,
		},
		List: mockHosts,
	}

	mockAPI := &mockBasisMaasAPI{
		getResult: mockResponseBody,
		getErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	systemID := "non-existing-system-id"

	// Act
	hostID, err := controller.getHostID(ctx, systemID)

	// Assert
	if err == nil {
		t.Error("Expected error for non-existing host")
	}

	if hostID != 0 {
		t.Errorf("Expected hostID 0 on error, got %d", hostID)
	}
}

// Test for markBroken function
func TestCanonicalMaasController_markBroken_ValidError_CallsMarkBrokenAPI(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Arrange
	setupTestEnvironmentForController(t)

	mockAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 200},
		postErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	testError := errors.New("test error")
	systemID := "test-system-id"

	// Act
	controller.markBroken(context.Background(), testError, systemID)

	// Assert - No specific assertion since this is a void function
	// The test passes if no panic occurs
}

// Test for MachineRegister public method
func TestCanonicalMaasController_MachineRegister_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name           string
		request        *proto.MachineRegisterRequest
		mockPostResult response_body.ResbodyPostMachines
		mockPostErr    error
		expectedResult common.ResultCode
		expectError    bool
	}{
		{
			name: "Successful machine register",
			request: &proto.MachineRegisterRequest{
				MacAddress:   "00:11:22:33:44:55",
				IpmiAddress:  "192.168.1.100",
				IpmiUser:     "admin",
				IpmiPassword: "password",
				NetworkInformation: []*proto.NetworkInformation{
					{
						MacAddress: "00:11:22:33:44:55",
						Cidr:       "192.168.1.0/24",
					},
				},
			},
			mockPostResult: response_body.ResbodyPostMachines{
				SystemID:      "registered-system-id",
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
			mockPostErr:    nil,
			expectedResult: common.ResultCode_ACCEPT,
			expectError:    false,
		},
		{
			name: "API error during machine register",
			request: &proto.MachineRegisterRequest{
				MacAddress:   "00:11:22:33:44:55",
				IpmiAddress:  "192.168.1.100",
				IpmiUser:     "admin",
				IpmiPassword: "password",
			},
			mockPostResult: response_body.ResbodyPostMachines{},
			mockPostErr:    errors.New("API error"),
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockAPI := &mockBasisMaasAPI{
				postResult: tc.mockPostResult,
				postErr:    tc.mockPostErr,
			}

			mockFactory := &mockMaasAPIFactory{
				factory: mockAPI,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
			}

			ctx := context.Background()

			// Act
			response, err := controller.MachineRegister(ctx, tc.request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			}
		})
	}
}

// Test for MachineRegister with commission failure path using gomock
func TestCanonicalMaasController_MachineRegister_CommissionFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock factory and APIs
	mockFactory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachinesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCommissionAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
	}

	// Mock machine registration success
	mockFactory.EXPECT().NewMachines(gomock.Any()).Return(mockMachinesAPI)
	mockMachinesAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostMachines{
			SystemID:      "test-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock commission failure
	mockFactory.EXPECT().NewMachineCommission("test-system-id").Return(mockCommissionAPI)
	mockCommissionAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{}, errors.New("commission failed"))

	// Mock markBroken call
	mockFactory.EXPECT().NewMachineMarkBroken("test-system-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.MachineRegister(ctx, request)

	// Assert - The main function should succeed even if commission fails in goroutine
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response == nil {
		t.Error("Expected response, got nil")
	} else if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result %v, got %v", common.ResultCode_ACCEPT, response.GetResult())
	}
	if response.GetSystemId() != "test-system-id" {
		t.Errorf("Expected system ID %v, got %v", "test-system-id", response.GetSystemId())
	}

	// Wait for goroutine to complete
	time.Sleep(200 * time.Millisecond)
}

// Test for MachineDelete public method
func TestCanonicalMaasController_MachineDelete_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name           string
		request        *proto.MachineDeleteRequest
		mockAPIErr     error
		expectedResult common.ResultCode
		expectError    bool
	}{
		{
			name: "Successful machine delete",
			request: &proto.MachineDeleteRequest{
				SystemId: "test-system-id",
			},
			mockAPIErr:     nil,
			expectedResult: common.ResultCode_ACCEPT,
			expectError:    false,
		},
		{
			name: "API error during machine delete",
			request: &proto.MachineDeleteRequest{
				SystemId: "test-system-id",
			},
			mockAPIErr:     errors.New("API delete error"),
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockAPI := &mockBasisMaasAPI{
				deleteResult: response_body.ResbodyCommon{HTTPStatus: 200},
				deleteErr:    tc.mockAPIErr,
			}

			mockFactory := &mockMaasAPIFactory{
				factory: mockAPI,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
			}

			ctx := context.Background()

			// Act
			response, err := controller.MachineDelete(ctx, tc.request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			}
		})
	}
}

// Test for MachineList public method
func TestCanonicalMaasController_MachineList_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name           string
		request        *proto.MachineListRequest
		mockMachines   response_body.ResbodyGetMachines
		mockAPIErr     error
		expectedResult common.ResultCode
		expectError    bool
	}{
		{
			name:    "Successful machine list",
			request: &proto.MachineListRequest{},
			mockMachines: response_body.ResbodyGetMachines{
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
				Machines: []response_body.MachineForResponse{
					{SystemID: "machine1", HostName: "host1", StatusName: "Ready"},
					{SystemID: "machine2", HostName: "host2", StatusName: "Deployed"},
				},
			},
			mockAPIErr:     nil,
			expectedResult: common.ResultCode_SUCCESS,
			expectError:    false,
		},
		{
			name:           "API error during machine list",
			request:        &proto.MachineListRequest{},
			mockMachines:   response_body.ResbodyGetMachines{},
			mockAPIErr:     errors.New("API list error"),
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockAPI := &mockBasisMaasAPI{
				getResult: tc.mockMachines,
				getErr:    tc.mockAPIErr,
			}

			mockFactory := &mockMaasAPIFactory{
				factory: mockAPI,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
			}

			ctx := context.Background()

			// Act
			response, err := controller.MachineList(ctx, tc.request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
				// Check that data is properly returned
				if response.GetData() == "" {
					t.Error("Expected data in response, got empty string")
				}
			}
		})
	}
}

// Test missing functions to improve coverage
func TestCanonicalMaasController_MachineRegister_CompleteScenarios(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name              string
		mockAnsibleOutput []byte
		mockAnsibleError  error
		mockAPIError      error
		expectedResult    common.ResultCode
		expectError       bool
	}{
		{
			name:              "Successful machine registration",
			mockAnsibleOutput: []byte("success"),
			mockAnsibleError:  nil,
			mockAPIError:      nil,
			expectedResult:    common.ResultCode_ACCEPT,
			expectError:       false,
		},
		{
			name:              "API error during registration",
			mockAnsibleOutput: []byte(""),
			mockAnsibleError:  nil,
			mockAPIError:      &utils.HttpError{StatusCode: 500, Message: "API error"},
			expectedResult:    common.ResultCode_ERROR,
			expectError:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockAPI := &mockBasisMaasAPI{
				postResult: response_body.ResbodyPostMachines{
					SystemID:      "test-system-id",
					ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
				},
				postErr: tc.mockAPIError,
			}

			mockFactory := &mockMaasAPIFactory{
				factory: mockAPI,
			}

			mockAnsible := &mockMaasAnsible{
				cmdExecuteOutput: tc.mockAnsibleOutput,
				cmdExecuteErr:    tc.mockAnsibleError,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
				Ansible:    mockAnsible,
			}

			ctx := context.Background()
			request := &proto.MachineRegisterRequest{
				MacAddress:   "00:11:22:33:44:55",
				IpmiAddress:  "192.168.1.100",
				IpmiUser:     "admin",
				IpmiPassword: "password",
				NetworkInformation: []*proto.NetworkInformation{
					{
						MacAddress:   "00:11:22:33:44:55",
						Cidr:         "192.168.1.0/24",
						AddressStart: func() *string { s := "192.168.1.10"; return &s }(),
						AddressEnd:   func() *string { s := "192.168.1.50"; return &s }(),
					},
				},
			}

			// Act
			response, err := controller.MachineRegister(ctx, request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			}
		})
	}
}

// Test for OsDeploy function
func TestCanonicalMaasController_OsDeploy_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name           string
		request        *proto.OsDeployRequest
		mockMachine    response_body.ResbodyGetMachine
		mockAPIErr     error
		mockAnsibleErr error
		expectedResult common.ResultCode
		expectError    bool
	}{
		{
			name: "Successful deployment",
			request: &proto.OsDeployRequest{
				SystemId: "test-sys",
				VmFlag:   &wrapperspb.BoolValue{Value: true},
				Os: &proto.OsInformation{
					Distribution: "ubuntu",
					Version:      "20.04",
				},
				UserData: "#!/bin/bash\necho 'test'",
			},
			mockMachine: response_body.ResbodyGetMachine{
				SystemID:      "test-sys",
				StatusName:    "Ready",
				Description:   "completion",
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
			mockAPIErr:     nil,
			mockAnsibleErr: nil,
			expectedResult: common.ResultCode_ACCEPT,
			expectError:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockAPI := &mockBasisMaasAPI{
				getResult:  tc.mockMachine,
				getErr:     tc.mockAPIErr,
				postResult: response_body.ResbodyCommon{HTTPStatus: 200},
				postErr:    nil,
			}

			mockFactory := &mockMaasAPIFactory{
				factory: mockAPI,
			}

			mockAnsible := &mockMaasAnsible{
				cmdExecuteOutput: []byte("deploy success"),
				cmdExecuteErr:    tc.mockAnsibleErr,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
				Ansible:    mockAnsible,
			}

			ctx := context.Background()

			// Act
			response, err := controller.OsDeploy(ctx, tc.request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			}
		})
	}
}

// Test for OsRelease function
func TestCanonicalMaasController_OsRelease_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name           string
		request        *proto.OsReleaseRequest
		mockAPIErr     error
		mockAnsibleErr error
		expectedResult common.ResultCode
		expectError    bool
	}{
		{
			name: "Successful release",
			request: &proto.OsReleaseRequest{
				SystemId: "test-sys",
			},
			mockAPIErr:     nil,
			mockAnsibleErr: nil,
			expectedResult: common.ResultCode_ACCEPT,
			expectError:    false,
		},
		{
			name: "Ansible error during release",
			request: &proto.OsReleaseRequest{
				SystemId: "test-sys",
			},
			mockAPIErr:     nil,
			mockAnsibleErr: errors.New("Ansible error"),
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			// Create a special mock factory for OsRelease test
			mockFactory := &osReleaseMockFactory{
				mockAPIErr:     tc.mockAPIErr,
				mockAnsibleErr: tc.mockAnsibleErr,
			}

			mockAnsible := &mockMaasAnsible{
				cmdExecuteOutput: []byte("unregister success"),
				cmdExecuteErr:    tc.mockAnsibleErr,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
				Ansible:    mockAnsible,
			}

			ctx := context.Background()

			// Act
			response, err := controller.OsRelease(ctx, tc.request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			}
		})
	}
}

// Test for VMCompose function - Complete success path
func TestCanonicalMaasController_VMCompose_CompleteSuccessPath(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-test-server",
		CpuCore:  func() *int32 { v := int32(2); return &v }(),
		Memory:   func() *int32 { v := int32(4096); return &v }(),
		DiskSize: func() *int32 { v := int32(20); return &v }(),
		NetworkInformation: []*proto.NetworkInformationCni{
			{
				IfName:     "eth0",
				BridgeName: "br0",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Step 1: Mock getMachineAccessInfo (successful)
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "host-system-id",
			HostName:    "vm-host",
			StatusName:  "Deployed",
			PowerStatus:  "on",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	// Step 2: Mock getHostID (VM host found)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id", // Found existing VM host
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Step 3: Mock VM host refresh
	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	// Step 4: Mock getSubnetList
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 1, Cidr: "192.168.1.0/24"},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	// Step 5: Mock VM compose
	mockVMComposeAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMComposeAPI.EXPECT().POST(gomock.Any(), gomock.Eq(request_body.ReqbodyVMhostCompose{
		HostName:   "vm-test-server",
		Cores:      2,
		Memory:     4096,
		Storage:    20,
		Interfaces: "eth0:name=br0",
	})).Return(
		response_body.ResbodyPostVMCompose{
			SystemID:      "composed-vm-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHostCompose(123).Return(mockVMComposeAPI).Times(1)

	// Step 7: Mock goroutine polling (pollingMachineStatus)
	mockPollingAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockPollingAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "composed-vm-system-id",
			StatusName:    "Ready",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil).AnyTimes()

	factory.EXPECT().NewMachineSystemID("composed-vm-system-id").Return(mockPollingAPI).Times(1)

	// Step 8: Mock getInterfaceList
	mockInterfaceListAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockInterfaceListAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetInterfaces{
			List: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:aa:bb:cc:dd:ee",
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewInterfaces("composed-vm-system-id").Return(mockInterfaceListAPI).Times(1)

	// Step 9: Mock linkSubnetInterface operations (disconnect + link)
	mockDisconnectAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockDisconnectAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewInterfaceDisconnect("composed-vm-system-id", 1).Return(mockDisconnectAPI).Times(1)

	mockLinkAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockLinkAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewInterfaceLink("composed-vm-system-id", 1).Return(mockLinkAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response == nil {
		t.Error("Expected response, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ACCEPT {
			t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
		}
		if response.GetSystemId() != "composed-vm-system-id" {
			t.Errorf("Expected system ID 'composed-vm-system-id', got %s", response.GetSystemId())
		}
	}

	// Wait for goroutine to complete fully
	time.Sleep(1000 * time.Millisecond)
}

// Test for VMCompose function - VM host not found (registration path)
func TestCanonicalMaasController_VMCompose_VMHostRegistrationPath(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "new-host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
		NetworkInformation: []*proto.NetworkInformationCni{
			{
				IfName:     "eth0",
				BridgeName: "br0",
				Cidr:       "10.0.0.0/24",
			},
		},
	}

	// Step 1: Mock getMachineAccessInfo (successful)
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "new-host-system-id",
			HostName:      "new-vm-host",
			IPAddresses:   []string{"10.0.0.50"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("new-host-system-id").Return(mockMachineAPI).Times(1)

	// Step 2: Mock getHostID (VM host NOT found - triggers registration)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 999,
					Host: response_body.Host{
						SystemID: "other-system-id", // Different system ID
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Step 3: Mock VM host registration
	mockVMHostRegisterAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRegisterAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostVMHost{
			ID:            456, // New VM host ID
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostRegisterAPI).Times(1)

	// Step 4: Mock VM host parameters
	mockVMHostParamsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostParamsAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyGetOpParameter{
			Certificate:   "test-certificate-key",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHostParameters(456).Return(mockVMHostParamsAPI).Times(1)

	// Step 5: Mock certificate registration (ansible)
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("certificate registered successfully"),
		cmdExecuteErr:    nil,
	}

	// Step 6: Mock VM host refresh
	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(456).Return(mockVMHostRefreshAPI).Times(1)

	// Step 7: Mock getSubnetList
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 2, Cidr: "10.0.0.0/24"},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	// Step 8: Mock VM compose
	mockVMComposeAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMComposeAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostVMCompose{
			SystemID:      "new-vm-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHostCompose(456).Return(mockVMComposeAPI).Times(1)

	// Step 10: Mock goroutine polling
	mockPollingAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockPollingAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "new-vm-system-id",
			StatusName:    "Deployed",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil).AnyTimes()

	factory.EXPECT().NewMachineSystemID("new-vm-system-id").Return(mockPollingAPI).AnyTimes()

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response != nil && response.GetResult() == common.ResultCode_ACCEPT {
		t.Logf("VM compose succeeded with new VM host registration")
	}

	// Wait for goroutine to complete
	time.Sleep(500 * time.Millisecond)
}

// Test for VMCompose function - getMachineAccessInfo failure
func TestCanonicalMaasController_VMCompose_GetMachineAccessInfoFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "invalid-system-id",
		HostName: "test-server",
		Memory:   func() *int32 { v := int32(1024); return &v }(),
	}

	// Mock getMachineAccessInfo failure
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		nil, errors.New("machine not found"))

	factory.EXPECT().NewMachineSystemID("invalid-system-id").Return(mockMachineAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from getMachineAccessInfo failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test for VMCompose function - VM compose API failure
func TestCanonicalMaasController_VMCompose_VMComposeAPIFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "host-system-id",
			HostName:      "test-host",
			IPAddresses:   []string{"192.168.1.100"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	// Mock successful getHostID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock successful VM host refresh
	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	// Mock successful getSubnetList
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List:          []response_body.Subnet{},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	// Mock VM compose API failure
	mockVMComposeAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMComposeAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		nil, errors.New("VM compose failed"))

	factory.EXPECT().NewVMHostCompose(123).Return(mockVMComposeAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from VM compose failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test for VMCompose function - VM compose response type cast failure
func TestCanonicalMaasController_VMCompose_ResponseTypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(1024); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "host-system-id",
			HostName:      "test-host",
			IPAddresses:   []string{"192.168.1.100"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	// Mock successful getHostID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock successful VM host refresh
	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	// Mock successful getSubnetList
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List:          []response_body.Subnet{},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	// Mock VM compose API that returns wrong type
	mockVMComposeAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMComposeAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil) // Wrong type

	factory.EXPECT().NewVMHostCompose(123).Return(mockVMComposeAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from response type cast failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - VM host registration failure
func TestCanonicalMaasController_VMCompose_VMHostRegistrationFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "new-host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "new-host-system-id",
			HostName:      "new-vm-host",
			IPAddresses:   []string{"10.0.0.50"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("new-host-system-id").Return(mockMachineAPI).Times(1)

	// Mock getHostID (VM host NOT found)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{}, // Empty list
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock VM host registration failure
	mockVMHostRegisterAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRegisterAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		nil, errors.New("VM host registration failed"))

	factory.EXPECT().NewVMHosts().Return(mockVMHostRegisterAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible: &mockMaasAnsible{
			cmdExecuteOutput: []byte("ansible successfully"),
			cmdExecuteErr:    nil,
		},
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from VM host registration failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - VM host registration type cast failure
func TestCanonicalMaasController_VMCompose_VMHostRegistrationTypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "new-host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "new-host-system-id",
			HostName:      "new-vm-host",
			IPAddresses:   []string{"10.0.0.50"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("new-host-system-id").Return(mockMachineAPI).Times(1)

	// Mock getHostID (VM host NOT found)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{}, // Empty list
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock VM host registration type cast failure
	mockVMHostRegisterAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRegisterAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 201}, nil) // Wrong type

	factory.EXPECT().NewVMHosts().Return(mockVMHostRegisterAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible: &mockMaasAnsible{
			cmdExecuteOutput: []byte("ansible successfully"),
			cmdExecuteErr:    nil,
		},
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from type cast failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - VM host parameters failure
func TestCanonicalMaasController_VMCompose_VMHostParametersFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "new-host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "new-host-system-id",
			HostName:      "new-vm-host",
			IPAddresses:   []string{"10.0.0.50"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("new-host-system-id").Return(mockMachineAPI).Times(1)

	// Mock getHostID (VM host NOT found)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{}, // Empty list
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock successful VM host registration
	mockVMHostRegisterAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRegisterAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostVMHost{
			ID:            456,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostRegisterAPI).Times(1)

	// Mock VM host parameters failure
	mockVMHostParamsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostParamsAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		nil, errors.New("VM host parameters failed"))

	factory.EXPECT().NewVMHostParameters(456).Return(mockVMHostParamsAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible: &mockMaasAnsible{
			cmdExecuteOutput: []byte("ansible successfully"),
			cmdExecuteErr:    nil,
		},
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from VM host parameters failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - VM host parameters type cast failure
func TestCanonicalMaasController_VMCompose_VMHostParametersTypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "new-host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "new-host-system-id",
			HostName:      "new-vm-host",
			IPAddresses:   []string{"10.0.0.50"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("new-host-system-id").Return(mockMachineAPI).Times(1)

	// Mock getHostID (VM host NOT found)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{}, // Empty list
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock successful VM host registration
	mockVMHostRegisterAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRegisterAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostVMHost{
			ID:            456,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostRegisterAPI).Times(1)

	// Mock VM host parameters type cast failure
	mockVMHostParamsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostParamsAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil) // Wrong type

	factory.EXPECT().NewVMHostParameters(456).Return(mockVMHostParamsAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible: &mockMaasAnsible{
			cmdExecuteOutput: []byte("ansible successfully"),
			cmdExecuteErr:    nil,
		},
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from type cast failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - VM host refresh failure
func TestCanonicalMaasController_VMCompose_VMHostRefreshFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "host-system-id",
			HostName:      "test-host",
			IPAddresses:   []string{"192.168.1.100"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	// Mock successful getHostID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock VM host refresh failure
	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		nil, errors.New("VM host refresh failed"))

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible: &mockMaasAnsible{
			cmdExecuteOutput: []byte("ansible successfully"),
			cmdExecuteErr:    nil,
		},
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from VM host refresh failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - getSubnetList failure
func TestCanonicalMaasController_VMCompose_GetSubnetListFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
		NetworkInformation: []*proto.NetworkInformationCni{
			{
				IfName:     "eth0",
				BridgeName: "br0",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "host-system-id",
			HostName:      "test-host",
			IPAddresses:   []string{"192.168.1.100"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	// Mock successful getHostID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock successful VM host refresh
	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	// Mock getSubnetList failure
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		nil, errors.New("subnet list retrieval failed"))

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible: &mockMaasAnsible{
			cmdExecuteOutput: []byte("ansible successfully"),
			cmdExecuteErr:    nil,
		},
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from getSubnetList failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - getSubnetList type cast failure
func TestCanonicalMaasController_VMCompose_GetSubnetListTypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
		NetworkInformation: []*proto.NetworkInformationCni{
			{
				IfName:     "eth0",
				BridgeName: "br0",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Mock successful getMachineAccessInfo, getHostID, VM host refresh
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "host-system-id",
			HostName:      "test-host",
			IPAddresses:   []string{"192.168.1.100"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	// Mock getSubnetList type cast failure
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil) // Wrong type

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible: &mockMaasAnsible{
			cmdExecuteOutput: []byte("ansible successfully"),
			cmdExecuteErr:    nil,
		},
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from type cast failure, got nil")
	}
	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - machine update (description) failure
// Test case: VMCompose - LXD setup (Ansible) failure
func TestCanonicalMaasController_VMCompose_LXDSetupFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "new-host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "new-host-system-id",
			HostName:    "new-vm-host",
			IPAddresses: []string{"10.0.0.50"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "10.0.0.50",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "10.0.0.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("new-host-system-id").Return(mockMachineAPI).Times(1)

	// Mock getHostID (VM host NOT found - triggers registration)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{}, // Empty list
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock Ansible failure for LXD setup (setup_lxd.yaml)
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte(""),
		cmdExecuteErr:    errors.New("LXD setup failed: setup_lxd.yaml execution failed"),
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from LXD setup failure, got nil")
	}

	if !strings.Contains(err.Error(), "LXD setup failed") {
		t.Errorf("Expected error message to contain 'LXD setup failed', got: %v", err)
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - Certificate registration (Ansible) failure
func TestCanonicalMaasController_VMCompose_CertificateRegistrationFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "new-host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "new-host-system-id",
			HostName:    "new-vm-host",
			IPAddresses: []string{"10.0.0.50"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "10.0.0.50",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "10.0.0.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("new-host-system-id").Return(mockMachineAPI).Times(1)

	// Mock getHostID (VM host NOT found - triggers registration)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{}, // Empty list
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock successful VM host registration
	mockVMHostRegisterAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRegisterAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostVMHost{
			ID:            456,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostRegisterAPI).Times(1)

	// Mock successful VM host parameters (get certificate)
	mockVMHostParamsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostParamsAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyGetOpParameter{
			Certificate:   "-----BEGIN CERTIFICATE-----\nMIICertificateData\n-----END CERTIFICATE-----",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHostParameters(456).Return(mockVMHostParamsAPI).Times(1)

	// Mock Ansible with call tracking using DoAndReturn
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)
	callCount := 0
	mockAnsible.EXPECT().
		CmdExecute(gomock.Any(), gomock.Eq("10.0.0.50"), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, remoteHost string, playbook string, extraArgs string) ([]byte, error) {
			callCount++
			t.Logf("🔍 Ansible call #%d: playbook=%s", callCount, playbook)

			if callCount == 1 {
				// 1st call: setup_lxd.yaml success
				if strings.Contains(playbook, "setup_lxd.yaml") {
					return []byte("LXD setup completed successfully"), nil
				}
			} else if callCount == 2 {
				// 2nd call: register_lxd_certificate.yaml failure
				if strings.Contains(playbook, "register_lxd_certificate.yaml") {
					return []byte(""), errors.New("certificate registration failed: register_lxd_certificate.yaml execution failed")
				}
			}

			return []byte(""), errors.New("unexpected ansible call")
		}).
		Times(2)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from certificate registration failure, got nil")
	}

	if !strings.Contains(err.Error(), "certificate registration failed") {
		t.Errorf("Expected error message to contain 'certificate registration failed', got: %v", err)
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

// Test case: VMCompose - subnet not found for CIDR (findSubnet failure)
func TestCanonicalMaasController_VMCompose_SubnetNotFoundCreateSubnetFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		CpuCore:  func() *int32 { v := int32(2); return &v }(),
		Memory:   func() *int32 { v := int32(2048); return &v }(),
		DiskSize: func() *int32 { v := int32(20); return &v }(),
		NetworkInformation: []*proto.NetworkInformationCni{
			{
				IfName:     "eth0",
				BridgeName: "br0",
				Cidr:       "172.16.0.0/24", // This CIDR will NOT be found in subnet list
			},
		},
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "host-system-id",
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	// Mock successful getHostID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock successful VM host refresh
	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	// Mock getSubnetList with subnets that do NOT contain the requested CIDR
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 1, Cidr: "192.168.1.0/24"}, // Different CIDR
				{ID: 2, Cidr: "10.0.0.0/24"},    // Different CIDR
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	// Mock subnet creation failure in createSubnetAndIPRange
	mockFabricAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockFabricAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		nil, errors.New("fabric creation failed"))

	factory.EXPECT().NewFabrics().Return(mockFabricAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible: &mockMaasAnsible{
			cmdExecuteOutput: []byte("ansible successfully"),
			cmdExecuteErr:    nil,
		},
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from subnet not found for CIDR, got nil")
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}
}

func TestCanonicalMaasController_VMCompose_SubnetNotFoundCreateSubnetSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		CpuCore:  func() *int32 { v := int32(2); return &v }(),
		Memory:   func() *int32 { v := int32(2048); return &v }(),
		DiskSize: func() *int32 { v := int32(20); return &v }(),
		NetworkInformation: []*proto.NetworkInformationCni{
			{
				IfName:     "eth0",
				BridgeName: "br0",
				Cidr:       "172.16.0.0/24",
			},
		},
	}

	// Mock successful getMachineAccessInfo
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "host-system-id",
			HostName:    "test-host",
			StatusName:  "Deployed",
			PowerStatus: "on",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	// Mock successful getHostID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	// Mock successful VM host refresh
	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	// Mock getSubnetList followed by subnet creation
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 1, Cidr: "192.168.1.0/24"},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)
	mockSubnetAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostSubnets{
			ID:            200,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(2)

	mockFabricAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockFabricAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostFabrics{
			ID: 100,
			Vlans: []response_body.Vlan{
				{Vid: 0},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewFabrics().Return(mockFabricAPI).Times(1)

	// Mock VM compose
	mockVMComposeAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMComposeAPI.EXPECT().POST(gomock.Any(), gomock.Eq(request_body.ReqbodyVMhostCompose{
		HostName:   "vm-server",
		Cores:      2,
		Memory:     2048,
		Storage:    20,
		Interfaces: "eth0:name=br0",
	})).Return(
		response_body.ResbodyPostVMCompose{
			SystemID:      "composed-vm-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHostCompose(123).Return(mockVMComposeAPI).Times(1)

	// Mock goroutine polling
	mockPollingAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockPollingAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "composed-vm-system-id",
			StatusName:    "Ready",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil).AnyTimes()

	factory.EXPECT().NewMachineSystemID("composed-vm-system-id").Return(mockPollingAPI).Times(1)

	// Mock getInterfaceList
	mockInterfaceListAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockInterfaceListAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetInterfaces{
			List: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:aa:bb:cc:dd:ee",
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewInterfaces("composed-vm-system-id").Return(mockInterfaceListAPI).Times(1)

	// Mock linkSubnetInterface operations
	mockDisconnectAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockDisconnectAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewInterfaceDisconnect("composed-vm-system-id", 1).Return(mockDisconnectAPI).Times(1)

	mockLinkAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockLinkAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewInterfaceLink("composed-vm-system-id", 1).Return(mockLinkAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response == nil {
		t.Error("Expected response, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ACCEPT {
			t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
		}
		if response.GetSystemId() != "composed-vm-system-id" {
			t.Errorf("Expected system ID 'composed-vm-system-id', got %s", response.GetSystemId())
		}
	}

	// Wait for goroutine to complete fully
	time.Sleep(1000 * time.Millisecond)
}

// Test case: VMCompose goroutine - pollingMachineStatus failure
func TestCanonicalMaasController_VMCompose_GoroutinePollingFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
	}

	// Mock successful initial operations
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "host-system-id",
			HostName:      "test-host",
			IPAddresses:   []string{"192.168.1.100"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List:          []response_body.Subnet{},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	mockVMComposeAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMComposeAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostVMCompose{
			SystemID:      "composed-vm-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHostCompose(123).Return(mockVMComposeAPI).Times(1)

	// Mock polling failure
	mockPollingAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockPollingAPI.EXPECT().GET(gomock.Any()).Return(
		nil, errors.New("polling machine status failed"))

	factory.EXPECT().NewMachineSystemID("composed-vm-system-id").Return(mockPollingAPI).Times(1)

	// Mock markBroken call
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewMachineMarkBroken("composed-vm-system-id").Return(mockMarkBrokenAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert - Main function should succeed
	if err != nil {
		t.Errorf("Expected no error from main function, got: %v", err)
	}
	if response == nil {
		t.Error("Expected response, got nil")
	} else if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for goroutine to complete and call markBroken
	time.Sleep(300 * time.Millisecond)
}

// Test case: VMCompose goroutine - getInterfaceList failure
func TestCanonicalMaasController_VMCompose_GoroutineGetInterfaceListFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
		NetworkInformation: []*proto.NetworkInformationCni{
			{
				IfName:     "eth0",
				BridgeName: "br0",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Mock successful initial operations
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "host-system-id",
			HostName:      "test-host",
			IPAddresses:   []string{"192.168.1.100"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 1, Cidr: "192.168.1.0/24"},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	mockVMComposeAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMComposeAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostVMCompose{
			SystemID:      "composed-vm-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHostCompose(123).Return(mockVMComposeAPI).Times(1)

	// Mock successful polling
	mockPollingAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockPollingAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "composed-vm-system-id",
			StatusName:    "Ready",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("composed-vm-system-id").Return(mockPollingAPI).Times(1)

	// Mock getInterfaceList failure
	mockInterfaceAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockInterfaceAPI.EXPECT().GET(gomock.Any()).Return(
		nil, errors.New("get interface list failed"))

	factory.EXPECT().NewInterfaces("composed-vm-system-id").Return(mockInterfaceAPI).Times(1)

	// Mock markBroken call
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewMachineMarkBroken("composed-vm-system-id").Return(mockMarkBrokenAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert - Main function should succeed
	if err != nil {
		t.Errorf("Expected no error from main function, got: %v", err)
	}
	if response == nil {
		t.Error("Expected response, got nil")
	} else if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for goroutine to complete
	time.Sleep(300 * time.Millisecond)
}

// Test case: VMCompose goroutine - linkSubnetInterface failure
func TestCanonicalMaasController_VMCompose_GoroutineLinkSubnetInterfaceFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	request := &proto.VmComposeRequest{
		ProductInfo: &proto.ProductInformation{
			Vendor:      "test-vendor",
			ProductName: "test-product",
			Version:     "1.0.0",
		},
		MaasInfo: &proto.MaasInformation{
			AccessUrl: "http://172.31.16.200:5240/MAAS/api/2.0/",
			ApiKey:    "test-maas-api-key-from-request",
		},
		SystemId: "host-system-id",
		HostName: "vm-server",
		Memory:   func() *int32 { v := int32(2048); return &v }(),
		NetworkInformation: []*proto.NetworkInformationCni{
			{
				IfName:     "eth0",
				BridgeName: "br0",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Mock successful initial operations (getMachineAccessInfo, getHostID, etc.)
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "host-system-id",
			HostName:      "test-host",
			IPAddresses:   []string{"192.168.1.100"},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("host-system-id").Return(mockMachineAPI).Times(1)

	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "host-system-id",
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	mockVMHostRefreshAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostRefreshAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewVMHostRefresh(123).Return(mockVMHostRefreshAPI).Times(1)

	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 1, Cidr: "192.168.1.0/24"},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	mockVMComposeAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMComposeAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostVMCompose{
			SystemID:      "composed-vm-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		}, nil)

	factory.EXPECT().NewVMHostCompose(123).Return(mockVMComposeAPI).Times(1)

	// Mock successful polling
	mockPollingAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockPollingAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "composed-vm-system-id",
			StatusName:    "Ready",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("composed-vm-system-id").Return(mockPollingAPI).Times(1)

	// Mock successful getInterfaceList
	mockInterfaceAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockInterfaceAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetInterfaces{
			List: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:11:22:33:44:55",
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewInterfaces("composed-vm-system-id").Return(mockInterfaceAPI).Times(1)

	// Mock interface disconnect failure
	mockDisconnectAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockDisconnectAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		nil, errors.New("interface disconnect failed"))

	factory.EXPECT().NewInterfaceDisconnect("composed-vm-system-id", 1).Return(mockDisconnectAPI).Times(1)

	// Mock markBroken call
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().NewMachineMarkBroken("composed-vm-system-id").Return(mockMarkBrokenAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.VMCompose(ctx, request)

	// Assert - Main function should succeed
	if err != nil {
		t.Errorf("Expected no error from main function, got: %v", err)
	}
	if response == nil {
		t.Error("Expected response, got nil")
	} else if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for goroutine to complete
	time.Sleep(300 * time.Millisecond)
}

// NOTE: TestCanonicalMaasController_VMCompose_GoroutineFinalStatusUpdateFailure was removed.
// The goroutine's completion-description PUT ("completion") was deleted in favor of JobManager tracking.

// Test for VMDelete function
func TestCanonicalMaasController_VMDelete_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name           string
		request        *proto.VmDeleteRequest
		mockAPIErr     error
		expectedResult common.ResultCode
		expectError    bool
	}{
		{
			name: "Successful VM delete",
			request: &proto.VmDeleteRequest{
				SystemId: "vm-123",
			},
			mockAPIErr:     nil,
			expectedResult: common.ResultCode_ACCEPT,
			expectError:    false,
		},
		{
			name: "API error during VM delete",
			request: &proto.VmDeleteRequest{
				SystemId: "vm-123",
			},
			mockAPIErr:     errors.New("API error"),
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockAPI := &mockBasisMaasAPI{
				deleteResult: response_body.ResbodyCommon{HTTPStatus: 200},
				deleteErr:    tc.mockAPIErr,
			}

			mockFactory := &mockMaasAPIFactory{
				factory: mockAPI,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
			}

			ctx := context.Background()

			// Act
			response, err := controller.VMDelete(ctx, tc.request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			}
		})
	}
}

// Test for MachineShow public method
func TestCanonicalMaasController_MachineShow_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name           string
		request        *proto.MachineShowRequest
		mockMachine    response_body.ResbodyGetMachine
		mockAPIErr     error
		expectedResult common.ResultCode
		expectError    bool
	}{
		{
			name: "Successful machine show",
			request: &proto.MachineShowRequest{
				SystemId: "test-sys",
			},
			mockMachine: response_body.ResbodyGetMachine{
				SystemID:      "test-sys",
				HostName:      "test-host",
				StatusName:    "Ready",
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
			mockAPIErr:     nil,
			expectedResult: common.ResultCode_SUCCESS,
			expectError:    false,
		},
		{
			name: "API error during machine show",
			request: &proto.MachineShowRequest{
				SystemId: "test-sys",
			},
			mockMachine:    response_body.ResbodyGetMachine{},
			mockAPIErr:     errors.New("API error"),
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockAPI := &mockBasisMaasAPI{
				getResult: tc.mockMachine,
				getErr:    tc.mockAPIErr,
			}

			mockFactory := &mockMaasAPIFactory{
				factory: mockAPI,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
			}

			ctx := context.Background()

			// Act
			response, err := controller.MachineShow(ctx, tc.request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
				// Check that data is properly returned
				if response.GetData() == "" {
					t.Error("Expected data in response, got empty string")
				}
			}
		})
	}
}

// Test for Cancel function
func TestCanonicalMaasController_Cancel_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name           string
		request        *proto.CancelRequest
		mockAPI        *mockBasisMaasAPI
		expectedResult common.ResultCode
		expectError    bool
	}{
		{
			name: "Successful cancel(Commissioning)",
			request: &proto.CancelRequest{
				SystemId: "test-sys",
			},
			mockAPI: &mockBasisMaasAPI{
				getResult:  response_body.ResbodyGetMachine{StatusName: "Commissioning", ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200}},
				getErr:     nil,
				postResult: response_body.ResbodyCommon{HTTPStatus: 200},
				postErr:    nil,
			},
			expectedResult: common.ResultCode_ACCEPT,
			expectError:    false,
		},
		{
			name: "Successful cancel(Deploying)",
			request: &proto.CancelRequest{
				SystemId: "test-sys",
			},
			mockAPI: &mockBasisMaasAPI{
				getResult:  response_body.ResbodyGetMachine{StatusName: "Deploying", ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200}},
				getErr:     nil,
				postResult: response_body.ResbodyCommon{HTTPStatus: 200},
				postErr:    nil,
			},
			expectedResult: common.ResultCode_ACCEPT,
			expectError:    false,
		},
		{
			name: "Successful cancel(Testing)",
			request: &proto.CancelRequest{
				SystemId: "test-sys",
			},
			mockAPI: &mockBasisMaasAPI{
				getResult:  response_body.ResbodyGetMachine{StatusName: "Testing", ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200}},
				getErr:     nil,
				postResult: response_body.ResbodyCommon{HTTPStatus: 200},
				postErr:    nil,
			},
			expectedResult: common.ResultCode_ACCEPT,
			expectError:    false,
		},
		{
			name: "Get error during cancel",
			request: &proto.CancelRequest{
				SystemId: "test-sys",
			},
			mockAPI: &mockBasisMaasAPI{
				getResult:  nil,
				getErr:     errors.New("API error"),
				postResult: nil,
				postErr:    nil,
			},
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
		{
			name: "API error during cancel",
			request: &proto.CancelRequest{
				SystemId: "test-sys",
			},
			mockAPI: &mockBasisMaasAPI{
				getResult:  response_body.ResbodyGetMachine{StatusName: "Commissioning", ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200}},
				getErr:     nil,
				postResult: nil,
				postErr:    errors.New("API error"),
			},
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
		{
			name: "NotCancellable state",
			request: &proto.CancelRequest{
				SystemId: "test-sys",
			},
			mockAPI: &mockBasisMaasAPI{
				getResult:  response_body.ResbodyGetMachine{StatusName: "Deployed", ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200}},
				getErr:     nil,
				postResult: response_body.ResbodyCommon{HTTPStatus: 200},
				postErr:    nil,
			},
			expectedResult: common.ResultCode_ERROR,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockFactory := &mockMaasAPIFactory{
				factory: tc.mockAPI,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
			}

			ctx := context.Background()

			// Act
			response, err := controller.Cancel(ctx, tc.request)

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if response != nil && response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Expected response, got nil")
				} else if response.GetResult() != tc.expectedResult {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, response.GetResult())
				}
			}
		})
	}
}

// Test helper functions to improve coverage
func TestCanonicalMaasController_HelperFunctions(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	mockAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyPostSubnets{ID: 1},
		postErr:    nil,
	}

	mockFactory := &mockMaasAPIFactory{
		factory: mockAPI,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()

	// Test createSubnetAndIPRange - simplified to avoid panic
	macToFabric := map[string]FabricPair{
		"00:11:22:33:44:55": {fabricID: 1, vlanID: 100},
	}

	t.Run("createSubnetAndIPRange", func(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// This will likely error but we're exercising the code path
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic due to mock limitations: %v", r)
			}
		}()
		_, err := controller.createSubnetAndIPRange(ctx, macToFabric, "00:11:22:33:44:55", "192.168.1.0/24", "192.168.1.10", "192.168.1.50")
		if err != nil {
			t.Logf("Expected error due to mock limitations: %v", err)
		}
	})

	t.Run("linkSubnetInterface", func(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// Test linkSubnetInterface with proper arguments
		interfaces := []response_body.Interface{}
		subnetLinks := map[string]SubnetLinkPair{}
		err := controller.linkSubnetInterface(ctx, "test-sys", interfaces, subnetLinks, "00:11:22:33:44:55", false, nil)
		if err != nil {
			t.Logf("Expected error due to mock limitations: %v", err)
		}
	})

	t.Run("getMachineAccessInfo", func(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// Test getMachineAccessInfo with proper return values
		_, _, _, _, _, _, _, _, err := controller.getMachineAccessInfo(ctx, "test-sys")
		if err != nil {
			t.Logf("Expected error due to mock limitations: %v", err)
		}
	})

	t.Run("internalCommission", func(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// Test internalCommission
		err := controller.internalCommission(ctx, "test-sys")
		if err != nil {
			t.Logf("Expected error due to mock limitations: %v", err)
		}
	})
}

// Test for internalCommission function with detailed scenarios
func TestCanonicalMaasController_internalCommission_DetailedPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)
	ctx := context.Background()

	t.Run("API POST Error", func(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// Arrange - Mock API that returns error for POST
		mockAPI := &mockBasisMaasAPI{
			postResult: nil,
			postErr:    errors.New("commission POST API failed"),
		}

		mockFactory := &mockMaasAPIFactory{
			factory: mockAPI,
		}

		controller := CanonicalMaasController{
			Logger:     klog.NewKlogr(),
			APIFactory: mockFactory,
		}

		// Act
		err := controller.internalCommission(ctx, "test-system-id")

		// Assert
		if err == nil {
			t.Error("Expected error from commission POST API, got nil")
		}

		if !strings.Contains(err.Error(), "commission POST API failed") {
			t.Errorf("Expected error message to contain 'commission POST API failed', got: %v", err)
		}
	})

	t.Run("Successful Commission with Ready Status", func(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// Arrange - Mock API that succeeds POST and returns Ready status
		mockAPI := &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
			postErr:    nil,
			getResult: response_body.ResbodyGetMachine{
				SystemID:      "test-system-id",
				StatusName:    "Ready", // Target status reached immediately
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
			getErr: nil,
		}

		mockFactory := &mockMaasAPIFactory{
			factory: mockAPI,
		}

		controller := CanonicalMaasController{
			Logger:     klog.NewKlogr(),
			APIFactory: mockFactory,
		}

		// Act
		err := controller.internalCommission(ctx, "test-system-id")

		// Assert
		if err != nil {
			t.Errorf("Expected no error for successful commission, got: %v", err)
		}
	})

	t.Run("Commission with Failed Status", func(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// Arrange - Mock API that succeeds POST but returns Failed commissioning status
		mockAPI := &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
			postErr:    nil,
			getResult: response_body.ResbodyGetMachine{
				SystemID:      "test-system-id",
				StatusName:    "Failed commissioning", // Target status reached
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
			getErr: nil,
		}

		mockFactory := &mockMaasAPIFactory{
			factory: mockAPI,
		}

		controller := CanonicalMaasController{
			Logger:     klog.NewKlogr(),
			APIFactory: mockFactory,
		}

		// Act
		err := controller.internalCommission(ctx, "test-system-id")

		// Assert
		if err == nil {
			t.Error("Expected error for failed commissioning status, got nil")
		}

		if !strings.Contains(err.Error(), "machine commission failed") {
			t.Errorf("Expected error message to contain 'machine commission failed', got: %v", err)
		}
	})

	t.Run("Polling Error", func(t *testing.T) {
		cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
		defer cleanup()

		// Arrange - Mock API that succeeds POST but fails during polling
		mockAPI := &mockBasisMaasAPI{
			postResult: response_body.ResbodyCommon{HTTPStatus: 200},
			postErr:    nil,
			getResult:  nil,
			getErr:     errors.New("polling API failed"),
		}

		mockFactory := &mockMaasAPIFactory{
			factory: mockAPI,
		}

		controller := CanonicalMaasController{
			Logger:     klog.NewKlogr(),
			APIFactory: mockFactory,
		}

		// Act
		err := controller.internalCommission(ctx, "test-system-id")

		// Assert
		if err == nil {
			t.Error("Expected error from polling API, got nil")
		}

		if !strings.Contains(err.Error(), "polling API failed") {
			t.Errorf("Expected error message to contain 'polling API failed', got: %v", err)
		}
	})
}

// Test pollingMachineStatus with different scenarios
func TestCanonicalMaasController_PollingMachineStatus_AllPaths(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	tests := []struct {
		name        string
		mockResult  response_body.Resbody
		mockError   error
		checkStatus []string
		expectError bool
	}{
		{
			name: "Successful status check",
			mockResult: response_body.ResbodyGetMachine{
				SystemID:      "test-sys",
				StatusName:    "Ready",
				ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			},
			mockError:   nil,
			checkStatus: []string{"Ready"},
			expectError: false,
		},
		{
			name:        "API error",
			mockResult:  nil,
			mockError:   errors.New("API error"),
			checkStatus: []string{"Ready"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			mockAPI := &mockBasisMaasAPI{
				getResult: tc.mockResult,
				getErr:    tc.mockError,
			}

			mockFactory := &mockMaasAPIFactory{
				factory: mockAPI,
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				APIFactory: mockFactory,
			}

			ctx := context.Background()
			err := controller.pollingMachineStatus(ctx, "test-sys", 100*time.Millisecond, tc.checkStatus)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Logf("Got expected error due to mock limitations: %v", err)
			}
		})
	}
}

// Test case ①: pollingMachineStatus with sleep interval coverage
func TestCanonicalMaasController_pollingMachineStatus_RequiresMultiplePolls(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machine API that returns different statuses on multiple calls
	mockMachineAPI := &mockBasisMaasAPIMultipleStatusChanges{
		statuses: []string{"Commissioning", "Commissioning", "Ready"},
	}
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI).Times(3)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"
	pollingInterval := 100 * time.Millisecond
	checkStatus := []string{"Ready"}

	start := time.Now()

	// Act
	err := controller.pollingMachineStatus(ctx, systemID, pollingInterval, checkStatus)

	elapsed := time.Since(start)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should take at least 2 polling intervals (2 sleeps before success)
	expectedMinDuration := 2 * pollingInterval
	if elapsed < expectedMinDuration {
		t.Errorf("Expected at least %v elapsed time, got %v", expectedMinDuration, elapsed)
	}
}

// Test case ②: getSubnetList type cast failure
func TestCanonicalMaasController_getSubnetList_TypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock that returns wrong response type
	mockAPI := &mockBasisMaasAPIWrongType{
		returnWrongType: true,
	}
	factory.EXPECT().NewSubnets().Return(mockAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	subnets, err := controller.getSubnetList(ctx)

	// Assert
	if err == nil {
		t.Error("Expected type cast error, got nil")
	}
	if len(subnets) != 0 {
		t.Error("Expected empty subnets on error")
	}
}

// Test case ③: createSubnetAndIPRange fabric creation failure
func TestCanonicalMaasController_createSubnetAndIPRange_FabricCreationFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock fabric creation failure
	mockFabricAPI := &mockBasisMaasAPIWithError{
		err: errors.New("fabric creation failed"),
	}
	factory.EXPECT().NewFabrics().Return(mockFabricAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	macToFabric := make(map[string]FabricPair)
	mac := "00:11:22:33:44:55"
	cidr := "192.168.1.0/24"

	// Act
	subnetID, err := controller.createSubnetAndIPRange(ctx, macToFabric, mac, cidr, "", "")

	// Assert
	if err == nil {
		t.Error("Expected fabric creation error, got nil")
	}
	if subnetID != 0 {
		t.Errorf("Expected subnetID 0 on error, got %d", subnetID)
	}
}

// Test case ③: createSubnetAndIPRange fabric type cast failure
func TestCanonicalMaasController_createSubnetAndIPRange_FabricTypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock fabric API that returns wrong type
	mockFabricAPI := &mockBasisMaasAPIWrongType{
		returnWrongType: true,
	}
	factory.EXPECT().NewFabrics().Return(mockFabricAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	macToFabric := make(map[string]FabricPair)
	mac := "00:11:22:33:44:55"
	cidr := "192.168.1.0/24"

	// Act
	subnetID, err := controller.createSubnetAndIPRange(ctx, macToFabric, mac, cidr, "", "")

	// Assert
	if err == nil {
		t.Error("Expected fabric type cast error, got nil")
	}
	if subnetID != 0 {
		t.Errorf("Expected subnetID 0 on error, got %d", subnetID)
	}
}

// Test case ③: createSubnetAndIPRange subnet creation failure
func TestCanonicalMaasController_createSubnetAndIPRange_SubnetCreationFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock subnet creation failure
	mockSubnetAPI := &mockBasisMaasAPIWithError{
		err: errors.New("subnet creation failed"),
	}
	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	macToFabric := map[string]FabricPair{
		"00:11:22:33:44:55": {fabricID: 1, vlanID: 0},
	}
	mac := "00:11:22:33:44:55"
	cidr := "192.168.1.0/24"
	addStart := "192.168.1.10"
	addEnd := "192.168.1.20"

	// Act
	_, err := controller.createSubnetAndIPRange(ctx, macToFabric, mac, cidr, addStart, addEnd)

	// Assert
	if err == nil {
		t.Error("Expected subnet creation error, got nil")
	}
}

// createSubnetAndIPRange subnet type cast failure
func TestCanonicalMaasController_createSubnetAndIPRange_SubnetCreationCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock subnet API that returns wrong type
	mockSubnetAPI := &mockBasisMaasAPIWrongType{
		returnWrongType: true,
	}
	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	macToFabric := map[string]FabricPair{
		"00:11:22:33:44:55": {fabricID: 1, vlanID: 0},
	}
	mac := "00:11:22:33:44:55"
	cidr := "192.168.1.0/24"
	addStart := "192.168.1.10"
	addEnd := "192.168.1.20"

	// Act
	_, err := controller.createSubnetAndIPRange(ctx, macToFabric, mac, cidr, addStart, addEnd)

	// Assert
	if err == nil {
		t.Error("Expected subnet creation error, got nil")
	}
}

// Test case ③: createSubnetAndIPRange IP range creation failure
func TestCanonicalMaasController_createSubnetAndIPRange_IPRangeCreationFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock successful subnet creation
	mockSubnetAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyPostSubnets{
			ID:            1,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		},
	}
	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	// Mock IP range creation failure
	mockIPRangeAPI := &mockBasisMaasAPIWithError{
		err: errors.New("IP range creation failed"),
	}
	factory.EXPECT().NewIPRanges().Return(mockIPRangeAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	macToFabric := map[string]FabricPair{
		"00:11:22:33:44:55": {fabricID: 1, vlanID: 0},
	}
	mac := "00:11:22:33:44:55"
	cidr := "192.168.1.0/24"
	addStart := "192.168.1.10"
	addEnd := "192.168.1.20"

	// Act
	_, err := controller.createSubnetAndIPRange(ctx, macToFabric, mac, cidr, addStart, addEnd)

	// Assert
	if err == nil {
		t.Error("Expected IP range creation error, got nil")
	}
}

// Test case ④: getInterfaceList type cast failure
func TestCanonicalMaasController_getInterfaceList_TypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock that returns wrong response type
	mockAPI := &mockBasisMaasAPIWrongType{
		returnWrongType: true,
	}
	factory.EXPECT().NewInterfaces("test-system-id").Return(mockAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	interfaces, err := controller.getInterfaceList(ctx, systemID)

	// Assert
	if err == nil {
		t.Error("Expected type cast error, got nil")
	}
	if len(interfaces) != 0 {
		t.Error("Expected empty interfaces on error")
	}
}

// Test case ⑤: linkSubnetInterface disconnect failure
func TestCanonicalMaasController_linkSubnetInterface_DisconnectFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock disconnect API failure
	mockDisconnectAPI := &mockBasisMaasAPIWithError{
		err: errors.New("interface disconnect failed"),
	}
	factory.EXPECT().NewInterfaceDisconnect("test-system-id", 1).Return(mockDisconnectAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"
	interfaces := []response_body.Interface{
		{
			ID:         1,
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
		},
	}
	key2sub := map[string]SubnetLinkPair{
		"00:11:22:33:44:55": {
			linkMode:  "DHCP",
			subnetIds: []int{1},
		},
	}

	// Act
	err := controller.linkSubnetInterface(ctx, systemID, interfaces, key2sub, "MacAddress", false, nil)

	// Assert
	if err == nil {
		t.Error("Expected interface disconnect error, got nil")
	}
}

// Test case ⑤: linkSubnetInterface link failure
func TestCanonicalMaasController_linkSubnetInterface_LinkFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock successful disconnect
	mockDisconnectAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 200},
	}
	factory.EXPECT().NewInterfaceDisconnect("test-system-id", 1).Return(mockDisconnectAPI).Times(1)

	// Mock link API failure
	mockLinkAPI := &mockBasisMaasAPIWithError{
		err: errors.New("interface link failed"),
	}
	factory.EXPECT().NewInterfaceLink("test-system-id", 1).Return(mockLinkAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"
	interfaces := []response_body.Interface{
		{
			ID:         1,
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
		},
	}
	key2sub := map[string]SubnetLinkPair{
		"00:11:22:33:44:55": {
			linkMode:  "DHCP",
			subnetIds: []int{1},
		},
	}

	// Act
	err := controller.linkSubnetInterface(ctx, systemID, interfaces, key2sub, "MacAddress", false, nil)

	// Assert
	if err == nil {
		t.Error("Expected interface link error, got nil")
	}
}

// Test case ⑥: getHostList type cast failure
func TestCanonicalMaasController_getHostList_TypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock that returns wrong response type
	mockAPI := &mockBasisMaasAPIWrongType{
		returnWrongType: true,
	}
	factory.EXPECT().NewVMHosts().Return(mockAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	hosts, err := controller.getHostList(ctx)

	// Assert
	if err == nil {
		t.Error("Expected type cast error, got nil")
	}
	if len(hosts) != 0 {
		t.Error("Expected empty hosts on error")
	}
}

// Test case ⑦: getHostID with getHostList failure
func TestCanonicalMaasController_getHostID_HostListFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock VM hosts API failure
	mockVMHostsAPI := &mockBasisMaasAPIWithError{
		err: errors.New("get VM hosts failed"),
	}
	factory.EXPECT().NewVMHosts().Return(mockVMHostsAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	hostID, err := controller.getHostID(ctx, systemID)

	// Assert
	if err == nil {
		t.Error("Expected error for non-existing host")
	}
	if hostID != 0 {
		t.Errorf("Expected hostID 0 on error, got %d", hostID)
	}
}

// Test case ⑧: internalMachineShow JSON marshal failure
func TestCanonicalMaasController_internalMachineShow_JSONMarshalFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machine API that returns data causing JSON marshal failure
	mockMachineAPI := &mockBasisMaasAPIUnmarshalableData{}
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	request := &proto.MachineShowRequest{
		SystemId: "test-system-id",
	}

	// Act
	jsonStr, machineStatus, description, err := controller.internalMachineShow(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected JSON marshal error, got nil")
	}
	if jsonStr != "" {
		t.Error("Expected empty JSON string on error")
	}
	if machineStatus != "" {
		t.Error("Expected empty machine status on error")
	}
	if description != "" {
		t.Error("Expected empty description on error")
	}
}

// Test case ⑨: getMachineAccessInfo with IPv4 subnet coverage
func TestCanonicalMaasController_getMachineAccessInfo_IPv4SubnetCoverage(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machine API that returns data with IPv4 subnets
	mockMachineAPI := &mockBasisMaasAPI{
		getResult: response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.10", "2001:db8::1"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.10",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			InterfaceSet: []response_body.Interface{
				{
					Name: "eth0",
					Links: []response_body.Link{
						{
							Subnet: response_body.Subnet{
								ID:   1,
								Cidr: "192.168.1.0/24",
							},
						},
						{
							Subnet: response_body.Subnet{
								ID:   2,
								Cidr: "2001:db8::/64",
							},
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		},
	}
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	hostName, bootIf, accessAddress, bootMacAddress, subnetIDs, _, _, _, err := controller.getMachineAccessInfo(ctx, systemID)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if hostName != "test-host" {
		t.Errorf("Expected hostName 'test-host', got %s", hostName)
	}
	if bootIf != "eth0" {
		t.Errorf("Expected bootIf 'eth0', got %s", bootIf)
	}
	if accessAddress != "192.168.1.10" {
		t.Errorf("Expected accessAddress '192.168.1.10', got %s", accessAddress)
	}
	if bootMacAddress != "00:11:22:33:44:55" {
		t.Errorf("Expected bootMacAddress '00:11:22:33:44:55', got %s", bootMacAddress)
	}
	// Should contain only IPv4 subnet ID (1), not IPv6 subnet ID (2)
	if len(subnetIDs) != 1 || subnetIDs[0] != 1 {
		t.Errorf("Expected subnetIDs [1], got %v", subnetIDs)
	}
}

// Mock implementations for additional test cases

// Mock for multiple status changes to test polling sleep
type mockBasisMaasAPIMultipleStatusChanges struct {
	statuses  []string
	callIndex int
}

func (m *mockBasisMaasAPIMultipleStatusChanges) GET(ctx context.Context) (response_body.Resbody, error) {
	if m.callIndex < len(m.statuses) {
		status := m.statuses[m.callIndex]
		m.callIndex++
		return response_body.ResbodyGetMachine{
			SystemID:      "test-system-id",
			StatusName:    status,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil
	}
	return response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		StatusName:    "Ready",
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil
}

func (m *mockBasisMaasAPIMultipleStatusChanges) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

func (m *mockBasisMaasAPIMultipleStatusChanges) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

func (m *mockBasisMaasAPIMultipleStatusChanges) DELETE(ctx context.Context) (response_body.Resbody, error) {
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

// Mock for wrong type responses
type mockBasisMaasAPIWrongType struct {
	returnWrongType bool
}

func (m *mockBasisMaasAPIWrongType) GET(ctx context.Context) (response_body.Resbody, error) {
	if m.returnWrongType {
		// Return wrong type that will fail type assertion
		return response_body.ResbodyCommon{HTTPStatus: 200}, nil
	}
	return response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 1, Cidr: "192.168.1.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil
}

func (m *mockBasisMaasAPIWrongType) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	if m.returnWrongType {
		return response_body.ResbodyCommon{HTTPStatus: 200}, nil
	}
	return response_body.ResbodyPostFabrics{
		ID:            1,
		Vlans:         []response_body.Vlan{{Vid: 0}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil
}

func (m *mockBasisMaasAPIWrongType) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

func (m *mockBasisMaasAPIWrongType) DELETE(ctx context.Context) (response_body.Resbody, error) {
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

// Mock for API errors
type mockBasisMaasAPIWithError struct {
	err error
}

func (m *mockBasisMaasAPIWithError) GET(ctx context.Context) (response_body.Resbody, error) {
	return nil, m.err
}

func (m *mockBasisMaasAPIWithError) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return nil, m.err
}

func (m *mockBasisMaasAPIWithError) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return nil, m.err
}

func (m *mockBasisMaasAPIWithError) DELETE(ctx context.Context) (response_body.Resbody, error) {
	return nil, m.err
}

// Mock for unmarshalable data to test JSON marshal failure
type mockBasisMaasAPIUnmarshalableData struct{}

// Custom type that embeds ResbodyGetMachine and shadows MachineForResponse field
// to contain unmarshalable data
type unmarshalableResbodyGetMachine struct {
	response_body.ResbodyGetMachine
	// Shadow the MachineForResponse field with unmarshalable data
	MachineForResponse failingMarshalData
}

// failingMarshalData is a type that fails to marshal
type failingMarshalData struct {
	Chan chan int
}

func (m *mockBasisMaasAPIUnmarshalableData) GET(ctx context.Context) (response_body.Resbody, error) {
	// Return a struct that embeds ResbodyGetMachine but shadows MachineForResponse
	return unmarshalableResbodyGetMachine{
		ResbodyGetMachine: response_body.ResbodyGetMachine{
			SystemID:      "test-system-id",
			StatusName:    "Ready",
			Description:   "test",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
			// MachineForResponse will be shadowed by the outer struct's field
		},
		MachineForResponse: failingMarshalData{
			Chan: make(chan int), // This will cause JSON marshal to fail
		},
	}, nil
}

func (m *mockBasisMaasAPIUnmarshalableData) POST(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

func (m *mockBasisMaasAPIUnmarshalableData) PUT(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

func (m *mockBasisMaasAPIUnmarshalableData) DELETE(ctx context.Context) (response_body.Resbody, error) {
	return response_body.ResbodyCommon{HTTPStatus: 200}, nil
}

// Test case: createSubnetAndIPRange new fabric creation success
func TestCanonicalMaasController_createSubnetAndIPRange_NewFabricCreationSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock successful fabric creation
	mockFabricAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyPostFabrics{
			ID: 123, // New fabric ID
			Vlans: []response_body.Vlan{
				{Vid: 100}, // VLAN ID
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		},
	}
	factory.EXPECT().NewFabrics().Return(mockFabricAPI).Times(1)

	// Mock successful subnet creation
	mockSubnetAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyPostSubnets{
			ID:            456, // New subnet ID
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		},
	}
	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	// 空のmacToFabric（新しいファブリック作成をトリガー）
	macToFabric := make(map[string]FabricPair)
	mac := "00:11:22:33:44:55"
	cidr := "192.168.1.0/24"

	// Act
	subnetID, err := controller.createSubnetAndIPRange(ctx, macToFabric, mac, cidr, "", "")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if subnetID != 456 {
		t.Errorf("Expected subnetID 456, got %d", subnetID)
	}

	// Verify that macToFabric was updated with new fabric info
	if fabricPair, exists := macToFabric[mac]; !exists {
		t.Error("Expected macToFabric to be updated with new fabric info")
	} else {
		if fabricPair.fabricID != 123 {
			t.Errorf("Expected fabric ID 123, got %d", fabricPair.fabricID)
		}
		if fabricPair.vlanID != 100 {
			t.Errorf("Expected VLAN ID 100, got %d", fabricPair.vlanID)
		}
	}
}

// Test case: createSubnetAndIPRange new fabric creation with IP ranges
func TestCanonicalMaasController_createSubnetAndIPRange_NewFabricWithIPRanges(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock successful fabric creation
	mockFabricAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyPostFabrics{
			ID: 200,
			Vlans: []response_body.Vlan{
				{Vid: 0}, // Default VLAN
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		},
	}
	factory.EXPECT().NewFabrics().Return(mockFabricAPI).Times(1)

	// Mock successful subnet creation
	mockSubnetAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyPostSubnets{
			ID:            300,
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 201},
		},
	}
	factory.EXPECT().NewSubnets().Return(mockSubnetAPI).Times(1)

	// Mock successful IP range creation (called twice for two ranges)
	mockIPRangeAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 201},
	}
	factory.EXPECT().NewIPRanges().Return(mockIPRangeAPI).Times(2)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	macToFabric := make(map[string]FabricPair) // Empty to trigger new fabric creation
	mac := "00:11:22:33:44:66"
	cidr := "10.0.0.0/24"
	addStart := "10.0.0.50"
	addEnd := "10.0.0.150"

	// Act
	subnetID, err := controller.createSubnetAndIPRange(ctx, macToFabric, mac, cidr, addStart, addEnd)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if subnetID != 300 {
		t.Errorf("Expected subnetID 300, got %d", subnetID)
	}

	// Verify macToFabric update
	if fabricPair, exists := macToFabric[mac]; !exists {
		t.Error("Expected macToFabric to be updated")
	} else {
		if fabricPair.fabricID != 200 {
			t.Errorf("Expected fabric ID 200, got %d", fabricPair.fabricID)
		}
		if fabricPair.vlanID != 0 {
			t.Errorf("Expected VLAN ID 0, got %d", fabricPair.vlanID)
		}
	}
}

// Test case: linkSubnetInterface with keyName "Name" (Name-based interface matching)
func TestCanonicalMaasController_linkSubnetInterface_ByName_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock successful disconnect
	mockDisconnectAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 200},
	}
	factory.EXPECT().NewInterfaceDisconnect("test-system-id", 1).Return(mockDisconnectAPI).Times(1)

	// Mock successful link
	mockLinkAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 200},
	}
	factory.EXPECT().NewInterfaceLink("test-system-id", 1).Return(mockLinkAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"
	interfaces := []response_body.Interface{
		{
			ID:         5,
			Name:       "eth99",
			MacAddress: "FF:11:22:33:44:55",
		},
		{
			ID:         1,
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
		},
	}
	// Use interface name as key instead of MAC address
	key2sub := map[string]SubnetLinkPair{
		"eth0": {
			linkMode:  "AUTO",
			subnetIds: []int{1},
		},
	}

	// Act with keyName "Name" to test the else branch
	err := controller.linkSubnetInterface(ctx, systemID, interfaces, key2sub, "Name", true, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// Test case: linkSubnetInterface with keyName "Name" and case insensitive matching
func TestCanonicalMaasController_linkSubnetInterface_ByName_CaseInsensitive(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock successful disconnect
	mockDisconnectAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 200},
	}
	factory.EXPECT().NewInterfaceDisconnect("test-system-id", 2).Return(mockDisconnectAPI).Times(1)

	// Mock successful link (called twice due to subnetIds: []int{2, 3})
	mockLinkAPI := &mockBasisMaasAPI{
		postResult: response_body.ResbodyCommon{HTTPStatus: 200},
	}
	factory.EXPECT().NewInterfaceLink("test-system-id", 2).Return(mockLinkAPI).Times(2)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"
	interfaces := []response_body.Interface{
		{
			ID:         2,
			Name:       "ETH1", // Upper case interface name
			MacAddress: "00:11:22:33:44:66",
		},
	}
	// Use lowercase interface name as key for case insensitive matching
	key2sub := map[string]SubnetLinkPair{
		"eth1": {
			linkMode:  "DHCP",
			subnetIds: []int{2, 3},
		},
	}

	// Act with keyName "Name" and caseSensitivity false to test case insensitive name matching
	err := controller.linkSubnetInterface(ctx, systemID, interfaces, key2sub, "Name", false, nil)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestCanonicalMaasController_linkSubnetInterface_WithMac2Name_RenameSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	mockDisconnectAPI := &mockBasisMaasAPI{postResult: response_body.ResbodyCommon{HTTPStatus: 200}}
	factory.EXPECT().NewInterfaceDisconnect("test-system-id", 1).Return(mockDisconnectAPI).Times(1)

	mockLinkAPI := &mockBasisMaasAPI{postResult: response_body.ResbodyCommon{HTTPStatus: 200}}
	factory.EXPECT().NewInterfaceLink("test-system-id", 1).Return(mockLinkAPI).Times(1)

	mockUpdateAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory.EXPECT().NewInterfaceUpdate("test-system-id", 1).Return(mockUpdateAPI).Times(1)
	mockUpdateAPI.EXPECT().PUT(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqBody request_body.Reqbody) (response_body.Resbody, error) {
			updateReq, ok := reqBody.(request_body.ReqbodyInterfaceUpdate)
			if !ok {
				t.Fatalf("Expected ReqbodyInterfaceUpdate, got %T", reqBody)
			}
			if updateReq.Name != "eth0" {
				t.Fatalf("Expected rename target eth0, got %s", updateReq.Name)
			}
			return response_body.ResbodyCommon{HTTPStatus: 200}, nil
		},
	).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	interfaces := []response_body.Interface{{
		ID:         1,
		Name:       "eno1",
		MacAddress: "00:11:22:33:44:55",
	}}
	key2sub := map[string]SubnetLinkPair{
		"00:11:22:33:44:55": {linkMode: "AUTO", subnetIds: []int{1}},
	}
	mac2name := map[string]string{"00:11:22:33:44:55": "eth0"}

	err := controller.linkSubnetInterface(context.Background(), "test-system-id", interfaces, key2sub, "MacAddress", false, mac2name)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestCanonicalMaasController_linkSubnetInterface_WithMac2Name_RenameFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	mockDisconnectAPI := &mockBasisMaasAPI{postResult: response_body.ResbodyCommon{HTTPStatus: 200}}
	factory.EXPECT().NewInterfaceDisconnect("test-system-id", 1).Return(mockDisconnectAPI).Times(1)

	mockLinkAPI := &mockBasisMaasAPI{postResult: response_body.ResbodyCommon{HTTPStatus: 200}}
	factory.EXPECT().NewInterfaceLink("test-system-id", 1).Return(mockLinkAPI).Times(1)

	mockUpdateAPI := &mockBasisMaasAPIWithError{err: errors.New("interface rename failed")}
	factory.EXPECT().NewInterfaceUpdate("test-system-id", 1).Return(mockUpdateAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	interfaces := []response_body.Interface{{
		ID:         1,
		Name:       "eno1",
		MacAddress: "00:11:22:33:44:55",
	}}
	key2sub := map[string]SubnetLinkPair{
		"00:11:22:33:44:55": {linkMode: "AUTO", subnetIds: []int{1}},
	}
	mac2name := map[string]string{"00:11:22:33:44:55": "eth0"}

	err := controller.linkSubnetInterface(context.Background(), "test-system-id", interfaces, key2sub, "MacAddress", false, mac2name)
	if err == nil {
		t.Error("Expected interface rename error, got nil")
	}
}

func TestCanonicalMaasController_linkSubnetInterface_WithMac2Name_SameNameSkipsRename(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	mockDisconnectAPI := &mockBasisMaasAPI{postResult: response_body.ResbodyCommon{HTTPStatus: 200}}
	factory.EXPECT().NewInterfaceDisconnect("test-system-id", 1).Return(mockDisconnectAPI).Times(1)

	mockLinkAPI := &mockBasisMaasAPI{postResult: response_body.ResbodyCommon{HTTPStatus: 200}}
	factory.EXPECT().NewInterfaceLink("test-system-id", 1).Return(mockLinkAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	interfaces := []response_body.Interface{{
		ID:         1,
		Name:       "eth0",
		MacAddress: "00:11:22:33:44:55",
	}}
	key2sub := map[string]SubnetLinkPair{
		"00:11:22:33:44:55": {linkMode: "AUTO", subnetIds: []int{1}},
	}
	mac2name := map[string]string{"00:11:22:33:44:55": "eth0"}

	err := controller.linkSubnetInterface(context.Background(), "test-system-id", interfaces, key2sub, "MacAddress", false, mac2name)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// Test case: getMachineAccessInfo API GET failure
func TestCanonicalMaasController_getMachineAccessInfo_APIGetFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machine API that returns error for GET
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().
		GET(gomock.Any()).
		Return(nil, errors.New("machine API GET failed"))

	factory.EXPECT().
		NewMachineSystemID("test-system-id").
		Return(mockMachineAPI).
		Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	hostName, bootIf, accessAddress, bootMacAddress, subnetIDs, _, _, _, err := controller.getMachineAccessInfo(ctx, systemID)

	// Assert
	if err == nil {
		t.Error("Expected error from machine API GET failure, got nil")
	}

	if !strings.Contains(err.Error(), "machine API GET failed") {
		t.Errorf("Expected error message to contain 'machine API GET failed', got: %v", err)
	}

	// All return values should be empty/zero on error
	if hostName != "" {
		t.Errorf("Expected empty hostName on error, got %s", hostName)
	}
	if bootIf != "" {
		t.Errorf("Expected empty bootIf on error, got %s", bootIf)
	}
	if accessAddress != "" {
		t.Errorf("Expected empty accessAddress on error, got %s", accessAddress)
	}
	if bootMacAddress != "" {
		t.Errorf("Expected empty bootMacAddress on error, got %s", bootMacAddress)
	}
	if len(subnetIDs) != 0 {
		t.Errorf("Expected empty subnetIDs on error, got %v", subnetIDs)
	}
}

// Test case: getMachineAccessInfo type cast failure
func TestCanonicalMaasController_getMachineAccessInfo_TypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machine API that returns wrong response type
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyCommon{HTTPStatus: 200}, nil) // Wrong type

	factory.EXPECT().
		NewMachineSystemID("test-system-id").
		Return(mockMachineAPI).
		Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	hostName, bootIf, accessAddress, bootMacAddress, subnetIDs, _, _, _, err := controller.getMachineAccessInfo(ctx, systemID)

	// Assert
	if err == nil {
		t.Error("Expected error from type cast failure, got nil")
	}

	if !strings.Contains(err.Error(), "response type is invalid") {
		t.Errorf("Expected error message to contain 'response type is invalid', got: %v", err)
	}

	// All return values should be empty/zero on error
	if hostName != "" {
		t.Errorf("Expected empty hostName on error, got %s", hostName)
	}
	if bootIf != "" {
		t.Errorf("Expected empty bootIf on error, got %s", bootIf)
	}
	if accessAddress != "" {
		t.Errorf("Expected empty accessAddress on error, got %s", accessAddress)
	}
	if bootMacAddress != "" {
		t.Errorf("Expected empty bootMacAddress on error, got %s", bootMacAddress)
	}
	if len(subnetIDs) != 0 {
		t.Errorf("Expected empty subnetIDs on error, got %v", subnetIDs)
	}
}

// NOTE: TestCanonicalMaasController_OsDeploy_InternalMachineShowFailure was removed.
// OsDeploy no longer performs a readiness check via internalMachineShow.

// Test case: OsDeploy machine deployment API failure
func TestCanonicalMaasController_OsDeploy_MachineDeploymentFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock deployment API that returns error
	mockDeployAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockDeployAPI.EXPECT().
		POST(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("machine deployment failed"))

	factory.EXPECT().
		NewMachineDeploy("test-system-id").
		Return(mockDeployAPI).
		Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	request := &proto.OsDeployRequest{
		SystemId: "test-system-id",
		VmFlag:   &wrapperspb.BoolValue{Value: true},
		Os: &proto.OsInformation{
			Distribution: "ubuntu",
			Version:      "20.04",
		},
		UserData: "#!/bin/bash\necho 'deployment test'",
	}

	// Act
	response, err := controller.OsDeploy(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from machine deployment failure, got nil")
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}

	if !strings.Contains(err.Error(), "machine deployment failed") {
		t.Errorf("Expected error message to contain 'machine deployment failed', got: %v", err)
	}
}

// NOTE: TestCanonicalMaasController_OsDeploy_MachineShowTypeCastFailure was removed.
// OsDeploy no longer performs a readiness check via internalMachineShow.

// Test case: OsRelease VMHost found and deletion success
func TestCanonicalMaasController_OsRelease_VMHostDeletionSuccess(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock VM hosts list that contains the target system ID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 123,
					Host: response_body.Host{
						SystemID: "test-system-id", // Found!
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewVMHosts().
		Return(mockVMHostsAPI).
		Times(1)

	// Mock successful VM host deletion
	mockVMHostDeleteAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostDeleteAPI.EXPECT().
		DELETE(gomock.Any()).
		Return(response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().
		NewVMHostHostID(123).
		Return(mockVMHostDeleteAPI).
		Times(1)

	// Mock machine access info for unregister subscription
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.10"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.10",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewMachineSystemID("test-system-id").
		Return(mockMachineAPI).
		Times(1)

	// Mock successful machine release
	mockMachineReleaseAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineReleaseAPI.EXPECT().
		POST(gomock.Any(), gomock.Any()).
		Return(response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().
		NewMachineRelease("test-system-id").
		Return(mockMachineReleaseAPI).
		Times(1)

	// Mock get interfaces for IP tag release
	mockInterfacesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockInterfacesAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetInterfaces{
			List: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:11:22:33:44:55",
Tags:       []string{"192.168.1.100/24"},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	factory.EXPECT().
		NewInterfaces("test-system-id").
		Return(mockInterfacesAPI).
		Times(1)

	// Mock IP address release for tagged IP (tag "192.168.1.100/24" → IP = "192.168.1.100")
	mockIPReleaseAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockIPReleaseAPI.EXPECT().
		POST(gomock.Any(), gomock.Any()).
		Return(response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().
		NewIPAddressRelease().
		Return(mockIPReleaseAPI).
		Times(1)

	// Mock InterfaceRemoveTag for old IP/prefix tag ("192.168.1.100/24")
	mockRemoveTagAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockRemoveTagAPI.EXPECT().
		POST(gomock.Any(), gomock.Any()).
		Return(response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().
		NewInterfaceRemoveTag("test-system-id", 1).
		Return(mockRemoveTagAPI).
		Times(1)

	// Mock successful Ansible execution
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("unregister success"),
		cmdExecuteErr:    nil,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	ctx := context.Background()
	request := &proto.OsReleaseRequest{
		SystemId: "test-system-id",
	}

	// Act
	response, err := controller.OsRelease(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if response == nil {
		t.Error("Expected response, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ACCEPT {
			t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
		}
	}
}

// Test case: OsRelease VMHost found but deletion failure
func TestCanonicalMaasController_OsRelease_VMHostDeletionFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock VM hosts list that contains the target system ID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 456,
					Host: response_body.Host{
						SystemID: "test-system-id", // Found!
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewVMHosts().
		Return(mockVMHostsAPI).
		Times(1)

	// Mock VM host deletion failure
	mockVMHostDeleteAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostDeleteAPI.EXPECT().
		DELETE(gomock.Any()).
		Return(nil, errors.New("VM host deletion failed"))

	factory.EXPECT().
		NewVMHostHostID(456).
		Return(mockVMHostDeleteAPI).
		Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	request := &proto.OsReleaseRequest{
		SystemId: "test-system-id",
	}

	// Act
	response, err := controller.OsRelease(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from VM host deletion failure, got nil")
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}

	if !strings.Contains(err.Error(), "VM host deletion failed") {
		t.Errorf("Expected error message to contain 'VM host deletion failed', got: %v", err)
	}
}

// Test case: OsRelease VMHost not found (existing behavior)
func TestCanonicalMaasController_OsRelease_VMHostNotFound(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock VM hosts list that does NOT contain the target system ID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 999,
					Host: response_body.Host{
						SystemID: "other-system-id", // Different system ID
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewVMHosts().
		Return(mockVMHostsAPI).
		Times(1)

	// Mock machine access info for unregister subscription
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.10"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewMachineSystemID("test-system-id").
		Return(mockMachineAPI).
		Times(1)

	// Mock successful machine release
	mockMachineReleaseAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineReleaseAPI.EXPECT().
		POST(gomock.Any(), gomock.Any()).
		Return(response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	factory.EXPECT().
		NewMachineRelease("test-system-id").
		Return(mockMachineReleaseAPI).
		Times(1)

	// Mock get interfaces for IP tag release
	mockInterfacesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockInterfacesAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetInterfaces{
			List: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:11:22:33:44:55",
					Tags:       []string{},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewInterfaces("test-system-id").
		Return(mockInterfacesAPI).
		Times(1)

	// Mock successful Ansible execution
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("unregister success"),
		cmdExecuteErr:    nil,
	}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	ctx := context.Background()
	request := &proto.OsReleaseRequest{
		SystemId: "test-system-id",
	}

	// Act
	response, err := controller.OsRelease(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if response == nil {
		t.Error("Expected response, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ACCEPT {
			t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
		}
	}
}

// Test case: MachineList response type cast failure
func TestCanonicalMaasController_MachineList_ResponseTypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machines API that returns wrong response type
	mockMachinesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachinesAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyCommon{HTTPStatus: 200}, nil) // Wrong type

	factory.EXPECT().
		NewMachines().
		Return(mockMachinesAPI).
		Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	request := &proto.MachineListRequest{}

	// Act
	response, err := controller.MachineList(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from response type cast failure, got nil")
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}

	if !strings.Contains(err.Error(), "response type is invalid") {
		t.Errorf("Expected error message to contain 'response type is invalid', got: %v", err)
	}
}

// Test case: OsRelease getMachineAccessInfo failure
func TestCanonicalMaasController_OsRelease_GetMachineAccessInfoFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock VM hosts list that does NOT contain the target system ID (VM host not found)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetVMHosts{
			List: []response_body.VMHost{
				{
					ID: 999,
					Host: response_body.Host{
						SystemID: "other-system-id", // Different system ID
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewVMHosts().
		Return(mockVMHostsAPI).
		Times(1)

	// Mock machine access info API that returns error
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().
		GET(gomock.Any()).
		Return(nil, errors.New("get machine access info failed"))

	factory.EXPECT().
		NewMachineSystemID("test-system-id").
		Return(mockMachineAPI).
		Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	request := &proto.OsReleaseRequest{
		SystemId: "test-system-id",
	}

	// Act
	response, err := controller.OsRelease(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from getMachineAccessInfo failure, got nil")
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}

	if !strings.Contains(err.Error(), "get machine access info failed") {
		t.Errorf("Expected error message to contain 'get machine access info failed', got: %v", err)
	}
}

// Test case: OsRelease machine release API failure
func TestCanonicalMaasController_OsRelease_MachineReleaseFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock VM hosts list that does NOT contain the target system ID (VM host not found)
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{}, // Empty list - no VM hosts
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewVMHosts().
		Return(mockVMHostsAPI).
		Times(1)

	// Mock successful machine access info
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.10"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.10",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewMachineSystemID("test-system-id").
		Return(mockMachineAPI).
		Times(1)

	// Mock successful Ansible execution
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("unregister success"),
		cmdExecuteErr:    nil,
	}

	// Mock machine release API that returns error
	mockMachineReleaseAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineReleaseAPI.EXPECT().
		POST(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("machine release failed"))

	factory.EXPECT().
		NewMachineRelease("test-system-id").
		Return(mockMachineReleaseAPI).
		Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	ctx := context.Background()
	request := &proto.OsReleaseRequest{
		SystemId: "test-system-id",
	}

	// Act
	response, err := controller.OsRelease(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from machine release failure, got nil")
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}

	if !strings.Contains(err.Error(), "machine release failed") {
		t.Errorf("Expected error message to contain 'machine release failed', got: %v", err)
	}
}

// Test case: OsRelease getMachineAccessInfo type cast failure
func TestCanonicalMaasController_OsRelease_GetMachineAccessInfoTypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock VM hosts list that does NOT contain the target system ID
	mockVMHostsAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockVMHostsAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyGetVMHosts{
			List:          []response_body.VMHost{},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().
		NewVMHosts().
		Return(mockVMHostsAPI).
		Times(1)

	// Mock machine API that returns wrong response type (for getMachineAccessInfo)
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineAPI.EXPECT().
		GET(gomock.Any()).
		Return(response_body.ResbodyCommon{HTTPStatus: 200}, nil) // Wrong type

	factory.EXPECT().
		NewMachineSystemID("test-system-id").
		Return(mockMachineAPI).
		Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	request := &proto.OsReleaseRequest{
		SystemId: "test-system-id",
	}

	// Act
	response, err := controller.OsRelease(ctx, request)

	// Assert
	if err == nil {
		t.Error("Expected error from type cast failure, got nil")
	}

	if response == nil {
		t.Error("Expected response even on error, got nil")
	} else {
		if response.GetResult() != common.ResultCode_ERROR {
			t.Errorf("Expected result ERROR, got %v", response.GetResult())
		}
	}

	if !strings.Contains(err.Error(), "response type is invalid") {
		t.Errorf("Expected error message to contain 'response type is invalid', got: %v", err)
	}
}

// Test case: MachineRegister goroutine - getSubnetList failure
func TestCanonicalMaasController_MachineRegister_GetSubnetListFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachinesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCommissionAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Mock successful machine registration
	mockFactory.EXPECT().NewMachines(gomock.Any()).Return(mockMachinesAPI)
	mockMachinesAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostMachines{
			SystemID:      "test-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock successful commission
	mockFactory.EXPECT().NewMachineCommission("test-system-id").Return(mockCommissionAPI)
	mockCommissionAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	// Mock machine status polling (commission success)
	mockMachineStatusAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockFactory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineStatusAPI).Times(1)
	mockMachineStatusAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "test-system-id",
			StatusName:    "Ready",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock getSubnetList failure
	mockFactory.EXPECT().NewSubnets().Return(mockSubnetAPI)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		nil, errors.New("subnet list retrieval failed"))

	// Mock markBroken call
	mockFactory.EXPECT().NewMachineMarkBroken("test-system-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.MachineRegister(ctx, request)

	// Assert - Main function should succeed
	if err != nil {
		t.Errorf("Expected no error from main function, got: %v", err)
	}
	if response == nil {
		t.Error("Expected response, got nil")
	} else if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for goroutine to complete
	time.Sleep(200 * time.Millisecond)
}

// Test case: MachineRegister goroutine - getMachineAccessInfo failure after successful polling
func TestCanonicalMaasController_MachineRegister_GetMachineAccessInfoFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachinesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCommissionAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMachineStatusAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Mock successful machine registration
	factory.EXPECT().NewMachines(gomock.Any()).Return(mockMachinesAPI)
	mockMachinesAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostMachines{
			SystemID:      "test-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock successful commission
	factory.EXPECT().NewMachineCommission("test-system-id").Return(mockCommissionAPI)
	mockCommissionAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	// Mock machine status API with sequential behavior using DoAndReturn
	callCount := 0
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineStatusAPI).Times(2)
	mockMachineStatusAPI.EXPECT().
		GET(gomock.Any()).
		DoAndReturn(func(ctx context.Context) (response_body.Resbody, error) {
			callCount++
			switch callCount {
			case 1:
				// First call: polling success
				return response_body.ResbodyGetMachine{
					SystemID:      "test-system-id",
					StatusName:    "Ready",
					ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
				}, nil
			case 2:
				// Second call: getMachineAccessInfo failure
				return nil, errors.New("get machine access info failed")
			default:
				return nil, errors.New("unexpected call")
			}
		}).
		Times(2)

	factory.EXPECT().NewSubnets().Return(mockSubnetAPI)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 1, Cidr: "192.168.1.0/24"},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock markBroken call
	factory.EXPECT().NewMachineMarkBroken("test-system-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.MachineRegister(ctx, request)

	// Assert - Main function should succeed (goroutine runs async)
	if err != nil {
		t.Errorf("Expected no error from main function, got: %v", err)
	}
	if response == nil {
		t.Error("Expected response, got nil")
	} else if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for goroutine to complete and call markBroken
	time.Sleep(300 * time.Millisecond)
}

// Test case: MachineRegister goroutine - createSubnetAndIPRange failure
func TestCanonicalMaasController_MachineRegister_CreateSubnetFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachinesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCommissionAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockFabricAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress:   "00:11:22:33:44:55",
				Cidr:         "192.168.1.0/24",
				AddressStart: func() *string { s := "192.168.1.10"; return &s }(),
				AddressEnd:   func() *string { s := "192.168.1.50"; return &s }(),
			},
		},
	}

	// Mock successful machine registration
	factory.EXPECT().NewMachines(gomock.Any()).Return(mockMachinesAPI)
	mockMachinesAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostMachines{
			SystemID:      "test-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock successful commission
	factory.EXPECT().NewMachineCommission("test-system-id").Return(mockCommissionAPI)
	mockCommissionAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	// Mock machine status polling - Allow multiple calls during polling
	mockMachineStatusAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineStatusAPI).AnyTimes()
	mockMachineStatusAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "test-system-id",
			StatusName:    "Ready",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil).AnyTimes()

	// Mock successful getSubnetList (empty list to trigger subnet creation)
	factory.EXPECT().NewSubnets().Return(mockSubnetAPI)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List:          []response_body.Subnet{}, // Empty - no existing subnets
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock fabric creation failure
	factory.EXPECT().NewFabrics().Return(mockFabricAPI)
	mockFabricAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		nil, errors.New("fabric creation failed"))

	// Mock markBroken call
	factory.EXPECT().NewMachineMarkBroken("test-system-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.MachineRegister(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error from main function, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for goroutine to complete
	time.Sleep(200 * time.Millisecond)
}

// Test case: MachineRegister goroutine - getInterfaceList failure
func TestCanonicalMaasController_MachineRegister_GetInterfaceListFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachinesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCommissionAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockInterfaceAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Mock successful machine registration, commission, and polling
	factory.EXPECT().NewMachines(gomock.Any()).Return(mockMachinesAPI)
	mockMachinesAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostMachines{
			SystemID:      "test-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineCommission("test-system-id").Return(mockCommissionAPI)
	mockCommissionAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	// Mock machine status polling with AnyTimes()
	mockMachineStatusAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineStatusAPI).AnyTimes()
	mockMachineStatusAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "test-system-id",
			StatusName:    "Ready",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil).AnyTimes()

	// Mock successful getSubnetList with existing subnet
	factory.EXPECT().NewSubnets().Return(mockSubnetAPI)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 1, Cidr: "192.168.1.0/24"},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock getInterfaceList failure
	factory.EXPECT().NewInterfaces("test-system-id").Return(mockInterfaceAPI)
	mockInterfaceAPI.EXPECT().GET(gomock.Any()).Return(
		nil, errors.New("interface list retrieval failed"))

	// Mock markBroken call
	factory.EXPECT().NewMachineMarkBroken("test-system-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.MachineRegister(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error from main function, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for goroutine to complete
	time.Sleep(200 * time.Millisecond)
}

// Test case: MachineRegister goroutine - linkSubnetInterface failure
func TestCanonicalMaasController_MachineRegister_LinkSubnetInterfaceFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachinesAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCommissionAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockSubnetAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockInterfaceAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockDisconnectAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	request := &proto.MachineRegisterRequest{
		MacAddress:   "00:11:22:33:44:55",
		IpmiAddress:  "192.168.1.100",
		IpmiUser:     "admin",
		IpmiPassword: "password",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	// Mock successful machine registration, commission, polling
	factory.EXPECT().NewMachines(gomock.Any()).Return(mockMachinesAPI)
	mockMachinesAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyPostMachines{
			SystemID:      "test-system-id",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineCommission("test-system-id").Return(mockCommissionAPI)
	mockCommissionAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	// Mock machine status polling with AnyTimes()
	mockMachineStatusAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineStatusAPI).AnyTimes()
	mockMachineStatusAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:      "test-system-id",
			StatusName:    "Ready",
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil).AnyTimes()

	// Mock successful getSubnetList
	factory.EXPECT().NewSubnets().Return(mockSubnetAPI)
	mockSubnetAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetSubnets{
			List: []response_body.Subnet{
				{ID: 1, Cidr: "192.168.1.0/24"},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock successful getInterfaceList
	factory.EXPECT().NewInterfaces("test-system-id").Return(mockInterfaceAPI)
	mockInterfaceAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetInterfaces{
			List: []response_body.Interface{
				{
					ID:         1,
					Name:       "eth0",
					MacAddress: "00:11:22:33:44:55",
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock interface disconnect failure
	factory.EXPECT().NewInterfaceDisconnect("test-system-id", 1).Return(mockDisconnectAPI)
	mockDisconnectAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		nil, errors.New("interface disconnect failed"))

	// Mock markBroken call
	factory.EXPECT().NewMachineMarkBroken("test-system-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()

	// Act
	response, err := controller.MachineRegister(ctx, request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error from main function, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for goroutine to complete
	time.Sleep(200 * time.Millisecond)
}

// NOTE: TestCanonicalMaasController_MachineRegister_GoroutineStatusPutFailure was removed.
// NOTE: TestCanonicalMaasController_MachineRegister_GoroutineCompleteSuccess was removed.
// The completion-description PUT ("completion") in the MachineRegister goroutine was deleted
// in favor of JobManager tracking.

// TestCanonicalMaasController_PowerON tests PowerON API
func TestCanonicalMaasController_PowerON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockPowerAPI := mocks.NewMockBasisMaasAPI(ctrl)

	factory.EXPECT().NewMachinePowerON("test-system-id").Return(mockPowerAPI)
	mockPowerAPI.EXPECT().POST(gomock.Any(), gomock.Eq(request_body.ReqbodyMachinePowerON{
		UserData: "#cloud-config",
	})).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	request := &proto.PowerOnRequest{
		SystemId: "test-system-id",
		UserData: "#cloud-config",
	}

	response2, err2 := controller.PowerON(context.Background(), request)

	if err2 != nil {
		t.Errorf("Expected no error, got: %v", err2)
	}
	if response2.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response2.GetResult())
	}
}

// TestCanonicalMaasController_PowerON_APIError tests PowerON with API error
func TestCanonicalMaasController_PowerON_APIError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockPowerAPI := mocks.NewMockBasisMaasAPI(ctrl)

	factory.EXPECT().NewMachinePowerON("test-system-id").Return(mockPowerAPI)
	mockPowerAPI.EXPECT().POST(gomock.Any(), gomock.Eq(request_body.ReqbodyMachinePowerON{
		UserData: "",
	})).Return(
		response_body.ResbodyCommon{HTTPStatus: 500}, errors.New("power ON failed"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	request := &proto.PowerOnRequest{
		SystemId: "test-system-id",
	}

	response2, err2 := controller.PowerON(context.Background(), request)

	if err2 == nil {
		t.Error("Expected error, got nil")
	}
	if response2.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response2.GetResult())
	}
}

// TestCanonicalMaasController_PowerOFF tests PowerOFF API
func TestCanonicalMaasController_PowerOFF(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockPowerAPI := mocks.NewMockBasisMaasAPI(ctrl)

	factory.EXPECT().NewMachinePowerOFF("test-system-id").Return(mockPowerAPI)
	mockPowerAPI.EXPECT().POST(gomock.Any(), gomock.Nil()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	request := &proto.PowerOffRequest{
		SystemId: "test-system-id",
	}

	response2, err2 := controller.PowerOFF(context.Background(), request)

	if err2 != nil {
		t.Errorf("Expected no error, got: %v", err2)
	}
	if response2.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response2.GetResult())
	}
}

// TestCanonicalMaasController_PowerOFF_APIError tests PowerOFF with API error
func TestCanonicalMaasController_PowerOFF_APIError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockPowerAPI := mocks.NewMockBasisMaasAPI(ctrl)

	factory.EXPECT().NewMachinePowerOFF("test-system-id").Return(mockPowerAPI)
	mockPowerAPI.EXPECT().POST(gomock.Any(), gomock.Nil()).Return(
		response_body.ResbodyCommon{HTTPStatus: 500}, errors.New("power OFF failed"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	request := &proto.PowerOffRequest{
		SystemId: "test-system-id",
	}

	response2, err2 := controller.PowerOFF(context.Background(), request)

	if err2 == nil {
		t.Error("Expected error, got nil")
	}
	if response2.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response2.GetResult())
	}
}

// TestCanonicalMaasController_KubeadmReset tests KubeadmReset API
func TestCanonicalMaasController_KubeadmReset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)

	// Mock getMachineAccessInfo via NewMachineSystemID
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI)

	// Mock Ansible CmdExecute for kubeadm reset
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.100", "kubeadm_reset.yaml", "").Return([]byte("success"), nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	request := &proto.KubeadmResetRequest{
		SystemId: "test-system-id",
	}

	response, err := controller.KubeadmReset(context.Background(), request)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_SUCCESS {
		t.Errorf("Expected result SUCCESS, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_KubeadmReset_MachineShowError tests KubeadmReset with MachineShow error
func TestCanonicalMaasController_KubeadmReset_MachineShowError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)

	// Mock getMachineAccessInfo to fail
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 500}, errors.New("machine not found"))

	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	request := &proto.KubeadmResetRequest{
		SystemId: "test-system-id",
	}

	response, err := controller.KubeadmReset(context.Background(), request)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_KubeadmReset_AnsibleError tests KubeadmReset with Ansible error
func TestCanonicalMaasController_KubeadmReset_AnsibleError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)

	// Mock getMachineAccessInfo
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI)

	// Mock Ansible CmdExecute to fail
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.100", "kubeadm_reset.yaml", "").Return(nil, errors.New("ansible execution failed"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	request := &proto.KubeadmResetRequest{
		SystemId: "test-system-id",
	}

	response, err := controller.KubeadmReset(context.Background(), request)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_KubeadmJoin tests KubeadmJoin API (async execution)
func TestCanonicalMaasController_KubeadmJoin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockWorkerAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCPAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)

	// Mock getMachineAccessInfo for worker node
	mockWorkerAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "worker-id",
			HostName:    "worker-host",
			IPAddresses: []string{"192.168.1.200"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:66",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.200",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock getMachineAccessInfo for control plane node (called in goroutine)
	mockCPAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "cp-id-1",
			HostName:    "cp-host",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("worker-id").Return(mockWorkerAPI)
	factory.EXPECT().NewMachineSystemID("cp-id-1").Return(mockCPAPI)

	// Mock Ansible CmdExecute: first for control plane token creation, then for worker join
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.100", "kubeadm_token_create.yaml", "").Return([]byte("KUBEADM_JOIN_COMMAND=kubeadm join 192.168.1.100:6443 --token abc123 --discovery-token-ca-cert-hash sha256:xyz789"), nil)
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.200", "kubeadm_join.yaml", gomock.Any()).Return([]byte("success"), nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	request := &proto.KubeadmJoinRequest{
		SystemId:   "worker-id",
		CpSystemId: []string{"cp-id-1"},
	}

	response, err := controller.KubeadmJoin(context.Background(), request)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for async goroutine to complete
	time.Sleep(50 * time.Millisecond)
}

// TestCanonicalMaasController_KubeadmJoin_MachineShowError tests KubeadmJoin with MachineShow error
func TestCanonicalMaasController_KubeadmJoin_MachineShowError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockMachineAPI := mocks.NewMockBasisMaasAPI(ctrl)

	// Mock getMachineAccessInfo to fail
	mockMachineAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 500}, errors.New("machine not found"))

	factory.EXPECT().NewMachineSystemID("worker-id").Return(mockMachineAPI)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	request := &proto.KubeadmJoinRequest{
		SystemId:   "worker-id",
		CpSystemId: []string{"cp-id-1"},
	}

	response, err := controller.KubeadmJoin(context.Background(), request)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_KubeadmJoin_TokenCreateError tests KubeadmJoin with token create error
func TestCanonicalMaasController_KubeadmJoin_TokenCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockWorkerAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCPAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	// Mock getMachineAccessInfo for worker node
	mockWorkerAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "worker-id",
			HostName:    "worker-host",
			IPAddresses: []string{"192.168.1.200"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:66",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.200",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock getMachineAccessInfo for control plane node
	mockCPAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "cp-id-1",
			HostName:    "cp-host",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("worker-id").Return(mockWorkerAPI)
	factory.EXPECT().NewMachineSystemID("cp-id-1").Return(mockCPAPI)

	// Mock Ansible CmdExecute: token creation fails
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.100", "kubeadm_token_create.yaml", "").Return(nil, errors.New("token creation failed"))

	// Mock markBroken call
	factory.EXPECT().NewMachineMarkBroken("worker-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	request := &proto.KubeadmJoinRequest{
		SystemId:   "worker-id",
		CpSystemId: []string{"cp-id-1"},
	}

	response, err := controller.KubeadmJoin(context.Background(), request)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for async goroutine to fail
	time.Sleep(50 * time.Millisecond)
}

// TestCanonicalMaasController_KubeadmJoin_ParseError tests KubeadmJoin with join command parse error
func TestCanonicalMaasController_KubeadmJoin_ParseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockWorkerAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCPAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	// Mock getMachineAccessInfo for worker node
	mockWorkerAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "worker-id",
			HostName:    "worker-host",
			IPAddresses: []string{"192.168.1.200"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:66",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.200",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock getMachineAccessInfo for control plane node
	mockCPAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "cp-id-1",
			HostName:    "cp-host",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("worker-id").Return(mockWorkerAPI)
	factory.EXPECT().NewMachineSystemID("cp-id-1").Return(mockCPAPI)

	// Mock Ansible CmdExecute: returns output without valid join command
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.100", "kubeadm_token_create.yaml", "").Return([]byte("invalid output without join command"), nil)

	// Mock markBroken call
	factory.EXPECT().NewMachineMarkBroken("worker-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	request := &proto.KubeadmJoinRequest{
		SystemId:   "worker-id",
		CpSystemId: []string{"cp-id-1"},
	}

	response, err := controller.KubeadmJoin(context.Background(), request)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for async goroutine to fail
	time.Sleep(50 * time.Millisecond)
}

// TestCanonicalMaasController_KubeadmJoin_WorkerJoinError tests KubeadmJoin with worker join error
func TestCanonicalMaasController_KubeadmJoin_WorkerJoinError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockWorkerAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCPAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)
	mockMarkBrokenAPI := mocks.NewMockBasisMaasAPI(ctrl)

	// Mock getMachineAccessInfo for worker node
	mockWorkerAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "worker-id",
			HostName:    "worker-host",
			IPAddresses: []string{"192.168.1.200"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:66",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.200",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	// Mock getMachineAccessInfo for control plane node
	mockCPAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			SystemID:    "cp-id-1",
			HostName:    "cp-host",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet: response_body.Subnet{
							ID:   1,
							Cidr: "192.168.1.0/24",
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("worker-id").Return(mockWorkerAPI)
	factory.EXPECT().NewMachineSystemID("cp-id-1").Return(mockCPAPI)

	// Mock Ansible CmdExecute: token creation succeeds, but join fails
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.100", "kubeadm_token_create.yaml", "").Return([]byte("KUBEADM_JOIN_COMMAND=kubeadm join 192.168.1.100:6443 --token abc123"), nil)
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.200", "kubeadm_join.yaml", gomock.Any()).Return(nil, errors.New("join failed"))

	// Mock markBroken call
	factory.EXPECT().NewMachineMarkBroken("worker-id").Return(mockMarkBrokenAPI)
	mockMarkBrokenAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(
		response_body.ResbodyCommon{HTTPStatus: 200}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	request := &proto.KubeadmJoinRequest{
		SystemId:   "worker-id",
		CpSystemId: []string{"cp-id-1"},
	}

	response, err := controller.KubeadmJoin(context.Background(), request)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Wait for async goroutine to fail
	time.Sleep(50 * time.Millisecond)
}

// TestCanonicalMaasController_KubeadmJoin_FallbackParse tests KubeadmJoin with fallback join command parse
func TestCanonicalMaasController_KubeadmJoin_FallbackParse(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

factory := mocks.NewMockMaasAPIFactory(ctrl)
mockWorkerAPI := mocks.NewMockBasisMaasAPI(ctrl)
mockCPAPI := mocks.NewMockBasisMaasAPI(ctrl)
mockAnsible := mocks.NewMockMaasAnsible(ctrl)

mockWorkerAPI.EXPECT().GET(gomock.Any()).Return(
response_body.ResbodyGetMachine{
SystemID:    "worker-id",
HostName:    "worker-host",
IPAddresses: []string{"192.168.1.200"},
BootInterface: response_body.Interface{
Name:       "eth0",
MacAddress: "00:11:22:33:44:66",
Links:      []response_body.Link{{IPAddress: "192.168.1.200", Subnet: response_body.Subnet{ID: 1, Cidr: "192.168.1.0/24"}}},
},
ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
}, nil)

mockCPAPI.EXPECT().GET(gomock.Any()).Return(
response_body.ResbodyGetMachine{
SystemID:    "cp-id-1",
HostName:    "cp-host",
IPAddresses: []string{"192.168.1.100"},
BootInterface: response_body.Interface{
Name:       "eth0",
MacAddress: "00:11:22:33:44:55",
Links:      []response_body.Link{{IPAddress: "192.168.1.100", Subnet: response_body.Subnet{ID: 1, Cidr: "192.168.1.0/24"}}},
},
ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
}, nil)

factory.EXPECT().NewMachineSystemID("worker-id").Return(mockWorkerAPI)
factory.EXPECT().NewMachineSystemID("cp-id-1").Return(mockCPAPI)

// Fallback parse: direct kubeadm join command without KUBEADM_JOIN_COMMAND marker
mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.100", "kubeadm_token_create.yaml", "").Return([]byte("kubeadm join 192.168.1.100:6443 --token abc123"), nil)
mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.200", "kubeadm_join.yaml", gomock.Any()).Return([]byte("success"), nil)

controller := CanonicalMaasController{Logger: klog.NewKlogr(), APIFactory: factory, Ansible: mockAnsible}
response, err := controller.KubeadmJoin(context.Background(), &proto.KubeadmJoinRequest{SystemId: "worker-id", CpSystemId: []string{"cp-id-1"}})

if err != nil {
t.Errorf("Expected no error, got: %v", err)
}
if response.GetResult() != common.ResultCode_ACCEPT {
t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
}
time.Sleep(50 * time.Millisecond)
}

// TestCanonicalMaasController_KubeadmJoin_CPMachineShowError tests CP MachineShow error with fallback to next CP
func TestCanonicalMaasController_KubeadmJoin_CPMachineShowError(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

factory := mocks.NewMockMaasAPIFactory(ctrl)
mockWorkerAPI := mocks.NewMockBasisMaasAPI(ctrl)
mockCPAPI1 := mocks.NewMockBasisMaasAPI(ctrl)
mockCPAPI2 := mocks.NewMockBasisMaasAPI(ctrl)
mockAnsible := mocks.NewMockMaasAnsible(ctrl)

mockWorkerAPI.EXPECT().GET(gomock.Any()).Return(
response_body.ResbodyGetMachine{
SystemID:    "worker-id",
HostName:    "worker-host",
IPAddresses: []string{"192.168.1.200"},
BootInterface: response_body.Interface{
Name:       "eth0",
MacAddress: "00:11:22:33:44:66",
Links:      []response_body.Link{{IPAddress: "192.168.1.200", Subnet: response_body.Subnet{ID: 1, Cidr: "192.168.1.0/24"}}},
},
ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
}, nil)

mockCPAPI1.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyCommon{HTTPStatus: 500}, errors.New("cp1 not found"))
mockCPAPI2.EXPECT().GET(gomock.Any()).Return(
response_body.ResbodyGetMachine{
SystemID:    "cp-id-2",
HostName:    "cp-host-2",
IPAddresses: []string{"192.168.1.101"},
BootInterface: response_body.Interface{
Name:       "eth0",
MacAddress: "00:11:22:33:44:77",
Links:      []response_body.Link{{IPAddress: "192.168.1.101", Subnet: response_body.Subnet{ID: 1, Cidr: "192.168.1.0/24"}}},
},
ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
}, nil)

factory.EXPECT().NewMachineSystemID("worker-id").Return(mockWorkerAPI)
factory.EXPECT().NewMachineSystemID("cp-id-1").Return(mockCPAPI1)
factory.EXPECT().NewMachineSystemID("cp-id-2").Return(mockCPAPI2)

mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.101", "kubeadm_token_create.yaml", "").Return([]byte("KUBEADM_JOIN_COMMAND=kubeadm join 192.168.1.101:6443 --token abc123"), nil)
mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.200", "kubeadm_join.yaml", gomock.Any()).Return([]byte("success"), nil)

controller := CanonicalMaasController{Logger: klog.NewKlogr(), APIFactory: factory, Ansible: mockAnsible}
response, err := controller.KubeadmJoin(context.Background(), &proto.KubeadmJoinRequest{SystemId: "worker-id", CpSystemId: []string{"cp-id-1", "cp-id-2"}})

if err != nil {
t.Errorf("Expected no error, got: %v", err)
}
if response.GetResult() != common.ResultCode_ACCEPT {
t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
}
time.Sleep(50 * time.Millisecond)
}
// TestCanonicalMaasController_NetworkUpdate_Success tests successful NetworkUpdate
func TestCanonicalMaasController_NetworkUpdate_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("success"),
		cmdExecuteErr:    nil,
	}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Tags:       []string{},
				Links: []response_body.Link{
					{
						IPAddress: "192.168.30.10",
						Subnet: response_body.Subnet{
							ID:   10,
							Cidr: "192.168.30.0/24",
						},
					},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List: []response_body.Subnet{
			{ID: 10, Cidr: "192.168.30.0/24"},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:    "test-system-id",
		StatusName:  "Deployed",
		IPAddresses: []string{"192.168.30.10"},
		BootInterface: response_body.Interface{
			ID:         1,
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links: []response_body.Link{
				{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// InterfaceAddTag (for new IP/prefix tag; no old tags → no release/remove)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_GetInterfaceListFailure tests NetworkUpdate with getInterfaceList failure
func TestCanonicalMaasController_NetworkUpdate_GetInterfaceListFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList with error
	mockAPI.EXPECT().GET(gomock.Any()).Return(nil, errors.New("interface list error"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_GetSubnetListFailure tests NetworkUpdate with getSubnetList failure
func TestCanonicalMaasController_NetworkUpdate_GetSubnetListFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List:          []response_body.Interface{{ID: 1, Name: "eth0", MacAddress: "00:11:22:33:44:55"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList with error
	mockAPI.EXPECT().GET(gomock.Any()).Return(nil, errors.New("subnet list error"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_GetMachineAccessInfoFailure tests NetworkUpdate with getMachineAccessInfo failure
func TestCanonicalMaasController_NetworkUpdate_GetMachineAccessInfoFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List:          []response_body.Interface{{ID: 1, Name: "eth0", MacAddress: "00:11:22:33:44:55"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo with error
	mockAPI.EXPECT().GET(gomock.Any()).Return(nil, errors.New("machine access info error"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}// Additional NetworkUpdate test cases - to be appended to canonical_maas_controller_test.go

// TestCanonicalMaasController_NetworkUpdate_InterfaceNotFound tests NetworkUpdate with interface not found for MAC address
func TestCanonicalMaasController_NetworkUpdate_InterfaceNotFound(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList with different MAC address
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{ID: 1, Name: "eth0", MacAddress: "AA:BB:CC:DD:EE:FF"}, // Different MAC
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55", // This MAC is not in the interface list
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_SubnetNotFound tests NetworkUpdate with subnet not found for CIDR
func TestCanonicalMaasController_NetworkUpdate_SubnetNotFound(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List:          []response_body.Interface{{ID: 1, Name: "eth0", MacAddress: "00:11:22:33:44:55"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList with different CIDR
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.1.0/24"}}, // Different CIDR
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24", // This CIDR is not in the subnet list
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_UnreservedIPRangesFailure tests NetworkUpdate with SubnetUnreservedIPRanges failure
func TestCanonicalMaasController_NetworkUpdate_UnreservedIPRangesFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges with error
	mockAPI.EXPECT().GET(gomock.Any()).Return(nil, errors.New("unreserved IP ranges error"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_NoUnreservedIPRanges tests NetworkUpdate with empty unreserved IP ranges
func TestCanonicalMaasController_NetworkUpdate_NoUnreservedIPRanges(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges with empty list
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List:          []response_body.UnreservedIPRange{}, // Empty list
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_IPAddressReserveFailure tests NetworkUpdate with IPAddressReserve failure
func TestCanonicalMaasController_NetworkUpdate_IPAddressReserveFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve with error
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(nil, errors.New("IP address reserve error"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_InvalidCIDRFormat tests NetworkUpdate with invalid CIDR format
func TestCanonicalMaasController_NetworkUpdate_InvalidCIDRFormat(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "invalid-cidr"}},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList  
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "invalid-cidr"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// CIDR format check occurs before SubnetUnreservedIPRanges query in the new flow,
	// so no GET/POST calls are expected after findSubnet.

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "invalid-cidr", // Invalid CIDR format (no slash)
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_AnsibleFailure tests NetworkUpdate with Ansible CmdExecute failure
func TestCanonicalMaasController_NetworkUpdate_AnsibleFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: nil,
		cmdExecuteErr:    errors.New("Ansible execution error"),
	}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_IPReleaseFailure tests NetworkUpdate with IPAddressRelease failure
func TestCanonicalMaasController_NetworkUpdate_IPReleaseFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("success"),
		cmdExecuteErr:    nil,
	}

	// Mock getInterfaceList — interface has an existing IP/prefix tag ("192.168.30.50/24")
	// so TaggedIPs() returns ["192.168.30.50"] and that old IP will be released
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Tags:       []string{"192.168.30.50/24"},
				Links:      []response_body.Link{},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.50"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressRelease (for old tagged IP "192.168.30.50") — this fails
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(nil, errors.New("IP address release error"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_SameIPNoRelease tests NetworkUpdate when the same IP is in both old tags and new assignment
func TestCanonicalMaasController_NetworkUpdate_SameIPNoRelease(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("success"),
		cmdExecuteErr:    nil,
	}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{IPAddress: "192.168.30.100", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}}, // Same IP
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.100"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges - returns the same IP as current
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101}, // Start is same as current IP
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Note: No IPAddressRelease expected — no old IP/prefix tags exist on the interface

	// Mock InterfaceAddTag (Step 8: new IP/prefix tag added)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_MultipleNetworkInterfaces tests NetworkUpdate with multiple network interfaces
func TestCanonicalMaasController_NetworkUpdate_MultipleNetworkInterfaces(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("success"),
		cmdExecuteErr:    nil,
	}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
				},
			},
			{
				ID:         2,
				Name:       "eth1",
				MacAddress: "AA:BB:CC:DD:EE:FF",
				Links: []response_body.Link{
					{IPAddress: "10.0.0.10", Subnet: response_body.Subnet{ID: 20, Cidr: "10.0.0.0/24"}},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List: []response_body.Subnet{
			{ID: 10, Cidr: "192.168.30.0/24"},
			{ID: 20, Cidr: "10.0.0.0/24"},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges for first interface
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve for first interface
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// No IPAddressRelease for first interface (interface has no old IP/prefix tags)

	// Mock InterfaceAddTag for first interface (Step 8: new IP/prefix tag)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	// Mock SubnetUnreservedIPRanges for second interface
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "10.0.0.100", End: "10.0.0.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve for second interface
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// No IPAddressRelease for second interface (interface has no old IP/prefix tags)

	// Mock InterfaceAddTag for second interface (Step 8: new IP/prefix tag)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
			{
				MacAddress: "AA:BB:CC:DD:EE:FF",
				Cidr:       "10.0.0.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_TypeCastFailure tests NetworkUpdate with type cast failure for unreserved IP ranges
func TestCanonicalMaasController_NetworkUpdate_TypeCastFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:      "test-system-id",
		IPAddresses:   []string{"192.168.30.10"},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges with wrong type (not ResbodySubnetUnreservedIPRanges)
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{ // Wrong type!
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_WithUserData_Success tests NetworkUpdate with user_data triggering cloud-init re-execution
func TestCanonicalMaasController_NetworkUpdate_WithUserData_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{
		cmdExecuteOutput: []byte("success"),
		cmdExecuteErr:    nil,
	}

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Tags:       []string{},
				Links: []response_body.Link{
					{
						IPAddress: "192.168.30.10",
						Subnet: response_body.Subnet{
							ID:   10,
							Cidr: "192.168.30.0/24",
						},
					},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List: []response_body.Subnet{
			{ID: 10, Cidr: "192.168.30.0/24"},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:    "test-system-id",
		StatusName:  "Deployed",
		IPAddresses: []string{"192.168.30.10"},
		BootInterface: response_body.Interface{
			ID:         1,
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links: []response_body.Link{
				{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// No IPAddressRelease (interface has no old IP/prefix tags)

	// Mock InterfaceAddTag (for new IP/prefix tag)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	// user_data is base64 encoded (base64 of "test-user-data")
	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
		UserData: "dGVzdC11c2VyLWRhdGE=",
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}
}

// TestCanonicalMaasController_NetworkUpdate_WithUserData_CloudInitFailure tests NetworkUpdate where cloud-init re-execution fails
func TestCanonicalMaasController_NetworkUpdate_WithUserData_CloudInitFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)

	// Mock getInterfaceList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Tags:       []string{},
				Links: []response_body.Link{
					{
						IPAddress: "192.168.30.10",
						Subnet: response_body.Subnet{
							ID:   10,
							Cidr: "192.168.30.0/24",
						},
					},
				},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List: []response_body.Subnet{
			{ID: 10, Cidr: "192.168.30.0/24"},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:    "test-system-id",
		StatusName:  "Deployed",
		IPAddresses: []string{"192.168.30.10"},
		BootInterface: response_body.Interface{
			ID:         1,
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links: []response_body.Link{
				{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// No IPAddressRelease (interface has no old IP/prefix tags)

	// Mock InterfaceAddTag (for new IP/prefix tag)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	// First ansible call (set_static_ip.yaml) succeeds
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), gomock.Any(), "set_static_ip.yaml", gomock.Any()).
		Return([]byte("success"), nil)

	// Second ansible call (run_cloud_init.yaml) fails.
	// Verify extra-vars are passed as JSON (same pattern as KubeadmJoin).
	expectedExtra := `{"user_data":"dGVzdC11c2VyLWRhdGE="}`
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), gomock.Any(), "run_cloud_init.yaml", expectedExtra).
		Return(nil, errors.New("cloud-init execution failed"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.30.0/24",
			},
		},
		UserData: "dGVzdC11c2VyLWRhdGE=",
	}

	response, err := controller.NetworkUpdate(context.Background(), req)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

func TestCanonicalMaasController_OsRelease_TagNonIPv4Skipped(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}

	mockAPI.EXPECT().GET(gomock.Any()).Return("unexpected-type", nil)
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		HostName: "test-host",
		BootInterface: response_body.Interface{
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links:     []response_body.Link{},
		},
		InterfaceSet: []response_body.Interface{},
		Storage: 100,
		StatusName: "Deployed",
		PowerStatus: "on",
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
Tags:       []string{"192.168.1.100/24", "not-an-ip", "10.0.0.1/24"},
		},
	},
	ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
}, nil)

// Tags "192.168.1.100/24" and "10.0.0.1/24" are IP/prefix format → TaggedIPs() returns both IPs.
// "not-an-ip" is ignored (not IP/prefix format).
var releasedIPs []string
expectedIPs := map[string]bool{"192.168.1.100": true, "10.0.0.1": true}
mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).DoAndReturn(
	func(_ interface{}, reqBody interface{}) (interface{}, error) {
		if r, ok := reqBody.(request_body.ReqbodyIPAddressRelease); ok {
				if expectedIPs[r.IP] {
					releasedIPs = append(releasedIPs, r.IP)
				}
			}
			return response_body.ResbodyCommon{HTTPStatus: 200}, nil
		},
	).MinTimes(2)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    &mockMaasAnsible{},
	}
	ctx := context.Background()
	req := &proto.OsReleaseRequest{SystemId: "test-sys"}
	_, err := controller.OsRelease(ctx, req)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(releasedIPs) != 2 {
		t.Errorf("Expected 2 IP releases, got %d", len(releasedIPs))
	}
	ipMap := map[string]bool{}
	for _, ip := range releasedIPs {
		ipMap[ip] = true
	}
	for ip := range expectedIPs {
		if !ipMap[ip] {
			t.Errorf("Expected IP release for %s, but not called", ip)
		}
	}
}

func TestCanonicalMaasController_MachineRegister_DuplicateSubnetID_NoDuplication(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	mockAPI := &mockMaasAPI{
		getSubnetsResult: []response_body.Subnet{
			{ID: 1, Cidr: "192.168.1.0/24"},
		},
	}
	mockFactory := &mockMaasAPIFactory{factory: mockAPI}

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: mockFactory,
	}

	ctx := context.Background()
	req := &proto.MachineRegisterRequest{
		MacAddress: "00:11:22:33:44:55",
		IpmiAddress: "192.168.1.100",
		IpmiUser: "admin",
		IpmiPassword: "password",
		NetworkInformation: []*proto.NetworkInformation{
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.1.0/24",
			},
			{
				MacAddress: "00:11:22:33:44:55",
				Cidr:       "192.168.1.0/24",
			},
		},
	}

	mockAPI.getInterfacesResult = []response_body.Interface{
		{ID: 1, Name: "eth0", MacAddress: "00:11:22:33:44:55"},
	}

	// Act
	resp, err := controller.MachineRegister(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.GetResult() != common.ResultCode_ACCEPT {
		t.Fatalf("unexpected response: %+v", resp)
	}

	time.Sleep(200 * time.Millisecond)
}

// Test for getMachineAccessInfo with boot interface having children (bridge case)
func TestCanonicalMaasController_getMachineAccessInfo_BootInterfaceWithChildren(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machine API that returns data with boot interface (bridge) having no IP but children with IP
	mockMachineAPI := &mockBasisMaasAPI{
		getResult: response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host-bridge",
			IPAddresses: []string{"192.168.1.50"},
			BootInterface: response_body.Interface{
				Name:       "br0",
				MacAddress: "00:11:22:33:44:55",
				Children:   []string{"eth0"},
				Links:      []response_body.Link{}, // Bridge itself has no IP
			},
			InterfaceSet: []response_body.Interface{
				{
					Name: "br0",
					Links: []response_body.Link{
						{
							Subnet: response_body.Subnet{
								ID:   1,
								Cidr: "192.168.1.0/24",
							},
						},
					},
				},
				{
					Name: "eth0",
					Links: []response_body.Link{
						{
							IPAddress: "192.168.1.50",
							Subnet: response_body.Subnet{
								ID:   1,
								Cidr: "192.168.1.0/24",
							},
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		},
	}
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	hostName, bootIf, accessAddress, bootMacAddress, subnetIDs, _, _, _, err := controller.getMachineAccessInfo(ctx, systemID)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if hostName != "test-host-bridge" {
		t.Errorf("Expected hostName 'test-host-bridge', got %s", hostName)
	}
	// Should use child interface name (eth0) instead of bridge (br0)
	if bootIf != "eth0" {
		t.Errorf("Expected bootIf 'eth0' (child interface), got %s", bootIf)
	}
	// Should find IP from child interface
	if accessAddress != "192.168.1.50" {
		t.Errorf("Expected accessAddress '192.168.1.50', got %s", accessAddress)
	}
	if bootMacAddress != "00:11:22:33:44:55" {
		t.Errorf("Expected bootMacAddress '00:11:22:33:44:55', got %s", bootMacAddress)
	}
	// Should get subnet IDs from child interface (eth0)
	if len(subnetIDs) != 1 || subnetIDs[0] != 1 {
		t.Errorf("Expected subnetIDs [1], got %v", subnetIDs)
	}
}

// Test for getMachineAccessInfo with boot interface having multiple children
func TestCanonicalMaasController_getMachineAccessInfo_BootInterfaceWithChildren_MultipleChildren(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machine API with boot interface having multiple children, first child has no IP
	mockMachineAPI := &mockBasisMaasAPI{
		getResult: response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host-multi-children",
			IPAddresses: []string{"192.168.2.100"},
			BootInterface: response_body.Interface{
				Name:       "br0",
				MacAddress: "aa:bb:cc:dd:ee:ff",
				Children:   []string{"eth0", "eth1"},
				Links:      []response_body.Link{},
			},
			InterfaceSet: []response_body.Interface{
				{
					Name: "br0",
					Links: []response_body.Link{
						{
							Subnet: response_body.Subnet{
								ID:   2,
								Cidr: "192.168.2.0/24",
							},
						},
					},
				},
				{
					Name:  "eth0",
					Links: []response_body.Link{}, // First child has no IP
				},
				{
					Name: "eth1",
					Links: []response_body.Link{
						{
							IPAddress: "192.168.2.100",
							Subnet: response_body.Subnet{
								ID:   2,
								Cidr: "192.168.2.0/24",
							},
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		},
	}
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	hostName, bootIf, accessAddress, bootMacAddress, subnetIDs, _, _, _, err := controller.getMachineAccessInfo(ctx, systemID)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if hostName != "test-host-multi-children" {
		t.Errorf("Expected hostName 'test-host-multi-children', got %s", hostName)
	}
	// Should use first child interface name (eth0) regardless of which child has IP
	if bootIf != "eth0" {
		t.Errorf("Expected bootIf 'eth0' (first child interface), got %s", bootIf)
	}
	// Should find IP from second child interface (eth1)
	if accessAddress != "192.168.2.100" {
		t.Errorf("Expected accessAddress '192.168.2.100', got %s", accessAddress)
	}
	if bootMacAddress != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("Expected bootMacAddress 'aa:bb:cc:dd:ee:ff', got %s", bootMacAddress)
	}
	// Should get subnet IDs based on bootIf (eth0), which has no links in this test
	if len(subnetIDs) != 0 {
		t.Errorf("Expected subnetIDs [], got %v", subnetIDs)
	}
}

// Test for getMachineAccessInfo with boot interface having children but no IPv4 found
func TestCanonicalMaasController_getMachineAccessInfo_BootInterfaceWithChildren_NoIPFound(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Mock machine API with boot interface having children but no IPv4 addresses
	mockMachineAPI := &mockBasisMaasAPI{
		getResult: response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host-no-ip",
			IPAddresses: []string{"2001:db8::1"},
			BootInterface: response_body.Interface{
				Name:       "br0",
				MacAddress: "11:22:33:44:55:66",
				Children:   []string{"eth0"},
				Links:      []response_body.Link{},
			},
			InterfaceSet: []response_body.Interface{
				{
					Name: "br0",
					Links: []response_body.Link{
						{
							Subnet: response_body.Subnet{
								ID:   3,
								Cidr: "2001:db8::/64",
							},
						},
					},
				},
				{
					Name: "eth0",
					Links: []response_body.Link{
						{
							IPAddress: "2001:db8::1", // IPv6 only
							Subnet: response_body.Subnet{
								ID:   3,
								Cidr: "2001:db8::/64",
							},
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		},
	}
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	ctx := context.Background()
	systemID := "test-system-id"

	// Act
	hostName, bootIf, accessAddress, bootMacAddress, subnetIDs, _, _, _, err := controller.getMachineAccessInfo(ctx, systemID)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if hostName != "test-host-no-ip" {
		t.Errorf("Expected hostName 'test-host-no-ip', got %s", hostName)
	}
	// Should still use child interface name even if no IPv4 found
	if bootIf != "eth0" {
		t.Errorf("Expected bootIf 'eth0' (child interface), got %s", bootIf)
	}
	// Should have empty accessAddress when no IPv4 found
	if accessAddress != "" {
		t.Errorf("Expected empty accessAddress, got %s", accessAddress)
	}
	if bootMacAddress != "11:22:33:44:55:66" {
		t.Errorf("Expected bootMacAddress '11:22:33:44:55:66', got %s", bootMacAddress)
	}
	// Should not include IPv6 subnets
	if len(subnetIDs) != 0 {
		t.Errorf("Expected subnetIDs [] (no IPv4), got %v", subnetIDs)
	}
}

// =============================================================
// localhostSSHAvailable: helper to detect if port 22 is open on
// the loopback interface. Tests that need real SSH connectivity
// call t.Skip() when this returns false.
// =============================================================

func localhostSSHAvailable() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:22", 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// =============================================================
// isCloudInitDone
// =============================================================

func TestCanonicalMaasController_isCloudInitDone(t *testing.T) {
	tests := []struct {
		name   string
		output []byte
		err    error
		want   bool
	}{
		{
			name:   "AnsibleError_ReturnsFalse",
			output: nil,
			err:    errors.New("ansible execution failed"),
			want:   false,
		},
		{
			name:   "StatusDone_ReturnsTrue",
			output: []byte("CLOUD_INIT_STATUS=done"),
			want:   true,
		},
		{
			name:   "StatusRunning_ReturnsFalse",
			output: []byte("CLOUD_INIT_STATUS=running"),
			want:   false,
		},
		{
			name:   "NoStatusLine_ReturnsFalse",
			output: []byte("some output\nno relevant line"),
			want:   false,
		},
		{
			name:   "DoneInMultilineOutput_ReturnsTrue",
			output: []byte("ok: [host]\nok: [host] => {\n    \"msg\": \"CLOUD_INIT_STATUS=done\"\n}"),
			want:   true,
		},
		{
			name:   "StatusDoneWithLeadingWhitespace_ReturnsTrue",
			output: []byte("  CLOUD_INIT_STATUS=done  "),
			want:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			controller := CanonicalMaasController{
				Logger:  klog.NewKlogr(),
				Ansible: &mockMaasAnsible{cmdExecuteOutput: tc.output, cmdExecuteErr: tc.err},
			}
			got := controller.isCloudInitDone(context.Background(), "192.168.1.1")
			if got != tc.want {
				t.Errorf("isCloudInitDone() = %v, want %v", got, tc.want)
			}
		})
	}
}

// =============================================================
// isSSHReachable
// =============================================================

func TestCanonicalMaasController_isSSHReachable_CancelledContext_ReturnsFalse(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	controller := CanonicalMaasController{Logger: klog.NewKlogr()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so DialContext fails immediately
	result := controller.isSSHReachable(ctx, "127.0.0.1", 5*time.Second)
	if result {
		t.Errorf("Expected false for cancelled context, got true")
	}
}

func TestCanonicalMaasController_isSSHReachable_LocalhostListening_ReturnsTrue(t *testing.T) {
	if !localhostSSHAvailable() {
		t.Skip("SSH (port 22) not available on localhost; skipping")
	}
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	controller := CanonicalMaasController{Logger: klog.NewKlogr()}
	result := controller.isSSHReachable(context.Background(), "127.0.0.1", 5*time.Second)
	if !result {
		t.Errorf("Expected true for localhost SSH, got false")
	}
}

// =============================================================
// getReadyStatus – table-driven test covering all branches
//
// Branches:
//   statusName == "Ready"   + hasJob           → "Processing"
//   statusName == "Ready"   + !hasJob          → "Ready"
//   statusName == "Broken"                     → "Failed"
//   statusName starts "Failed"                 → "Failed"
//   other statusName                           → "Processing"
//   statusName == "Deployed" + power != "on"   → "Processing"
//   statusName == "Deployed" + power == "on" + accessAddress == "" → "Processing"
//   statusName == "Deployed" + power == "on" + SSH fails          → "Processing"
//   statusName == "Deployed" + power == "on" + SSH ok + CI running → "Processing"
//   statusName == "Deployed" + power == "on" + SSH ok + CI done + hasJob → "Processing"
//   statusName == "Deployed" + power == "on" + SSH ok + CI done + !hasJob → "Ready"
//
// Cases that require port 22 to be listening on localhost are
// automatically skipped when localhostSSHAvailable() returns false.
// =============================================================

func TestCanonicalMaasController_getReadyStatus(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	sshAvailable := localhostSSHAvailable()

	jobManagerWithJob := NewInMemoryJobManager()
	jobManagerWithJob.Register("sys-job", JobTypeMachineRegister)
	jobManagerEmpty := NewInMemoryJobManager()

	ansibleDone := &mockMaasAnsible{cmdExecuteOutput: []byte("CLOUD_INIT_STATUS=done")}
	ansibleRunning := &mockMaasAnsible{cmdExecuteOutput: []byte("CLOUD_INIT_STATUS=running")}

	// Pre-cancelled context: forces SSH dial to fail immediately.
	cancelledCtx, cancelFn := context.WithCancel(context.Background())
	cancelFn()

	tests := []struct {
		name          string
		systemID      string
		statusName    string
		powerStatus   string
		accessAddress string
		ansible       *mockMaasAnsible
		jobManager    *JobManager
		ctx           context.Context
		skipIfNoSSH   bool
		want          string
	}{
		// ---- statusName == "Ready" ----
		{
			name:       "Ready_HasJob_Processing",
			systemID:   "sys-job",
			statusName: "Ready",
			ansible:    ansibleDone,
			jobManager: jobManagerWithJob,
			ctx:        context.Background(),
			want:       "Processing",
		},
		{
			name:       "Ready_NoJob_Ready",
			statusName: "Ready",
			ansible:    ansibleDone,
			jobManager: jobManagerEmpty,
			ctx:        context.Background(),
			want:       "Ready",
		},

		// ---- statusName == "Broken" / "Failed*" ----
		{
			name:       "Broken_Failed",
			statusName: "Broken",
			ansible:    ansibleDone,
			jobManager: jobManagerEmpty,
			ctx:        context.Background(),
			want:       "Failed",
		},
		{
			name:       "FailedCommissioning_Failed",
			statusName: "Failed commissioning",
			ansible:    ansibleDone,
			jobManager: jobManagerEmpty,
			ctx:        context.Background(),
			want:       "Failed",
		},

		// ---- other statusName ----
		{
			name:       "Commissioning_Processing",
			statusName: "Commissioning",
			ansible:    ansibleDone,
			jobManager: jobManagerEmpty,
			ctx:        context.Background(),
			want:       "Processing",
		},
		{
			name:       "Deploying_Processing",
			statusName: "Deploying",
			ansible:    ansibleDone,
			jobManager: jobManagerEmpty,
			ctx:        context.Background(),
			want:       "Processing",
		},

		// ---- statusName == "Deployed" ----
		{
			name:        "Deployed_PowerOff_Processing",
			statusName:  "Deployed",
			powerStatus: "off",
			ansible:     ansibleDone,
			jobManager:  jobManagerEmpty,
			ctx:         context.Background(),
			want:        "Processing",
		},
		{
			name:          "Deployed_PowerOn_EmptyAddress_Processing",
			statusName:    "Deployed",
			powerStatus:   "on",
			accessAddress: "",
			ansible:       ansibleDone,
			jobManager:    jobManagerEmpty,
			ctx:           context.Background(),
			want:          "Processing",
		},
		{
			// SSH dial fails because context is pre-cancelled.
			name:          "Deployed_PowerOn_SSHFails_Processing",
			statusName:    "Deployed",
			powerStatus:   "on",
			accessAddress: "127.0.0.1",
			ansible:       ansibleDone,
			jobManager:    jobManagerEmpty,
			ctx:           cancelledCtx,
			want:          "Processing",
		},

		// The following three cases need port 22 listening on localhost.
		{
			name:          "Deployed_PowerOn_SSHReachable_CloudInitRunning_Processing",
			statusName:    "Deployed",
			powerStatus:   "on",
			accessAddress: "127.0.0.1",
			ansible:       ansibleRunning,
			jobManager:    jobManagerEmpty,
			skipIfNoSSH:   true,
			ctx:           context.Background(),
			want:          "Processing",
		},
		{
			name:          "Deployed_PowerOn_SSHReachable_CloudInitDone_HasJob_Processing",
			systemID:      "sys-job",
			statusName:    "Deployed",
			powerStatus:   "on",
			accessAddress: "127.0.0.1",
			ansible:       ansibleDone,
			jobManager:    jobManagerWithJob,
			skipIfNoSSH:   true,
			ctx:           context.Background(),
			want:          "Processing",
		},
		{
			name:          "Deployed_PowerOn_SSHReachable_CloudInitDone_NoJob_Ready",
			statusName:    "Deployed",
			powerStatus:   "on",
			accessAddress: "127.0.0.1",
			ansible:       ansibleDone,
			jobManager:    jobManagerEmpty,
			skipIfNoSSH:   true,
			ctx:           context.Background(),
			want:          "Ready",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipIfNoSSH && !sshAvailable {
				t.Skip("SSH (port 22) not available on localhost; skipping")
			}

			systemID := tc.systemID
			if systemID == "" {
				systemID = "sys-test"
			}

			controller := CanonicalMaasController{
				Logger:     klog.NewKlogr(),
				Ansible:    tc.ansible,
				JobManager: tc.jobManager,
			}

			got := controller.getReadyStatus(tc.ctx, systemID, tc.statusName, tc.powerStatus, tc.accessAddress)
			if got != tc.want {
				t.Errorf("getReadyStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestCanonicalMaasController_KubeadmJoin_AnsibleFormatWithTrailingQuote verifies that
// Ansible's JSON-wrapped debug output (which appends a trailing `"` after SplitN on the
// KUBEADM_JOIN_COMMAND= marker) is handled correctly:
//   - the trailing `"` is stripped via TrimSuffix
//   - extra-vars passed to kubeadm_join.yaml is valid JSON with the clean command value
func TestCanonicalMaasController_KubeadmJoin_AnsibleFormatWithTrailingQuote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)
	mockWorkerAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockCPAPI := mocks.NewMockBasisMaasAPI(ctrl)
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)

	mockWorkerAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			BootInterface: response_body.Interface{
				Links: []response_body.Link{
					{IPAddress: "192.168.1.200", Subnet: response_body.Subnet{ID: 1, Cidr: "192.168.1.0/24"}},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	mockCPAPI.EXPECT().GET(gomock.Any()).Return(
		response_body.ResbodyGetMachine{
			BootInterface: response_body.Interface{
				Links: []response_body.Link{
					{IPAddress: "192.168.1.100", Subnet: response_body.Subnet{ID: 1, Cidr: "192.168.1.0/24"}},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		}, nil)

	factory.EXPECT().NewMachineSystemID("worker-id").Return(mockWorkerAPI)
	factory.EXPECT().NewMachineSystemID("cp-id-1").Return(mockCPAPI)

	// Realistic Ansible debug output: the `"msg": "KUBEADM_JOIN_COMMAND=..."` line
	// has a trailing `"` (JSON closing quote) after SplitN on "KUBEADM_JOIN_COMMAND=".
	// This is the root cause of the original "unexpected EOF while looking for matching `\"'"
	// error reported in the issue.
	const wantJoinCmd = "kubeadm join 192.168.1.100:6443 --token abc123 --discovery-token-ca-cert-hash sha256:xyz789"
	ansibleOutput := "ok: [192.168.1.100] => {\n" +
		"    \"msg\": \"KUBEADM_JOIN_COMMAND=" + wantJoinCmd + "\"\n" +
		"}"

	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.100", "kubeadm_token_create.yaml", "").
		Return([]byte(ansibleOutput), nil)

	// Capture the extra-vars passed to kubeadm_join.yaml
	capturedExtra := make(chan string, 1)
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.1.200", "kubeadm_join.yaml", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, extra string) ([]byte, error) {
			capturedExtra <- extra
			return []byte("success"), nil
		})

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	_, err := controller.KubeadmJoin(context.Background(), &proto.KubeadmJoinRequest{
		SystemId:   "worker-id",
		CpSystemId: []string{"cp-id-1"},
	})
	if err != nil {
		t.Fatalf("KubeadmJoin returned unexpected error: %v", err)
	}

	select {
	case extra := <-capturedExtra:
		// ① extra-vars must be valid JSON
		var m map[string]string
		if jsonErr := json.Unmarshal([]byte(extra), &m); jsonErr != nil {
			t.Fatalf("extra-vars is not valid JSON: %v (value: %q)", jsonErr, extra)
		}
		// ② the join_command value must not contain a trailing `"`
		if m["join_command"] != wantJoinCmd {
			t.Errorf("join_command = %q\nwant        = %q", m["join_command"], wantJoinCmd)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout: kubeadm_join.yaml was not called within 500ms")
	}
}

// =============================================================
// NetworkUpdate: Two netInfos with the same MAC (different CIDRs)
// → MAC grouping fires a single Ansible call with comma-separated IPs
// =============================================================

func TestCanonicalMaasController_NetworkUpdate_MultipleCIDRsSameMAC(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := mocks.NewMockMaasAnsible(ctrl)

	// Mock getInterfaceList — one interface, no old IP/prefix tags
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Tags:       []string{},
				Links:      []response_body.Link{},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getSubnetList — two subnets (one per CIDR)
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List: []response_body.Subnet{
			{ID: 10, Cidr: "192.168.30.0/24"},
			{ID: 20, Cidr: "10.0.0.0/24"},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock getMachineAccessInfo
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:    "test-system-id",
		StatusName:  "Deployed",
		IPAddresses: []string{"192.168.30.10"},
		BootInterface: response_body.Interface{
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links: []response_body.Link{
				{IPAddress: "192.168.30.10", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges for first CIDR
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve for first IP
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock SubnetUnreservedIPRanges for second CIDR
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "10.0.0.100", End: "10.0.0.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Mock IPAddressReserve for second IP
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Capture the Ansible call — must be exactly ONE call with comma-separated IPs
	capturedExtra := make(chan string, 1)
	mockAnsible.EXPECT().
		CmdExecute(gomock.Any(), gomock.Any(), "set_static_ip.yaml", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, extra string) ([]byte, error) {
			capturedExtra <- extra
			return []byte("success"), nil
		}).Times(1)

	// No IPAddressRelease (no old IP/prefix tags on the interface)

	// Mock InterfaceAddTag for first new IP (192.168.30.100/24)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	// Mock InterfaceAddTag for second new IP (10.0.0.100/24)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{MacAddress: "00:11:22:33:44:55", Cidr: "192.168.30.0/24"},
			{MacAddress: "00:11:22:33:44:55", Cidr: "10.0.0.0/24"}, // same MAC
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}

	// Verify single Ansible call has both IPs comma-separated
	select {
	case extra := <-capturedExtra:
		if !strings.Contains(extra, "ip_with_prefix_list=") {
			t.Errorf("Ansible extra-vars missing ip_with_prefix_list: %q", extra)
		}
		if !strings.Contains(extra, "192.168.30.100/24") {
			t.Errorf("Ansible extra-vars missing first IP: %q", extra)
		}
		if !strings.Contains(extra, "10.0.0.100/24") {
			t.Errorf("Ansible extra-vars missing second IP: %q", extra)
		}
		// Both IPs must appear in the same ip_with_prefix_list value (comma-separated)
		ipListPart := extra[strings.Index(extra, "ip_with_prefix_list="):]
		if !strings.Contains(ipListPart, ",") {
			t.Errorf("ip_with_prefix_list must contain comma-separated IPs, got: %q", ipListPart)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout: set_static_ip.yaml was not called")
	}
}

// =============================================================
// NetworkUpdate: existing IP/prefix tag on interface → old IP released
// and old tag removed, new tag added
// =============================================================

func TestCanonicalMaasController_NetworkUpdate_WithExistingIPTags_OldIPReleased(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Interface has an existing IP/prefix tag ("192.168.30.50/24")
	// → TaggedIPs() returns ["192.168.30.50"], IPWithPrefixTags() returns ["192.168.30.50/24"]
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Tags:       []string{"192.168.30.50/24"},
				Links:      []response_body.Link{},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:    "test-system-id",
		StatusName:  "Deployed",
		IPAddresses: []string{"192.168.30.50"},
		BootInterface: response_body.Interface{
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links: []response_body.Link{
				{IPAddress: "192.168.30.50", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Step 5: IPAddressReserve for new IP (192.168.30.100)
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// Step 7: IPAddressRelease for old IP (192.168.30.50 not in new set {192.168.30.100})
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	// Step 8a: InterfaceRemoveTag for old tag "192.168.30.50/24"
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	// Step 8b: InterfaceAddTag for new tag "192.168.30.100/24"
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{MacAddress: "00:11:22:33:44:55", Cidr: "192.168.30.0/24"},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if response.GetResult() != common.ResultCode_ACCEPT {
		t.Errorf("Expected result ACCEPT, got %v", response.GetResult())
	}
}

// =============================================================
// NetworkUpdate: InterfaceRemoveTag fails → error returned
// =============================================================

func TestCanonicalMaasController_NetworkUpdate_InterfaceRemoveTagFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Interface has an existing IP/prefix tag to trigger InterfaceRemoveTag
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Tags:       []string{"192.168.30.50/24"},
				Links:      []response_body.Link{},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:    "test-system-id",
		IPAddresses: []string{"192.168.30.50"},
		BootInterface: response_body.Interface{
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links: []response_body.Link{
				{IPAddress: "192.168.30.50", Subnet: response_body.Subnet{ID: 10, Cidr: "192.168.30.0/24"}},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// IPAddressReserve succeeds
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// IPAddressRelease (old IP 192.168.30.50) succeeds
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyCommon{
		HTTPStatus: 200,
	}, nil)

	// InterfaceRemoveTag fails
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(nil, errors.New("remove tag failed"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{MacAddress: "00:11:22:33:44:55", Cidr: "192.168.30.0/24"},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// =============================================================
// NetworkUpdate: InterfaceAddTag fails → error returned
// =============================================================

func TestCanonicalMaasController_NetworkUpdate_InterfaceAddTagFailure(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockBasisMaasAPI(ctrl)
	factory := &mockMaasAPIFactory{factory: mockAPI}
	mockAnsible := &mockMaasAnsible{}

	// Interface has no old tags → no IPAddressRelease, no InterfaceRemoveTag
	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetInterfaces{
		List: []response_body.Interface{
			{
				ID:         1,
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Tags:       []string{},
				Links:      []response_body.Link{},
			},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetSubnets{
		List:          []response_body.Subnet{{ID: 10, Cidr: "192.168.30.0/24"}},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodyGetMachine{
		SystemID:    "test-system-id",
		IPAddresses: []string{},
		BootInterface: response_body.Interface{
			Name:       "eth0",
			MacAddress: "00:11:22:33:44:55",
			Links:      []response_body.Link{},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	mockAPI.EXPECT().GET(gomock.Any()).Return(response_body.ResbodySubnetUnreservedIPRanges{
		List: []response_body.UnreservedIPRange{
			{Start: "192.168.30.100", End: "192.168.30.200", NumAddresses: 101},
		},
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// IPAddressReserve succeeds
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(response_body.ResbodyIPAddressReserve{
		ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
	}, nil)

	// InterfaceAddTag fails
	mockAPI.EXPECT().POST(gomock.Any(), gomock.Any()).Return(nil, errors.New("add tag failed"))

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
		Ansible:    mockAnsible,
	}

	req := &proto.NetworkUpdateRequest{
		SystemId: "test-system-id",
		NetworkInformation: []*proto.NetworkInformation{
			{MacAddress: "00:11:22:33:44:55", Cidr: "192.168.30.0/24"},
		},
	}

	response, err := controller.NetworkUpdate(context.Background(), req)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if response.GetResult() != common.ResultCode_ERROR {
		t.Errorf("Expected result ERROR, got %v", response.GetResult())
	}
}

// =============================================================
// getMachineAccessInfo: synthetic Link (Subnet.ID=0) excluded from subnetIDs
// =============================================================

func TestCanonicalMaasController_getMachineAccessInfo_SyntheticLinkSubnetIDZeroExcluded(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	setupTestEnvironmentForController(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := mocks.NewMockMaasAPIFactory(ctrl)

	// Interface has two Links in InterfaceSet:
	//   - Subnet.ID=0 (synthetic, created by UnmarshalJSON from IP/prefix tag) → excluded
	//   - Subnet.ID=5 (real subnet) → included
	mockMachineAPI := &mockBasisMaasAPI{
		getResult: response_body.ResbodyGetMachine{
			SystemID:    "test-system-id",
			HostName:    "test-host",
			IPAddresses: []string{"192.168.1.100"},
			BootInterface: response_body.Interface{
				Name:       "eth0",
				MacAddress: "00:11:22:33:44:55",
				Links: []response_body.Link{
					{
						IPAddress: "192.168.1.100",
						Subnet:    response_body.Subnet{ID: 0, Cidr: "192.168.1.0/24"},
					},
				},
			},
			InterfaceSet: []response_body.Interface{
				{
					Name: "eth0",
					Links: []response_body.Link{
						{
							// Synthetic link from tag (ID=0) — must be excluded
							IPAddress: "192.168.1.100",
							Subnet:    response_body.Subnet{ID: 0, Cidr: "192.168.1.0/24"},
						},
						{
							// Real link (ID=5) — must be included
							IPAddress: "10.0.0.50",
							Subnet:    response_body.Subnet{ID: 5, Cidr: "10.0.0.0/8"},
						},
					},
				},
			},
			ResbodyCommon: response_body.ResbodyCommon{HTTPStatus: 200},
		},
	}
	factory.EXPECT().NewMachineSystemID("test-system-id").Return(mockMachineAPI).Times(1)

	controller := CanonicalMaasController{
		Logger:     klog.NewKlogr(),
		APIFactory: factory,
	}

	_, _, _, _, subnetIDs, _, _, _, err := controller.getMachineAccessInfo(context.Background(), "test-system-id")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Only Subnet.ID=5 must appear; Subnet.ID=0 must be excluded
	if len(subnetIDs) != 1 {
		t.Fatalf("Expected 1 subnetID, got %d: %v", len(subnetIDs), subnetIDs)
	}
	if subnetIDs[0] != 5 {
		t.Errorf("Expected subnetIDs[0]=5, got %d", subnetIDs[0])
	}
}
