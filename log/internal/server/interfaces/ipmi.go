package interfaces

type IPMITargetList struct {
	ServerID     string
	IPMIAddress  string
	IPMIUser     string
	IPMIPassword string
	ProductInfo   		string
	ExtraParameters 	string
}

// interface of IPMI
type IPMI interface {
	Init() error
	Finalize()
	Collection(targetList []IPMITargetList)
}
