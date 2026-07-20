package core

import "fmt"

// ScopedMessage returns a message prefixed with the plugin name and component.
func ScopedMessage(component, msg string) string {
	return fmt.Sprintf("[%s] %s", component, msg)
}
