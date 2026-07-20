package cache

import "errors"

var (
	ErrZoneNotFound = errors.New("zone not found")
	ErrZoneHasNoSOA = errors.New("zone has no SOA")
)
