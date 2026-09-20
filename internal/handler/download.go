package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

// DownloadSong handles GET /api/v1/songs/{id}/download.
// It serves the physical audio file with appropriate Content-Disposition and Range support.
func DownloadSong(db *database.DB) http.HandlerFunc {
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
			slog.Error("failed to query song for download", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		if _, err := os.Stat(song.FilePath); err != nil {
			if os.IsNotExist(err) {
				respondError(w, http.StatusNotFound, "audio file not found on storage")
				return
			}
			slog.Error("failed to stat audio file", "error", err, "path", song.FilePath)
			respondError(w, http.StatusInternalServerError, "failed to access audio file")
			return
		}

		filename := song.Filename
		if filename == "" {
			filename = filepath.Base(song.FilePath)
		}

		// Set inline disposition by default for streaming; attachment for download triggers
		disposition := "inline"
		if r.URL.Query().Get("download") == "true" {
			disposition = "attachment"
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filename))
		w.Header().Set("Accept-Ranges", "bytes")

		// Map explicit audio mime-type if known
		switch song.Format {
		case "mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
		case "m4a", "aac":
			w.Header().Set("Content-Type", "audio/mp4")
		case "flac":
			w.Header().Set("Content-Type", "audio/flac")
		case "wav":
			w.Header().Set("Content-Type", "audio/wav")
		case "opus", "ogg":
			w.Header().Set("Content-Type", "audio/ogg")
		}

		http.ServeFile(w, r, song.FilePath)
	}
}
