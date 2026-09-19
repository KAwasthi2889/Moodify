package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/analyzer"
	"github.com/KAwasthi2889/Moodify/internal/database"
)

func TestAnalyzeSong_InvalidID(t *testing.T) {
	t.Parallel()

	h := AnalyzeSong(nil, nil)
	r := chi.NewRouter()
	r.Post("/api/v1/songs/{id}/analyze", h)

	req := httptest.NewRequest("POST", "/api/v1/songs/not-a-valid-uuid/analyze", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestGetSongFeatures_InvalidID(t *testing.T) {
	t.Parallel()

	h := GetSongFeatures(nil)
	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/features", h)

	req := httptest.NewRequest("GET", "/api/v1/songs/invalid-uuid/features", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestAnalyzeSong_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/analyze.py"
	audioPath := "../../local/Without you.mp3"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found")
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skip("skipping test: audio file not found")
	}

	az := analyzer.NewAnalyzer(pythonBin, scriptPath)

	// 1. Create a dummy song pointing to the real test audio file
	song := &database.Song{
		Filename:     "without_you_test.mp3",
		OriginalName: "Without you.mp3",
		Format:       "m4a",
		FilePath:     audioPath,
		SizeBytes:    1024 * 1024,
		Status:       "uploaded",
	}
	if err := db.CreateSong(ctx, song); err != nil {
		t.Fatalf("failed to create song: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1", song.ID)
	}()

	r := chi.NewRouter()
	r.Post("/api/v1/songs/{id}/analyze", AnalyzeSong(db, az))
	r.Get("/api/v1/songs/{id}/features", GetSongFeatures(db))

	// 2. Before analysis, GET /features must return 404
	reqGet := httptest.NewRequest("GET", "/api/v1/songs/"+song.ID.String()+"/features", nil)
	recGet := httptest.NewRecorder()
	r.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found before analysis, got %d", recGet.Code)
	}

	// 3. POST /analyze must execute successfully
	reqPost := httptest.NewRequest("POST", "/api/v1/songs/"+song.ID.String()+"/analyze", nil)
	recPost := httptest.NewRecorder()
	r.ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from analyze, got %d (body: %s)", recPost.Code, recPost.Body.String())
	}

	var postResp map[string]any
	if err := json.Unmarshal(recPost.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("failed to decode analyze response: %v", err)
	}

	if postResp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", postResp["status"])
	}

	// 4. After analysis, GET /features must return 200 OK
	recGetAfter := httptest.NewRecorder()
	r.ServeHTTP(recGetAfter, reqGet)

	if recGetAfter.Code != http.StatusOK {
		t.Errorf("expected 200 OK after analysis, got %d", recGetAfter.Code)
	}
}

func TestAnalyzeSong_NonExistentSong(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	r := chi.NewRouter()
	r.Post("/api/v1/songs/{id}/analyze", AnalyzeSong(db, nil))

	nonExistentID := uuid.New().String()
	req := httptest.NewRequest("POST", "/api/v1/songs/"+nonExistentID+"/analyze", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", rec.Code)
	}
}
