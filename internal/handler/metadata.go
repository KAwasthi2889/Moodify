package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
	"github.com/KAwasthi2889/Moodify/internal/storage"
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

		_ = db.UpdateSongStatus(r.Context(), songID, "tagged")

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

// EmbedTags handles POST /api/v1/songs/{id}/embed.
// It embeds the stored metadata into the physical audio file using mutagen.
func EmbedTags(db *database.DB, embedder *metadata.Embedder) http.HandlerFunc {
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

		meta, _, err := db.GetMetadata(r.Context(), songID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusBadRequest, "cannot embed tags: metadata has not been saved for this song yet")
				return
			}
			slog.Error("failed to query metadata", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		if err := embedder.EmbedTags(r.Context(), song.FilePath, meta); err != nil {
			slog.Error("failed to embed tags into audio file", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "failed to embed tags: "+err.Error())
			return
		}

		slog.Info("tags embedded successfully", "song_id", songID, "title", meta.Title)

		respondJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"song_id":       songID,
			"file_path":     song.FilePath,
			"embedded_tags": meta,
		})
	}
}

// RenameSong handles POST /api/v1/songs/{id}/rename.
// It renames the physical file using the Title-only convention:
// `Title.<ext>` or `(Original) | (English).<ext>` when transliteration exists.
func RenameSong(db *database.DB, store storage.FileStore) http.HandlerFunc {
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

		meta, _, err := db.GetMetadata(r.Context(), songID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusBadRequest, "cannot rename song: metadata has not been saved for this song yet")
				return
			}
			slog.Error("failed to query metadata", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		// Compute title-only base name
		var baseName string
		if meta.EnglishTitle != "" && !strings.EqualFold(meta.EnglishTitle, meta.Title) {
			baseName = fmt.Sprintf("(%s) | (%s)", meta.Title, meta.EnglishTitle)
		} else if meta.Title != "" {
			baseName = meta.Title
		} else {
			baseName = strings.TrimSuffix(song.OriginalName, filepath.Ext(song.OriginalName))
		}

		baseName = sanitizeFilename(baseName)
		ext := "." + song.Format
		targetFilename := fmt.Sprintf("%s%s", baseName, ext)

		// Rename file on storage with collision prevention
		newPath, finalFilename, err := store.Rename(r.Context(), song.FilePath, targetFilename)
		if err != nil {
			slog.Error("failed to rename file in storage", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "failed to rename file: "+err.Error())
			return
		}

		// Update database with new filename and path
		if err := db.UpdateSongFile(r.Context(), songID, finalFilename, newPath); err != nil {
			slog.Error("failed to update song path in database", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database update failed: "+err.Error())
			return
		}

		slog.Info("song file renamed successfully", "song_id", songID, "new_filename", finalFilename)

		respondJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"song_id":      songID,
			"new_filename": finalFilename,
			"new_path":     newPath,
		})
	}
}
