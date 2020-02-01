package sshutil

import (
	"context"
	"fmt"

	"github.com/glaucusio/ssh"
	"github.com/glaucusio/xerrors"
)

func Callback(callbacks ...ssh.ConfigCallback) ssh.ConfigCallback {
	return func(ctx context.Context, network, address string) (*ssh.Config, error) {
		for _, cb := range callbacks {
			cfg, err := cb(ctx, network, address)
			if err == nil {
				return cfg, nil
			}
			if xerrors.Is(err, ssh.ErrConfigNotFound) {
				continue
			}
			return nil, err
		}
		return nil, ssh.ErrConfigNotFound
	}
}

func LazyCallback(fns ...func() ssh.ConfigCallback) ssh.ConfigCallback {
	return func(ctx context.Context, network, address string) (*ssh.Config, error) {
		var callbacks []ssh.ConfigCallback

		for _, fn := range fns {
			callbacks = append(callbacks, fn())
		}

		return Callback(callbacks...)(ctx, network, address)
	}
}

func PatchCallback(cb ssh.ConfigCallback, patch func(context.Context, *ssh.Config) error) ssh.ConfigCallback {
	return func(ctx context.Context, network, address string) (*ssh.Config, error) {
		cfg, err := cb(ctx, network, address)
		if err != nil {
			return nil, err
		}

		if err := patch(ctx, cfg); err != nil {
			return nil, fmt.Errorf("failed to patch config: %w", err)
		}

		return cfg, nil
	}
}
