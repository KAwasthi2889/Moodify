package handler

import (
	"net/http"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

// Health returns a handler that checks database connectivity.
func Health(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Pool.Ping(r.Context()); err != nil {
			respondJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  "database unreachable",
			})
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"status":  "healthy",
			"service": "moodifydb",
		})
	}
}
