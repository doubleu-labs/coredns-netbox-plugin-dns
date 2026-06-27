package core

// Views represent a set of views that can be included or excluded when
// querying the Netbox API.
type Views struct {
	Include []string
	Exclude []string
}
