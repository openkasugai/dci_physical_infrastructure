package interfaces

type ServerTargetList struct {
	ServerID      		string
	SystemID  			string
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
