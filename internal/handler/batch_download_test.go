package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

func TestBatchDownload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "moodify-batch-download-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "track1.mp3")
	if err := os.WriteFile(file1, []byte("audio-content-1"), 0o644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	file2 := filepath.Join(tmpDir, "track2.mp3")
	if err := os.WriteFile(file2, []byte("audio-content-2"), 0o644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping db integration test: %v", err)
	}
	defer db.Close()

	song1 := &database.Song{
		SessionID:    "batch-dl-session",
		Filename:     "track1.mp3",
		OriginalName: "Song One.mp3",
		Format:       "mp3",
		FilePath:     file1,
		SizeBytes:    15,
		Status:       "ready",
	}
	song2 := &database.Song{
		SessionID:    "batch-dl-session",
		Filename:     "track2.mp3",
		OriginalName: "Song Two.mp3",
		Format:       "mp3",
		FilePath:     file2,
		SizeBytes:    15,
		Status:       "ready",
	}

	if err := db.CreateSong(ctx, song1); err != nil {
		t.Fatalf("failed to create song1: %v", err)
	}
	if err := db.CreateSong(ctx, song2); err != nil {
		t.Fatalf("failed to create song2: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(ctx, "DELETE FROM songs WHERE session_id = 'batch-dl-session'")
	}()

	r := chi.NewRouter()
	r.Post("/api/v1/songs/batch/download", BatchDownload(db))

	reqBody, _ := json.Marshal(BatchDownloadRequest{
		SongIDs: []string{song1.ID.String(), song2.ID.String()},
	})
	req := httptest.NewRequest("POST", "/api/v1/songs/batch/download", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	if rec.Header().Get("Content-Type") != "application/zip" {
		t.Errorf("expected Content-Type application/zip, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Content-Disposition") != `attachment; filename="moodify_songs.zip"` {
		t.Errorf("expected moodify_songs.zip, got %s", rec.Header().Get("Content-Disposition"))
	}

	// Verify zip contents
	zipReader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("failed to open zip output: %v", err)
	}

	if len(zipReader.File) != 2 {
		t.Fatalf("expected 2 files in zip, got %d", len(zipReader.File))
	}

	foundNames := make(map[string]bool)
	for _, f := range zipReader.File {
		foundNames[f.Name] = true
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("failed to open zip file entry %s: %v", f.Name, err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		if len(data) == 0 {
			t.Errorf("expected non-empty content in %s", f.Name)
		}
	}

	if !foundNames["Song_One.mp3"] && !foundNames["Song One.mp3"] {
		t.Errorf("expected Song One.mp3 in zip, found: %v", foundNames)
	}
}
