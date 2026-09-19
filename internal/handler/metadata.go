package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

// MetadataResponse represents the stored metadata record for HTTP responses.
type MetadataResponse struct {
	ID     string `json:"id"`
	SongID string `json:"song_id"`
	metadata.SongMetadata
}

// SaveMetadata handles PUT /api/v1/songs/{id}/metadata.
// It persists user-approved or custom-edited metadata for a song.
func SaveMetadata(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		songID, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid song id format")
			return
		}

		// Ensure the song exists
		if _, err := db.GetSong(r.Context(), songID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusNotFound, "song not found")
				return
			}
			slog.Error("failed to check song existence", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		var meta metadata.SongMetadata
		if err := json.NewDecoder(r.Body).Decode(&meta); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json payload: "+err.Error())
			return
		}

		metaID, err := db.UpsertMetadata(r.Context(), songID, &meta)
		if err != nil {
			slog.Error("failed to save metadata", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "failed to save metadata")
			return
		}

		respondJSON(w, http.StatusOK, MetadataResponse{
			ID:           metaID.String(),
			SongID:       songID.String(),
			SongMetadata: meta,
		})
	}
}

// GetMetadata handles GET /api/v1/songs/{id}/metadata.
// It retrieves the stored metadata for a song.
func GetMetadata(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		songID, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid song id format")
			return
		}

		meta, metaID, err := db.GetMetadata(r.Context(), songID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusNotFound, "metadata not found for this song")
				return
			}
			slog.Error("failed to query song metadata", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		respondJSON(w, http.StatusOK, MetadataResponse{
			ID:           metaID.String(),
			SongID:       songID.String(),
			SongMetadata: *meta,
		})
	}
}
