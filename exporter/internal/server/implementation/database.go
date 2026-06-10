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

package implementation

import (
	"context"
	"fmt"
	"common/models/db"
	commonUtils "common/utils"
	"exporter_module/internal/server/interfaces" // import for interface
	localUtils "exporter_module/internal/server/utils"
	"k8s.io/klog/v2"
)

// struct of Database
type DatabaseImplement struct {
	Logger 		klog.Logger
	API    		interfaces.API
	AccessURL	string
	JWT	   		string
}

func (l *DatabaseImplement) Init() (err error) {
	l.Logger.V(2).Info("start Init")
	defer func() {
		l.Logger.V(2).Info("end Init", "err", err)
	}()

	l.Logger.Info("initializing database connection")

	l.AccessURL = localUtils.GetConfig().DbAccessURL
	
	// Get JWT from Kubernetes Secret
	l.Logger.Info("retrieving JWT from Kubernetes Secret")
	l.JWT, err = commonUtils.GetSecretData("postgrest-jwt", "postgrest", "jwt")
	if err != nil {
		l.Logger.Error(err, "failed to retrieve JWT from Kubernetes Secret")
		return fmt.Errorf("failed to retrieve JWT: %w", err)
	}
	l.Logger.V(2).Info("branch: JWT retrieved successfully")

	l.Logger.V(2).Info("branch: database connection established successfully")
	l.Logger.Info("database initialization completed successfully")
	return
}

func (l *DatabaseImplement) Finalize() {
	l.Logger.V(2).Info("start Finalize")
	defer func() {
		l.Logger.V(2).Info("end Finalize")
	}()

	// Nothing to do.
	return
}

func (l *DatabaseImplement) SelectNwSwitchTable() (targetList []interfaces.NetworkTargetList, err error) {
	l.Logger.V(2).Info("start SelectNwSwitchTable")
	defer func() {
		l.Logger.V(2).Info("end SelectNwSwitchTable", "targetListCount", len(targetList), "err", err)
	}()

	l.Logger.Info("selecting network switch table data")

	// API call to get network switch data
	nw := db.TNwSwitch{}
	resp, err := l.API.APIExecuteJWTAUth(context.Background(), "GET", l.AccessURL, nw.TableName(), l.JWT, "")
	if err != nil {
		l.Logger.V(2).Info("branch: API call to get network switch data failed", "error", err.Error())
		return
	}

	// Parse response and populate targetList
	nws, err := nw.Parse(resp)
	if err != nil {
		l.Logger.V(2).Info("branch: invalid data format in API response", "error", err.Error())
		return
	}
	l.Logger.V(2).Info("branch: database query successful", "recordCount", len(nws))

	for _, nw := range nws {
		targetList = append(targetList, interfaces.NetworkTargetList{
			IPAddress: nw.NwIPAddress,
			LoginUser: nw.NwUser,
			ProductInfo: nw.ProductInfo,
			ExtraParameters: nw.ExtraParameters,
		})
		l.Logger.V(2).Info("branch: added network target", "ipAddress", nw.NwIPAddress, "loginUser", nw.NwUser)
	}

	l.Logger.Info("network switch table selection completed", "targetCount", len(targetList))
	return
}

