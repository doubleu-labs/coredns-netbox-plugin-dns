package tokens

// noinspection LongLine
import (
	"net/url"

	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var tURLName = "url"

func init() {
	config.RegisterToken(tURLName, tURL{})
}

type tURL struct{}

func (tURL) Parse(c *caddy.Controller, cfg *config.Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tURLName)
	}
	u, err := url.Parse(c.Val())
	if err != nil {
		return config.ErrTokenParse(c, tURLName, err)
	}
	cfg.URL = u
	return nil
}

func (tURL) Validate(_ *caddy.Controller, _ *config.Config) error {
	return nil
}
