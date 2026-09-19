package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/audio"
	"github.com/KAwasthi2889/Moodify/internal/database"
)

func TestGetSimilarSongs_InvalidID(t *testing.T) {
	t.Parallel()

	h := GetSimilarSongs(nil)
	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/similar", h)

	req := httptest.NewRequest("GET", "/api/v1/songs/not-a-valid-uuid/similar", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestGetSimilarSongs_InvalidLimit(t *testing.T) {
	t.Parallel()

	h := GetSimilarSongs(nil)
	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/similar", h)

	cases := []string{
		"/api/v1/songs/" + uuid.New().String() + "/similar?limit=abc",
		"/api/v1/songs/" + uuid.New().String() + "/similar?limit=0",
		"/api/v1/songs/" + uuid.New().String() + "/similar?limit=-5",
		"/api/v1/songs/" + uuid.New().String() + "/similar?limit=300",
	}

	for _, url := range cases {
		req := httptest.NewRequest("GET", url, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("url %s: expected 400 Bad Request, got %d", url, rec.Code)
		}
	}
}

func TestGetSimilarSongs_InvalidThreshold(t *testing.T) {
	t.Parallel()

	h := GetSimilarSongs(nil)
	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/similar", h)

	cases := []string{
		"/api/v1/songs/" + uuid.New().String() + "/similar?threshold=abc",
		"/api/v1/songs/" + uuid.New().String() + "/similar?threshold=-0.1",
		"/api/v1/songs/" + uuid.New().String() + "/similar?threshold=1.5",
	}

	for _, url := range cases {
		req := httptest.NewRequest("GET", url, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("url %s: expected 400 Bad Request, got %d", url, rec.Code)
		}
	}
}

func TestGetSimilarSongs_NotFound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/similar", GetSimilarSongs(db))

	randomID := uuid.New().String()
	req := httptest.NewRequest("GET", "/api/v1/songs/"+randomID+"/similar", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for unanalyzed song, got %d", rec.Code)
	}
}

func TestGetSimilarSongs_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	// Create test target and peer song
	createSongWithVector := func(name string, vec []float32) *database.Song {
		s := &database.Song{
			SessionID:    "similar-handler-test",
			Filename:     name + ".mp3",
			OriginalName: name,
			Format:       "mp3",
			FilePath:     "/tmp/" + name + ".mp3",
			SizeBytes:    2048,
			Status:       "ready",
		}
		if err := db.CreateSong(ctx, s); err != nil {
			t.Fatalf("CreateSong failed: %v", err)
		}
		feat := &database.SongFeatures{
			SongID: s.ID,
			AcousticFeatures: audio.AcousticFeatures{
				DurationSec: 180.0,
				TempoBPM:    128.0,
				Energy:      0.8,
			},
			MatchedMoods: []string{"High Energy"},
			MoodVector:   vec,
		}
		if _, err := db.UpsertFeatures(ctx, feat); err != nil {
			t.Fatalf("UpsertFeatures failed: %v", err)
		}
		return s
	}

	vec1 := make([]float32, 36)
	vec1[0] = 0.8
	vec1[1] = 0.6

	vec2 := make([]float32, 36)
	vec2[0] = 0.7
	vec2[1] = 0.7

	song1 := createSongWithVector("sim_handler_1", vec1)
	song2 := createSongWithVector("sim_handler_2", vec2)

	defer func() {
		_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id IN ($1, $2)", song1.ID, song2.ID)
	}()

	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/similar", GetSimilarSongs(db))

	req := httptest.NewRequest("GET", "/api/v1/songs/"+song1.ID.String()+"/similar?limit=5", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status       string                 `json:"status"`
		SongID       uuid.UUID              `json:"song_id"`
		Threshold    float64                `json:"threshold"`
		Count        int                    `json:"count"`
		SimilarSongs []database.SimilarSong `json:"similar_songs"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %s", resp.Status)
	}
	if resp.SongID != song1.ID {
		t.Errorf("expected song_id %s, got %s", song1.ID, resp.SongID)
	}
	if resp.Threshold != 0.70 {
		t.Errorf("expected default threshold 0.70, got %f", resp.Threshold)
	}
	if resp.Count < 1 {
		t.Errorf("expected at least 1 similar song, got %d", resp.Count)
	}
	if len(resp.SimilarSongs) == 0 || resp.SimilarSongs[0].ID != song2.ID {
		t.Errorf("expected song2 (%s) in similar songs", song2.ID)
	}

	// Request with strict threshold (0.995): should return 0 similar songs dynamically
	reqStrict := httptest.NewRequest("GET", "/api/v1/songs/"+song1.ID.String()+"/similar?threshold=0.995", nil)
	recStrict := httptest.NewRecorder()
	r.ServeHTTP(recStrict, reqStrict)

	if recStrict.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from strict threshold query, got %d", recStrict.Code)
	}
	var respStrict struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	if err := json.Unmarshal(recStrict.Body.Bytes(), &respStrict); err != nil {
		t.Fatalf("failed to decode strict response: %v", err)
	}
	if respStrict.Count != 0 {
		t.Errorf("expected 0 songs for strict 0.995 threshold, got %d", respStrict.Count)
	}
}

func TestGetMoodClusters_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/songs/clusters", GetMoodClusters(db))

	req := httptest.NewRequest("GET", "/api/v1/songs/clusters", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status        string                 `json:"status"`
		TotalClusters int                    `json:"total_clusters"`
		TotalSongs    int                    `json:"total_songs"`
		Clusters      []database.MoodCluster `json:"clusters"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode clusters response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %s", resp.Status)
	}
	if resp.Clusters == nil {
		t.Errorf("expected non-nil clusters list")
	}
}
