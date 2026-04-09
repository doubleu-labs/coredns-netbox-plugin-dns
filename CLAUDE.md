# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`netboxdns` is a CoreDNS plugin that turns a NetBox instance running the
[netbox-plugin-dns](https://github.com/sys4/netbox-plugin-dns) extension
into a fully functional **hidden primary DNS server**: it answers
recursive lookups, serves outgoing AXFR/IXFR to secondaries, and
publishes RFC 9432 catalog zones so secondaries auto-discover what to
transfer. Requires netbox-plugin-dns ≥ 1.5.4 / NetBox ≥ 4.5.4.

The plugin is consumed by building a custom CoreDNS binary that imports
this module via `plugin.cfg`. **Critical**: the entry must be inserted
**after `transfer:transfer`**, otherwise AXFR queries are routed to
`ServeDNS` instead of `Transfer()` and forwarded to NetBox as
`?type=AXFR`, which returns HTTP 400.

```sh
sed -i '/^transfer:transfer/a netboxdns:github.com/doubleu-labs/coredns-netbox-plugin-dns' plugin.cfg
```

## Common commands

Development uses `just` with a Podman-backed NetBox test instance:

- `just instance-start` — start NetBox + Postgres + Valkey, wait healthy, seed dataset (`.testing/init/init.go`).
- `just instance-stop` — tear down with volumes.
- `just test` — depends on `instance-start`; runs `go test` with coverage over the root package and `internal/netbox`.
- `just coverage` — opens HTML coverage report.
- Run a single test: `go test -run TestName .` (NetBox instance must already be running, since some tests hit the live API at `http://localhost:9999`).
- Race-clean unit-only run (no live NetBox): `go test -short -race ./...`.

NetBox dev UI: http://localhost:9999, `admin:admin`.

## Architecture

### Plugin entry points (CoreDNS shape)

- `setup.go` — registers the plugin via `plugin.Register`. After parsing it
  (a) probes NetBox to fail fast on a misconfigured `view` (NetBox returns
  HTTP 400 on unknown views, which would otherwise silently SERVFAIL every
  query), (b) wraps the HTTP client `Transport` in `instrumentedTransport`
  so every NetBox round-trip is observed by Prometheus, and (c) hooks
  `OnStartup`/`OnShutdown` for the IXFR snapshot poller.
- `parse.go` — Corefile parser. Token-funcs: `token`, `url`, `timeout`,
  `fallthrough`, `tls` (0–3 args), `view`, `poll_interval`, `ixfr_history`.
- `netboxdns.go` — defines `NetboxDNS` (`plugin.Handler`). `ServeDNS`
  matches the qname against configured zones, calls `lookup`, writes an
  authoritative reply / NXDOMAIN / non-authoritative delegation, or falls
  through to the next plugin. Per-request latency and rcode are recorded
  via deferred timer + return-site counter bumps.
- `lookup.go` — query resolution against NetBox. Classifies results as
  `lookupSuccess`, `lookupNameError`, or `lookupDelegation`, assembles
  `Answer`/`Ns`/`Extra`. `fixQType` collapses A/AAAA based on the request
  family (so an A query over IPv6 transparently resolves AAAA).
- `record.go` — converts NetBox record rows into `dns.RR` values.

### Phase-by-phase code (this is the order it was built and the layering each phase added)

1. **Phase 1 — `view` filter** (`parse.go`, `setup.go`, `internal/netbox/zone.go`).
   The Corefile takes a single `view <name>`; `GetZones`/`GetCatalogZones`/
   `GetRecordsQuery` always pin `?view=<name>` so a hidden primary only
   ever sees the zones it is meant to serve. The setup-time probe is the
   guard against NetBox's HTTP 400 on unknown views.

2. **Phase 2 — outgoing AXFR** (`transfer.go`, `record.go`).
   `Transfer` implements `transfer.Transferer`. The contract from
   `coredns/plugin/transfer` is implemented verbatim:
   - `serial == 0`            → AXFR (full zone, SOA first **and** last per RFC 5936 §2.2).
   - `serial >= current`      → IXFR no-op (single SOA, channel closed).
   - `serial <  current`      → IXFR (Phase 3+ delta if cache can satisfy, else AXFR fallback).
   The opening/closing SOA is **always synthesised** from NetBox zone
   metadata via `buildSOA`/`buildSOAWithSerial`, never from the SOA
   `Record` returned by `/records/`. SOA records that come back from
   `/records/` are dropped before `recordsToRR` because storing them in
   the snapshot would only churn diffs.

3. **Phase 3 — IXFR snapshot poller** (`poller.go`, `internal/zonecache/`).
   - `internal/zonecache/zonecache.go` is a per-zone ring buffer
     (`New(maxSize)`, `Put`, `Latest`, `Diff`, `Len`). `Put` is
     idempotent on equal serial via in-place RR refresh. Zone names
     normalised lowercase no-dot. `cloneRRs` deep-copies via `dns.Copy`
     so callers cannot mutate cached state.
   - `internal/zonecache/diff.go` does a multiset diff keyed by
     `<lower owner>|<type>|<class>|<ttl>|<rdata>`. SOA records are filtered
     out of the diff (the Transfer path supplies its own old/new SOAs).
   - `poller.go` runs an immediate `pollOnce()` synchronously inside the
     poller goroutine *before* the first tick so the cache is warm
     enough to serve the first IXFR request without racing the ticker.
     Errors are logged + swallowed: the poller must never crash the
     plugin, so a single bad zone is skipped, not propagated.
   - `transfer.go` adds `streamIXFR` (RFC 1995 §4 wire layout: newSOA →
     oldSOA → removed → newSOA → added → newSOA), `sendBatched`, and the
     IXFR delta branch in `Transfer`.

4. **Phase 4 — RFC 9432 catalog zones** (`catalog.go`, `internal/netbox/zone.go`, `transfer.go`, `poller.go`).
   - **Detection is convention-driven, not Corefile-driven**: a zone is
     a catalog **iff** its NetBox `status` is `parked` *and* its name has
     prefix `cat.` (case-insensitive). To turn it on, create a zone with
     name `cat.example.com`, set status `parked`, fill SOA + nameservers,
     put it in the same view as its members. There is **no Corefile
     setting** for catalogs by design — the user explicitly chose this
     over `catalog { ... }` to avoid double-config.
   - Active vs catalog split is enforced by always querying NetBox with
     `?status=active` for member zones (`GetZones`) and `?status=parked`
     for catalogs (`GetCatalogZones`); the catalog list is then
     client-side filtered by the `cat.` prefix. This was committed in a
     **separate `fix:` commit before Phase 4** because filtering out
     parked/deprecated/reserved zones is independently correct semantics
     ("records-of-record, not zones to publish on the wire").
   - `buildCatalog(catalog, members)` returns: one `NS` per
     `catalog.NameServers`, the mandatory `version.<catalog>. TXT "2"`
     marker, and one PTR per member with owner
     `id-<member.ID>.zones.<catalog>.` and target `<member.Name>.`
     Self-references are excluded by ID equality, not name (defensive:
     even if the member list bizarrely contains the catalog itself, no
     self PTR is emitted).
   - **Catalog SOA serial — option C** (`catalogTracker` in `catalog.go`).
     The catalog's `soa_serial` field in NetBox is **deliberately
     ignored**. Per catalog name we keep an `uint32` counter that
     initialises to `time.Now().Unix()` on first use and bumps by **+1**
     only when the membership fingerprint (sorted comma-joined zone IDs)
     changes. Per-record edits inside member zones therefore do *not*
     churn the catalog serial. This was the user's explicit choice over
     options A/B because it gives stable, monotonic-on-real-change
     serials without round-tripping through NetBox state.
   - `transferCatalog` reuses `streamIXFR` and the same cache snapshot
     plumbing as member zones; the diff/cache layer does not need to
     know catalogs exist. On a cold start (cache disabled or never
     populated for this catalog) it builds the body inline by calling
     `GetZones` once, then writes a snapshot.

5. **Phase 5 — Prometheus metrics** (`metrics.go`, `metrics_test.go`, instrumentation in
   `setup.go`, `netboxdns.go`, `transfer.go`, `poller.go`).
   See **Metrics** section below. Important detail: `internal/netbox`
   stays free of any prometheus import. All NetBox API observability
   comes from `instrumentedTransport`, an `http.RoundTripper` wrapper
   installed on the plugin's `*http.Client` in `setup`. The endpoint
   label is reduced to a low-cardinality bucket (`zones`/`records`/
   `nameservers`/`views`/…/`other`) by `netboxEndpointLabel`, which
   strips the `/api/plugins/netbox-dns/` prefix and keeps only the first
   path segment.

### NetBox API client (`internal/netbox/`)

- `api.go` — `APIRequestClient` (Client + URL + Token + UserAgent),
  `doGet`, generic `get[T]` and `getMany[T]` (auto-paginates on `next`).
- Authorization header is **`Bearer <token>`**, not the legacy
  `Token <token>`. NetBox 4.x accepts both, but Bearer is the RFC 6750
  form newer NetBox-side tooling expects, so it's preferred going
  forward. README documents this so token-audit logs make sense.
- `zone.go` — `Zone` struct (with `Status` field), `GetZones(client, view)`
  pinning `?status=active`, `GetCatalogZones(client, view)` pinning
  `?status=parked` and client-side filtering on `cat.` prefix.
- `record.go` — `Record` + `RecordQuery` and `GetRecordsQuery`. Records
  carry an `AbsoluteValue` field that `record.go`/`recordsToRR` parses
  into typed `dns.RR` values.

### Metrics (`metrics.go`)

All collectors live under `coredns_netboxdns_` and use `promauto` against
the default Prometheus registerer (CoreDNS's `prometheus` plugin scrapes
that registerer, so no `metrics.MustRegister` plumbing is needed). The
full set:

| Metric | Type | Labels | Where it's bumped |
| --- | --- | --- | --- |
| `requests_total` | counter | `zone`, `rcode` | `ServeDNS` per return site (lookup error → SERVFAIL is counted, not lost) |
| `request_duration_seconds` | histogram | `zone` | `ServeDNS` deferred timer |
| `netbox_requests_total` | counter | `endpoint`, `code` | `instrumentedTransport.RoundTrip` (error → label `error`) |
| `netbox_request_duration_seconds` | histogram | `endpoint` | same |
| `transfers_total` | counter | `zone`, `kind` | `Transfer` and `transferCatalog`, one bump per branch (`axfr`, `ixfr_delta`, `ixfr_noop`, `ixfr_fallback`, `catalog_axfr`, `catalog_ixfr_delta`, `catalog_ixfr_noop`) |
| `poll_cycles_total` | counter | `result` | `pollOnce` (success/error split on the initial `GetZones` outcome) |
| `poll_duration_seconds` | histogram | — | `pollOnce` deferred timer |
| `zone_serial` | gauge | `zone` | `pollOnce` after each member `Put`, and `pollCatalogs` after each catalog `Put` |
| `cache_snapshots` | gauge | `zone` | as above, set to `cache.Len(zone)` |
| `catalog_members` | gauge | `catalog` | `pollCatalogs` via `catalogMemberCount` (excludes the catalog itself) |

Histograms use `ExponentialBuckets` aligned with each path's expected
latency: 0.5 ms base for DNS request handling, 1 ms base for NetBox
round-trips, 10 ms base for poll cycles (which fan out across all zones).

## Important conventions and gotchas

- **`plugin.cfg` ordering**: `netboxdns` MUST be after `transfer:transfer`.
  This was the root cause of the "AXFR returns 400 from NetBox" bug
  during the Phase 2 smoke test.
- **SOA is always synthesised**, never round-tripped from NetBox's SOA
  record string. Keeps the wire form deterministic and avoids depending
  on `miekg/dns` parsing the netbox-formatted SOA value.
- **Zone status filter is intentional**: only `status=active` zones are
  served on every path (lookup, AXFR, IXFR poller). Switching a zone
  from `active` to `parked` in NetBox takes it out of service on the
  next poll cycle without deleting it.
- **Catalog detection is NetBox-side, not Corefile-side**, by user
  decision. Don't add a `catalog { ... }` Corefile block.
- **Catalog SOA serial is owned by the plugin**, not by NetBox. Only a
  membership change bumps it; per-record edits inside members don't.
- **Both member zones and the catalog must be listed in the Corefile
  server block AND in the `transfer` directive**, e.g.
  `firma.sk cat.firma.sk:53 { netboxdns firma.sk cat.firma.sk { … } transfer firma.sk cat.firma.sk { to * } }`.
- **Poller errors are swallowed**, never propagated. A bad single zone
  is skipped; only a top-level `GetZones` failure marks the cycle
  `result="error"`.
- **`internal/netbox` has no prometheus import**. If you need to observe
  a new HTTP code path, do it through `instrumentedTransport`, not by
  adding metrics into the client package.
- **Zone-name normalisation**: the cache stores names lowercase + no
  trailing dot. Anything that crosses the cache boundary should go
  through the same normalisation; mismatches will look like silent
  cache misses.

## Test layout

- `unit_test.go` — `newMockPlugin` builds a `*NetboxDNS` wired against an
  `httptest.Server`. **It bypasses `NewNetboxDNS`**, so any new field
  with a non-nil zero invariant (like `catalogTracker`) must be added to
  the literal here too — there is a real bug history of nil-deref panics
  from this.
- `catalog_test.go` — covers `buildCatalog`, `catalogTracker` semantics
  (stable on unchanged membership, +1 on change, per-catalog
  independent), cold-start AXFR, IXFR no-op, and `pollOnce` populating
  the catalog cache. Fixtures use deterministic IDs (3, 7, 99) so PTR
  owner names are stable across runs.
- `transfer_ixfr_test.go`, `setup_view_test.go`, `poller_test.go`,
  `lookup_unit_test.go`, `servedns_unit_test.go` cover their respective
  phases.
- `metrics_test.go` — exercises `netboxEndpointLabel`. The metric
  collectors themselves are validated end-to-end via the staging
  smoke runs, not via mocked Prometheus assertions.
- `internal/zonecache/diff_test.go`, `zonecache_test.go`,
  `internal/netbox/{api,record,zone}_test.go` round out the layered
  tests.

## Live staging environment (lastmile.sk)

Used for end-to-end smoke between phases (NOT for CI). Notes are kept
here so future sessions don't have to rediscover them:

- NetBox: http://netbox.if.lastmile.sk (also https://; HTTP works fine
  for token-auth tests).
