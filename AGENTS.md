# AGENTS.md

Compact guidance for OpenCode sessions working in this repo.

## Build & Verification

- **Build**: `bash deploy/build.sh` — cross-compiles `dist/aggregator-linux-amd64`, `dist/probe-linux-amd64`, `dist/probe-linux-arm64`.
- **No tests. No lint config.** Do not run `go test` or expect a linter.
- **Go version**: `1.25` (go.mod). Build flags used: `CGO_ENABLED=0 -trimpath -ldflags "-s -w"`.

## Architecture Gotchas

- Two binaries share `internal/`: `cmd/aggregator` (HTTP server + embedded SPA) and `cmd/probe` (agent).
- **Static files live in `cmd/aggregator/static/`**, not a root `static/` dir. Embedded via `//go:embed static`.
- **SPA is vanilla JS/CSS** (no framework, no build step). Edit `cmd/aggregator/static/{index.html,app.css,app.js}` directly.
- Aggregator migrations are automatic on startup (`store.Open` runs `migrate`). No separate migration tool.

## Env & Credentials

- Secrets live in two gitignored files: `info` (Bangumi cookie, API token) and `.ingest_secret` (aggregator ingest token). Deploy scripts `grep`/`cut` from `info` directly.
- Aggregator requires `DB_DSN` and at least one of `INGEST_SECRET` (legacy single token) or `INGEST_SECRETS` (per-operator). `INGEST_SECRETS` format: `token1:prefix1;token2:prefix2`. Prefixes must not overlap.
- Probe credentials (`BGM_COOKIE`, `BGM_API_TOKEN`) are optional. **If both are omitted, probe runs guest-only** (skips Auth checks entirely).
- `REGION` must be a valid ISO 3166-1 alpha-2 country code (lowercase). Validation is hardcoded in `internal/region/region.go`; invalid values are rejected at startup and by the aggregator ingest endpoint.

## Operational Logic

- **Quorum**: status only escalates from OK when `≥ ceil(2/3 × active probes)` agree (minimum 2). This means with fewer than 3 probes, nothing ever escalates above OK.
- **Telegram alerts** debounce: 2 consecutive bad observations before firing. Only `Auth`-kind components on `bgm.tv`/`bangumi.tv`/`chii.in` trigger messages. API subdomains are silent.
- **Cache**: `/api/status` has a 20s HTTP cache header, but the server-side cache is stale-after 30s with a background refresh loop every 20s. Ingest calls invalidate the cache immediately.
- **Daily report** fires at CST (UTC+8) midnight, not UTC.
- **Data retention**: checks older than ~35 days are purged every 6 hours. `online_counts` is kept indefinitely.

## Deploy & CI

- `deploy/build.sh` is the canonical build command. Do not use `go build` alone for releases — the flags matter.
- `deploy/deploy-all.sh` expects `INGEST_SECRET` in env and hardcodes SSH host names (`Tokyo-1`, `Eden`, `Osaka-*`, `Shenzhen-1`).
- `deploy/deploy-probe.sh` reads the local `info` file, converts cookie JSON to a header string with Python, and generates a per-probe env file.
- **Dockerfile only builds the probe** (scratch-based image). Aggregator is deployed as a bare binary behind Caddy.
- **GitHub Actions** (`.github/workflows/release.yml`) triggers on `v*` tags: builds binaries + creates release, and builds/pushes a multi-arch Docker image for the probe only (`ghcr.io/.../bangumi-status-probe`).

## Style & Conventions

- Standard Go layout. No code generation.
- Prefer editing existing files over creating new ones. The codebase is intentionally small and monolithic.
