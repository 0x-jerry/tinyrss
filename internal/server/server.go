// Package server owns the HTTP layer: routing, middleware, JSON handlers and
// the embedded SPA. Domain logic lives in internal/feeds.
package server

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"tinyrss/internal/feeds"
)

type Server struct {
	repo    *feeds.Repo
	fetcher *feeds.Fetcher
	token   []byte // empty = auth disabled
	spa     fs.FS
}

func New(repo *feeds.Repo, fetcher *feeds.Fetcher, token string, spa fs.FS) *Server {
	s := &Server{repo: repo, fetcher: fetcher, spa: spa}
	if token != "" {
		s.token = []byte(token)
	}
	return s
}

func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	s.routes(api)

	apiChain := s.withRecovery(s.withRequestID(s.withLogging(s.withAuth(api))))
	staticChain := s.withRecovery(s.withRequestID(s.withLogging(spaHandler(s.spa))))

	mux := http.NewServeMux()
	mux.Handle("/api/", apiChain)
	mux.Handle("/", staticChain)
	return mux
}

// ---- middleware ----

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		r = r.WithContext(contextWithReqID(r.Context(), id))
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s -> %d (%s) [%s]",
			r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond), reqID(r.Context()))
	})
}

func (s *Server) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic serving %s %s: %v", r.Method, r.URL.Path, rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withAuth guards every /api/* route except /api/health when a token is set.
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.token) == 0 || r.URL.Path == "/api/health" || s.authorized(r) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusUnauthorized, "unauthorized")
	})
}

func (s *Server) authorized(r *http.Request) bool {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	tok := strings.TrimSpace(auth[len(prefix):])
	if len(tok) != len(s.token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tok), s.token) == 1
}

// ---- response helpers ----

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

// spaHandler serves the embedded frontend, falling back to index.html so
// history-mode routes resolve to the SPA shell.
func spaHandler(spa fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := spa.Open(path); err == nil {
			defer f.Close()
			if st, err := f.Stat(); err == nil && !st.IsDir() {
				if rs, ok := f.(io.ReadSeeker); ok {
					http.ServeContent(w, r, path, st.ModTime(), rs)
					return
				}
				data, _ := io.ReadAll(f)
				http.ServeContent(w, r, path, st.ModTime(), bytes.NewReader(data))
				return
			}
		}
		if f, err := spa.Open("index.html"); err == nil {
			defer f.Close()
			st, _ := f.Stat()
			if rs, ok := f.(io.ReadSeeker); ok {
				http.ServeContent(w, r, "index.html", st.ModTime(), rs)
				return
			}
			data, _ := io.ReadAll(f)
			http.ServeContent(w, r, "index.html", st.ModTime(), bytes.NewReader(data))
			return
		}
		http.NotFound(w, r)
	})
}
