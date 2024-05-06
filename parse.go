package netboxdns

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/tls"
)

type tokenFuncMap map[string]func(*caddy.Controller, *NetboxDNS) error

var tokenFuncs tokenFuncMap

func init() {
	tokenFuncs = tokenFuncMap{
		"fallthrough": parseFallthrough,
		"timeout":     parseTimeout,
		"tls":         parseTLS,
		"token":       parseToken,
		"url":         parseUrl,
	}
}

// Parse netboxdns configuration
func Parse(controller *caddy.Controller, netboxdns *NetboxDNS) error {
	instances := 0
	for controller.Next() {
		if instances > 0 {
			return plugin.ErrOnce
		}
		instances++
		parseZones(controller, netboxdns)
		if err := parseConfigTokens(controller, netboxdns); err != nil {
			return err
		}
	}

	if err := parseValidate(controller, netboxdns); err != nil {
		return err
	}

	fullPluginURL := netboxdns.requestClient.NetboxURL.JoinPath(
		"api",
		"plugins",
		"netbox-dns",
	)
	netboxdns.requestClient.NetboxURL = fullPluginURL

	netboxdns.requestClient.UserAgent = fmt.Sprintf(
		"coredns plugin %s",
		pluginName,
	)

	return nil
}

func parseZones(controller *caddy.Controller, netboxdns *NetboxDNS) {
	zones := plugin.OriginsFromArgsOrServerBlock(
		controller.RemainingArgs(),
		controller.ServerBlockKeys,
	)
	if len(zones) > 0 {
		netboxdns.zones = zones
	}
}

func parseConfigTokens(controller *caddy.Controller, netboxdns *NetboxDNS) error {
	for controller.NextBlock() {
		tokenName := controller.Val()
		tokenFunc, ok := tokenFuncs[tokenName]
		if !ok {
			return unknownToken(controller, tokenName)
		}
		if err := tokenFunc(controller, netboxdns); err != nil {
			return err
		}
	}
	return nil
}

func unknownToken(controller *caddy.Controller, unknownToken string) error {
	expectedTokenString := ""
	i := 0
	for tokenName := range tokenFuncs {
		expectedTokenString += fmt.Sprintf("%q", tokenName)
		if i+1 < len(tokenFuncs) {
			expectedTokenString += ", "
		}
		if i == len(tokenFuncs)-2 {
			expectedTokenString += "or "
		}
		i++
	}
	return controller.Errf(
		"unknown token %q; expected %s",
		unknownToken,
		expectedTokenString,
	)
}

func parseFallthrough(
	controller *caddy.Controller,
	netboxdns *NetboxDNS,
) error {
	netboxdns.fall.SetZonesFromArgs(controller.RemainingArgs())
	return nil
}

func parseTimeout(controller *caddy.Controller, netboxdns *NetboxDNS) error {
	if !controller.NextArg() {
		return controller.Err(`no value for "timeout" provided`)
	}
	duration, err := time.ParseDuration(controller.Val())
	if err != nil {
		return controller.Errf(
			`there was an error parsing "timeout": %q`,
			err.Error(),
		)
	}
	netboxdns.requestClient.Client.Timeout = duration
	return nil
}

func parseTLS(controller *caddy.Controller, netboxdns *NetboxDNS) error {
	args := controller.RemainingArgs()
	tlsConfig, err := tls.NewTLSConfigFromArgs(args...)
	if err != nil {
		return err
	}
	netboxdns.requestClient.Client.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	return nil
}

func parseToken(controller *caddy.Controller, netboxdns *NetboxDNS) error {
	if !controller.NextArg() {
		return controller.Err(`no value for "token" provided`)
	}
	netboxdns.requestClient.Token = controller.Val()
	return nil
}

func parseUrl(controller *caddy.Controller, netboxdns *NetboxDNS) error {
	if !controller.NextArg() {
		return controller.Err(`no value for "url" provided`)
	}
	netboxUrl, err := url.Parse(controller.Val())
	if err != nil {
		return controller.Errf(
			`there was an error parsing "url": %q`,
			err.Error(),
		)
	}
	netboxdns.requestClient.NetboxURL = netboxUrl
	return nil
}

func parseValidate(controller *caddy.Controller, netboxdns *NetboxDNS) error {
	tokenEmpty := netboxdns.requestClient.Token == ""
	urlEmpty := netboxdns.requestClient.NetboxURL == nil ||
		netboxdns.requestClient.NetboxURL.Host == ""
	if tokenEmpty && urlEmpty {
		return controller.Err(
			`values are required for "token" and "url"`,
		)
	}
	if tokenEmpty {
		return controller.Err(`value is required for "token"`)
	}
	if urlEmpty {
		return controller.Err(`value is required for "url"`)
	}
	return nil
}
