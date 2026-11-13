// Package server owns the HTTP layer: routing, middleware, JSON handlers and
// the embedded SPA. Domain logic lives in internal/feeds; persistence and the
// data model in internal/repository.
package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"tinyrss/internal/feeds"
	"tinyrss/internal/repository"
	"tinyrss/internal/server/middlewares"
)

type Server struct {
	repo    *repository.Repo
	fetcher *feeds.Fetcher
	token   []byte // empty = auth disabled
	spa     fs.FS
}

func New(repo *repository.Repo, fetcher *feeds.Fetcher, token string, spa fs.FS) *Server {
	s := &Server{repo: repo, fetcher: fetcher, spa: spa}
	if token != "" {
		s.token = []byte(token)
	}
	return s
}

func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	s.routes(api)

	// /api/health lets clients probe liveness; /api/favicon is fetched by <img>
	// tags (which cannot send a bearer token) and only proxies public icons.
	authMW := middlewares.Auth(s.token, middlewares.ExcludePaths("/api/health", "/api/favicon"))
	apiChain := middlewares.Recovery(middlewares.RequestID(middlewares.Logging(authMW(api))))
	staticChain := middlewares.Recovery(middlewares.RequestID(middlewares.Logging(spaHandler(s.spa))))

	mux := http.NewServeMux()
	mux.Handle("/api/", apiChain)
	mux.Handle("/", staticChain)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
