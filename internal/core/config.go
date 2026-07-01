package core

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
)

// ValidateConfig validates plugin configuration.
// Reflection is used to determine if a field is required and returns an error
// if the field is missing, empty, or zero.
func ValidateConfig[T any](c *caddy.Controller, cfg *T) error {
	cfgType := reflect.TypeOf(cfg).Elem()
	cfgValue := reflect.ValueOf(cfg).Elem()

	if cfgType.Kind() != reflect.Struct {
		// this error should only be encountered during development.
		// a properly constructed plugin configuration would use a struct for
		// configuration.
		return c.Err(
			ScopedMessage(
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
			// this error should only be encountered during development.
			// a properly constructed plugin configuration struct would not have
			// this error.
			return c.Err(
				ScopedMessage(
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
