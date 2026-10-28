package interfaces

import (
	"context"
)

// interface of Ansible
type MaasAnsible interface {

	// Command execution
	CmdExecute(ctx context.Context, remoteHost string, playbook string, extrArgs string, retryCount ...int) (output []byte, err error)
}
