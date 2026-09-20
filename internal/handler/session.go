package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

// DeleteSong handles DELETE /api/v1/songs/{id}.
// It atomically removes the song record from PostgreSQL (cascading metadata, features, lyrics)
// and deletes the stored audio file from disk.
func DeleteSong(db *database.DB, store storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		songID, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid song id: must be a valid UUID")
			return
		}

		deleted, err := db.DeleteSong(r.Context(), songID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusNotFound, "song not found")
				return
			}
			slog.Error("failed to delete song from database", "song_id", songID, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to delete song")
			return
		}

		if err := store.Delete(r.Context(), deleted.FilePath); err != nil {
			slog.Warn("failed to delete physical song file", "song_id", songID, "file_path", deleted.FilePath, "error", err)
		}

		slog.Info("song deleted",
			"song_id", songID,
			"session_id", deleted.SessionID,
			"filename", deleted.Filename,
		)

		respondJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"deleted_id": songID,
			"filename":   deleted.Filename,
		})
	}
}

// DeleteSession handles DELETE /api/v1/sessions/{id}.
// It deletes all songs registered under the session ID and their associated files on disk.
func DeleteSession(db *database.DB, store storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "id")
		if sessionID == "" {
			respondError(w, http.StatusBadRequest, "session id cannot be empty")
			return
		}

		deletedSongs, err := db.DeleteSession(r.Context(), sessionID)
		if err != nil {
			slog.Error("failed to delete session songs from database", "session_id", sessionID, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to delete session")
			return
		}

		for _, s := range deletedSongs {
			if err := store.Delete(r.Context(), s.FilePath); err != nil {
				slog.Warn("failed to delete physical session file",
					"session_id", sessionID,
					"song_id", s.ID,
					"file_path", s.FilePath,
					"error", err,
				)
			}
		}

		slog.Info("session deleted",
			"session_id", sessionID,
			"deleted_count", len(deletedSongs),
		)

		respondJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"session_id":    sessionID,
			"deleted_count": len(deletedSongs),
		})
	}
}
