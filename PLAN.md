# tinyrss — RSS Reader Architecture (Go + SQLite + Vue)

## 1. Goal & success criteria

Build a self-hosted, **single-user** RSS reader: one Go binary serves a JSON API plus an embedded static Vue 3 build, and a background goroutine polls feeds into a local SQLite database. Classic 3-pane web UI.

**Success criteria**
- `tinyrss` binary runs from a single artifact: `go build` after `bun run build` embeds the Vue dist; no CGO, no external services.
- Add/edit/delete feeds and folders, manual + scheduled refresh, read/unread, star, OPML import/export, and full-text search all work end to end.
- 3-pane UI: feeds/folders tree (with unread badges) | article list (virtualized, paginated) | reading view (mark-read on view).
- Feeds are fetched with conditional GET, deduped by GUID, and the app stays responsive while prints happen (async, WAL).
- Every non-trivial backend path has one runnable check (fixture-based parser/store/handler tests).

## 2. Repo layout (monorepo)

```
tinyrss/
├── go.mod / go.sum              # module path: tinyrss ; go 1.22+
├── main.go                      # wiring: config → db → fetcher → server; graceful shutdown
├── internal/
│   ├── config/                  # flag/env parsing (-addr :8087, -db data/tinyrss.db, -refresh 15m, -token)
│   ├── store/                   # sql.DB open, PRAGMAs, schema migrations (embedded .sql)
│   ├── feeds/                   # domain: repository over items/feeds/folders + OPML + search
│   └── server/                  # HTTP router, handlers, auth middleware, static SPA embedding
│       └── assets/              # //go:embed web/dist (built output)
├── web/                         # Vue 3 + Vite + TS app
│   ├── vite.config.ts, package.json, index.html
│   └── src/
│       ├── main.ts App.vue router.ts
│       ├── api/                 # typed REST client (base URL, fetch wrappers, Bearer header, error handling)
│       ├── providers/           # provide/inject state (typed InjectionKey per domain)
│       │   ├── keys.ts          # InjectionKey<T> symbols for each provider
│       │   ├── auth.ts          # provideAuth / injectAuth (token, isAuthenticated)
│       │   ├── feedsTree.ts     # folders+feeds+unread badges
│       │   ├── selection.ts     # current folder/feed/item
│       │   └── items.ts         # article list, pagination, filters
│       ├── composables/         # useAuth, useFeeds, useItems, useAutoRefresh, useKeyboard (thin, non-state)
│       ├── types/               # models mirroring backend JSON
│       ├── components/shared/   # Button, Icon, Badge, EmptyState, ConfirmDialog, LoginForm
│       ├── components/business/ # FeedTree, FolderItem, ArticleList, ArticlePreview, ReaderPane
│       └── views/               # LoginView, FeedLayout (3-pane shell), SettingsView, SearchView
├── data/                        # runtime tinyrss.db (gitignored)
└── Makefile                     # build (bun→go), run, test, clean, embed check
```

Packages split by domain, not strict layers: `feeds` owns item/feed/folder/OPML/search data access + fetch logic; `server` owns HTTP; `store` owns the raw connection only.

## 3. Data model (SQLite, WAL mode)

PRAGMAs on open: `journal_mode=WAL`, `foreign_keys=ON`, `busy_timeout=5000`, `synchronous=NORMAL`. Driver: `modernc.org/sqlite` (pure Go, no CGO → static cross-compilable binary).

Tables:
- `feeds`: id PK, title, feed_url UNIQUE, site_url, description, folder_id → folders.id, etag, last_modified, last_fetched_at, fetch_error, created_at, updated_at.
- `folders`: id PK, name UNIQUE, sort_order.
- `items`: id PK, feed_id → feeds.id ON DELETE CASCADE, guid, title, url, author, summary, content, published_at, is_read (0/1), is_starred (0/1), created_at; `UNIQUE(feed_id, guid)` for dedup.
- `items_fts`: SQLite **FTS5** virtual table (title, summary, content) kept in sync via triggers on items insert/update/delete — enables fast, ranked search over large content.
- `schema_migrations`: versioned `.sql` files applied in order on startup (embedded `//go:embed`).

Query patterns: unread badges = `COUNT(*) WHERE feed_id=? AND is_read=0`; article list filtered by feed_id/folder_id/unread/starred/search with `LIMIT/OFFSET` pagination; FTS5 via `MATCH ?`.

## 4. Backend (Go)

