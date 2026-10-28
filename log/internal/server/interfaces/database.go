package interfaces

// interface of Database
type Database interface {
	Init() (error)
	Finalize()
	SelectCDITable() ([]CDITargetList, error)
	SelectServerTable() ([]IPMITargetList, error)
	SelectMaasTable() ([]MaasServerTargetList, error)
}
