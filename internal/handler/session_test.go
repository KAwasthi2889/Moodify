package handler

import (
	"bytes"
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
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

func TestDeleteSong_InvalidUUID(t *testing.T) {
	t.Parallel()

	h := DeleteSong(nil, nil)
	r := chi.NewRouter()
	r.Delete("/api/v1/songs/{id}", h)

	req := httptest.NewRequest("DELETE", "/api/v1/songs/not-a-valid-uuid", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid UUID, got %d", rec.Code)
	}
}

func TestDeleteSong_NotFound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
	}
	defer db.Close()

	tempDir := t.TempDir()
	store, _ := storage.NewLocalStore(tempDir)

	h := DeleteSong(db, store)
	r := chi.NewRouter()
	r.Delete("/api/v1/songs/{id}", h)

	randomID := uuid.New().String()
	req := httptest.NewRequest("DELETE", "/api/v1/songs/"+randomID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for non-existent song, got %d", rec.Code)
	}
}

func TestDeleteSong_Integration(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
	}
	defer db.Close()

	tempDir := t.TempDir()
	store, err := storage.NewLocalStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}

	// 1. Create file and song
	savedPath, err := store.Save(ctx, "delete_target.mp3", bytes.NewReader([]byte("song payload")))
	if err != nil {
		t.Fatalf("failed to save file: %v", err)
	}

	song := &database.Song{
		SessionID:    "del-session-" + uuid.New().String(),
		Filename:     "delete_target.mp3",
		OriginalName: "Target",
		Format:       "mp3",
		FilePath:     savedPath,
		SizeBytes:    int64(len("song payload")),
		Status:       "ready",
	}
	if err := db.CreateSong(ctx, song); err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	// 2. Execute DELETE handler
	h := DeleteSong(db, store)
	r := chi.NewRouter()
	r.Delete("/api/v1/songs/{id}", h)

	req := httptest.NewRequest("DELETE", "/api/v1/songs/"+song.ID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}

	// 3. Verify file deleted from disk
	if _, err := os.Stat(savedPath); !os.IsNotExist(err) {
		t.Errorf("expected physical file %s to be deleted from disk", savedPath)
	}

	// 4. Verify DB record deleted
	if _, err := db.GetSong(ctx, song.ID); err == nil {
		t.Errorf("expected song to be deleted from database")
	}
}

func TestDeleteSession_Integration(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
	}
	defer db.Close()

	tempDir := t.TempDir()
	store, err := storage.NewLocalStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}

	sessionID := "bulk-session-" + uuid.New().String()

	p1, err := store.Save(ctx, "session1.mp3", bytes.NewReader([]byte("song1")))
	if err != nil {
		t.Fatalf("failed to save p1: %v", err)
	}
	p2, err := store.Save(ctx, "session2.mp3", bytes.NewReader([]byte("song2")))
	if err != nil {
		t.Fatalf("failed to save p2: %v", err)
	}

	s1 := &database.Song{
		SessionID:    sessionID,
		Filename:     "session1.mp3",
		OriginalName: "S1",
		Format:       "mp3",
		FilePath:     p1,
		SizeBytes:    int64(len("song1")),
		Status:       "ready",
	}
	s2 := &database.Song{
		SessionID:    sessionID,
		Filename:     "session2.mp3",
		OriginalName: "S2",
		Format:       "mp3",
		FilePath:     p2,
		SizeBytes:    int64(len("song2")),
		Status:       "ready",
	}
	if err := db.CreateSong(ctx, s1); err != nil {
		t.Fatalf("failed to create s1: %v", err)
	}
	if err := db.CreateSong(ctx, s2); err != nil {
		t.Fatalf("failed to create s2: %v", err)
	}

	// Execute DELETE session handler
	h := DeleteSession(db, store)
	r := chi.NewRouter()
	r.Delete("/api/v1/sessions/{id}", h)

	req := httptest.NewRequest("DELETE", "/api/v1/sessions/"+sessionID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if count, ok := resp["deleted_count"].(float64); !ok || int(count) != 2 {
		t.Errorf("expected deleted_count 2, got %v", resp["deleted_count"])
	}

	// Verify both files deleted from disk
	if _, err := os.Stat(p1); !os.IsNotExist(err) {
		t.Errorf("expected p1 %s to be deleted", p1)
	}
	if _, err := os.Stat(p2); !os.IsNotExist(err) {
		t.Errorf("expected p2 %s to be deleted", p2)
	}

	// Verify session songs empty in DB
	remaining, err := db.GetSongsBySession(ctx, sessionID)
	if err != nil {
		t.Fatalf("failed to check session songs: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 remaining songs in session, got %d", len(remaining))
	}
}
