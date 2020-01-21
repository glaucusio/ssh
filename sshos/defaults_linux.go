package sshos

import (
	"os/user"
	"path/filepath"
)

var (
	DefaultUserConfig = filepath.Join(home, ".ssh", "config")

	DefaultSystemConfig = filepath.FromSlash("/etc/ssh/ssh_config")

	DefaultKnownHosts = filepath.Join(home, ".ssh", "known_hosts")

	DefaultIdentity = []string{
		filepath.Join(home, ".ssh", "id_dsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
	}
)

var home = currentUserHomeDir()

func currentUserHomeDir() string {
	u, err := user.Current()
	if err != nil {
		panic("unexpected error reading home dir: " + err.Error())
	}

	return u.HomeDir
}
