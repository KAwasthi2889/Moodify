package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// respondJSON serializes data as JSON with the specified HTTP status code.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response json", "error", err)
	}
}

// respondError writes a standardized JSON error response.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
