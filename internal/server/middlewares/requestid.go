package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type ctxKey struct{}

// ContextWithReqID stores the request id in ctx.
func ContextWithReqID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// ReqID returns the request id stored in ctx, or "" when absent.
func ReqID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// RequestID ensures every request carries an X-Request-ID, generating one when
// absent, echoing it on the response, and exposing it via ReqID in the context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		r = r.WithContext(ContextWithReqID(r.Context(), id))
		next.ServeHTTP(w, r)
	})
}
