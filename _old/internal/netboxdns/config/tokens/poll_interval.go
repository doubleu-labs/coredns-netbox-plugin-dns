package tokens

// noinspection LongLine
import (
	"time"

	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var (
	tPollIntervalName   = "poll_interval"
	defaultPollInterval = 300 * time.Second
)

func init() {
	config.RegisterToken(tPollIntervalName, tPollInterval{})
}

type tPollInterval struct {
	set bool
}

func (t tPollInterval) Parse(c *caddy.Controller, cfg *config.Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tPollIntervalName)
	}
	d, err := time.ParseDuration(c.Val())
	if err != nil {
		return config.ErrTokenParse(c, tPollIntervalName, err)
	}
	cfg.PollInterval = d
	t.set = true
	return nil
}

func (t tPollInterval) Validate(c *caddy.Controller, cfg *config.Config) error {
	if !t.set {
		cfg.PollInterval = defaultPollInterval
		return nil
	}
	if cfg.PollInterval <= 0 {
		return config.ErrTokenValidate(
			c,
			tPollIntervalName,
			"must be greater than zero",
		)
	}
	return nil
}
