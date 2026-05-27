# AdGuard Home Prometheus Exporter

A single-binary Prometheus exporter for one or more [AdGuard Home](https://adguard.com/en/adguard-home/overview.html) instances. Exposes `/metrics` with all scraped data labelled by instance name so a Grafana dashboard can filter by instance.

## Features

- **Multi-instance** — scrape any number of AdGuard Home instances in a single process.
- **Parallel scraping** — each `/metrics` pull fans out one goroutine per instance with a 10 s timeout; a failing instance never blocks or pollutes others.
- **Three configuration formats** — indexed env vars (with secret-file support), CSV env vars, or repeatable CLI flags. All three can be mixed freely in one run.
- **OS service integration** — install, start, stop, restart, and uninstall as a system service (systemd / launchd / Windows SCM) via `--service`.
- **Single static binary** — no CGO, no runtime dependencies.
- **Graceful shutdown** — drains in-flight scrapes on `SIGINT`/`SIGTERM`.

---

## Requirements

- Go 1.23+
- AdGuard Home reachable over HTTP or HTTPS

---

## Build

```bash
# Development build
make build          # → bin/adguardhome-exporter

# Cross-compiled release binaries
make release        # → dist/adguardhome-exporter-{linux-amd64,linux-arm64,darwin-arm64,windows-amd64.exe}

# Run tests
make test

# Lint
make lint
```

---

## Quick Start

```bash
# Single instance via env vars
export ADGUARD_URL_1=http://192.168.1.1
export ADGUARD_USERNAME_1=admin
export ADGUARD_PASSWORD_1=secret
./bin/adguardhome-exporter

# Single instance via CLI flag
./bin/adguardhome-exporter \
  --instance "url=http://192.168.1.1,username=admin,password=secret,name=home"

# Two instances
./bin/adguardhome-exporter \
  --instance "url=http://192.168.1.1,username=admin,password=pass1,name=primary" \
  --instance "url=http://192.168.1.2,username=admin,password=pass2,name=secondary"
```

Metrics are available at `http://localhost:9100/metrics`. Visiting `/` redirects there automatically.

---

## Configuration

### CLI Flags

| Flag         | Default | Env override              | Description                                               |
|--------------|---------|---------------------------|-----------------------------------------------------------|
| `--port`     | `9100`  | `ADGUARD_EXPORTER_PORT`   | Port to expose `/metrics` on.                             |
| `--instance` | —       | —                         | Inline instance spec (repeatable). See Format C below.    |
| `--service`  | —       | —                         | Service action: `install`, `uninstall`, `start`, `stop`, `restart`. |

**Port precedence:** `ADGUARD_EXPORTER_PORT` env var always wins over `--port`.

---

### Instance Configuration

Instances are declared via environment variables or CLI flags — there is no config file. All three formats can be mixed; instances are collected in order **Format A → Format B → Format C**.

After collecting all instances, duplicate resolved names are a fatal error. Zero instances configured is also a fatal error (the HTTP server will not start).

---

#### Format A — Indexed env vars (recommended; supports secret files)

Per-instance variables suffixed with `_N` (1-based integer). Indices do not need to be contiguous.

| Variable                    | Required | Description                                                      |
|-----------------------------|----------|------------------------------------------------------------------|
| `ADGUARD_URL_<N>`           | yes      | Instance URL (`http://` or `https://`).                          |
| `ADGUARD_NAME_<N>`          | no       | Display name / Prometheus label. Default: `host[:port]` from URL.|
| `ADGUARD_USERNAME_<N>`      | one of * | Inline username.                                                 |
| `ADGUARD_USERNAME_FILE_<N>` | one of * | Path to a file containing the username (trimmed).                |
| `ADGUARD_PASSWORD_<N>`      | one of † | Inline password.                                                 |
| `ADGUARD_PASSWORD_FILE_<N>` | one of † | Path to a file containing the password (trimmed).                |
| `ADGUARD_SKIP_TLS_<N>`      | no       | Skip TLS verification (`true`/`false`). Default: `true`.         |

\* Exactly one of `USERNAME` / `USERNAME_FILE` must be set.
† Exactly one of `PASSWORD` / `PASSWORD_FILE` must be set. Setting both is a fatal error.

**Inline credentials:**

```bash
ADGUARD_URL_1=http://192.168.1.1
ADGUARD_USERNAME_1=admin
ADGUARD_PASSWORD_1=secret
ADGUARD_NAME_1=home-primary

ADGUARD_URL_2=https://192.168.1.2
ADGUARD_USERNAME_2=admin
ADGUARD_PASSWORD_2=secret2
ADGUARD_SKIP_TLS_2=false
```

**Secret files (Docker secrets / Kubernetes secrets):**

```bash
ADGUARD_URL_1=http://192.168.1.1
ADGUARD_USERNAME_FILE_1=/run/secrets/adguard_username
ADGUARD_PASSWORD_FILE_1=/run/secrets/adguard_password
```

Secret file contents are read once at startup, whitespace-trimmed, and never logged.

---

#### Format B — CSV env vars

Parallel comma-separated arrays. All arrays must have the same number of entries.

| Variable             | Required | Description                                                     |
|----------------------|----------|-----------------------------------------------------------------|
| `ADGUARD_URLS`       | yes      | Comma-separated list of instance URLs.                          |
| `ADGUARD_USERNAMES`  | yes      | Comma-separated usernames (same count as `ADGUARD_URLS`).       |
| `ADGUARD_PASSWORDS`  | yes      | Comma-separated passwords (same count as `ADGUARD_URLS`).       |
| `ADGUARD_NAMES`      | no       | Comma-separated display names. Default: `host[:port]` per URL.  |
| `ADGUARD_SKIP_TLS`   | no       | Comma-separated `true`/`false`. Default: `true` per instance.   |

```bash
ADGUARD_URLS=http://192.168.1.1,http://192.168.1.2
ADGUARD_USERNAMES=admin,admin
ADGUARD_PASSWORDS=pass1,pass2
ADGUARD_NAMES=primary,secondary
```

Length mismatches between arrays are a fatal error. Secret files are not supported in Format B.

---

#### Format C — `--instance` CLI flag (repeatable)

Comma-separated `key=value` pairs. Repeat the flag for multiple instances.

```bash
./bin/adguardhome-exporter \
  --instance "url=http://192.168.1.1,username=admin,password=pass1,name=primary" \
  --instance "url=http://192.168.1.2,username=admin,password=pass2,name=secondary,skip_tls=false"
```

Recognized keys: `url` (required), `username`, `password`, `username_file`, `password_file`, `name`, `skip_tls`.

Same mutual-exclusion rules apply: setting both `username` and `username_file` (or both password variants) on the same flag is a fatal error.

---

## Metrics Reference

All metrics carry the label `instance="<name>"`.

### Scalar metrics

| Metric                               | Type    | Description                                                    |
|--------------------------------------|---------|----------------------------------------------------------------|
| `adguard_up`                         | Gauge   | `1` if the instance is reachable, `0` if the scrape failed.   |
| `adguard_protection_enabled`         | Gauge   | `1` if DNS protection is enabled, `0` otherwise.               |
| `adguard_dns_queries_total`          | Gauge   | Total DNS queries in the current stats window.                 |
| `adguard_blocked_filtering_total`    | Gauge   | Queries blocked by filter lists.                               |
| `adguard_blocked_safebrowsing_total` | Gauge   | Queries blocked by safe browsing (malware/phishing).           |
| `adguard_blocked_parental_total`     | Gauge   | Queries blocked by parental controls.                          |
| `adguard_enforced_safesearch_total`  | Gauge   | Queries with safe search enforced.                             |
| `adguard_avg_processing_time_seconds`| Gauge   | Average DNS resolution time in seconds.                        |
| `adguard_scrape_duration_seconds`    | Gauge   | Time taken to scrape this instance.                            |
| `adguard_scrape_errors_total`        | Counter | Total failed scrapes since process start (monotonic).          |

> The ADH "total" fields are exposed as **gauges** because AdGuard Home returns rolling-window values
> that can decrease over time. Using `Counter` for them would break `rate()` in Prometheus.
> `adguard_scrape_errors_total` is the only true counter — it is generated by the exporter and is
> always monotonically increasing.

### Top-N metrics

Expose per-entry gauges with an additional label. The exporter emits whatever the API returns (typically up to 25 entries) — no artificial limit is applied.

| Metric                                  | Extra label | Description                              |
|-----------------------------------------|-------------|------------------------------------------|
| `adguard_top_clients`                   | `client`    | DNS queries from top clients.            |
| `adguard_top_queried_domains`           | `domain`    | Most queried domains.                    |
| `adguard_top_blocked_domains`           | `domain`    | Most blocked domains.                    |
| `adguard_top_upstreams`                 | `upstream`  | Responses per upstream DNS server.       |
| `adguard_top_upstreams_avg_time_seconds`| `upstream`  | Average response time per upstream (s).  |

Top-N entries are regenerated fresh on every scrape — stale entries never linger in the output.

---

## Service Mode

Install the exporter as a system service so it starts automatically with the OS.

```bash
# Install (captures current ADGUARD_* env vars and flags into the service unit)
sudo ADGUARD_URL_1=http://192.168.1.1 \
     ADGUARD_USERNAME_1=admin \
     ADGUARD_PASSWORD_1=secret \
     ./bin/adguardhome-exporter --service install

# Start / stop / restart
sudo ./bin/adguardhome-exporter --service start
sudo ./bin/adguardhome-exporter --service stop
sudo ./bin/adguardhome-exporter --service restart

# Remove
sudo ./bin/adguardhome-exporter --service uninstall
```

At install time, all `ADGUARD_*` environment variables present in the current shell are captured into the service definition (`Environment=` lines in the systemd unit). Any `--port` and `--instance` flags passed at install time are also preserved. The service therefore has full configuration available when the OS starts it — no additional setup required.

Service metadata:
- **Name:** `adguardhome-exporter`
- **Display name:** `AdGuard Home Prometheus Exporter`

---

## Docker / Compose

```yaml
services:
  adguard-exporter:
    image: ghcr.io/t0mer/aghexporter:latest
    ports:
      - "9100:9100"
    environment:
      ADGUARD_URL_1: http://adguardhome:3000
      ADGUARD_USERNAME_1: admin
      ADGUARD_PASSWORD_1: secret
    restart: unless-stopped
```

With Docker secrets:

```yaml
services:
  adguard-exporter:
    image: ghcr.io/t0mer/aghexporter:latest
    ports:
      - "9100:9100"
    environment:
      ADGUARD_URL_1: http://adguardhome:3000
      ADGUARD_USERNAME_FILE_1: /run/secrets/adguard_username
      ADGUARD_PASSWORD_FILE_1: /run/secrets/adguard_password
    secrets:
      - adguard_username
      - adguard_password
    restart: unless-stopped

secrets:
  adguard_username:
    file: ./secrets/username.txt
  adguard_password:
    file: ./secrets/password.txt
```

---

## Prometheus Configuration

```yaml
scrape_configs:
  - job_name: adguardhome
    static_configs:
      - targets: ["localhost:9100"]
```

---

## Grafana

Point a Prometheus datasource at the scrape job above. All metrics are labelled with `instance` so you can filter or group by individual AdGuard Home instances. Panels covering the following areas work out of the box:

- Status & protection state
- DNS traffic and query counts
- Security & filtering (safe browsing, parental, safe search)
- Top clients, queried domains, and blocked domains
- Processing time and upstream response latency

---

## License

See [LICENSE](LICENSE).
