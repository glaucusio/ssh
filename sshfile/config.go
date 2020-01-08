package sshfile

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/glaucusio/ssh"

	"github.com/spf13/pflag"
)

var DefaultHostConfig = &HostConfig{}

type HostConfig struct {
	Port                  int      `json:"port,string,omitempty"`
	StrictHostKeyChecking *Bool    `json:"stricthostkeychecking,omitempty"`
	GlobalKnownHostsFile  string   `json:"globalknownhostsfile,omitempty"`
	UserKnownHostsFile    string   `json:"userknownhostsfile,omitempty"`
	TcpKeepAlive          *Bool    `json:"tcpkeepalive,omitempty"`
	ConnectTimeout        Duration `json:"connecttimeout,omitempty"`
	ConnectionAttempts    int      `json:"connectionattempts,string,omitempty"`
	ServerAliveInterval   Duration `json:"serveraliveinterval,omitempty"`
	ServerAliveCountMax   int      `json:"serveralivecountmax,string,omitempty"`
	Hostname              string   `json:"hostname,omitempty"`
	User                  string   `json:"user,omitempty"`
	IdentityFile          string   `json:"identityfile,omitempty"`
}

func (hc *HostConfig) Merge(in *HostConfig) error {
	return merge(hc, in)
}

var _ = new(HostConfig).clone()

func (hc *HostConfig) clone() *HostConfig {
	if hc == nil {
		panic("called clone() on nil object")
	}

	var hcCopy HostConfig

	if err := merge(&hcCopy, hc); err != nil {
		panic("unexpected error: " + err.Error())
	}

	return &hcCopy
}

type Config struct {
	global *HostConfig
	local  []*HostConfig
	hosts  []struct {
		r   *regexp.Regexp
		ref int
	}
}

var _ ssh.Option = (*Config)(nil)

func (c *Config) Host(hostname string) *HostConfig {
	for _, h := range c.hosts {
		if h.r.MatchString(hostname) {
			cfg := c.global.clone()

			if err := merge(cfg, c.local[h.ref]); err != nil {
				panic("unexpected error: " + err.Error())
			}

			return cfg
		}
	}
	return nil
}

func (c *Config) Apply(cfg *ssh.Config) *ssh.Config {
	return cfg
}

var globToRegexp = strings.NewReplacer(
	".", `\.`,
	"*", ".*",
	"?", ".",
)

func ParseConfig(r io.Reader) (*Config, error) {
	const (
		stateGlobal = 1 << iota
		stateHost
	)

	var (
		scanner = bufio.NewScanner(r)
		cfg     = &Config{global: new(HostConfig)}
		tmp     = make(map[string]string)
		state   = stateGlobal
		lineno  = 1
	)

	for ; scanner.Scan(); lineno++ {
		s := scanner.Text()
		ts := strings.TrimSpace(s)

		switch {
		case strings.HasPrefix(ts, "#") || ts == "":
			// ignore line
		case strings.HasPrefix(s, " ") || strings.HasPrefix(s, "\t"):
			switch state {
			case stateGlobal:
				return nil, fmt.Errorf("unexpected indentation at line %d", lineno)
			case stateHost:
				k, v, err := parsekv(ts)
				if err != nil {
					return nil, fmt.Errorf("unexpected line %d: %s", lineno, err)

				}

				tmp[strings.ToLower(k)] = v
			}
		case strings.HasPrefix(s, "Host "):
			switch state {
			case stateGlobal:
				if err := merge(cfg.global, tmp); err != nil {
					return nil, fmt.Errorf("unexpected host configuration at line %d: %+v (%s)", lineno, tmp, err)
				}

				tmp = make(map[string]string)
				state = stateHost
			case stateHost:
				if err := merge(cfg.local[len(cfg.local)-1], tmp); err != nil {
					return nil, fmt.Errorf("unexpected host configuration at line %d: %+v (%s)", lineno, tmp, err)
				}

				tmp = make(map[string]string)
			}

			for _, host := range strings.Split(strings.TrimSpace(strings.TrimPrefix(ts, "Host")), " ") {
				r, err := regexp.Compile(globToRegexp.Replace(strings.TrimSpace(host)))
				if err != nil {
					return nil, fmt.Errorf("unexpected host at line %d: %q (%s)", lineno, host, err)
				}

				cfg.hosts = append(cfg.hosts, struct {
					r   *regexp.Regexp
					ref int
				}{
					r:   r,
					ref: len(cfg.local),
				})
			}

			cfg.local = append(cfg.local, new(HostConfig))
		default:
			switch state {
			case stateGlobal:
				k, v, err := parsekv(ts)
				if err != nil {
					return nil, fmt.Errorf("unexpected line %d: %s", lineno, err)
				}

				tmp[k] = v
			case stateHost:
				return nil, fmt.Errorf("unexpected line %d", lineno)
			}
		}
	}

	if len(tmp) != 0 && state == stateHost {
		if err := merge(cfg.local[len(cfg.local)-1], tmp); err != nil {
			return nil, fmt.Errorf("unexpected host configuration at line %d: %+v (%s)", lineno, tmp, err)
		}

		tmp = nil
	}

	return cfg, nil
}

func ParseFlags(flags []string) (*HostConfig, error) {
	var kv []string

	f := pflag.NewFlagSet("ssh", pflag.ContinueOnError)
	f.StringArrayVarP(&kv, "option", "o", nil, "")

	if err := f.Parse(flags); err != nil {
		return nil, fmt.Errorf("unable to parse flags: %w", err)
	}

	tmp := make(map[string]string)

	for _, kv := range kv {
		k, v, err := parsekv(kv)
		if err != nil {
			return nil, fmt.Errorf("unexpected %q flag: %w", kv, err)
		}

		tmp[strings.ToLower(k)] = v
	}

	hc := new(HostConfig)

	if err := merge(hc, tmp); err != nil {
		return nil, fmt.Errorf("unexpected flags: %w", err)
	}

	return hc, nil
}

func merge(orig interface{}, in ...interface{}) error {
	if len(in) == 0 {
		return nil
	}
	for _, v := range in {
		p, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}
		if err := json.Unmarshal(p, orig); err != nil {
			return fmt.Errorf("failed to unmarshal: %w", err)
		}
	}
	return nil
}

func umin(i, j int) int {
	if j > -1 && (i > j || i == -1) {
		return j
	}
	return i
}

func parsekv(line string) (k, v string, err error) {
	i := umin(strings.IndexRune(line, ' '), strings.IndexRune(line, '='))
	if i == -1 {
		return "", "", errors.New("delimiter not found")
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), nil
}
