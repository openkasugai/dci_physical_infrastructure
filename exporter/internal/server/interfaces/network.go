package interfaces

type NetworkTargetList struct {
	IPAddress 		string
	LoginUser 		string
	ProductInfo   	string
	ExtraParameters string
	WritedMetrics  	[]MetricLabel
}

// interface of Network
type Network interface {
	Init() error
	Finalize()
	Collection(targetList []NetworkTargetList)
}
