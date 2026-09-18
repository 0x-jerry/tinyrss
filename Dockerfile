# syntax=docker/dockerfile:1

# Multi-stage build: the single Go binary embeds the Vue frontend (web/dist),
# so we build the frontend first, then compile Go with it, then ship just the
# static binary + CA certs.

# ---- Frontend: build web/dist with bun ----
FROM oven/bun:1.4.2 AS web
WORKDIR /src
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ .
RUN bun run build

# ---- Backend: build the Go binary (embeds web/dist) ----
FROM golang:1.27-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY --from=web /src/dist web/dist
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -buildvcs=false -o /out/tinyrss .

# ---- Runtime ----
FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 tinyrss
WORKDIR /app
COPY --from=build /out/tinyrss ./tinyrss
RUN mkdir -p /data && chown tinyrss:tinyrss /data
USER tinyrss
VOLUME ["/data"]
EXPOSE 8087
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8087/api/health >/dev/null || exit 1
# Override with -e TINYRSS_TOKEN=<hex> to require auth (recommended: the
# container listens on 0.0.0.0).
ENV TINYRSS_TOKEN=
CMD ["/app/tinyrss", "-addr", ":8087", "-db", "/data/tinyrss.db"]
