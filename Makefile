.PHONY: build web run dev test clean

## Build the single binary: builds the Vue frontend, then compiles Go with the dist embedded.
build: web
	go build -trimpath -ldflags="-s -w" -o tinyrss .

## Build just the frontend bundle.
web:
	cd web && bun install && bun run build

## Run in place (dev).
run:
	go run . -addr 127.0.0.1:8087 -db data/tinyrss.db

## Dev mode: Go backend on :8087 + Vite dev server (hot reload) on :5173,
## with Vite proxying /api to the backend. Auth uses the fixed dev token
## "tinyrss"; log in with it on the login page. Ctrl-C stops both.
dev:
	mkdir -p data
	@TINYRSS_TOKEN=tinyrss go run . -addr 127.0.0.1:8087 -db data/tinyrss.db & \
	BACKEND=$$!; \
	trap 'kill $$BACKEND 2>/dev/null' EXIT INT TERM; \
	cd web && bun run dev

## Run backend tests (and frontend tests if present).
test:
	go test ./...
	-cd web && bun run test 2>/dev/null || true

## Clean build artifacts.
clean:
	rm -rf web/dist data tinyrss
