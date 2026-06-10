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
	"encoding/json"
    "github.com/go-playground/validator/v10"
	"common/models"
	"common/models/db"
	"common/models/extra_parameters"
	commonUtils "common/utils"
	"log_module/internal/server/interfaces" // import for interface
	localUtils "log_module/internal/server/utils"
	"k8s.io/klog/v2"
)

var validate *validator.Validate

func init() {
    validate = validator.New(validator.WithRequiredStructEnabled())
}

// struct of Database
type DatabaseImplement struct {
	Logger klog.Logger
	API    interfaces.API
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

func (l *DatabaseImplement) SelectCDITable() (targetList []interfaces.CDITargetList, err error) {
	l.Logger.V(2).Info("start SelectCDITable")
	defer func() {
		l.Logger.V(2).Info("end SelectCDITable", "targetListCount", len(targetList), "err", err)
	}()

	l.Logger.Info("selecting CDI table data")

	// API call to get cdi data
	cdi := db.TCdi{}
	resp, err := l.API.APIExecuteJWTAUth(context.Background(), "GET", l.AccessURL, cdi.TableName(), l.JWT, "")
	if err != nil {
		l.Logger.V(2).Info("branch: API call to get cdi data failed", "error", err.Error())
		return
	}
	// Parse response and populate targetList
	cdis, err := cdi.Parse(resp)
	if err != nil {
		l.Logger.V(2).Info("branch: invalid data format in API response", "error", err.Error())
		return
	}
	l.Logger.V(2).Info("branch: database query successful", "recordCount", len(cdis))

	for _, cdi := range cdis {
		product := models.ParseProductTypeFromJSON[models.CDIProductType](cdi.ProductInfo)
		if (product == models.PG_CDI_1_0 || product == models.PG_CDI_1_1) {	// PG-CDI
			// parse extra parameter
			extraParameter, err := ParseExtraParameter(cdi.ExtraParameters)
			if err != nil {
				continue
			}

			targetList = append(targetList, interfaces.CDITargetList{
				CDIHost:            cdi.RemoteHost,
				CDIHostUser:		cdi.RemoteUser,
				CDISoftUser:		extraParameter.CDIUser,
				CDISoftPassword:	extraParameter.CDIPassword,
				ProductInfo	: 		cdi.ProductInfo,
				ExtraParameters:	cdi.ExtraParameters,
			})
		} else {
			l.Logger.V(2).Info("branch: ", "target is unsupport product", "ProductInfo", cdi.ProductInfo)
			continue
		}

		l.Logger.V(2).Info("branch: added CDI target", "cdiHost", cdi.RemoteHost, "ProductInfo", cdi.ProductInfo)
	}

	l.Logger.Info("CDI table selection completed", "targetCount", len(targetList))
	return
}

func (l *DatabaseImplement) SelectServerTable() (targetList []interfaces.IPMITargetList, err error) {
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

		var target interfaces.IPMITargetList

		// in caes os COTS server
		if server.CotsServerID != nil {
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
			target.ServerID = cots.ServerID
			target.IPMIAddress = cots.IpmiAddress
			target.IPMIUser = cots.IpmiUser
			target.IPMIPassword = cots.IpmiPassword
			target.ProductInfo = cots.ProductInfo
			target.ExtraParameters = cots.ExtraParameters
			l.Logger.V(2).Info("branch: COTS server processed", "serverID", cotsServer.ServerID, "ipmiAddress", cotsServer.IpmiAddress)
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
			cdi := cdiServers[0]
			target.ServerID = cdi.ServerID
			target.IPMIAddress = cdi.IpmiAddress
			target.IPMIUser = cdi.IpmiUser
			target.IPMIPassword = cdi.IpmiPassword
			target.ProductInfo = cdi.ProductInfo
			target.ExtraParameters = cdi.ExtraParameters
			l.Logger.V(2).Info("branch: CDI server processed", "serverID", cdiServer.ServerID, "ipmiAddress", cdiServer.IpmiAddress)
		}

		targetList = append(targetList, target)
		l.Logger.V(2).Info("branch: server target added", "serverID", target.ServerID, "ProductInfo", target.ProductInfo)
	}

	l.Logger.Info("server table selection completed", "targetCount", len(targetList))
	return
}

func (l DatabaseImplement) SelectMaasTable() (targetList []interfaces.MaasServerTargetList, err error) {
	l.Logger.V(2).Info("start SelectMaasTable")
	defer func() {
		l.Logger.V(2).Info("end SelectMaasTable", "targetListCount", len(targetList), "err", err)
	}()

	l.Logger.Info("selecting maas table data")

	// API call to get Maas server data
	maasServer := db.TMaas{}
	resp, err := l.API.APIExecuteJWTAUth(context.Background(), "GET", l.AccessURL, maasServer.TableName(), l.JWT, "")
	if err != nil {
		l.Logger.V(2).Info("branch: maas server query failed", "error", err.Error())
		return
	}
	// Parse response and populate targetList
	servers, err := maasServer.Parse(resp)
	if err != nil {
		l.Logger.V(2).Info("branch: invalid data format in API response", "error", err.Error())
		return
	}
	l.Logger.V(2).Info("branch: maas server query successful", "serverCount", len(servers))
	
	for _, server := range servers {
		l.Logger.V(2).Info("processing maas server", "maasId", server.MaasID, "physicalInfraId", server.PhysicalInfraId)

		var target interfaces.MaasServerTargetList
		target.MaasAccessUrl = server.AccessURL
		target.MaasApiKey = server.ApiKey
		target.ProductInfo = server.ProductInfo
		targetList = append(targetList, target)

		l.Logger.V(2).Info("branch: maas server target added", "maasId", server.MaasID, "physicalInfraId", server.PhysicalInfraId)
	}

	l.Logger.Info("maas server table selection completed", "targetCount", len(targetList))
	return
}

// ParseExtraParameter parses JSON string from extra_parameter field
func ParseExtraParameter(extraParamStr string) (*extra_parameters.PgCDIExtraParameters, error) {

	var param extra_parameters.PgCDIExtraParameters
	if err := json.Unmarshal([]byte(extraParamStr), &param); err != nil {
		return nil, err
	}

	// validation
    if err := validate.Struct(&param); err != nil {
		return nil, err
	}

	return &param, nil
}
