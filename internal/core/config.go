package core

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
)

type Config interface {
	PluginName() string
}

// ValidateConfig validates plugin configuration.
// Reflection is used to determine if a field is required and returns an error
// if the field is missing, empty, or zero.
func ValidateConfig[T Config](c *caddy.Controller, cfg *T) error {
	cfgType := reflect.TypeOf(cfg).Elem()
	cfgValue := reflect.ValueOf(cfg).Elem()

	if cfgType.Kind() != reflect.Struct {
		return c.Err(
			ScopedMessage(
				(*cfg).PluginName(),
				"config",
				"config supplied is not a struct",
			),
		)
	}

	for i := 0; i < cfgType.NumField(); i++ {
		field := cfgType.Field(i)
		tagRequired := field.Tag.Get("required")
		if tagRequired == "" {
			continue
		}
		required, err := strconv.ParseBool(tagRequired)
		if err != nil {
			return c.Err(
				ScopedMessage(
					(*cfg).PluginName(),
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
				ScopedMessage(
					(*cfg).PluginName(),
					"config",
					fmt.Sprintf(
						"error validating token %q; token is required",
						tagName,
					),
				),
			)
		}
	}

	return nil
}

// ParseZones gets the configured origins for the plugin.
// Will return the plugin-specific origins or the server block origins if
// no plugin-specific origins are configured.
func ParseZones(c *caddy.Controller) []string {
	return plugin.OriginsFromArgsOrServerBlock(
		c.RemainingArgs(),
		c.ServerBlockKeys,
	)
}
