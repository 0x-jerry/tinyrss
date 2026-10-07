package server

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// spaHandler serves the embedded frontend, falling back to index.html so
// history-mode routes resolve to the SPA shell.
func spaHandler(spa fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := shellPath(spa, strings.TrimPrefix(r.URL.Path, "/"))
		f, err := spa.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		var modTime time.Time
		if st, err := f.Stat(); err == nil {
			modTime = st.ModTime()
		}
		setStaticHeaders(w, name)
		if rs, ok := f.(io.ReadSeeker); ok {
			http.ServeContent(w, r, name, modTime, rs)
			return
		}
		data, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "read asset", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, name, modTime, bytes.NewReader(data))
	})
}

// shellPath is the embedded file to serve: the request path when it names a real
// file, otherwise index.html.
func shellPath(spa fs.FS, name string) string {
	if name == "" {
		return "index.html"
	}
	if st, err := fs.Stat(spa, name); err == nil && !st.IsDir() {
		return name
	}
	return "index.html"
}

func setStaticHeaders(w http.ResponseWriter, name string) {
	// Go's MIME registry (and mime.types on macOS/Alpine) has no .webmanifest
	// entry, so ServeContent would sniff it as text/plain.
	if path.Ext(name) == ".webmanifest" {
		w.Header().Set("Content-Type", "application/manifest+json")
	}
	w.Header().Set("Cache-Control", cacheControlFor(name))
}

func cacheControlFor(name string) string {
	switch {
	// The shell, the worker, and the manifest must never be served stale: they
	// carry the pointers to each deploy's hashed assets.
	case name == "index.html" || name == "sw.js" || name == "manifest.webmanifest":
		return "no-cache"
	// Vite content-hashes asset filenames.
	case strings.HasPrefix(name, "assets/"):
		return "public, max-age=31536000, immutable"
	default:
		return "public, max-age=3600"
	}
}
