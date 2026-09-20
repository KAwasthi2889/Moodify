package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// LyricEmotion represents a single emotion classification and confidence probability.
type LyricEmotion struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

// SongLyrics represents the synchronized and plain lyrics, language, and 28-D emotion vector for a song.
type SongLyrics struct {
	ID             uuid.UUID      `json:"id"`
	SongID         uuid.UUID      `json:"song_id"`
	PlainLyrics    string         `json:"plain_lyrics"`
	SyncedLyrics   string         `json:"synced_lyrics"`
	IsSynced       bool           `json:"is_synced"`
	IsInstrumental bool           `json:"is_instrumental"`
	Language       string         `json:"language"`
	TopEmotions    []LyricEmotion `json:"top_emotions"`
	EmotionVector  []float32      `json:"emotion_vector,omitempty"`
	SyncedAt       time.Time      `json:"synced_at"`
}

// UpsertLyrics stores or updates synchronized/plain lyrics and the 28-D emotion vector for a song.
func (db *DB) UpsertLyrics(ctx context.Context, l *SongLyrics) error {
	topEmotionsJSON, err := json.Marshal(l.TopEmotions)
	if err != nil {
		return fmt.Errorf("marshal top emotions: %w", err)
	}

	vecStr := FormatVector(l.EmotionVector)
	if len(l.EmotionVector) == 0 {
		vecStr = FormatVector(make([]float32, 28))
	}

	return db.Pool.QueryRow(ctx, `
		INSERT INTO song_lyrics (
			song_id, plain_lyrics, synced_lyrics, is_synced,
			is_instrumental, language, top_emotions, emotion_vector, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (song_id) DO UPDATE SET
			plain_lyrics = EXCLUDED.plain_lyrics,
			synced_lyrics = EXCLUDED.synced_lyrics,
			is_synced = EXCLUDED.is_synced,
			is_instrumental = EXCLUDED.is_instrumental,
			language = EXCLUDED.language,
			top_emotions = EXCLUDED.top_emotions,
			emotion_vector = EXCLUDED.emotion_vector,
			synced_at = NOW()
		RETURNING id, synced_at
	`,
		l.SongID, l.PlainLyrics, l.SyncedLyrics, l.IsSynced,
		l.IsInstrumental, l.Language, topEmotionsJSON, vecStr,
	).Scan(&l.ID, &l.SyncedAt)
}

// GetLyrics retrieves lyrics and the 28-D emotion vector for a song by song_id.
func (db *DB) GetLyrics(ctx context.Context, songID uuid.UUID) (*SongLyrics, error) {
	var l SongLyrics
	l.SongID = songID

	var emotionsJSON []byte
	var vecStr string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, plain_lyrics, synced_lyrics, is_synced,
		       is_instrumental, language, top_emotions, emotion_vector::text, synced_at
		FROM song_lyrics
		WHERE song_id = $1
	`, songID).Scan(
		&l.ID, &l.PlainLyrics, &l.SyncedLyrics, &l.IsSynced,
		&l.IsInstrumental, &l.Language, &emotionsJSON, &vecStr, &l.SyncedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("query lyrics for %s: %w", songID, err)
	}

	if len(emotionsJSON) > 0 {
		_ = json.Unmarshal(emotionsJSON, &l.TopEmotions)
	}
	if l.TopEmotions == nil {
		l.TopEmotions = []LyricEmotion{}
	}

	l.EmotionVector, err = ParseVector(vecStr)
	if err != nil {
		return nil, fmt.Errorf("parse emotion vector: %w", err)
	}

	return &l, nil
}
