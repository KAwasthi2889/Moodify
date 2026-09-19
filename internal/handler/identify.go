package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

// IdentifyResponse is returned by POST /api/v1/songs/{id}/identify.
type IdentifyResponse struct {
	Status              string           `json:"status"`
	SongID              string           `json:"song_id"`
	FingerprintDuration float64          `json:"fingerprint_duration"`
	AutoSaved           bool             `json:"auto_saved"`
	Matches             []metadata.Match `json:"matches"`
	Warning             string           `json:"warning,omitempty"`
	Note                string           `json:"note,omitempty"`
}

// Identify handles POST /api/v1/songs/{id}/identify.
// It fingerprints the audio file, looks up metadata via AcoustID + MusicBrainz,
// and optionally persists the top match if ?auto_save=true is specified.
func Identify(db *database.DB, identifier *metadata.Identifier, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		songID, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid song id format")
			return
		}

		// Retrieve song from database
		song, err := db.GetSong(r.Context(), songID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusNotFound, "song not found")
				return
			}
			slog.Error("failed to query song", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		// Run identification sidecar
		result, err := identifier.Identify(r.Context(), song.FilePath, apiKey)
		if err != nil {
			slog.Error("identification failed", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "identification failed: "+err.Error())
			return
		}

		autoSave := r.URL.Query().Get("auto_save") == "true"
		var autoSaved bool

		if autoSave && len(result.Matches) > 0 {
			top := result.Matches[0]
			if _, err := db.UpsertMetadata(r.Context(), songID, &top.SongMetadata); err != nil {
				slog.Error("failed to auto-save metadata", "error", err, "song_id", songID)
			} else {
				autoSaved = true
			}
		}

		respondJSON(w, http.StatusOK, IdentifyResponse{
			Status:              result.Status,
			SongID:              songID.String(),
			FingerprintDuration: result.FingerprintDuration,
			AutoSaved:           autoSaved,
			Matches:             result.Matches,
			Warning:             result.Warning,
			Note:                result.Note,
		})
	}
}
