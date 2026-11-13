package middlewares

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// AuthOptions configures the Auth middleware.
type AuthOptions struct {
	// Exclude lists exact request paths served without a token even when auth
	// is enabled (e.g. liveness probes and <img>-fetched public assets).
	Exclude []string
}

// AuthOption mutates AuthOptions; used as the variadic options for Auth.
type AuthOption func(*AuthOptions)

// ExcludePaths adds paths to the set served without authentication.
func ExcludePaths(paths ...string) AuthOption {
	return func(o *AuthOptions) {
		o.Exclude = append(o.Exclude, paths...)
	}
}

// Auth guards the wrapped handler with a bearer token. An empty token disables
// auth entirely. Paths named by ExcludePaths bypass the check.
func Auth(token []byte, opts ...AuthOption) func(http.Handler) http.Handler {
	var o AuthOptions
	for _, opt := range opts {
		opt(&o)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(token) == 0 || o.excludes(r.URL.Path) || authorized(r, token) {
				next.ServeHTTP(w, r)
				return
			}
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		})
	}
}

func (o *AuthOptions) excludes(path string) bool {
	for _, p := range o.Exclude {
		if path == p {
			return true
		}
	}
	return false
}

// authorized reports whether the request carries the expected bearer token,
// compared in constant time.
func authorized(r *http.Request, token []byte) bool {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	tok := strings.TrimSpace(auth[len(prefix):])
	if len(tok) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tok), token) == 1
}
