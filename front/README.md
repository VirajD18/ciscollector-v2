# KloudDB Shield — DBA Console (Vanilla Frontend)

Enterprise PostgreSQL security dashboard built with **HTML5**, **CSS3**, and **vanilla JavaScript (ES modules)**.

## UI vs data (important)

| Source | Role |
|--------|------|
| **`dba-console-prototype.html`** (repo root) | **Reference UI only** — layout, CSS, copy, demo charts. **Do not use its hardcoded `STRATEGIC_BY_RANGE` / `FLEET_CATEGORIES` as product data.** |
| **`front/`** (this folder) | **Production UI** served by `main-server`. All fleet/host/violation numbers come from **HTTP APIs**. |
| **SQLite** (`~/.klouddb/klouddbshield.db` or `storage` in `kshieldconfig.toml`) | **Single source of truth** — collectors persist scan runs; dashboard reads via `pkg/reportstore`. |

```
ciscollector → reportstore (SQLite)
                    ↓
main-server /api/strategic, /api/fleet/categories, /api/hosts, …
                    ↓
front/scripts/api/*  →  front/scripts/app/prototype-app.js (render)
```

## Structure

| Path | Purpose |
|------|---------|
| `index.html` | App entry shell |
| `pages/` | One HTML file per dashboard view |
| `components/` | Sidebar, topbar, flow banner |
| `styles/` | Design tokens + layout CSS |
| `scripts/` | Router, app bootstrap, dashboard logic |
| `scripts/api/` | `fetch()` client + service modules (under `/scripts/api/` to avoid clashing with Go `/api/*` routes) |
| `mock-data/` | JSON fixtures until Go APIs are live |
| `utilities/` | Shared formatters |
| `vendor/` | Optional local vendor assets |
| `layouts/` | Architecture notes |

## Local development

Serve via the Go backend (required for ES modules and mock JSON):

```sh
# From repo root — syncs front/ into embed dist and builds
make front-sync
go build -o main-server.exe ./cmd/main-server
./main-server.exe -addr :8081
```

Open [http://localhost:8081](http://localhost:8081)

Do **not** open `index.html` via `file://` — dynamic `import()` and `fetch()` need HTTP.

## Regenerate from prototype

After editing `dba-console-prototype.html`:

```sh
node front/scripts/extract-prototype.mjs
```

## API integration

- Set `useMock: false` in `scripts/api/config.js` (default) so the UI reads **live SQLite-backed** endpoints from `cmd/main-server`.
- Boot load: `scripts/app/init.js` fetches hosts, fleet categories, strategic (30d), violations, findings, overview, runs.
- Range change / **Refresh** on Fleet Overview: `scripts/app/strategic-loader.js` re-fetches `/api/strategic?range=` and `/api/fleet/categories` from the DB.
- `mock-data/*.json` is only used when `useMock: true` or the server is unreachable.

## Charts

Strategic visuals use CSS (prototype parity). [ApexCharts](https://apexcharts.com/) is loaded for optional enhancements via `scripts/charts/apex-theme.js`.
