package sshfile

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/glaucusio/ssh"
	"github.com/glaucusio/ssh/sshutil"

	xssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var globalHost = Host{
	Regexp: regexp.MustCompile(".*"),
}

type Config struct {
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
	Host                  Host     `json:"host,omitempty"`
}

func (c *Config) Merge(in *Config) error {
	return merge(c, in)
}

var _ = new(Config).clone()

func (c *Config) clone() *Config {
	if c == nil {
		panic("called clone() on nil object")
	}

	var cCopy Config

	if err := merge(&cCopy, c); err != nil {
		panic("unexpected error: " + err.Error())
	}

	return &cCopy
}

func (c *Config) build() (*ssh.Config, error) {
	cfg := &ssh.Config{
		ClientConfig: xssh.ClientConfig{
			User:    c.User,
			Timeout: c.ConnectTimeout.Duration(),
		},
		Network:   "tcp",
		Address:   c.Hostname,
		KeepAlive: true,
		ServerAlive: ssh.Heartbeat{
			Interval: c.ServerAliveInterval.Duration(),
			MaxCount: c.ServerAliveCountMax,
		},
	}

	if c.TcpKeepAlive != nil {
		cfg.KeepAlive = c.TcpKeepAlive.Bool()
	}

	if c.StrictHostKeyChecking == nil || c.StrictHostKeyChecking.Bool() {
		if files := nonempty(c.GlobalKnownHostsFile, c.UserKnownHostsFile); len(files) != 0 {
			known, err := knownhosts.New(files...)
			if err != nil {
				return nil, fmt.Errorf("failed to build known hosts list: %w", err)
			}

			cfg.HostKeyCallback = known
		}
	} else {
		cfg.HostKeyCallback = xssh.InsecureIgnoreHostKey()
	}

	if c.Port != 0 {
		cfg.Address = net.JoinHostPort(cfg.Address, strconv.Itoa(c.Port))
	} else {
		cfg.Address = net.JoinHostPort(cfg.Address, "22")
	}

	if c.IdentityFile != "" {
		auth, err := IdentityAuth(c.IdentityFile)
		if err != nil {
			return nil, fmt.Errorf("failed to build identity auth: %w", err)
		}

		cfg.Auth = append(cfg.Auth, auth)
	}

	// todo?

	return cfg, nil
}

func (c *Config) Callback() ssh.ConfigCallback {
	return func(ctx context.Context, network, address string) (*ssh.Config, error) {
		if !c.Host.MatchString(address) {
			return nil, ssh.ErrConfigNotFound
		}

		cfg, err := c.build()
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %w", err)
		}

		return cfg, nil
	}
}

type Host struct {
	*regexp.Regexp
}

var (
	_ json.Marshaler   = Host{}
	_ json.Unmarshaler = (*Host)(nil)
)

func (h Host) MarshalJSON() ([]byte, error) {
	if h.Regexp == nil {
		return []byte(`""`), nil
	}
	return json.Marshal(h.String())
}

func (h *Host) UnmarshalJSON(p []byte) error {
	var expr string

	if err := json.Unmarshal(p, &expr); err != nil {
		return err
	}

	if expr == "" {
		h.Regexp = nil
		return nil
	}

	r, err := regexp.Compile(expr)
	if err != nil {
		return err
	}

	h.Regexp = r

	return nil
}

func (h Host) Equal(rhs Host) bool {
	if h.Regexp == nil && rhs.Regexp == nil {
		return true
	}
	return h.Regexp != nil && rhs.Regexp != nil && h.String() == rhs.String()
}

func (h Host) String() string {
	if h.Regexp != nil {
		return h.Regexp.String()
	}
	return "<nil>"
}

type Configs []*Config

func (c Configs) Callback() ssh.ConfigCallback {
	var callbacks []ssh.ConfigCallback

	for _, c := range c {
		callbacks = append(callbacks, c.Callback())
	}

	return sshutil.Callback(callbacks...)
}

func (c Configs) LazyCallback() ssh.ConfigCallback {
	var fns []func() ssh.ConfigCallback

	for _, c := range c {
		fns = append(fns, c.Callback)
	}

	return sshutil.LazyCallback(fns...)
}

func debug(text string, v interface{}) {
	fmt.Printf("(DEBUG) %s = ", text)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	enc.Encode(v)
}

func (c Configs) Merge(in Configs) Configs {
	merged := make(Configs, len(c)+len(in))

	debug("c", c)
	debug("in", in)

	if len(c) > 0 {
		if c[len(c)-1].Host.Equal(globalHost) {
			merged[len(merged)-2] = c[len(c)-1]
		}

		copy(merged, c[:len(c)-1])
	}

	debug("merged", merged)

	if len(in) > 0 {
		if in[len(in)-1].Host.Equal(globalHost) {
			merged[len(merged)-1] = in[len(in)-1]
		}

		copy(merged[len(c):], in[:len(in)-1])
	}

	debug("merged", merged)

	return merged
}

func (c Configs) append(cfg *Config, hosts ...Host) Configs {
	for _, host := range hosts {
		cfg := cfg.clone()
		cfg.Host = host

		c = append(c, cfg)
	}

	return c
}

var globToRegexp = strings.NewReplacer(
	".", `\.`,
	"*", ".*",
	"?", ".",
)

func ParseConfigFile(path string) (Configs, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseConfig(f)
}

func ParseConfig(r io.Reader) (Configs, error) {
	const (
		stateGlobal = 1 << iota
		stateHost
	)

	var (
		scanner = bufio.NewScanner(r)
		global  = new(Config)
		local   *Config
		configs Configs
		hosts   []Host
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
				if err := merge(global, tmp); err != nil {
					return nil, fmt.Errorf("unexpected host configuration at line %d: %+v (%s)", lineno, tmp, err)
				}

				state = stateHost
			case stateHost:
				if err := merge(local, tmp); err != nil {
					return nil, fmt.Errorf("unexpected host configuration at line %d: %+v (%s)", lineno, tmp, err)
				}

				configs = configs.append(local, hosts...)
			}

			tmp, local, hosts = make(map[string]string), new(Config), hosts[:0]

			for _, host := range strings.Split(strings.TrimSpace(strings.TrimPrefix(ts, "Host")), " ") {
				r, err := regexp.Compile(globToRegexp.Replace(strings.TrimSpace(host)))
				if err != nil {
					return nil, fmt.Errorf("unexpected host at line %d: %q (%s)", lineno, host, err)
				}

				hosts = append(hosts, Host{r})
			}
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

	if len(tmp) != 0 && len(hosts) != 0 && state == stateHost {
		if err := merge(local, tmp); err != nil {
			return nil, fmt.Errorf("unexpected host configuration at line %d: %+v (%s)", lineno, tmp, err)
		}

		configs = configs.append(local, hosts...)

		tmp, local, hosts = nil, nil, hosts[:0]
	}

	configs = configs.append(global, globalHost)

	return configs, nil
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