- Vault: `https://vault.int.lastmile.sk`. **Always set `VAULT_ADDR`**
  before invoking `vault` — the local default points at `127.0.0.1:8200`
  and will fail with "connection refused".
- Token used by the plugin smoke runs lives in
  `server-secrets/k8s/prod/netbox`:
  - `coredns_api_token` — v2 (`nbt_…`), read-only, used for production-
    style smoke. Returns 403 on POST.
  - `netbox_v1_superadmin` — v1 (40-char hex), full access, used when
    the smoke needs to create/delete zones (catalog membership churn).
    Sent as `Authorization: Token <token>`. The older
    `superuser_api_token` is rejected by NetBox 4.5 ("Invalid v1 token").
- Nameserver IDs in the staging instance: `ns1.firma.sk` = 3,
  `ns2.firma.sk` = 4. Used as SOA MNAME / nameserver references when
  creating test zones via the API.
- The plugin is built into a custom CoreDNS binary at `/tmp/coredns/`
  using `go mod edit -replace` against the local checkout, then started
  with `/tmp/Corefile` listening on port 1053 plus the `prometheus`
  plugin on `127.0.0.1:9153`.

## Implementation plan

`implementation-plan-en.md` is the source of truth for phase scope and
ordering. The five phases (0 tests, 1 view, 2 AXFR, 3 IXFR, 4 catalog,
5 metrics) are all complete in master. Use it as the entry point if you
need to add a Phase 6.
