package broadcom_sonic_network

import (
	edgecore_sonic_network "network_module/internal/server/implementation/edgecore_sonic_network" // import edgecore sonic network implement
)

// struct of Broadcom-SONiC network controller
// Embeds EdgeCoreSonicNetworkController to reuse its VLAN operations
type BroadcomSonicNetworkController struct {
	edgecore_sonic_network.EdgeCoreSonicNetworkController
}
