package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/queue"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

func TestBatchUpload_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
	}
	defer db.Close()

	tempDir := t.TempDir()
	store, _ := storage.NewLocalStore(tempDir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Session field
	sessionID := "batch-session-" + uuid.New().String()
	_ = writer.WriteField("session_id", sessionID)

	// File 1: FLAC
	f1, _ := writer.CreateFormFile("files", "song1.flac")
	_, _ = f1.Write([]byte("fLaC\x00\x00\x00\x22"))
	_, _ = f1.Write(bytes.Repeat([]byte{0x01}, 256))

	// File 2: MP3
	f2, _ := writer.CreateFormFile("files", "song2.mp3")
	_, _ = f2.Write([]byte("ID3\x03\x00\x00\x00\x00\x00\x10"))
	_, _ = f2.Write(bytes.Repeat([]byte{0x02}, 256))

	_ = writer.Close()

	h := BatchUpload(db, store, 35)
	req := httptest.NewRequest("POST", "/api/v1/songs/batch/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["uploaded_count"] != float64(2) {
		t.Errorf("expected uploaded_count 2, got %v", resp["uploaded_count"])
	}

	// Verify header propagated
	if rec.Header().Get("X-Session-ID") != sessionID {
		t.Errorf("expected X-Session-ID %q, got %q", sessionID, rec.Header().Get("X-Session-ID"))
	}

	// Cleanup songs
	songs, err := db.GetSongsBySession(ctx, sessionID)
	if err == nil {
		for _, s := range songs {
			_, _ = db.DeleteSong(ctx, s.ID)
			_ = os.Remove(s.FilePath)
		}
	}
}

func TestGeneratePlaylist_M3U8(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
	}
	defer db.Close()

	h := GeneratePlaylist(db)
	payload := `{"title": "Chill Afternoon", "format": "m3u8", "limit": 5}`
	req := httptest.NewRequest("POST", "/api/v1/playlists/generate", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "mpegurl") {
		t.Errorf("expected audio/x-mpegurl content type, got %s", contentType)
	}

	bodyStr := rec.Body.String()
	if !strings.HasPrefix(bodyStr, "#EXTM3U") {
		t.Errorf("expected #EXTM3U playlist prefix, got: %s", bodyStr)
	}
}

func TestBatchAnalyze_Enqueue(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
	}
	defer db.Close()

	workerPool := queue.NewWorkerPoolQueue(db, nil, 2, 50)
	sqsQueue := queue.NewSQSQueue("", workerPool)

	h := BatchAnalyze(db, sqsQueue)
	req := httptest.NewRequest("POST", "/api/v1/songs/batch/analyze", strings.NewReader(`{"limit": 5}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", rec.Code, rec.Body.String())
	}
}
