package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/analyzer"
	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

// IdentifyResponse is returned by POST /api/v1/songs/{id}/identify.
type IdentifyResponse struct {
	Status              string                 `json:"status"`
	SongID              string                 `json:"song_id"`
	FingerprintDuration float64                `json:"fingerprint_duration"`
	AutoSaved           bool                   `json:"auto_saved"`
	MoodifyEnabled      bool                   `json:"moodify_enabled,omitempty"`
	Matches             []metadata.Match       `json:"matches"`
	Features            *database.SongFeatures `json:"features,omitempty"`
	Warning             string                 `json:"warning,omitempty"`
	Note                string                 `json:"note,omitempty"`
}

// Identify handles POST /api/v1/songs/{id}/identify.
// It fingerprints the audio file, looks up metadata via AcoustID + MusicBrainz,
// and optionally persists the top match if ?auto_save=true is specified.
// If ?moodify=true is specified, it runs librosa feature extraction concurrently in parallel.
func Identify(db *database.DB, identifier *metadata.Identifier, az *analyzer.Analyzer, apiKey string) http.HandlerFunc {
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

		moodify := r.URL.Query().Get("moodify") == "true"
		autoSave := r.URL.Query().Get("auto_save") == "true"

		var (
			identResult *metadata.IdentifyResult
			identErr    error
			featResult  *database.SongFeatures
			featErr     error
			wg          sync.WaitGroup
		)

		// 1. Dispatch AcoustID identification
		wg.Add(1)
		go func() {
			defer wg.Done()
			identResult, identErr = identifier.Identify(r.Context(), song.FilePath, apiKey)
		}()

		// 2. Dispatch audio feature analysis concurrently in parallel if opted in
		if moodify && az != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				featResult, featErr = az.Analyze(r.Context(), song.FilePath)
			}()
		}

		wg.Wait()

		if identErr != nil {
			slog.Error("identification failed", "error", identErr, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "identification failed: "+identErr.Error())
			return
		}

		var autoSaved bool
		if autoSave && len(identResult.Matches) > 0 {
			top := identResult.Matches[0]
			if _, err := db.UpsertMetadata(r.Context(), songID, &top.SongMetadata); err != nil {
				slog.Error("failed to auto-save metadata", "error", err, "song_id", songID)
			} else {
				autoSaved = true
			}
		}

		// If moodify was run and succeeded, persist features to DB
		if moodify && featResult != nil {
			featResult.SongID = songID
			if _, err := db.UpsertFeatures(r.Context(), featResult); err != nil {
				slog.Error("failed to save concurrent features", "error", err, "song_id", songID)
			}
		} else if moodify && featErr != nil {
			slog.Warn("moodify feature extraction failed during identify", "error", featErr, "song_id", songID)
		}

		respondJSON(w, http.StatusOK, IdentifyResponse{
			Status:              identResult.Status,
			SongID:              songID.String(),
			FingerprintDuration: identResult.FingerprintDuration,
			AutoSaved:           autoSaved,
			MoodifyEnabled:      moodify,
			Matches:             identResult.Matches,
			Features:            featResult,
			Warning:             identResult.Warning,
			Note:                identResult.Note,
		})
	}
}
