package ssh

type Client struct {
}

func NewClient(cfg *Config) (*Client, error) {
	c := &Client{}

	return c, nil
}

func (c *Client) Connect() (*Session, error) {
	return nil, nil
}
