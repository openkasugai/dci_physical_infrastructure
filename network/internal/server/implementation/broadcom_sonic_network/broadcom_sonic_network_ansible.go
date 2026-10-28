package broadcom_sonic_network

import (
	edgecore_sonic_network "network_module/internal/server/implementation/edgecore_sonic_network" // import edgecore sonic network implement
)

// struct of Broadcom-SONiC network ansible
// Embeds EdgeCoreSonicAnsible with AnsibleSubDir set to "broadcom_sonic_network"
type BroadcomSonicAnsible struct {
	edgecore_sonic_network.EdgeCoreSonicAnsible
}
