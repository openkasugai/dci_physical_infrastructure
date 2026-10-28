package interfaces

type MaasServerTargetList struct {
	MaasAccessUrl        string
	MaasApiKey   		string
	ProductInfo   		string
}

// interface of MaasServer
type MaasServer interface {
	Init() error
	Finalize()
	Collection(targetList []MaasServerTargetList)
}
