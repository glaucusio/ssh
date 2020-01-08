package sshfile

type Hosts []struct {
	Host   string
	Config *HostConfig
}

func (c *Config) Hosts() Hosts {
	var hosts Hosts

	for _, h := range c.hosts {
		host := h.r.String()
		cfg := c.global.clone()

		if err := merge(cfg, c.local[h.ref]); err != nil {
			panic("unexpected error: " + err.Error())
		}

		hosts = append(hosts, struct {
			Host   string
			Config *HostConfig
		}{
			Host:   host,
			Config: cfg,
		})
	}

	return hosts
}
