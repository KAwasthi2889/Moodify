package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/audio"
)

func TestFormatAndParseVector(t *testing.T) {
	t.Parallel()

	t.Run("Standard 36-D roundtrip", func(t *testing.T) {
		t.Parallel()
		original := make([]float32, 36)
		for i := range original {
			original[i] = float32(i) * 0.025
		}

		formatted := FormatVector(original)
		parsed, err := ParseVector(formatted)
		if err != nil {
			t.Fatalf("ParseVector failed: %v", err)
		}

		if len(parsed) != len(original) {
			t.Fatalf("expected length %d, got %d", len(original), len(parsed))
		}

		for i := range original {
			diff := parsed[i] - original[i]
			if diff < -0.0001 || diff > 0.0001 {
				t.Errorf("dim %d mismatch: expected %f, got %f", i, original[i], parsed[i])
			}
		}
	})

	t.Run("Negative numbers and zeros", func(t *testing.T) {
		t.Parallel()
		input := []float32{-0.85, 0.0, 0.42, -1.0}
		formatted := FormatVector(input)
		parsed, err := ParseVector(formatted)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parsed) != 4 {
			t.Fatalf("expected 4 elements, got %d", len(parsed))
		}
		if parsed[0] < -0.8501 || parsed[0] > -0.8499 {
			t.Errorf("expected -0.85, got %f", parsed[0])
		}
	})

	t.Run("Empty vector string", func(t *testing.T) {
		t.Parallel()
		parsed, err := ParseVector("[]")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(parsed) != 0 {
			t.Errorf("expected empty slice, got %v", parsed)
		}
	})

	t.Run("Whitespace-padded vector", func(t *testing.T) {
		t.Parallel()
		parsed, err := ParseVector("[ 0.1 ,  0.25 , 0.5 ]")
		if err != nil {
			t.Fatalf("unexpected error on whitespace-padded vector: %v", err)
		}
		if len(parsed) != 3 || parsed[1] != 0.25 {
			t.Errorf("unexpected parsed result: %v", parsed)
		}
	})

	t.Run("Malformed non-numeric vector returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseVector("[0.1, corrupt_data, 0.5]")
		if err == nil {
			t.Fatal("expected error parsing non-numeric vector, got nil")
		}
	})
}

func TestSongFeaturesIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://ever:first_commit@localhost:5432/moods?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Connect(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	// 1. Create a dummy song record
	song := &Song{
		Filename:     "test_vector_song.mp3",
		OriginalName: "Test Song.mp3",
		Format:       "mp3",
		FilePath:     "/tmp/test_vector_song.mp3",
		SizeBytes:    1024,
		Status:       "uploaded",
	}
	if err := db.CreateSong(ctx, song); err != nil {
		t.Fatalf("CreateSong failed: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1", song.ID)
	}()

	// 2. Prepare 36-D feature vector
	vec := make([]float32, 36)
	for i := range vec {
		vec[i] = 0.15
	}

	feat := &SongFeatures{
		SongID: song.ID,
		AcousticFeatures: audio.AcousticFeatures{
			DurationSec:     180.5,
			TempoBPM:        128.0,
			Energy:          0.75,
			Brightness:      2200.0,
			HarmonicRatio:   0.65,
			PercussiveRatio: 0.35,
			BeatImpact:      1.2,
			DistortionZCR:   0.08,
		},
		MatchedMoods: []string{"Euphoric / Uplifting", "Irresistible Groove / Dance"},
		VibeScores: map[string]float64{
			"Euphoric / Uplifting":        0.85,
			"Irresistible Groove / Dance": 0.80,
		},
		MoodVector: vec,
	}

	// 3. Upsert features
	featID, err := db.UpsertFeatures(ctx, feat)
	if err != nil {
		t.Fatalf("UpsertFeatures failed: %v", err)
	}
	if featID == uuid.Nil {
		t.Fatal("expected non-nil featID")
	}

	// 4. Retrieve features
	retrieved, err := db.GetFeatures(ctx, song.ID)
	if err != nil {
		t.Fatalf("GetFeatures failed: %v", err)
	}

	if retrieved.TempoBPM != 128.0 {
		t.Errorf("expected tempo 128.0, got %f", retrieved.TempoBPM)
	}
	if len(retrieved.MoodVector) != 36 {
		t.Errorf("expected 36 dimensions, got %d", len(retrieved.MoodVector))
	}
	if len(retrieved.MatchedMoods) != 2 {
		t.Errorf("expected 2 matched moods, got %v", retrieved.MatchedMoods)
	}
	if retrieved.VibeScores["Euphoric / Uplifting"] != 0.85 {
		t.Errorf("expected vibe score 0.85, got %v", retrieved.VibeScores)
	}

	// 5. Test Upsert conflict (updating features for same song)
	feat.TempoBPM = 140.0
	feat.Energy = 0.90
	updatedFeatID, err := db.UpsertFeatures(ctx, feat)
	if err != nil {
		t.Fatalf("UpsertFeatures on conflict failed: %v", err)
	}
	if updatedFeatID != featID {
		t.Errorf("expected same feature ID %s, got %s", featID, updatedFeatID)
	}

	reRetrieved, err := db.GetFeatures(ctx, song.ID)
	if err != nil {
		t.Fatalf("GetFeatures after update failed: %v", err)
	}
	if reRetrieved.TempoBPM != 140.0 {
		t.Errorf("expected updated tempo 140.0, got %f", reRetrieved.TempoBPM)
	}

	// 6. Test Foreign Key Cascade: Deleting the song must cascade-delete song_features!
	_, err = db.Pool.Exec(ctx, "DELETE FROM songs WHERE id = $1", song.ID)
	if err != nil {
		t.Fatalf("delete song failed: %v", err)
	}

	_, err = db.GetFeatures(ctx, song.ID)
	if err == nil {
		t.Fatal("expected error after song deletion, got nil")
	}
	if !errors.Is(err, pgx.ErrNoRows) && !isNotFound(err) {
		t.Errorf("expected not found / no rows error, got: %v", err)
	}
}

func isNotFound(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) || errors.Unwrap(err) == pgx.ErrNoRows)
}
