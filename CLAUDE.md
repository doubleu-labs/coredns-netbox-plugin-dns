# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`netboxdns` is a CoreDNS plugin that resolves DNS queries against zones/records managed by the [netbox-plugin-dns](https://github.com/sys4/netbox-plugin-dns) extension to NetBox. It is consumed by building a custom CoreDNS binary that imports this module via `plugin.cfg` (inserted after `cache`). Requires netbox-plugin-dns >= 1.5.4, NetBox >= 4.5.4.

## Common commands

Development uses `just` with a Podman-backed NetBox test instance:

- `just instance-start` — start NetBox + Postgres + Valkey via podman compose, wait for healthy, seed dataset (`.testing/init/init.go`).
- `just instance-stop` — tear down with volumes.
- `just test` — depends on `instance-start`; runs `go test` with coverage over the root package and `internal/netbox`.
- `just coverage` — opens HTML coverage report.
- Run a single test: `go test -run TestName .` (the NetBox instance must already be running, since tests hit the live API at `http://localhost:9999`).

NetBox dev UI: http://localhost:9999, `admin:admin`.

## Architecture

Plugin entry points follow the standard CoreDNS plugin shape:

- `setup.go` registers the plugin with `caddy`/CoreDNS via `plugin.Register`.
- `parse.go` parses the Corefile `netboxdns { ... }` block (`token`, `url`, `timeout`, `fallthrough`, `tls` with 0–3 args; see README for TLS modes).
- `netboxdns.go` defines the `NetboxDNS` handler. `ServeDNS` matches the qname against configured zones, calls `lookup`, and either writes an authoritative reply, returns NXDOMAIN, marks the response non-authoritative for delegations, or falls through to the next plugin.
- `lookup.go` implements query resolution against NetBox: it classifies results as `lookupSuccess`, `lookupNameError`, or `lookupDelegation`, and assembles `Answer`/`Ns`/`Extra` sections. `fixQType` collapses A/AAAA based on the request family.
- `record.go` converts NetBox record rows into `dns.RR` values.
- `internal/netbox/` is the NetBox API client (`api.go`, `zone.go`, `record.go`) used via `APIRequestClient` on the `NetboxDNS` struct. All HTTP/TLS configuration funnels through this client.

Test data and the NetBox instance live under `.testing/` (compose file + `init/init.go` seeder). Tests in `netboxdns_test.go` and `setup_test.go` exercise both Corefile parsing and end-to-end lookups against the seeded instance.
