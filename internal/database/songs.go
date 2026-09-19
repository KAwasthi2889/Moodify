package database

import (
	"context"
	"fmt"
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
