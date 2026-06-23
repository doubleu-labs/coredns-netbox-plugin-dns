package netbox

import (
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	*http.Client
	NetboxURL *url.URL
	token     string
	UserAgent string
}

func (c *Client) String() string {
	var tokenValue string
	if c.HasToken() {
		tokenValue = "[REDACTED]"
	}
	return fmt.Sprintf(
		"Client{NetboxURL: %s, token: %s, UserAgent: %s, Client: %#v",
		c.NetboxURL,
		tokenValue,
		c.UserAgent,
		c.Client,
	)
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) HasToken() bool {
	return c.token != ""
}
