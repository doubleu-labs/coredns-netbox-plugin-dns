# Changelog

All notable changes to this project are documented in this file.

## Unreleased

### Added

- **View filtering** (`view` directive): restrict served zones to a single
  NetBox DNS view. Setup-time validation rejects unknown view names.
- **Multi-view support** (`view` with multiple values, `view_exclude`):
  whitelist or blacklist multiple views. Client-side filtering when more
  than one view is specified; ambiguity warning when the same zone exists
  in multiple views without a view filter.
- **Outgoing AXFR** via the CoreDNS `transfer` plugin (`Transfer()`
  implements `transfer.Transferer`). SOA is always synthesised from NetBox
  zone metadata.
- **IXFR delta transfers** with a background snapshot poller
  (`poll_interval`, `ixfr_history`). Configurable ring-buffer depth per
  zone; RFC 1995 wire format with diff computed from snapshot multiset.
- **RFC 9432 catalog zones**: detected automatically via NetBox-side
  convention (`status=parked` + `cat.` name prefix). Catalog content is
  synthesised from active member zones in the same view. SOA serial managed
  by an internal counter that bumps only on membership change.
- **Catalog SOA/NS query support**: secondaries can poll `cat.<zone> SOA`
  before initiating AXFR (previously returned NXDOMAIN because catalog
  zones are `parked`).
- **Prometheus metrics** (`coredns_netboxdns_*`): request count/latency,
  NetBox API round-trip observability (via HTTP RoundTripper wrapper),
  transfer counters by kind, poller cycle count/duration, zone serial
  gauge, cache depth gauge, catalog member count gauge.
- **Defensive AXFR/IXFR guard** in `ServeDNS`: zone transfer queries are
  forwarded to the next plugin regardless of `plugin.cfg` ordering, so a
  mis-ordered config no longer produces HTTP 400 from NetBox.

### Changed

- API authentication header switched from `Token` to `Bearer` (RFC 6750).
  NetBox 4.x accepts both; no user action required.
- Zone status filter: only `status=active` zones are served on every path
  (lookup, AXFR, IXFR poller). Parked/deprecated/reserved zones are
  excluded by design.

### Fixed

- `plugin.cfg` ordering: documentation now instructs inserting after
  `transfer:transfer` instead of `cache:cache`, preventing AXFR routing
  through the wrong code path.
- `catalog_members` gauge now uses the same name-based exclusion as
  `buildCatalog`, ensuring the count exactly matches the synthesised PTR
  count.
