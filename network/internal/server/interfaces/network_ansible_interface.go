package interfaces

import (
	"context"
    common "common/api/proto"    // import of common protobuf
)

// interface of network API
type NetworkAnsible interface {

	// Command execution
	CmdExecute(ctx context.Context, remoteHost string, remoteUser string, sshPrivateKeyFile string, playbook string, extrArgs string) ([]byte, *common.ErrorMessage)
}
