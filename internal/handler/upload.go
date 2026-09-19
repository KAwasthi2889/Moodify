package handler

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/audio"
	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

const maxUploadSize = 50 << 20 // 50 MB

// Upload returns a handler that accepts multipart audio file uploads.
// It detects the true format via magic bytes, saves the file, and inserts a DB record.
func Upload(db *database.DB, store storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enforce max upload size.
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		file, header, err := r.FormFile("file")
		if err != nil {
			if err.Error() == "http: request body too large" {
				respondError(w, http.StatusRequestEntityTooLarge,
					fmt.Sprintf("file exceeds maximum upload size of %d MB", maxUploadSize>>20))
				return
			}
			respondError(w, http.StatusBadRequest, "missing or invalid 'file' field")
			return
		}
		defer file.Close()

		// Read the first 12 bytes for magic byte detection.
		headerBytes := make([]byte, 12)
		n, err := io.ReadFull(file, headerBytes)
		if err != nil && err != io.ErrUnexpectedEOF {
			respondError(w, http.StatusBadRequest, "unable to read file header")
			return
		}
		headerBytes = headerBytes[:n]

		// Detect true format from magic bytes.
		detectedFormat := audio.DetectFormat(headerBytes)
		if detectedFormat == "" {
			respondError(w, http.StatusBadRequest,
				"unsupported audio format: file must be MP3, M4A, FLAC, WAV, or OPUS")
			return
		}

		// Check extension vs detected format; log warning if mismatch.
		ext := strings.TrimPrefix(filepath.Ext(header.Filename), ".")
		extFormat := extensionToFormat(strings.ToLower(ext))
		if extFormat != "" && extFormat != detectedFormat {
			slog.Warn("file extension does not match detected format",
				"filename", header.Filename,
				"extension", ext,
				"detected_format", detectedFormat,
			)
		}

		// Reset reader by combining read header with remaining stream.
		fullReader := io.MultiReader(bytes.NewReader(headerBytes), file)

		// Generate storage filename using the detected format extension.
		storedName := generateFilename(header.Filename, string(detectedFormat))

		// Save file to storage.
		savedPath, err := store.Save(r.Context(), storedName, fullReader)
		if err != nil {
			slog.Error("failed to save file", "error", err, "filename", storedName)
			respondError(w, http.StatusInternalServerError, "failed to store file")
			return
		}

		// Insert song record in database.
		song := &database.Song{
			Filename:     storedName,
			OriginalName: header.Filename,
			Format:       string(detectedFormat),
			FilePath:     savedPath,
			SizeBytes:    header.Size,
			Status:       "uploaded",
		}
		if err := db.CreateSong(r.Context(), song); err != nil {
			slog.Error("failed to insert song record", "error", err)
			_ = store.Delete(r.Context(), storedName)
			respondError(w, http.StatusInternalServerError, "failed to record upload")
			return
		}

		slog.Info("file uploaded",
			"song_id", song.ID,
			"original_name", header.Filename,
			"detected_format", detectedFormat,
			"size_bytes", header.Size,
		)

		respondJSON(w, http.StatusCreated, map[string]any{
			"id":              song.ID,
			"filename":        storedName,
			"original_name":   header.Filename,
			"detected_format": detectedFormat,
			"size_bytes":      header.Size,
			"status":          "uploaded",
		})
	}
}

// extensionToFormat maps file extensions to audio.AudioFormat.
func extensionToFormat(ext string) audio.AudioFormat {
	switch ext {
	case "mp3":
		return audio.FormatMP3
	case "m4a", "aac", "mp4":
		return audio.FormatM4A
	case "flac":
		return audio.FormatFLAC
	case "wav":
		return audio.FormatWAV
	case "opus", "ogg":
		return audio.FormatOPUS
	default:
		return ""
	}
}

// generateFilename creates a unique storage filename with the correct extension.
func generateFilename(original string, format string) string {
	base := strings.TrimSuffix(original, filepath.Ext(original))
	base = sanitizeFilename(base)
	return fmt.Sprintf("%s_%d.%s", base, time.Now().UnixNano(), format)
}

// sanitizeFilename removes or replaces characters unsafe for filesystems.
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"\x00", "",
	)
	name = replacer.Replace(name)
	name = strings.Trim(name, ". ")

	if name == "" {
		name = "UNKNOWN"
	}
	return name
}
