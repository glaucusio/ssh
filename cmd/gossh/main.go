package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/glaucusio/ssh"
	"github.com/glaucusio/ssh/sshfile"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func die(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}

type app struct {
	configfile    string
	identityfiles []string
}

func (a *app) register(f *pflag.FlagSet) {
	f.StringVarP(&a.configfile, "config", "F", sshfile.DefaultConfig, "")
	f.StringArrayVarP(&a.identityfiles, "identity", "i", sshfile.DefaultIdentity, "")
}

func (a *app) run(cmd *cobra.Command, args []string) error {
	filecfg, err := a.parseConfigFile()
	if err != nil {
		return err
	}

	identity, err := sshfile.IdentityAuth(a.identityfiles...)
	if err != nil && !errors.Is(err, sshfile.NoAuthMethods) {
		return err
	}

	cfg := new(ssh.Config).
		With(filecfg).
		WithAuth(identity)

	client, err := ssh.NewClient(cfg)
	if err != nil {
		return err
	}

	_ = client

	return nil
}

func (a *app) parseConfigFile() (*sshfile.Config, error) {
	f, err := os.Open(a.configfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return sshfile.ParseConfig(f)
}

func main() {
	app := new(app)
	cmd := newCommand(app)

	if err := cmd.Execute(); err != nil {
		die(err)
	}
}

func newCommand(a *app) *cobra.Command {
	m := &cobra.Command{
		Use:   "gossh",
		Short: "Command line interface to glaucusio/ssh",
		Args:  cobra.NoArgs,
		RunE:  a.run,
	}

	a.register(pflag.CommandLine)

	return m
}
