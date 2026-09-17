# tinyrss

A self-hosted, single-user RSS reader: one Go binary serving a JSON API plus an
embedded Vue 3 frontend, with a background goroutine polling feeds into SQLite.

See [PLAN.md](PLAN.md) for the full architecture.

## Quick start

```sh
make build                          # builds web/ via bun, then compiles the binary
TINYRSS_TOKEN=$(openssl rand -hex 32) ./tinyrss -addr 127.0.0.1:8087 -db data/tinyrss.db
# open http://127.0.0.1:8087 and paste the token to log in
```

- Token auth: set `TINYRSS_TOKEN` (or `-token`) to require `Authorization: Bearer <token>`
  on all `/api/*` routes. Omit it to run auth-free on loopback.
- Stack: Go 1.22+ (`net/http`, `modernc.org/sqlite`, gofeed) · SQLite (WAL + FTS5) · Vue 3 + Vite + TypeScript (bun).
