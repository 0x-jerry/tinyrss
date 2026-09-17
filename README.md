# tinyrss

A self-hosted, single-user RSS reader. One Go binary serves a JSON API plus an
embedded Vue 3 frontend, and a background goroutine polls your feeds into a
local SQLite database. Classic 3-pane web UI: feeds/folders tree · article list
· reading view.

## Features

- Add/rename/delete feeds and folders; unread badges and totals
- Background feed polling (default every 15m) with conditional GET (304 → skip)
- Read/unread, star, mark-all-read; article list filters (all/unread/starred) + search
- Full-text search (SQLite FTS5)
- OPML import + export
- Single-access-token auth (optional)
- Self-contained binary: the Vue build is embedded into the Go executable

## Stack

- **Backend:** Go 1.22+ — stdlib `net/http` (Go 1.22 `ServeMux`), `modernc.org/sqlite`
  (pure-Go, no CGO), `github.com/mmcdole/gofeed`
- **Storage:** SQLite in WAL mode, FTS5 full-text index, versioned migrations
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
| `-refresh` | — | `15m` | Poll interval for due feeds |
| `-token` | `TINYRSS_TOKEN` | *(empty)* | Access token; env wins over the flag |

## Backend layout

```
main.go                 wiring: config → db → feeds → server; graceful shutdown
internal/config         flags/env
internal/store          SQLite open, PRAGMAs, versioned migrations
internal/feeds          repository (folders/feeds/items), fetch+scheduler, search, OPML
internal/server         net/http router, handlers, auth middleware, embedded SPA
```

## Tests

- Backend: fixture-driven tests covering feed parsing (RSS/Atom), repository
  CRUD + GUID dedup, fetcher conditional-GET/new-item detection, API handlers +
  auth, and OPML round-trip — `go test ./...`
- Frontend: Vitest for the API client and provider logic — `bun run test`
