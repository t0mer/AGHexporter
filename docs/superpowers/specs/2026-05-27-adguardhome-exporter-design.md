# AdGuard Home Prometheus Exporter — Design Spec

**Date:** 2026-05-27
**Module:** `github.com/t0mer/AGHexporter`
**Status:** Approved

---

## 1. Goal

A single-binary Prometheus exporter for one or more AdGuard Home (ADH) instances. Exposes `/metrics` with all scraped data labelled by `instance` name. Runs as a foreground process or as an OS service via `--service`.

---

## 2. Architecture

Single registered `prometheus.Collector` (Option A). On each `/metrics` pull, `Collect()` fans out one goroutine per instance, each with a 10 s timeout. Results are sent over a channel; `Collect()` returns when all goroutines complete. No background poller, no caching — scrape is always fresh and on-demand.

### Package layout

```
cmd/adguardhome-exporter/main.go
    — flag parsing (pflag), port resolution, instance discovery,
      service dispatch or server start

internal/instances/
    instance.go      — Instance struct + per-field validation
    discover.go      — Format A (indexed env) + B (CSV env) + C (--instance flag);
                       combines in A→B→C order, deduplicates names
    secrets.go       — *_FILE reader: read once at startup, TrimSpace, never log contents
    discover_test.go — full test coverage (see §6)

internal/adguard/
    client.go        — http.Client with Basic Auth, per-instance TLS config
    types.go         — Go structs matching swagger ServerStatus + Stats + TopArrayEntry
    status.go        — GET /control/status
    stats.go         — GET /control/stats
    client_test.go   — httptest-based stubs

internal/collector/
    descriptors.go   — all prometheus.Desc declarations (single source of truth)
    collector.go     — prometheus.Collector implementation; GaugeVec.Reset() before
                       repopulating top-N tables each scrape
    collector_test.go

internal/server/
    server.go        — /metrics handler, http.Server, graceful shutdown on SIGINT/SIGTERM

internal/svc/
    service.go       — kardianos/service wrapper; captures ADGUARD_* env vars +
                       --instance flags into service definition at install time
```

**Dependency direction (no cycles):**
`cmd → svc → server → collector → adguard → instances`

`instances` is a pure leaf with no sibling imports.

---

## 3. Configuration

### Port resolution (env wins over flag)

```
ADGUARD_EXPORTER_PORT set? → use it
else --port set?           → use flag value
else                       → default 9100
```

This precedence is a tested invariant (unit test in `discover_test.go`).

### Instance discovery — three additive formats

Results from all three formats are concatenated in order: **A then B then C**.

**Format A — indexed env vars (preferred; supports secret files)**

Scan environment for `ADGUARD_URL_<N>` keys; collect all indices. Per index N:

| Variable | Required |
|---|---|
| `ADGUARD_URL_<N>` | yes |
| `ADGUARD_NAME_<N>` | no — defaults to `host[:port]` |
| `ADGUARD_USERNAME_<N>` | one-of* |
| `ADGUARD_USERNAME_FILE_<N>` | one-of* |
| `ADGUARD_PASSWORD_<N>` | one-of† |
| `ADGUARD_PASSWORD_FILE_<N>` | one-of† |
| `ADGUARD_SKIP_TLS_<N>` | no — default `true` |

\* Both set on the same index → **fatal error** (no silent pick).
† Same rule.

**Format B — CSV env vars**

`ADGUARD_URLS`, `ADGUARD_USERNAMES`, `ADGUARD_PASSWORDS` (required, must have equal length).
Optional: `ADGUARD_NAMES`, `ADGUARD_SKIP_TLS` (each must match `ADGUARD_URLS` length).
Length mismatch → **fatal error**.

**Format C — `--instance` CLI flag (repeatable)**

Value is `key=value,...`. Keys: `url` (required), `username`, `password`, `username_file`, `password_file`, `name`, `skip_tls`. Same mutual-exclusion rules as Format A.

### Post-collection validation

1. Resolve name: explicit value if given, else `host[:port]` from URL.
2. Duplicate resolved names → fatal error.
3. Zero instances → fatal error with message explaining all three formats; HTTP server does **not** start.
4. URL must parse with scheme `http` or `https`.
5. Both `username` + `username_file` (or password equivalents) set → fatal error.

### Secret file handling

Read once at startup. `strings.TrimSpace()` the contents. Never log contents or the resolved path at INFO level (DEBUG only on read error).

---

## 4. Service Mode (`--service`)

Uses `github.com/kardianos/service`. Accepted values: `install`, `uninstall`, `start`, `stop`, `restart`.

At `install` time:
- Filter `os.Environ()` for keys matching `ADGUARD_*` and `ADGUARD_EXPORTER_PORT`.
- Serialize any `--instance` flag values back to `key=value,...` form.
- Pass both as `EnvVars` in the `service.Config` → systemd gets `Environment=` lines.

