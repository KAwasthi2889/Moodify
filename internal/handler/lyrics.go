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
	"github.com/KAwasthi2889/Moodify/internal/lyrics"
)

type syncLyricsRequest struct {
	Track  string `json:"track,omitempty"`
	Artist string `json:"artist,omitempty"`
	Album  string `json:"album,omitempty"`
	Mock   string `json:"mock,omitempty"`
}

// SyncLyrics handles POST /api/v1/songs/{id}/lyrics/sync.
// It retrieves synchronized/plain lyrics and the 28-D emotion vector from LRCLIB and RoBERTa,
// and persists the result to the song_lyrics table.
func SyncLyrics(db *database.DB, lc *lyrics.Client) http.HandlerFunc {
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

		var req syncLyricsRequest
		if r.Header.Get("Content-Type") == "application/json" && r.ContentLength > 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}

		// Allow query parameter overrides as well
		if mockParam := r.URL.Query().Get("mock"); mockParam != "" {
			req.Mock = mockParam
		}
		if trackParam := r.URL.Query().Get("track"); trackParam != "" {
			req.Track = trackParam
		}
		if artistParam := r.URL.Query().Get("artist"); artistParam != "" {
			req.Artist = artistParam
		}

		var songLyric *database.SongLyrics
		if req.Mock != "" {
			songLyric, err = lc.FetchLyricsMock(r.Context(), req.Mock)
		} else {
			track := req.Track
			artist := req.Artist
			album := req.Album

			// If track or artist not specified in request, retrieve from stored metadata
			if track == "" || artist == "" {
				meta, _, metaErr := db.GetMetadata(r.Context(), songID)
				if metaErr == nil && meta != nil {
					if track == "" {
						track = meta.Title
					}
					if artist == "" {
						artist = meta.Artist
					}
					if album == "" {
						album = meta.Album
					}
				}
			}

			if track == "" {
				track = song.OriginalName
			}

			songLyric, err = lc.FetchLyrics(r.Context(), track, artist, album, 0)
		}

		if err != nil {
			slog.Error("failed to fetch/analyze lyrics", "error", err, "song_id", songID)
			respondError(w, http.StatusBadGateway, "lyrics fetch failed: "+err.Error())
			return
		}

		songLyric.SongID = song.ID
		if err := db.UpsertLyrics(r.Context(), songLyric); err != nil {
			slog.Error("failed to persist song lyrics", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "failed to store lyrics")
			return
		}

		slog.Info("song lyrics synchronized and stored",
			"song_id", songID,
			"is_synced", songLyric.IsSynced,
			"language", songLyric.Language,
			"emotions_count", len(songLyric.TopEmotions),
		)

		respondJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"song_id": songID,
			"lyrics":  songLyric,
		})
	}
}

// GetSongLyrics handles GET /api/v1/songs/{id}/lyrics.
// It retrieves stored synchronized lyrics, plain text, and emotional sentiment for a song.
func GetSongLyrics(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		songID, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid song id format")
			return
		}

		l, err := db.GetLyrics(r.Context(), songID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondError(w, http.StatusNotFound, "lyrics have not been synchronized for this song yet")
				return
			}
			slog.Error("failed to query song lyrics", "error", err, "song_id", songID)
			respondError(w, http.StatusInternalServerError, "database query failed")
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"song_id": songID,
			"lyrics":  l,
		})
	}
}
