# tinyrss

A self-hosted, single-user RSS reader. One Go binary serves a JSON API plus an
embedded Vue 3 frontend, and a background goroutine polls your feeds into a
local SQLite database. Classic 3-pane web UI: feeds/folders tree · article list
· reading view.

## Features

- Classic 3-pane web UI: feeds/folders tree · article list · reading view, with
  keyboard shortcuts (`j`/`k` navigate, `m` toggle read)
- Add/rename/delete feeds and folders, unread badges and totals
- Subscribe by a single URL (with feed autodiscovery) and auto-fill feed details via **Detect**
- Background polling with conditional GET (304 → skip); configurable auto-refresh interval
- Read/unread, star, article filters + full-text search (SQLite FTS5)
- Built-in reader that extracts article content server-side
- Statistics page with per-feed article trends
- OPML import/export; optional token auth
- Self-contained single binary — the Vue frontend is embedded into the Go executable

## Stack

- **Backend:** Go (stdlib `net/http`, SQLite via `modernc.org/sqlite`, `gofeed` for parsing)
- **Storage:** SQLite — WAL mode, FTS5 full-text index, versioned migrations
- **Frontend:** Vue 3 + Vite + TypeScript, managed with **bun**

## Quick start

```sh
# 1. Build the single binary (frontend via bun, then Go embeds the dist)
make build

# 2. Run it with a token
TINYRSS_TOKEN=$(openssl rand -hex 32) ./tinyrss -addr 127.0.0.1:8087 -db data/tinyrss.db
# open http://127.0.0.1:8087 and paste the token to log in
```

## Docker

A multi-stage `Dockerfile` builds the frontend with bun, embeds it into the Go
binary, and ships a slim Alpine image with a `/data` volume for the SQLite DB.

```sh
# Build and run with the bundled Compose file
TINYRSS_TOKEN=$(openssl rand -hex 32) docker compose up -d --build
# open http://127.0.0.1:8087 and paste the token to log in
```

The `docker-compose.yml` maps port 8087, persists the DB in a named volume
(`tinyrss-data`), and reads `TINYRSS_TOKEN` from the environment (set it before
`up`, or edit the file). The container listens on `0.0.0.0:8087`, so a token is
required. To run a one-off container instead:

```sh
docker build -t tinyrss .
docker run -d -p 8087:8087 -v tinyrss-data:/data \
  -e TINYRSS_TOKEN=$(openssl rand -hex 32) tinyrss
```

## Make targets

| Target | What it does |
| --- | --- |
| `make build` | `bun install` + `vite build` in `web/`, then `go build -o tinyrss .` (dist embedded) |
| `make dev` | Run Go backend on `:8087` (dev token **`tinyrss`**) + Vite dev server with hot reload on `:5173`, `/api` proxied to the backend; Ctrl-C stops both |
| `make run` | `go run .` on `127.0.0.1:8087` |
| `make test` | `go test ./...` + `bun run test` |
| `make clean` | Remove `web/dist`, runtime `data/`, and the `tinyrss` binary |

### Dev mode

```sh
make dev      # open http://localhost:5173 and log in with the token: tinyrss
```

## Auth

Set `TINYRSS_TOKEN` (or the `-token` flag) to require
`Authorization: Bearer <token>` on all `/api/*` routes except `/api/health`
(compared in constant time). If no token is configured, auth is off — only bind
to loopback in that case. In dev (`make dev`) the token is fixed to `tinyrss`.

## Configuration

| Flag | Env | Default | Purpose |
| --- | --- | --- | --- |
| `-addr` | — | `127.0.0.1:8087` | Listen address |
| `-db` | — | `data/tinyrss.db` | SQLite database path (migrations auto-applied on first run) |
| `-refresh` | — | `15m` | Startup default poll interval; once running, the interval set in Settings (`refresh_interval_minutes`, default 15) governs |
| `-token` | `TINYRSS_TOKEN` | *(empty)* | Access token; env wins over the flag |

## Backend layout

```
main.go                 wiring: config → db → repository → feeds → server; graceful shutdown
internal/config         flags/env
internal/store          SQLite open, PRAGMAs, versioned migrations
internal/repository     persistence + data model: folders/feeds/items, fetch logs, settings, OPML
internal/feeds          fetch + scheduler + article render service
internal/server         net/http router, handlers, auth middleware, embedded SPA
```

## Tests

- Backend: fixture-driven tests covering feed parsing (RSS/Atom), repository
  CRUD + GUID dedup, fetcher conditional-GET/new-item detection, API handlers +
  auth, and OPML round-trip — `go test ./...`
- Frontend: Vitest for the API client and provider logic — `bun run test`
