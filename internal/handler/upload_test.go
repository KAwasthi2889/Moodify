package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal clean string", "normal_song", "normal_song"},
		{"Forward slash traversal", "../../etc/passwd", "etc_passwd"},
		{"Backslash traversal", "..\\..\\Windows\\System32", "Windows_System32"},
		{"Embedded null bytes", "song\x00_title", "song_title"},
		{"Leading and trailing dots", "...leading_dots...", "leading_dots"},
		{"Spaces and dots", "  .my_track.  ", "my_track"},
		{"Empty string defaults to UNKNOWN", "", "UNKNOWN"},
		{"Only slashes and dots", "/.../\\", "UNKNOWN"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func testDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return "postgres://ever:first_commit@localhost:5432/moods?sslmode=disable"
	}
	return dsn
}

func TestUploadHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping upload handler test, database unavailable: %v", err)
		return
	}
	defer db.Close()

	tempDir := t.TempDir()
	store, err := storage.NewLocalStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}

	uploadHandler := Upload(db, store)

	t.Run("Missing file field returns 400 Bad Request", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/songs/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		uploadHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Unsupported binary returns 400 Bad Request", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "image.png")
		if err != nil {
			t.Fatalf("create form file failed: %v", err)
		}
		// Write PNG header bytes
		_, _ = part.Write([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR"))
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/songs/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		uploadHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("M4A audio with .mp3 extension gets detected and corrected", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "Without_you.mp3")
		if err != nil {
			t.Fatalf("create form file failed: %v", err)
		}
		// Write M4A magic bytes
		_, _ = part.Write([]byte("\x00\x00\x00\x20ftypM4A \x00\x00\x00\x00"))
		// Write some dummy audio payload
		_, _ = part.Write(bytes.Repeat([]byte{0xAA}, 1024))
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/songs/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		uploadHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		if resp["detected_format"] != "m4a" {
			t.Errorf("expected detected_format 'm4a', got %v", resp["detected_format"])
		}
		if resp["extension_corrected"] != true {
			t.Errorf("expected extension_corrected true, got %v", resp["extension_corrected"])
		}
		if resp["corrected_filename"] != "Without_you.m4a" {
			t.Errorf("expected corrected_filename 'Without_you.m4a', got %v", resp["corrected_filename"])
		}
		if resp["format_warning"] == nil {
			t.Errorf("expected format_warning in response, got nil")
		}

		// Cleanup created song record and file
		if songIDStr, ok := resp["id"].(string); ok {
			_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1", songIDStr)
		}
		if filename, ok := resp["filename"].(string); ok {
			_ = os.Remove(filepath.Join(tempDir, filename))
		}
	})

	t.Run("Valid FLAC upload succeeds without extension correction", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "acoustic.flac")
		if err != nil {
			t.Fatalf("create form file failed: %v", err)
		}
		// Write FLAC header
		_, _ = part.Write([]byte("fLaC\x00\x00\x00\x22"))
		_, _ = part.Write(bytes.Repeat([]byte{0xBB}, 512))
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/songs/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		uploadHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", rec.Code)
		}

		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["detected_format"] != "flac" {
			t.Errorf("expected 'flac', got %v", resp["detected_format"])
		}
		if resp["extension_corrected"] != false {
			t.Errorf("expected extension_corrected false, got %v", resp["extension_corrected"])
		}
		if resp["format_warning"] != nil {
			t.Errorf("expected nil format_warning, got %v", resp["format_warning"])
		}

		// Cleanup
		if songIDStr, ok := resp["id"].(string); ok {
			_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1", songIDStr)
		}
		if filename, ok := resp["filename"].(string); ok {
			_ = os.Remove(filepath.Join(tempDir, filename))
		}
	})

	t.Run("Upload with explicit X-Session-ID", func(t *testing.T) {
		customSession := "client-session-12345"
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "track.flac")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		_, _ = part.Write([]byte("fLaC\x00\x00\x00\x22"))
		_, _ = part.Write(bytes.Repeat([]byte{0xCC}, 256))
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/songs/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-Session-ID", customSession)
		rec := httptest.NewRecorder()

		uploadHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", rec.Code)
		}

		if gotHeader := rec.Header().Get("X-Session-ID"); gotHeader != customSession {
			t.Errorf("expected X-Session-ID response header %q, got %q", customSession, gotHeader)
		}

		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["session_id"] != customSession {
			t.Errorf("expected session_id %q in response, got %v", customSession, resp["session_id"])
		}

		// Cleanup
		if songIDStr, ok := resp["id"].(string); ok {
			_, _ = db.Pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1", songIDStr)
		}
		if filename, ok := resp["filename"].(string); ok {
			_ = os.Remove(filepath.Join(tempDir, filename))
		}
	})
}

