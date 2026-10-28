package interfaces

import "context"

// interface of Ansible
type Ansible interface {
	CmdExecute(ctx context.Context, remoteHost string, remotUser string, playbook string, extraArgs string) (output interface{}, err error)
}
