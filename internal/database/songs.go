package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

// Song represents a song record in the database.
type Song struct {
	ID           uuid.UUID
	SessionID    string
	Filename     string
	OriginalName string
	Format       string
	FilePath     string
	SizeBytes    int64
	UploadedAt   time.Time
	Status       string
}

// CreateSong inserts a new song record into the database.
func (db *DB) CreateSong(ctx context.Context, s *Song) error {
	return db.Pool.QueryRow(ctx, `
		INSERT INTO songs (session_id, filename, original_name, format, file_path, file_size, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, uploaded_at
	`,
		s.SessionID, s.Filename, s.OriginalName, s.Format, s.FilePath, s.SizeBytes, s.Status,
	).Scan(&s.ID, &s.UploadedAt)
}

// GetSong retrieves a song record by its UUID.
func (db *DB) GetSong(ctx context.Context, id uuid.UUID) (*Song, error) {
	var s Song
	s.ID = id
	err := db.Pool.QueryRow(ctx, `
		SELECT session_id, filename, original_name, format, file_path, file_size, uploaded_at, status
		FROM songs
		WHERE id = $1
	`, id).Scan(
		&s.SessionID, &s.Filename, &s.OriginalName, &s.Format, &s.FilePath,
		&s.SizeBytes, &s.UploadedAt, &s.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("query song %s: %w", id, err)
	}
	return &s, nil
}

// UpsertMetadata inserts or updates the metadata for a given song.
func (db *DB) UpsertMetadata(ctx context.Context, songID uuid.UUID, meta *metadata.SongMetadata) (uuid.UUID, error) {
	if meta.Source == "" {
		meta.Source = "user"
	}

	var metaID uuid.UUID
	err := db.Pool.QueryRow(ctx, `
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
		return uuid.Nil, fmt.Errorf("upsert metadata for song %s: %w", songID, err)
	}
	return metaID, nil
}

// GetMetadata retrieves the stored metadata for a given song.
func (db *DB) GetMetadata(ctx context.Context, songID uuid.UUID) (*metadata.SongMetadata, uuid.UUID, error) {
	var m metadata.SongMetadata
	var metaID, sid uuid.UUID

	err := db.Pool.QueryRow(ctx, `
		SELECT id, song_id, source, title, artist, album, album_artist,
		       release_year, genre, track_number, musicbrainz_id,
		       acoustid_score, english_title
		FROM song_metadata
		WHERE song_id = $1
	`, songID).Scan(
		&metaID, &sid, &m.Source, &m.Title, &m.Artist, &m.Album, &m.AlbumArtist,
		&m.ReleaseYear, &m.Genre, &m.TrackNumber, &m.MusicbrainzID,
		&m.AcoustidScore, &m.EnglishTitle,
	)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("query metadata for song %s: %w", songID, err)
	}
	return &m, metaID, nil
}

// UpdateSongFile updates the stored filename and path for a song.
func (db *DB) UpdateSongFile(ctx context.Context, id uuid.UUID, filename, filePath string) error {
	res, err := db.Pool.Exec(ctx, `
		UPDATE songs SET filename = $1, file_path = $2 WHERE id = $3
	`, filename, filePath, id)
	if err != nil {
		return fmt.Errorf("update song file %s: %w", id, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("song %s not found", id)
	}
	return nil
}

// SongFeatures represents the 36-D acoustic feature extraction result stored in the database.
type SongFeatures struct {
	ID              uuid.UUID          `json:"id"`
	SongID          uuid.UUID          `json:"song_id"`
	DurationSec     float32            `json:"duration_sec"`
	TempoBPM        float32            `json:"tempo_bpm"`
	Energy          float32            `json:"energy"`
	Brightness      float32            `json:"brightness"`
	HarmonicRatio   float32            `json:"harmonic_ratio"`
	PercussiveRatio float32            `json:"percussive_ratio"`
	BeatImpact      float32            `json:"beat_impact"`
	DistortionZCR   float32            `json:"distortion_zcr"`
	MatchedMoods    []string           `json:"matched_moods"`
	VibeScores      map[string]float64 `json:"vibe_scores"`
	MoodVector      []float32          `json:"mood_vector"`
	AnalyzedAt      time.Time          `json:"analyzed_at"`
}

// UpsertFeatures inserts or updates the acoustic feature record for a song.
func (db *DB) UpsertFeatures(ctx context.Context, f *SongFeatures) (uuid.UUID, error) {
	vibeJSON, err := json.Marshal(f.VibeScores)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal vibe scores: %w", err)
	}

	vecStr := FormatVector(f.MoodVector)

	var featID uuid.UUID
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO song_features (
			song_id, duration_sec, tempo_bpm, energy, brightness,
			harmonic_ratio, percussive_ratio, beat_impact, distortion_zcr,
			matched_moods, vibe_scores, mood_vector
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::vector)
		ON CONFLICT (song_id) DO UPDATE SET
			duration_sec = EXCLUDED.duration_sec,
			tempo_bpm = EXCLUDED.tempo_bpm,
			energy = EXCLUDED.energy,
			brightness = EXCLUDED.brightness,
			harmonic_ratio = EXCLUDED.harmonic_ratio,
			percussive_ratio = EXCLUDED.percussive_ratio,
			beat_impact = EXCLUDED.beat_impact,
			distortion_zcr = EXCLUDED.distortion_zcr,
			matched_moods = EXCLUDED.matched_moods,
			vibe_scores = EXCLUDED.vibe_scores,
			mood_vector = EXCLUDED.mood_vector,
			analyzed_at = NOW()
		RETURNING id, analyzed_at
	`,
		f.SongID, f.DurationSec, f.TempoBPM, f.Energy, f.Brightness,
		f.HarmonicRatio, f.PercussiveRatio, f.BeatImpact, f.DistortionZCR,
		f.MatchedMoods, vibeJSON, vecStr,
	).Scan(&featID, &f.AnalyzedAt)

	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert features for song %s: %w", f.SongID, err)
	}
	f.ID = featID
	return featID, nil
}

