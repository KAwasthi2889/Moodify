package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

func TestDownloadSong_InvalidID(t *testing.T) {
	t.Parallel()

	h := DownloadSong(nil)
	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/download", h)

	req := httptest.NewRequest("GET", "/api/v1/songs/not-a-valid-uuid/download", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestDownloadSong_NotFound(t *testing.T) {
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
	r.Get("/api/v1/songs/{id}/download", DownloadSong(db))

	randomID := uuid.New().String()
	req := httptest.NewRequest("GET", "/api/v1/songs/"+randomID+"/download", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for non-existent song, got %d", rec.Code)
	}
}

func TestDownloadSong_MissingFileOnDisk(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	song := &database.Song{
		SessionID:    "download-missing-file",
		Filename:     "ghost.mp3",
		OriginalName: "ghost.mp3",
		Format:       "mp3",
		FilePath:     "/tmp/ghost_file_that_does_not_exist_12345.mp3",
		SizeBytes:    1024,
		Status:       "ready",
	}
	if err := db.CreateSong(ctx, song); err != nil {
		t.Fatalf("CreateSong failed: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1", song.ID)
	}()

	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/download", DownloadSong(db))

	req := httptest.NewRequest("GET", "/api/v1/songs/"+song.ID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found when file missing on disk, got %d", rec.Code)
	}
}

func TestDownloadSong_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping test, postgres unavailable: %v", err)
		return
	}
	defer db.Close()

	// Write mock audio file to temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_audio.mp3")
	dummyContent := []byte("ID3v2_mock_audio_stream_data_1234567890")
	if err := os.WriteFile(filePath, dummyContent, 0o644); err != nil {
		t.Fatalf("failed to write dummy audio: %v", err)
	}

	song := &database.Song{
		SessionID:    "download-success-session",
		Filename:     "Test_Track.mp3",
		OriginalName: "Test_Track.mp3",
		Format:       "mp3",
		FilePath:     filePath,
		SizeBytes:    int64(len(dummyContent)),
		Status:       "ready",
	}
	if err := db.CreateSong(ctx, song); err != nil {
		t.Fatalf("CreateSong failed: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1", song.ID)
	}()

	r := chi.NewRouter()
	r.Get("/api/v1/songs/{id}/download", DownloadSong(db))

	// 1. Test standard full download
	req := httptest.NewRequest("GET", "/api/v1/songs/"+song.ID.String()+"/download", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	disposition := rec.Header().Get("Content-Disposition")
	expectedDisposition := `attachment; filename="Test_Track.mp3"`
	if disposition != expectedDisposition {
		t.Errorf("expected Content-Disposition %s, got %s", expectedDisposition, disposition)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "audio/mpeg" {
		t.Errorf("expected Content-Type audio/mpeg, got %s", contentType)
	}

	if rec.Body.String() != string(dummyContent) {
		t.Errorf("body content mismatch: got %q, want %q", rec.Body.String(), string(dummyContent))
	}

	// 2. Test HTTP 206 Partial Content (Range Request)
	reqRange := httptest.NewRequest("GET", "/api/v1/songs/"+song.ID.String()+"/download", nil)
	reqRange.Header.Set("Range", "bytes=0-4")
	recRange := httptest.NewRecorder()

	r.ServeHTTP(recRange, reqRange)

	if recRange.Code != http.StatusPartialContent {
		t.Fatalf("expected 206 Partial Content, got %d", recRange.Code)
	}

	expectedSlice := string(dummyContent[0:5])
	if recRange.Body.String() != expectedSlice {
		t.Errorf("range body mismatch: got %q, want %q", recRange.Body.String(), expectedSlice)
	}
}
