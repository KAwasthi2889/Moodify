package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

const maxUploadSize = 50 << 20 // 50 MB

// AudioFormat represents a detected audio file format.
type AudioFormat string

const (
	FormatMP3  AudioFormat = "mp3"
	FormatM4A  AudioFormat = "m4a"
	FormatFLAC AudioFormat = "flac"
	FormatWAV  AudioFormat = "wav"
	FormatOPUS AudioFormat = "opus"
)

// Upload returns a handler that accepts multipart audio file uploads.
// It detects the true format via magic bytes, saves the file, and inserts a DB record.
func Upload(db *database.DB, store storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enforce max upload size.
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		file, header, err := r.FormFile("file")
		if err != nil {
			if err.Error() == "http: request body too large" {
				writeError(w, http.StatusRequestEntityTooLarge,
					fmt.Sprintf("file exceeds maximum upload size of %d MB", maxUploadSize>>20))
				return
			}
			writeError(w, http.StatusBadRequest, "missing or invalid 'file' field")
			return
		}
		defer file.Close()

		// Read the first 12 bytes for magic byte detection.
		headerBytes := make([]byte, 12)
		n, err := io.ReadFull(file, headerBytes)
		if err != nil && err != io.ErrUnexpectedEOF {
			writeError(w, http.StatusBadRequest, "unable to read file header")
			return
		}
		headerBytes = headerBytes[:n]

		detectedFormat := detectFormat(headerBytes)
		if detectedFormat == "" {
			writeError(w, http.StatusUnsupportedMediaType,
				"unsupported audio format; accepted: mp3, m4a, flac, wav, opus")
			return
		}

		// Check for extension mismatch.
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(header.Filename), "."))
		extFormat := extensionToFormat(ext)
		if extFormat != "" && extFormat != detectedFormat {
			slog.Warn("file extension does not match detected format",
				"filename", header.Filename,
				"extension", ext,
				"detected_format", detectedFormat,
			)
		}

		// Reconstruct a reader with the peeked bytes prepended.
		fullReader := io.MultiReader(bytes.NewReader(headerBytes), file)

		// Generate a unique filename preserving the correct extension.
		storedName := generateFilename(header.Filename, string(detectedFormat))

		// Save to storage.
		filePath, err := store.Save(r.Context(), storedName, fullReader)
		if err != nil {
			slog.Error("failed to save file", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to save file")
			return
		}

		// Insert record into DB.
		songID, err := insertSong(r.Context(), db.Pool,
			storedName, header.Filename, string(detectedFormat), filePath, header.Size)
		if err != nil {
			slog.Error("failed to insert song record", "error", err)
			// Attempt to clean up the saved file.
			_ = store.Delete(r.Context(), filePath)
			writeError(w, http.StatusInternalServerError, "failed to record upload")
			return
		}

		slog.Info("file uploaded",
			"song_id", songID,
			"original_name", header.Filename,
			"detected_format", detectedFormat,
			"size_bytes", header.Size,
		)

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":              songID,
			"filename":        storedName,
			"original_name":   header.Filename,
			"detected_format": detectedFormat,
			"size_bytes":      header.Size,
			"status":          "uploaded",
		})
	}
}

// detectFormat identifies the audio format from the first bytes of the file.
func detectFormat(header []byte) AudioFormat {
	if len(header) < 3 {
		return ""
	}

	// FLAC: starts with "fLaC"
	if len(header) >= 4 && string(header[:4]) == "fLaC" {
		return FormatFLAC
	}

	// WAV: starts with "RIFF" and has "WAVE" at offset 8
	if len(header) >= 12 && string(header[:4]) == "RIFF" && string(header[8:12]) == "WAVE" {
		return FormatWAV
	}

	// OGG/Opus: starts with "OggS"
	if len(header) >= 4 && string(header[:4]) == "OggS" {
		return FormatOPUS
	}

	// M4A/MP4: has "ftyp" at offset 4
	if len(header) >= 8 && string(header[4:8]) == "ftyp" {
		return FormatM4A
	}

	// MP3: ID3v2 tag header
	if string(header[:3]) == "ID3" {
		return FormatMP3
	}

	// MP3: MPEG sync word (0xFF followed by 0xE0 mask)
	if header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
		return FormatMP3
	}

	return ""
}

// extensionToFormat maps file extensions to AudioFormat.
func extensionToFormat(ext string) AudioFormat {
	switch ext {
	case "mp3":
		return FormatMP3
	case "m4a", "aac", "mp4":
		return FormatM4A
	case "flac":
		return FormatFLAC
	case "wav":
		return FormatWAV
	case "opus", "ogg":
		return FormatOPUS
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

func insertSong(ctx context.Context, pool *pgxpool.Pool,
	filename, originalName, format, filePath string, fileSize int64,
) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO songs (filename, original_name, format, file_path, file_size, status)
		VALUES ($1, $2, $3, $4, $5, 'uploaded')
		RETURNING id
	`, filename, originalName, format, filePath, fileSize).Scan(&id)
	return id, err
}
