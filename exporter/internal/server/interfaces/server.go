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

package interfaces

type ServerTargetList struct {
	ServerID      		string
	HostIPAddress 		string
	LoginUser     		string
	IpmiAddress   		string
	IpmiUser      		string
	IpmiPassword  		string
	P2PEnable			bool
	PowerON				bool
	UptimeSeconds  		int64
	CdiInfo				CdiInfo
	ProductInfo   		string
	ExtraParameters 	string
	WritedMetrics  		[]MetricLabel
}

type CdiInfo struct {
	RemoteHost        	string
	RemoteUser        	string
	MachineName			string
	GroupName         	string
	ProductInfo   		string
	ExtraParameters 	string
}

// interface of Server
type Server interface {
	Init() error
	Finalize()
	Colloction(targetList []ServerTargetList)
}
