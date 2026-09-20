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

func testDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return "postgres://ever:first_commit@localhost:5432/moods?sslmode=disable"
	}
	return dsn
}

func TestSongFeaturesIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Connect(ctx, testDSN())
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
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected not found / no rows error, got: %v", err)
	}
}

func TestFindSimilarSongsIntegration(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test: database not reachable: %v", err)
	}
	defer db.Close()

	// 1. Test non-existent song ID returns pgx.ErrNoRows
	randomID := uuid.New()
	_, err = db.FindSimilarSongs(ctx, randomID, 0.70, 5, "acoustic")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows for unanalyzed song, got: %v", err)
	}

	// 2. Create 3 test songs: Target A, Similar B, Dissimilar C
	createTestSongWithFeatures := func(name string, vec []float32, moods []string) (*Song, *SongFeatures) {
		s := &Song{
			SessionID:    "sim-test-session",
			Filename:     name + ".mp3",
			OriginalName: name,
			Format:       "mp3",
			FilePath:     "/tmp/" + name + ".mp3",
			SizeBytes:    1024,
			Status:       "ready",
		}
		if err := db.CreateSong(ctx, s); err != nil {
			t.Fatalf("failed to create test song %s: %v", name, err)
		}
		feat := &SongFeatures{
			SongID: s.ID,
			AcousticFeatures: audio.AcousticFeatures{
				DurationSec: 150.0,
				TempoBPM:    120.0,
				Energy:      0.7,
				Brightness:  2000.0,
			},
			MatchedMoods: moods,
			VibeScores:   map[string]float64{"Energetic": 0.8},
			MoodVector:   vec,
		}
		if _, err := db.UpsertFeatures(ctx, feat); err != nil {
			t.Fatalf("failed to upsert features for %s: %v", name, err)
		}
		return s, feat
	}

	// Vectors of dimension 36:
	// Target A: [1, 0, 0, ...]
	vecA := make([]float32, 36)
	vecA[0] = 1.0

	// Similar B: [0.95, 0.31, 0, ...] (close to A in cosine space, sim ~ 0.95)
	vecB := make([]float32, 36)
	vecB[0] = 0.95
	vecB[1] = 0.31

	// Dissimilar C: [0, 1, 0, ...] (orthogonal to A, sim ~ 0.0)
	vecC := make([]float32, 36)
	vecC[1] = 1.0

	songA, _ := createTestSongWithFeatures("song_target_a", vecA, []string{"Club / Dance"})
	songB, _ := createTestSongWithFeatures("song_similar_b", vecB, []string{"Club / Dance"})
	songC, _ := createTestSongWithFeatures("song_dissimilar_c", vecC, []string{"Chill / Ambient"})

	defer func() {
		_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id IN ($1, $2, $3)", songA.ID, songB.ID, songC.ID)
	}()

	// Query with high threshold (0.80): only Song B should qualify (Song C is ~0.0 similarity)
	simsFiltered, err := db.FindSimilarSongs(ctx, songA.ID, 0.80, 10, "acoustic")
	if err != nil {
		t.Fatalf("FindSimilarSongs with threshold failed: %v", err)
	}
	if len(simsFiltered) != 1 {
		t.Fatalf("expected exactly 1 similar song above 0.80 threshold, got %d", len(simsFiltered))
	}
	if simsFiltered[0].ID != songB.ID {
		t.Errorf("expected Song B, got %s", simsFiltered[0].ID)
	}

	// Query with broad threshold (0.0): both Song B and Song C should qualify, ordered by similarity
	simsAll, err := db.FindSimilarSongs(ctx, songA.ID, 0.0, 10, "acoustic")
	if err != nil {
		t.Fatalf("FindSimilarSongs with broad threshold failed: %v", err)
	}
	if len(simsAll) < 2 {
		t.Fatalf("expected at least 2 similar songs with threshold 0.0, got %d", len(simsAll))
	}

	// Song B should be first (closest cosine distance to A)
	if simsAll[0].ID != songB.ID {
		t.Errorf("expected closest song to be B (%s), got %s", songB.ID, simsAll[0].ID)
	}
	if simsAll[0].SimilarityScore <= simsAll[1].SimilarityScore {
		t.Errorf("expected B similarity (%f) > C similarity (%f)", simsAll[0].SimilarityScore, simsAll[1].SimilarityScore)
	}
	if simsAll[0].Distance >= simsAll[1].Distance {
		t.Errorf("expected B distance (%f) < C distance (%f)", simsAll[0].Distance, simsAll[1].Distance)
	}
}

func TestDatabase_NotFoundWrapping(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
	}
	defer db.Close()

	randomID := uuid.New()

	// D1: GetSong must wrap pgx.ErrNoRows with %w
	_, err = db.GetSong(ctx, randomID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("GetSong for non-existent song did not wrap pgx.ErrNoRows: %v", err)
	}

	// D2: GetMetadata must wrap pgx.ErrNoRows with %w
	_, _, err = db.GetMetadata(ctx, randomID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("GetMetadata for non-existent song did not wrap pgx.ErrNoRows: %v", err)
	}

	// D3: UpdateSongFile must wrap pgx.ErrNoRows with %w
	err = db.UpdateSongFile(ctx, randomID, "new.mp3", "/tmp/new.mp3")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("UpdateSongFile for non-existent song did not wrap pgx.ErrNoRows: %v", err)
	}
}

