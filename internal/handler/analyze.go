package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/analyzer"
	"github.com/KAwasthi2889/Moodify/internal/database"
)

// AnalyzeSong handles POST /api/v1/songs/{id}/analyze.
// It extracts 36-D acoustic features and the mood vector via the Python sidecar,
// and persists the result to the song_features table.
func AnalyzeSong(db *database.DB, az *analyzer.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		songID, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid song id format")
			return
		}

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

		feat, err := az.Analyze(r.Context(), song.FilePath)
		if err != nil {
			slog.Error("failed to extract audio features", "error", err, "song_id", songID, "file", song.FilePath)
			respondError(w, http.StatusInternalServerError, "audio analysis failed: "+err.Error())
			return
		}

		feat.SongID = song.ID
		featID, err := db.UpsertFeatures(r.Context(), feat)
		if err != nil {
			slog.Error("failed to persist song features", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "failed to store audio features")
			return
		}

		slog.Info("audio features analyzed and stored",
			"song_id", songID,
			"feature_id", featID,
			"tempo_bpm", feat.TempoBPM,
			"vector_dims", len(feat.MoodVector),
		)

		respondJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"song_id":  songID,
			"features": feat,
		})
	}
}

// GetSongFeatures handles GET /api/v1/songs/{id}/features.
// It retrieves cached feature analysis from the database.
func GetSongFeatures(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		songID, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid song id format")
			return
		}

		feat, err := db.GetFeatures(r.Context(), songID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusNotFound, "features have not been analyzed for this song yet")
				return
			}
			slog.Error("failed to query song features", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"song_id":  songID,
			"features": feat,
		})
	}
}
