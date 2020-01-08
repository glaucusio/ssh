package sshfile

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

var NoAuthMethods = errors.New("no auth methods could be used")

var DefaultConfig = filepath.Join(home, ".ssh", "config")

var DefaultIdentity = []string{
	filepath.Join(home, ".ssh", "id_dsa"),
	filepath.Join(home, ".ssh", "id_ecdsa"),
	filepath.Join(home, ".ssh", "id_ed25519"),
	filepath.Join(home, ".ssh", "id_rsa"),
}

func IdentityAuth(files ...string) (ssh.AuthMethod, error) {
	if len(files) == 0 {
		files = DefaultIdentity
	}

	var signers []ssh.Signer

	for _, file := range files {
		p, err := ioutil.ReadFile(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error reading %q file: %w", file, err)
		}

		signer, err := ssh.ParsePrivateKey(p)
		if err != nil {
			return nil, fmt.Errorf("error parsing %q file: %w", file, err)
		}

		signers = append(signers, signer)

	}

	if len(signers) == 0 {
		return nil, NoAuthMethods
	}

	return ssh.PublicKeys(signers...), nil
}

var home = currentUserHomeDir()

func currentUserHomeDir() string {
	u, err := user.Current()
	if err != nil {
		panic("unexpected error reading home dir: " + err.Error())
	}

	return u.HomeDir
}