// GetFeatures retrieves the feature analysis for a given song.
func (db *DB) GetFeatures(ctx context.Context, songID uuid.UUID) (*SongFeatures, error) {
	var f SongFeatures
	var vibeJSON []byte
	var vecStr string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, song_id, duration_sec, tempo_bpm, energy, brightness,
		       harmonic_ratio, percussive_ratio, beat_impact, distortion_zcr,
		       matched_moods, vibe_scores, mood_vector::text, analyzed_at
		FROM song_features
		WHERE song_id = $1
	`, songID).Scan(
		&f.ID, &f.SongID, &f.DurationSec, &f.TempoBPM, &f.Energy, &f.Brightness,
		&f.HarmonicRatio, &f.PercussiveRatio, &f.BeatImpact, &f.DistortionZCR,
		&f.MatchedMoods, &vibeJSON, &vecStr, &f.AnalyzedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query features for song %s: %w", songID, err)
	}

	if len(vibeJSON) > 0 {
		_ = json.Unmarshal(vibeJSON, &f.VibeScores)
	}

	f.MoodVector, err = ParseVector(vecStr)
	if err != nil {
		return nil, fmt.Errorf("parse mood vector: %w", err)
	}

	return &f, nil
}

// FormatVector converts a float32 slice into a PostgreSQL vector literal string: "[0.1,0.2,...]".
func FormatVector(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, val := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatFloat(float64(val), 'f', 5, 32))
	}
	sb.WriteByte(']')
	return sb.String()
}

// ParseVector converts a PostgreSQL vector literal string "[0.1,0.2,...]" into a []float32.
func ParseVector(s string) ([]float32, error) {
	s = strings.Trim(s, "[] ")
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	res := make([]float32, len(parts))
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, err
		}
		res[i] = float32(f)
	}
	return res, nil
}
