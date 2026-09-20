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

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/lyrics"
)

func TestSyncLyrics_InvalidID(t *testing.T) {
	t.Parallel()

	h := SyncLyrics(nil, nil)
	r := chi.NewRouter()
	r.Post("/api/v1/songs/{id}/lyrics/sync", h)

	req := httptest.NewRequest("POST", "/api/v1/songs/invalid-uuid/lyrics/sync", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestGetSongLyrics_InvalidID(t *testing.T) {
	t.Parallel()

	h := GetSongLyrics(nil)
	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/lyrics", h)

	req := httptest.NewRequest("GET", "/api/v1/songs/invalid-uuid/lyrics", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestGetSongLyrics_NotFound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	h := GetSongLyrics(db)
	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/lyrics", h)

	nonExistentID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/songs/"+nonExistentID.String()+"/lyrics", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", rec.Code)
	}
}

func TestSyncLyrics_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/lyrics.py"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found")
	}

	lc := lyrics.NewClient(pythonBin, scriptPath, "")

	// 1. Create a dummy song record
	song := &database.Song{
		Filename:     "test_lyrics_song.mp3",
		OriginalName: "Without You.mp3",
		Format:       "mp3",
		FilePath:     "test_lyrics_song.mp3",
		SizeBytes:    1024,
		Status:       "ready",
	}
	if err := db.CreateSong(ctx, song); err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	r := chi.NewRouter()
	r.Post("/api/v1/songs/{id}/lyrics/sync", SyncLyrics(db, lc))
	r.Get("/api/v1/songs/{id}/lyrics", GetSongLyrics(db))

	// 2. Sync lyrics with mock mode for deterministic offline testing
	syncReq := httptest.NewRequest("POST", "/api/v1/songs/"+song.ID.String()+"/lyrics/sync?mock=sad_hopeful", nil)
	syncRec := httptest.NewRecorder()
	r.ServeHTTP(syncRec, syncReq)

	if syncRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from sync, got %d: %s", syncRec.Code, syncRec.Body.String())
	}

	var syncResp struct {
		Status string                `json:"status"`
		Lyrics *database.SongLyrics `json:"lyrics"`
	}
	if err := json.Unmarshal(syncRec.Body.Bytes(), &syncResp); err != nil {
		t.Fatalf("failed to unmarshal sync response: %v", err)
	}

	if !syncResp.Lyrics.IsSynced {
		t.Errorf("expected IsSynced true, got %v", syncResp.Lyrics.IsSynced)
	}
	if syncResp.Lyrics.SyncedLyrics == "" {
		t.Error("expected non-empty SyncedLyrics")
	}
	if len(syncResp.Lyrics.EmotionVector) != 28 {
		t.Errorf("expected 28-D emotion vector, got %d", len(syncResp.Lyrics.EmotionVector))
	}

	// 3. Get lyrics via GET endpoint
	getReq := httptest.NewRequest("GET", "/api/v1/songs/"+song.ID.String()+"/lyrics", nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from GET lyrics, got %d", getRec.Code)
	}

	var getResp struct {
		Status string                `json:"status"`
		Lyrics *database.SongLyrics `json:"lyrics"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to unmarshal GET response: %v", err)
	}

	if getResp.Lyrics.PlainLyrics != syncResp.Lyrics.PlainLyrics {
		t.Errorf("expected matched plain lyrics, got %s vs %s", getResp.Lyrics.PlainLyrics, syncResp.Lyrics.PlainLyrics)
	}
}