Service metadata:
- Name: `adguardhome-exporter`
- Display name: `AdGuard Home Prometheus Exporter`
- Description: `Scrapes one or more AdGuard Home instances and exposes Prometheus metrics.`

---

## 5. Metrics

All metrics carry `instance="<resolved name>"`.

### Scalar metrics

| Metric | Type | Source |
|---|---|---|
| `adguard_up` | Gauge | 1 = scrape OK, 0 = failed |
| `adguard_protection_enabled` | Gauge | `protection_enabled` (`/control/status`) |
| `adguard_dns_queries_total` | Gauge | `num_dns_queries` |
| `adguard_blocked_filtering_total` | Gauge | `num_blocked_filtering` |
| `adguard_blocked_safebrowsing_total` | Gauge | `num_replaced_safebrowsing` |
| `adguard_blocked_parental_total` | Gauge | `num_replaced_parental` |
| `adguard_enforced_safesearch_total` | Gauge | `num_replaced_safesearch` |
| `adguard_avg_processing_time_seconds` | Gauge | `avg_processing_time` (already seconds per swagger) |
| `adguard_scrape_duration_seconds` | Gauge | measured per instance per scrape |
| `adguard_scrape_errors_total` | Counter | incremented on each failed instance scrape |

> All ADH "totals" are `Gauge` — ADH returns rolling-window values that can decrease, so `Counter` would break `rate()`. `adguard_scrape_errors_total` is the only true `Counter` (exporter-generated, monotonic).

### Top-N GaugeVecs

| Metric | Extra label | Source field |
|---|---|---|
| `adguard_top_clients` | `client` | `top_clients` |
| `adguard_top_queried_domains` | `domain` | `top_queried_domains` |
| `adguard_top_blocked_domains` | `domain` | `top_blocked_domains` |
| `adguard_top_upstreams` | `upstream` | `top_upstreams_responses` |
| `adguard_top_upstreams_avg_time_seconds` | `upstream` | `top_upstreams_avg_time` (already seconds) |

GaugeVecs are `Reset()` at the start of each per-instance scrape goroutine, then repopulated — stale top-N entries do not linger.

---

## 6. Scrape Lifecycle

1. Prometheus calls `Collect(ch chan<- prometheus.Metric)`.
2. Collector spawns one goroutine per instance (`sync.WaitGroup`), each with a `context` timeout of `scrapeTimeout = 10 * time.Second`.
3. Each goroutine:
   - Records start time.
   - `GetStatus()` → failure: `adguard_up=0`, increment `adguard_scrape_errors_total`, log WARN, return.
   - `GetStats()` → same failure path.
   - Emits scalar metrics, resets + repopulates top-N GaugeVecs.
   - Emits `adguard_scrape_duration_seconds`.
4. `Collect()` waits for all goroutines via `WaitGroup`, then returns.
5. A failing instance cannot affect other instances or cause `Collect()` to error.

---

## 7. Testing

**`internal/instances/discover_test.go`**
- Format A inline credentials (single and multi-index)
- Format A secret files via `os.CreateTemp`
- Format A: inline + `_FILE` both set → fatal error
- Format B: happy path + length-mismatch rejection
- Format C: repeated `--instance` flag parsing
- All three formats combined in one run (A→B→C order)
- Duplicate resolved names → fatal error
- Zero instances → fatal error
- Port precedence: `ADGUARD_EXPORTER_PORT` beats `--port`

**`internal/adguard/client_test.go`**
- `httptest.NewServer` stub for `/control/status` and `/control/stats`
- TLS path via `httptest.NewTLSServer` with skip-verify
- Basic Auth header present on every request

**`internal/collector/collector_test.go`**
- Uses `stubADH(t, status, stats)` helper returning `*httptest.Server`
- Registers collector against `prometheus.NewRegistry()`
- `Gather()` asserts all expected metric families + correct `instance` labels
- One instance down: `adguard_up=0` for that instance, other instance unaffected

---

## 8. Build & Dependencies

- **Go:** 1.22+, no CGO, single static binary
- **`github.com/prometheus/client_golang`** — metrics + HTTP handler
- **`github.com/kardianos/service`** — OS service management
- **`github.com/spf13/pflag`** — repeatable `--instance` flag
- **stdlib only** for HTTP client (no ADH SDK)

Makefile targets: `build`, `test`, `lint`, `run`, `release` (cross-compile linux/amd64, linux/arm64, darwin/arm64, windows/amd64).

---

## 9. Key Invariants (must not be broken)

- `ADGUARD_EXPORTER_PORT` always wins over `--port` — unit tested.
- Inline credential + `_FILE` variant both set on same entry → always fatal.
- Duplicate resolved instance names → always fatal.
- Zero configured instances → server never starts.
- ADH "total" fields are always `Gauge`, never `Counter`.
- `avg_processing_time` and `top_upstreams_avg_time` exposed as-is (already seconds per swagger.yml).
