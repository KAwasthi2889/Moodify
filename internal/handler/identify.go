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
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid song id format"})
			return
		}

		// Check if song exists and retrieve file path
		var filePath string
		err = db.Pool.QueryRow(r.Context(),
			"SELECT file_path FROM songs WHERE id = $1",
			songID,
		).Scan(&filePath)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "song not found"})
				return
			}
			slog.Error("failed to query song", "error", err, "song_id", songID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database query failed"})
			return
		}

		// Run identification sidecar
		result, err := identifier.Identify(r.Context(), filePath, apiKey)
		if err != nil {
			slog.Error("identification failed", "error", err, "song_id", songID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "identification failed: " + err.Error()})
			return
		}

		autoSave := r.URL.Query().Get("auto_save") == "true"
		var autoSaved bool

		if autoSave && len(result.Matches) > 0 {
			top := result.Matches[0]
			_, err = db.Pool.Exec(r.Context(), `
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
			`,
				songID, top.Source, top.Title, top.Artist, top.Album, top.AlbumArtist,
				top.ReleaseYear, top.Genre, top.TrackNumber, top.MusicbrainzID,
				top.AcoustidScore, top.EnglishTitle,
			)
			if err != nil {
				slog.Error("failed to auto-save metadata", "error", err, "song_id", songID)
			} else {
				autoSaved = true
			}
		}

		resp := IdentifyResponse{
			Status:              result.Status,
			SongID:              songID.String(),
			FingerprintDuration: result.FingerprintDuration,
			AutoSaved:           autoSaved,
			Matches:             result.Matches,
			Warning:             result.Warning,
			Note:                result.Note,
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