### Auth (single access token)
- A single opaque access token is configured at startup from the **`TINYRSS_TOKEN` env var** first, else the `-token` flag (env preferred — a flag is visible via `ps`). The value must be non-empty and non-trivial (recommend `openssl rand -hex 32`).
- All `/api/*` routes (except `/api/health`) are guarded by an **auth middleware** that checks `Authorization: Bearer <token>`. Mismatch/missing → `401 {"error": "unauthorized"}`.
- If **no token is configured**, auth is disabled and the server behaves as the original loopback single-user mode (documented: only bind to `127.0.0.1` in that case). Presenting a token turns auth on.
- Token is compared with a constant-time check (`crypto/subtle`) and is never logged.

### HTTP layer (`internal/server`)
- stdlib `net/http` `ServeMux` (Go 1.22 method+path patterns). No framework.
- Middleware: request logging, panic recovery, `Content-Type: application/json`, request ID, **auth guard** (above).
- Routes `/api/*` → JSON handlers; everything else → embedded SPA (serve `index.html` fallback for history routing).
- Consistent responses: success = plain JSON object/array; error = `{"error": "message"}` with proper status (400/401/404/409/422/502).

### API (REST, JSON)
- **Feeds**: `GET/POST /api/feeds` (list with per-feed unread + folder grouping; POST takes a **direct feed URL** — must parse as RSS/Atom), `GET/PUT/DELETE /api/feeds/{id}`, `POST /api/feeds/{id}/refresh`.
- **Folders**: `GET/POST /api/folders`, `PUT/DELETE /api/folders/{id}` (delete folds feeds to "Uncategorized").
- **Items**: `GET /api/items?feed_id&folder_id&unread&search&starred&page&limit`; `GET /api/items/{id}` (full content); `POST .../read|unread|star|unstar`; `POST /api/items/read-all?feed_id=&folder_id=`; mark-read-on-view is a client-triggered `read` call.
- **OPML**: `POST /api/opml/import` (multipart), `GET /api/opml/export`.
- **System**: `GET /api/health`, `POST /api/refresh` (global), `GET /api/stats` (total/unread counts for sidebar).

### Fetching & parsing (`internal/feeds`)
- HTTP client: timeout, gzip, redirect limit, and conditional GET using stored `etag`/`Last-Modified` → **304 = skip** (update `last_fetched_at` only, save bandwidth).
- Parsing via `github.com/mmcdole/gofeed` (handles RSS 2.0/1.0 + Atom + dates via one API). Users paste a **direct feed URL** — no HTML discovery step, so `golang.org/x/net/html` is not needed.
- Dedup: upsert items keyed by `(feed_id, guid)`; only newly seen items become unread.
- Scheduler: a `fetcher` goroutine with a ticker (`-refresh` interval, default 15m) that fetches only due feeds, staggered; bounded worker pool (default 4); `singleflight` per feed to dedupe manual + scheduled refreshes. Manual `refresh` blocks until done and returns new-item/error info to the caller.
- Graceful shutdown: on SIGINT/SIGTERM, stop scheduler, `http.Server.Shutdown(ctx)` with drain.

### SQLite concurrency note
WAL gives concurrent readers + one writer; fetcher and API share one `*sql.DB`. To keep writes serialized and avoid `database is locked`, connections are pooled with `SetMaxOpenConns(1)` write path via a small write mutex if contention appears (documented ceiling + upgrade path). `busy_timeout` covers transient locks.

## 5. Frontend (Vue 3)

