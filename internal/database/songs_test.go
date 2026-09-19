package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFormatAndParseVector(t *testing.T) {
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
		SongID:          song.ID,
		DurationSec:     180.5,
		TempoBPM:        128.0,
		Energy:          0.75,
		Brightness:      2200.0,
		HarmonicRatio:   0.65,
		PercussiveRatio: 0.35,
		BeatImpact:      1.2,
		DistortionZCR:   0.08,
		MatchedMoods:    []string{"Euphoric / Uplifting", "Irresistible Groove / Dance"},
		VibeScores: map[string]float64{
			"Euphoric / Uplifting":       0.85,
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
}
