package ssh

import "golang.org/x/crypto/ssh"

type Option interface {
	Apply(*Config) *Config
}

type Config struct {
	ssh.ClientConfig
}

func (cfg *Config) With(opts ...Option) *Config {
	for _, opt := range opts {
		cfg = opt.Apply(cfg)
	}
	return cfg
}

func (cfg *Config) WithAuth(methods ...ssh.AuthMethod) *Config {
	cfg.ClientConfig.Auth = append(cfg.ClientConfig.Auth, methods...)
	return cfg
}