func (l *DatabaseImplement) SelectServerTable() (targetList []interfaces.ServerTargetList, err error) {
	l.Logger.V(2).Info("start SelectServerTable")
	defer func() {
		l.Logger.V(2).Info("end SelectServerTable", "targetListCount", len(targetList), "err", err)
	}()

	l.Logger.Info("selecting server table data")

	// API call to get logical server data
	logicalServer := db.TLogicalServer{ Status: 1 }
	resp, err := l.API.APIExecuteJWTAUth(context.Background(), "GET", l.AccessURL, logicalServer.TableName(), l.JWT, logicalServer.QueryParameter())
	if err != nil {
		l.Logger.V(2).Info("branch: logical server query failed", "error", err.Error())
		return
	}
	// Parse response and populate targetList
	servers, err := logicalServer.Parse(resp)
	if err != nil {
		l.Logger.V(2).Info("branch: invalid data format in API response", "error", err.Error())
		return
	}
	l.Logger.V(2).Info("branch: logical server query successful", "serverCount", len(servers))

	for _, server := range servers {
		l.Logger.V(2).Info("processing server", "serverCotsID", server.CotsServerID, "serverCdiID", server.CdiComputeServerID)

		var target interfaces.ServerTargetList

		// in caes os COTS server or VM
		if server.CotsServerID != nil {

			// in case of VM
			if server.ServerType == 2 {
			l.Logger.V(2).Info("branch: processing VM", "cotsServerID", *server.CotsServerID)
				target.P2PEnable = server.P2PEnabled
				target.ServerID = *server.CotsServerID
				l.Logger.V(2).Info("branch: VM processed", "serverID", *server.CotsServerID)
			} else {	// in case of COTS server
				l.Logger.V(2).Info("branch: processing COTS server", "cotsServerID", *server.CotsServerID)

				// API call to get COTS server data
				var resp interface{}
				cotsServer := db.TCotsServerPool{ ServerID: *server.CotsServerID }
				resp, err = l.API.APIExecuteJWTAUth(context.Background(), "GET", l.AccessURL, cotsServer.TableName(), l.JWT, cotsServer.QueryParameter())
				if err != nil {
					l.Logger.V(2).Info("branch: COTS server query failed", "error", err.Error())
					return
				}
				var cotsServers []db.TCotsServerPool
				cotsServers, err = cotsServer.Parse(resp)
				if err != nil || len(cotsServers) == 0 {
					l.Logger.V(2).Info("branch: invalid data format in COTS server API response", "error", err.Error())
					return
				}
				cots := cotsServers[0]
				target.P2PEnable = server.P2PEnabled
				target.ServerID = cots.ServerID
				target.IpmiAddress = cots.IpmiAddress
				target.IpmiUser = cots.IpmiUser
				target.IpmiPassword = cots.IpmiPassword
				target.ProductInfo = cots.ProductInfo
				target.ExtraParameters = cots.ExtraParameters
				l.Logger.V(2).Info("branch: COTS server processed", "serverID", cots.ServerID, "ipmiAddress", cots.IpmiAddress)
			}
		} else { // in case os Composed server
			l.Logger.V(2).Info("branch: processing CDI server", "cdiComputeServerID", *server.CdiComputeServerID)

			// API call to get CDI server data
			var resp interface{}
			cdiServer := db.TCdiComputePool{ ServerID: *server.CdiComputeServerID }
			resp, err = l.API.APIExecuteJWTAUth(context.Background(), "GET", l.AccessURL, cdiServer.TableName(), l.JWT, cdiServer.QueryParameter())
			if err != nil {
				l.Logger.V(2).Info("branch: CDI server query failed", "error", err.Error())
				return
			}
			var cdiServers []db.TCdiComputePool
			cdiServers, err = cdiServer.Parse(resp)
			if err != nil || len(cdiServers) == 0 {
				l.Logger.V(2).Info("branch: invalid data format in CDI server API response", "error", err.Error())
				return
			}
			cdiSv := cdiServers[0]

			cdiTbl := db.TCdi{ CdiID: cdiSv.CdiID }
			resp, err = l.API.APIExecuteJWTAUth(context.Background(), "GET", l.AccessURL, cdiTbl.TableName(), l.JWT, cdiTbl.QueryParameter())
			if err != nil {
				l.Logger.V(2).Info("branch: CDI table query failed", "error", err.Error())
				return
			}
			var cdiTbls []db.TCdi
			cdiTbls, err = cdiTbl.Parse(resp)
			if err != nil || len(cdiTbls) == 0 {
				l.Logger.V(2).Info("branch: invalid data format in CDI table API response", "error", err.Error())
				return
			}
			cdi := cdiTbls[0]

			target.P2PEnable = server.P2PEnabled
			target.ServerID = cdiSv.ServerID
			target.IpmiAddress = cdiSv.IpmiAddress
			target.IpmiUser = cdiSv.IpmiUser
			target.IpmiPassword = cdiSv.IpmiPassword
			target.ProductInfo = cdiSv.ProductInfo
			target.ExtraParameters = cdiSv.ExtraParameters
			target.CdiInfo = interfaces.CdiInfo{
				RemoteHost: cdi.RemoteHost,
				RemoteUser: cdi.RemoteUser,
				MachineName: *server.CdiMachineName,
				GroupName: cdi.GroupName,
				ProductInfo: cdi.ProductInfo,
				ExtraParameters: cdi.ExtraParameters,
			}
			l.Logger.V(2).Info("branch: CDI server processed", "serverID", cdiSv.ServerID, "ipmiAddress", cdiSv.IpmiAddress)
		}

		// API call to get OS info data
		var resp interface{}
		osInfo := db.TOsInfo{ ID: *server.OsID }
		resp, err = l.API.APIExecuteJWTAUth(context.Background(), "GET", l.AccessURL, osInfo.TableName(), l.JWT, osInfo.QueryParameter())
		if err != nil {
			l.Logger.V(2).Info("branch: OS info query failed", "error", err.Error())
			return
		}
		var osInfos []db.TOsInfo
		osInfos, err = osInfo.Parse(resp)
		if err != nil || len(osInfos) == 0 {
			l.Logger.V(2).Info("branch: invalid data format in OS info API response", "error", err.Error())
			return
		}
		os := osInfos[0]

		l.Logger.V(2).Info("branch: OS info retrieved", "osID", server.OsID, "loginUser", os.LoginUser)

		if server.MgrIPAddress != nil {
			target.HostIPAddress = *server.MgrIPAddress
			l.Logger.V(2).Info("branch: manager IP address set", "mgrIPAddress", *server.MgrIPAddress)
		} else {
			l.Logger.V(2).Info("branch: manager IP address is nil")
		}
		target.LoginUser = os.LoginUser
		targetList = append(targetList, target)

		l.Logger.V(2).Info("branch: server target added", "serverID", target.ServerID, "ProductInfo", target.ProductInfo)
	}

	l.Logger.Info("server table selection completed", "targetCount", len(targetList))
	return
}
