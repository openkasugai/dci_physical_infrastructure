package interfaces

import (
    common "common/api/proto"    // import of common protobuf
	"context"
)

// interface of PRIMAGY-CDI API
type CDIAnsible interface {

	// Command execution
	CmdExecute(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extrArgs string) (*common.ErrorMessage, map[string]interface{})
}
