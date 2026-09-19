package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

// GetSimilarSongs handles GET /api/v1/songs/{id}/similar.
// It returns acoustic neighbors that exceed the similarity threshold,
// ordered in decreasing order of similarity.
func GetSimilarSongs(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		songID, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid song id format")
			return
		}

		threshold := 0.70
		threshParam := r.URL.Query().Get("threshold")
		if threshParam == "" {
			threshParam = r.URL.Query().Get("min_similarity")
		}
		if threshParam != "" {
			parsedThresh, err := strconv.ParseFloat(threshParam, 32)
			if err != nil || parsedThresh < 0.0 || parsedThresh > 1.0 {
				respondError(w, http.StatusBadRequest, "threshold must be a float between 0.0 and 1.0")
				return
			}
			threshold = parsedThresh
		}

		limit := 50
		limitParam := r.URL.Query().Get("limit")
		if limitParam != "" {
			parsedLimit, err := strconv.Atoi(limitParam)
			if err != nil || parsedLimit < 1 || parsedLimit > 200 {
				respondError(w, http.StatusBadRequest, "limit must be an integer between 1 and 200")
				return
			}
			limit = parsedLimit
		}

		sims, err := db.FindSimilarSongs(r.Context(), songID, float32(threshold), limit)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusNotFound, "features have not been analyzed for this song yet")
				return
			}
			slog.Error("failed to find similar songs", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"song_id":       songID,
			"threshold":     threshold,
			"count":         len(sims),
			"similar_songs": sims,
		})
	}
}

// GetMoodClusters handles GET /api/v1/songs/clusters.
// It groups the analyzed audio library into mood clusters.
func GetMoodClusters(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clusters, err := db.GetMoodClusters(r.Context())
		if err != nil {
			slog.Error("failed to get mood clusters", "error", err)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		totalSongs := 0
		for _, c := range clusters {
			totalSongs += c.Count
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"status":         "ok",
			"total_clusters": len(clusters),
			"total_songs":    totalSongs,
			"clusters":       clusters,
		})
	}
}
