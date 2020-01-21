package ssh

import "golang.org/x/crypto/ssh"

type Conn struct {
	*ssh.Client
}
