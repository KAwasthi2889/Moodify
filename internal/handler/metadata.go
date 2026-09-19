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
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid song id format"})
			return
		}

		// Ensure the song exists
		var exists bool
		err = db.Pool.QueryRow(r.Context(),
			"SELECT EXISTS(SELECT 1 FROM songs WHERE id = $1)",
			songID,
		).Scan(&exists)
		if err != nil {
			slog.Error("failed to check song existence", "error", err, "song_id", songID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database query failed"})
			return
		}
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "song not found"})
			return
		}

		var meta metadata.SongMetadata
		if err := json.NewDecoder(r.Body).Decode(&meta); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload: " + err.Error()})
			return
		}

		if meta.Source == "" {
			meta.Source = "user"
		}

		var metaID uuid.UUID
		err = db.Pool.QueryRow(r.Context(), `
			INSERT INTO song_metadata (
				song_id, source, title, artist, album, album_artist,
				release_year, genre, track_number, musicbrainz_id,
				acoustid_score, english_title
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (song_id) DO UPDATE SET
				source = EXCLUDED.source,
				title = EXCLUDED.title,
				artist = EXCLUDED.artist,
				album = EXCLUDED.album,
				album_artist = EXCLUDED.album_artist,
				release_year = EXCLUDED.release_year,
				genre = EXCLUDED.genre,
				track_number = EXCLUDED.track_number,
				musicbrainz_id = EXCLUDED.musicbrainz_id,
				acoustid_score = EXCLUDED.acoustid_score,
				english_title = EXCLUDED.english_title
			RETURNING id
		`,
			songID, meta.Source, meta.Title, meta.Artist, meta.Album, meta.AlbumArtist,
			meta.ReleaseYear, meta.Genre, meta.TrackNumber, meta.MusicbrainzID,
			meta.AcoustidScore, meta.EnglishTitle,
		).Scan(&metaID)

		if err != nil {
			slog.Error("failed to upsert metadata", "error", err, "song_id", songID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save metadata"})
			return
		}

		writeJSON(w, http.StatusOK, MetadataResponse{
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
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid song id format"})
			return
		}

		var resp MetadataResponse
		var metaUUID, songUUID uuid.UUID
		err = db.Pool.QueryRow(r.Context(), `
			SELECT id, song_id, source, title, artist, album, album_artist,
			       release_year, genre, track_number, musicbrainz_id,
			       acoustid_score, english_title
			FROM song_metadata
			WHERE song_id = $1
		`, songID).Scan(
			&metaUUID, &songUUID, &resp.Source, &resp.Title, &resp.Artist, &resp.Album, &resp.AlbumArtist,
			&resp.ReleaseYear, &resp.Genre, &resp.TrackNumber, &resp.MusicbrainzID,
			&resp.AcoustidScore, &resp.EnglishTitle,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "metadata not found for this song"})
				return
			}
			slog.Error("failed to query song metadata", "error", err, "song_id", songID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database query failed"})
			return
		}

		resp.ID = metaUUID.String()
		resp.SongID = songUUID.String()
		writeJSON(w, http.StatusOK, resp)
	}
}
