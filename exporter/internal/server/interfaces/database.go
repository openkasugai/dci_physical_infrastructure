package interfaces

// interface of Database
type Database interface {
	Init() (error)
	Finalize()
	SelectServerTable() ([]ServerTargetList, error)
	SelectNwSwitchTable() ([]NetworkTargetList, error)
}
