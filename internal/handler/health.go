package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

// Health returns a handler that checks database connectivity.
func Health(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Pool.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  "database unreachable",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "healthy",
			"service": "moodifydb",
		})
	}
}

// writeJSON is a helper to send JSON responses with a status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError is a helper to send JSON error responses.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
