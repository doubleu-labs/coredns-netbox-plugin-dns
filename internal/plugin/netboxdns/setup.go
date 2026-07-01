package netboxdns

// noinspection LongLine
import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	iplugin "github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/config"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/poller"
)

var (
	isSetupTest    bool
	isTestRegister bool
)

func SetTestRegister() {
	isTestRegister = true
}

func Register() {
	if isTestRegister {
		core.RegisterPlugin(pluginName, setup).After("transfer")
		return
	}
	core.RegisterPlugin(pluginName, setup)
}

func setup(c *caddy.Controller) error {
	logger := log.NewWithPlugin(pluginName)

	cfg, err := config.Parse(c, logger)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	ctx := getServerContext(c, cfg)

	if !isSetupTest {
		if purgeErr := purgeInvalidViews(logger, ctx, cfg); purgeErr != nil {
			return plugin.Error(pluginName, purgeErr)
		}
	}

	views := &core.Views{
		Include: cfg.Views,
		Exclude: cfg.ViewsExclude,
	}

	netboxdns := &netboxDNS{
		client:      ctx.APIClient,
		logger:      logger,
		noop:        cfg.NoOp,
		serverState: ctx,
		views:       views,
		zones:       cfg.Zones,
	}

	if cfg.ViewPollerEnabled {
		p, pErr := poller.NewViewPoller(cfg.ViewPollerDuration)
		if pErr != nil {
			return plugin.Error(pluginName, pErr)
		}
		netboxdns.viewPoller = p
	}

	if cfg.Fall != nil {
		netboxdns.fall = *cfg.Fall
	}

	c.OnStartup(
		func() error {
			if cfg.ViewPollerEnabled {
				netboxdns.viewPoller.Start()
			}
			return nil
		},
	)

	c.OnShutdown(
		func() error {
			if cfg.ViewPollerEnabled {
				netboxdns.viewPoller.Stop()
			}
			return nil
		},
	)

	dnsserver.GetConfig(c).AddPlugin(
		func(next plugin.Handler) plugin.Handler {
			netboxdns.Next = next
			return netboxdns
		},
	)

	return nil
}

func getServerContext(
	c *caddy.Controller,
	cfg *config.Config,
) *iplugin.ServerContext {
	var ctx *iplugin.ServerContext
	if value, ok := c.Get(iplugin.ContextKey).(*iplugin.ServerContext); ok {
		ctx = value
	} else {
		ctx = &iplugin.ServerContext{
			APIClient: api.NewClient(cfg.Token, cfg.NetboxURL),
		}
		if cfg.Timeout != 0 {
			ctx.APIClient.SetTimeout(cfg.Timeout)
		}
		if cfg.TLS != nil {
			ctx.APIClient.SetTLSConfig(cfg.TLS)
		}
		ctx.APIClient.SetUserAgent(
			fmt.Sprintf("coredns.plugin.%s", pluginName),
		)
	}
	return ctx
}

func purgeInvalidViews(
	l log.P,
	sCtx *iplugin.ServerContext,
	cfg *config.Config,
) error {
	rq := &api.ViewsQuery{
		Brief: true,
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()
	views, err := rq.GetViews(ctx, sCtx.APIClient)
	if err != nil {
		return err
	}

	cfg.Views = slices.DeleteFunc(
		cfg.Views, func(view string) bool {
			exists := slices.ContainsFunc(
				views, func(v api.View) bool {
					return v.Name == view
				},
			)
			if !exists {
				l.Warning(
					core.ScopedMessage(
						"setup",
						fmt.Sprintf(
							"view %q specified by `views` does not "+
								"exist on the Netbox instance; removing",
							view,
						),
					),
				)
				return true
			}
			return false
		},
	)
	if len(cfg.Views) == 0 {
		cfg.Views = nil
	}

	cfg.ViewsExclude = slices.DeleteFunc(
		cfg.ViewsExclude, func(view string) bool {
			exists := slices.ContainsFunc(
				views, func(v api.View) bool {
					return v.Name == view
				},
			)
			if !exists {
				l.Warning(
					core.ScopedMessage(
						"setup",
						fmt.Sprintf(
							"view %q specified by `views_exclude` does "+
								"not exist on the Netbox instance; removing",
							view,
						),
					),
				)
				return true
			}
			return false
		},
	)
	if len(cfg.ViewsExclude) == 0 {
		cfg.ViewsExclude = nil
	}

	return nil
}
