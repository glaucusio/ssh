package main

import (
	"context"
	"fmt"
	"os"

	"github.com/glaucusio/ssh/sshos"
	"github.com/glaucusio/ssh/sshtrace"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func die(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}

type app struct {
	*sshos.Loader
	verbose bool
}

func (a *app) register(f *pflag.FlagSet) {
	f.StringVarP(&a.UserConfig, "config", "F", a.UserConfig, "")
	f.StringArrayVarP(&a.Identity, "identity", "i", a.Identity, "")
	f.StringArrayVarP(&a.Options, "option", "o", a.Options, "")
	f.BoolVarP(&a.verbose, "verbose", "v", false, "")
}

func (a *app) run(cmd *cobra.Command, args []string) error {
	c, err := sshos.NewClient()
	if err != nil {
		return err
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
	app := &app{
		Loader: sshos.DefaultLoader,
	}
	cmd := newCommand(app)

	if err := cmd.Execute(); err != nil {
		die(err)
	}
}

func newCommand(a *app) *cobra.Command {
	m := &cobra.Command{
		Use:   "gossh",
		Short: "Command line interface to glaucusio/ssh",
		Args:  cobra.ArbitraryArgs,
		RunE:  a.run,
	}

	a.register(pflag.CommandLine)

	return m
}

func processContext() context.Context {
	return context.Background()
}