- Vite + Vue 3 + **TypeScript**, Composition API `<script setup lang="ts">` throughout (per vue skill: `XXXProps`/`XXXEmits` interfaces, `defineModel`, typed emits tuple form). Package manager **bun** (`bun install` / `bun run build` / `bun run dev:web`).
- **Router**: `vue-router` history mode (SPA fallback already served by backend).
- **State via `provide`/`inject`** (no Pinia): app-scoped singleton state lives in provider modules under `src/providers/`, each exporting a typed `InjectionKey<T>` (`keys.ts`) plus a pair — `provideXxx` (call once at the app root) and `injectXxx` (used by business/page components). State is a `reactive()` object exposed `readonly` (mutations only through the provider's returned actions), keeping shared components free of business logic and preserving the one-way page → business → shared dependency. Providers: `auth` (token, isAuthenticated), `feedsTree` (folders+feeds+unread badges), `selection` (current folder/feed/item), `items` (list + pagination + filters).
- **API client** (`src/api/`): typed `fetch` wrappers + a small validation layer; every endpoint has a matching TS type; errors surfaced via a shared `useApiResult`/toast. Attaches `Authorization: Bearer <token>` (read from the auth provider).
- **Auth**: an `auth` provider holds the token (persisted in `localStorage`) and an authenticated flag, exposed via `injectAuth`. A `LoginView` prompts for the token once and stores it. A **router guard** redirects to `LoginView` when unauthenticated; a `401` from any API call clears the token and bounces back to login. No separate login page when no token is configured is fine (API client detects auth disabled via `/api/health` or a 401-less first call).
- **3-pane layout** (classic):
  - Left: `FeedTree` — folders + feeds with unread badges, add-feed form (URL → discovery), folder management, refresh-all.
  - Middle: `ArticleList` — virtualized long list (virtual scrolling), filters (unread/starred/all, search), infinite scroll or pagination, selection sync.
  - Right: `ReaderPane` — renders full article content; **`DOMPurify`-sanitized** before `v-html` (XSS guard); marks read on open/focus; star/prev/next; unread toggle.
- **Auto-refresh**: `useIntervalFn` polls `/api/feeds` or `/api/stats` (e.g. every 60s) to update unread badges; manual refresh buttons.
- **XSS**: article `content`/`summary` rendered only through DOMPurify → `v-html`. All other output Vue-escaped.
- **De-risking deps**: minimal — `vue-router`, DOMPurify; `@vueuse/core` for hooks (useIntervalFn, virtual scrolling); no Pinia (state via `provide`/`inject`), no CSS framework by default (scoped plain CSS in shared components); Tailwind noted as optional later.

## 6. Build & run pipeline

- Package manager: **bun**. Frontend: `bun install` → `bun run build` (`vite build` → `web/dist`); dev server `bun run dev:web`.
- Backend: `go:embed` `internal/server/assets` reads `web/dist` at build time. Two-step build documented in README + Makefile targets: `make build` = `(cd web && bun install && bun run build) && go build ./...`.
- Run: `TINYRSS_TOKEN=$(openssl rand -hex 32) ./tinyrss -addr 127.0.0.1:8087 -db data/tinyrss.db`. First run auto-applies migrations, creates `data/`. (Omit the token for auth-less loopback mode.)
- `.gitignore`: `data/`, `web/dist`, `web/node_modules` (dist is a build artifact, not committed; embed reference documents the prerequisite).

## 7. Tests & acceptance criteria

- **Go** (fixture-driven, one runnable check per non-trivial path):
  - Parser: canned RSS + Atom fixtures → normalized structs (dates, missing fields).
  - Store: CRUD + dedup (same GUID twice → one item) + unread counting against `:memory:` sqlite with migrations.
  - Fetcher: 304 path, new-item detection, singleflight dedup.
  - API handlers: `httptest` against `:memory:` db — feeds/items/folders/OPML/search, error statuses.
  - OPML: round-trip import→export equals input structure.
- **Frontend** (light): Vitest for the `api` client and `selection`/`items` provider logic; no heavy E2E initially — optional Playwright smoke (boot app, add feed, see item) deferred.
- **Acceptance checks**: `go test ./...`, `bun run build` clean, `make build` embeds dist, manual smoke of the full add→refresh→read→star→OPML→search loop against a live feed (e.g. a real well-known RSS feed).

## 8. Edge cases & failure modes

- Non-feed URL on add → return a clear 422 error instructing the user to paste a direct RSS/Atom URL (no discovery).
- Empty/unparseable feed → record `fetch_error`, keep feed alive, retry next cycle.
- HTTP 304 → skip parsing, update `last_fetched_at` only.
- Redirect loops / oversized bodies → client limits; cap fetched body size.
- Malformed/unsupported dates → fall back to fetch time.
- GUID collision / regenerating GUIDs → upsert by `(feed_id,guid)` keeps single copy.
- Unread-count correctness under concurrent fetch+read → count via SQL at read time, not a cached denormalized field (avoids drift); WAL + write mutex avoids lock errors.
- Large feeds → paginated list, virtualized rendering, capped content sizes, FTS5 indexed search (escape user query; handle `*:""` empty).
- Network/upstream downtime → per-feed `fetch_error` + backoff so one dead feed never blocks others or the scheduler.
- Manual refresh while scheduled fetch in flight → `singleflight` coalesces to one fetch.

## 9. Assumptions / explicit decisions

- **Single user, token auth** — an access token configured via `TINYRSS_TOKEN` env or `-token` guards all `/api/*` (Bearer header). If no token is configured, auth is off (loopback default only). Multi-user/accounts remain deferred.
- Core-reader scope only: folders (not tags), star, basic FTS search, OPML. Deferred (noted, not built): multi-user accounts/user management, read-later, sync clients, TTS/listen, dark mode theming, keyboard-shortcut reference sheet, and feed auto-discovery from a webpage URL (users paste direct feed URLs; add discovery later only if needed).
- `modernc.org/sqlite` (pure Go) chosen over `mattn/go-sqlite3` to keep the binary CGO-free and cross-compilable.
- Libraries allowed for parsing/search/sanitization (gofeed, FTS5, DOMPurify); everything else stdlib + minimal deps per the "laziness ladder".
