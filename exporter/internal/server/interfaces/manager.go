package interfaces

// interface of Manager
type Manager interface {
	ServerList([]ServerTargetList)
	NetworkList([]NetworkTargetList)
	SetP2POn([]ServerTargetList)
}
