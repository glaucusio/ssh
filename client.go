package ssh

import (
	"context"
	"errors"
)

var ErrConfigNotFound = errors.New("config not found")

type ConfigCallback func(ctx context.Context, network, address string) (*Config, error)

type Client struct {
	ConfigCallback ConfigCallback
}
