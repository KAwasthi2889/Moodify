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
	"github.com/KAwasthi2889/Moodify/internal/fusion"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

// AnalyzeSong handles POST /api/v1/songs/{id}/analyze.
// It extracts 36-D acoustic features and the mood vector via the Python sidecar,
// fuses with lyrics (if available) into the 64-D multimodal vector, and persists the result.
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

		// Check if lyrics already exist for this song to compute multimodal fusion
		var topEmotions []database.LyricEmotion
		var lyricalVector []float32
		if lyr, lyrErr := db.GetLyrics(r.Context(), songID); lyrErr == nil && lyr != nil {
			topEmotions = lyr.TopEmotions
			lyricalVector = lyr.EmotionVector
		}

		// Compute continuous 64-D joint multimodal vector
		feat.MultimodalVector = fusion.ComputeMultimodalVector(feat.MoodVector, lyricalVector)

		// Derive nuanced mood and inferred genre
		primaryMood, allMoods := fusion.DeriveNuancedMood(feat.MatchedMoods, feat.Energy, feat.Brightness, topEmotions)
		feat.MatchedMoods = allMoods

		featID, err := db.UpsertFeatures(r.Context(), feat)
		if err != nil {
			slog.Error("failed to persist song features", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "failed to store audio features")
			return
		}

		// Infer genre and save to song_metadata
		inferredGenre := fusion.InferGenre(
			feat.TempoBPM, feat.Energy, feat.Brightness, feat.HarmonicRatio,
			feat.PercussiveRatio, feat.BeatImpact, feat.DistortionZCR, primaryMood, topEmotions,
		)

		meta, _, metaErr := db.GetMetadata(r.Context(), songID)
		if metaErr != nil || meta == nil {
			meta = &metadata.SongMetadata{
				Title:  song.OriginalName,
				Source: "inferred",
			}
		}
		meta.InferredGenre = inferredGenre
		_, _ = db.UpsertMetadata(r.Context(), songID, meta)

		slog.Info("audio features analyzed and stored",
			"song_id", songID,
			"feature_id", featID,
			"tempo_bpm", feat.TempoBPM,
			"primary_mood", primaryMood,
			"inferred_genre", inferredGenre,
			"multimodal_dims", len(feat.MultimodalVector),
		)

		respondJSON(w, http.StatusOK, map[string]any{
			"status":         "ok",
			"song_id":        songID,
			"inferred_genre": inferredGenre,
			"features":       feat,
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
