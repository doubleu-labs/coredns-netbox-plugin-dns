package config

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

// Config is the shape of a plugin configuration.
type Config interface {
	SetZones([]string)
}

func validate[T Config](c *caddy.Controller, cfg *T) error {
	cfgType := reflect.TypeOf(*cfg).Elem()
	cfgValue := reflect.ValueOf(*cfg).Elem()

	numField := cfgType.NumField()
	for i := 0; i < numField; i++ {
		field := cfgType.Field(i)
		tagRequired := field.Tag.Get("required")
		if tagRequired == "" {
			continue
		}
		required, err := strconv.ParseBool(tagRequired)
		if err != nil {
			// this error should only be encountered during development.
			// a properly constructed plugin configuration struct would not have
			// this error.
			return c.Err(
				core.ScopedMessage(
					"config",
					fmt.Sprintf(
						"`required` tag on field %q is not a boolean",
						field.Name,
					),
				),
			)
		}
		if required && cfgValue.Field(i).IsZero() {
			tagName := field.Tag.Get("name")
			return c.Err(
				core.ScopedMessage(
					"config",
					fmt.Sprintf(
						"error validating token %q; token is required", tagName,
					),
				),
			)
		}
	}
	return nil
}

func parseZones(c *caddy.Controller) []string {
	return plugin.OriginsFromArgsOrServerBlock(
		c.RemainingArgs(),
		c.ServerBlockKeys,
	)
}
