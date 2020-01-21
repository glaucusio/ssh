package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/glaucusio/ssh"
	"github.com/glaucusio/ssh/sshfile"
	"github.com/glaucusio/ssh/sshos"
	"github.com/glaucusio/ssh/sshtrace"
	"github.com/glaucusio/ssh/sshutil"

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
	options       []string
	verbose       bool
}

func (a *app) register(f *pflag.FlagSet) {
	f.StringVarP(&a.configfile, "config", "F", "", "")
	f.StringArrayVarP(&a.identityfiles, "identity", "i", nil, "")
	f.StringArrayVarP(&a.options, "option", "o", nil, "")
	f.BoolVarP(&a.verbose, "verbose", "v", false, "")
}

func (a *app) run(cmd *cobra.Command, args []string) error {
	auth, err := sshfile.IdentityAuth(a.identityfiles...)
	if err != nil && !errors.Is(err, sshfile.NoAuthMethods) {
		return err
	}

	c, err := sshos.NewClient(a.configfile, a.options)
	if err != nil {
		return err
	}

	if auth != nil {
		c.ConfigCallback = sshutil.PatchCallback(c.ConfigCallback, func(_ context.Context, cfg *ssh.Config) error {
			cfg.Auth = append(cfg.Auth, auth)
			return nil
		})
	}

	ctx := processContext()

	if a.verbose {
		ctx = sshtrace.WithClientTrace(ctx, sshtrace.Debug("/tmp/gossh"))
	}

	for _, arg := range args {
		cfg, err := c.ConfigCallback(ctx, "tcp", arg)
		if err != nil {
			return err
		}

		_ = cfg
	}

	_ = c

	return nil
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

func processContext() context.Context {
	return context.Background()
}
