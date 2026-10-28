package interfaces

type CDITargetList struct {
	CDIHost             string
	CDIHostUser		 	string
	CDISoftUser		 	string
	CDISoftPassword		string
	ProductInfo   		string
	ExtraParameters 	string
}

// interface of CDI
type CDI interface {
	Init() error
	Finalize()
	Collection(targetList []CDITargetList)
}
