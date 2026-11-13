package middlewares

import (
	"encoding/json"
	"log"
	"net/http"
)

// writeJSONError writes {"error": msg} as JSON, mirroring the server's error
// shape so middleware-generated errors match handler-generated ones.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": msg}); err != nil {
		log.Printf("encode middleware error response: %v", err)
	}
}
