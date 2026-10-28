package interfaces

import "context"

// interface of Ansible
type Ansible interface {
	CmdExecute(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extraArgs string) (output interface{}, err error)
}

// interface of CDI Ansible
type CdiAnsible interface {
	CmdExecute(ctx context.Context, remoteHost string, remotUser string, sshPrivateKeyFile string, playbook string, extraArgs string) (errMsg *string, jsonData map[string]interface{})
}
