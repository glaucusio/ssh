package sshos

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/glaucusio/ssh"
	"github.com/glaucusio/ssh/sshfile"
	"github.com/glaucusio/ssh/sshtrace"
	"github.com/glaucusio/ssh/sshutil"
	"golang.org/x/crypto/ssh/knownhosts"
)

func NewClient(config string, options []string) (*ssh.Client, error) {
	var custom sshfile.Configs

	if config != "" {
		var err error
		if custom, err = sshfile.ParseConfigFile(config); err != nil {
			return nil, fmt.Errorf("failed to parse %q custom config: %w", config, err)
		}
	}

	var mixin *sshfile.Config

	if len(options) != 0 {
		var err error
		if mixin, err = sshfile.ParseOptions(options); err != nil {
			return nil, fmt.Errorf("failed to parse options: %w", err)
		}
	}

	usr, err := sshfile.ParseConfigFile(DefaultUserConfig)
	if err != nil && !is(err, os.ErrNotExist, os.ErrPermission) {
		return nil, fmt.Errorf("failed to parse %q user config: %w", DefaultUserConfig, err)
	}

	sys, err := sshfile.ParseConfigFile(DefaultSystemConfig)
	if err != nil && !is(err, os.ErrNotExist, os.ErrPermission) {
		return nil, fmt.Errorf("failed to parse %q system config: %w", DefaultSystemConfig, err)
	}

	cfgfile := custom.Merge(usr).Merge(sys) // todo: fix move global at the end (merge globals?)

	if mixin != nil {
		for i := range cfgfile {
			if err := cfgfile[i].Merge(mixin); err != nil {
				return nil, fmt.Errorf("%d: unable to apply custom options: %w", i, err)
			}
		}
	}

	cb := cfgfile.Callback()

	auth, err := sshfile.IdentityAuth(DefaultIdentity...)
	if err != nil && !is(err, sshfile.NoAuthMethods) {
		return nil, fmt.Errorf("failed to build identity auth: %w", err)
	}

	if auth != nil {
		cb = sshutil.PatchCallback(cb, func(_ context.Context, cfg *ssh.Config) error {
			cfg.Auth = append(cfg.Auth, auth)
			return nil
		})
	}

	known, err := knownhosts.New(DefaultKnownHosts)
	if err != nil && !is(err, os.ErrNotExist, os.ErrPermission) {
		return nil, fmt.Errorf("failed to parse %q known hosts file: %w", DefaultKnownHosts, err)
	}

	if known != nil {
		cb = sshutil.PatchCallback(cb, func(_ context.Context, cfg *ssh.Config) error {
			if cfg.HostKeyCallback == nil {
				cfg.HostKeyCallback = known
			}
			return nil
		})
	}

	var once sync.Once

	cb = sshutil.PatchCallback(cb, func(ctx context.Context, cfg *ssh.Config) error {
		if ct := sshtrace.ContextClientTrace(ctx); ct != nil {
			once.Do(func() { ct.GotFileConfig(cfgfile) })
			ct.GotConfig(cfg)
		}
		return nil
	})

	c := &ssh.Client{
		ConfigCallback: cb,
	}

	return c, nil
}
