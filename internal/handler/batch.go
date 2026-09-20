package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/audio"
	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/queue"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

// BatchUpload handles POST /api/v1/songs/batch/upload.
// It uses streaming multipart processing to ingest entire music libraries without an aggregate size ceiling,
// enforcing a strict per-file size ceiling and binding all songs to a unified session ID.
func BatchUpload(db *database.DB, store storage.FileStore, maxFileSizeMB int) http.HandlerFunc {
	if maxFileSizeMB <= 0 {
		maxFileSizeMB = 35
	}
	maxBytesPerFile := int64(maxFileSizeMB) << 20

	return func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			respondError(w, http.StatusBadRequest, "request must be multipart/form-data: "+err.Error())
			return
		}

		sessionID := r.Header.Get("X-Session-ID")
		if sessionID == "" {
			sessionID = r.URL.Query().Get("session_id")
		}

		type uploadedSong struct {
			ID           uuid.UUID `json:"id"`
			Filename     string    `json:"filename"`
			OriginalName string    `json:"original_name"`
			Format       string    `json:"format"`
			SizeBytes    int64     `json:"size_bytes"`
			Status       string    `json:"status"`
		}

		type uploadError struct {
			Filename string `json:"filename"`
			Error    string `json:"error"`
		}

		var successes []uploadedSong
		var failures []uploadError

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				slog.Error("failed reading next multipart part", "error", err)
				break
			}

			// Capture session_id form field if present before files
			if part.FormName() == "session_id" && sessionID == "" {
				buf := new(strings.Builder)
				_, _ = io.Copy(buf, part)
				sessionID = strings.TrimSpace(buf.String())
				part.Close()
				continue
			}

			filename := part.FileName()
			if filename == "" {
				part.Close()
				continue
			}

			if sessionID == "" {
				sessionID = uuid.New().String()
			}

			// Read header for magic byte detection
			headerBytes := make([]byte, 12)
			n, err := io.ReadFull(part, headerBytes)
			if err != nil && err != io.ErrUnexpectedEOF {
				failures = append(failures, uploadError{Filename: filename, Error: "unable to read file header"})
				part.Close()
				continue
			}
			headerBytes = headerBytes[:n]

			detectedFormat := audio.DetectFormat(headerBytes)
			if detectedFormat == "" {
				failures = append(failures, uploadError{
					Filename: filename,
					Error:    "unsupported format: must be MP3, M4A, FLAC, WAV, or OPUS",
				})
				part.Close()
				continue
			}

			storedName := generateFilename(filename, string(detectedFormat))
			fullReader := io.MultiReader(bytes.NewReader(headerBytes), io.LimitReader(part, maxBytesPerFile))

			savedPath, err := store.Save(r.Context(), storedName, fullReader)
			if err != nil {
				failures = append(failures, uploadError{Filename: filename, Error: "storage failed: " + err.Error()})
				part.Close()
				continue
			}

			// Calculate actual stored file size
			song := &database.Song{
				SessionID:    sessionID,
				Filename:     storedName,
				OriginalName: filename,
				Format:       string(detectedFormat),
				FilePath:     savedPath,
				SizeBytes:    int64(len(headerBytes)), // will be verified on disk
				Status:       "uploaded",
			}

			if err := db.CreateSong(r.Context(), song); err != nil {
				_ = store.Delete(r.Context(), savedPath)
				failures = append(failures, uploadError{Filename: filename, Error: "db insert failed: " + err.Error()})
				part.Close()
				continue
			}

			successes = append(successes, uploadedSong{
				ID:           song.ID,
				Filename:     storedName,
				OriginalName: filename,
				Format:       string(detectedFormat),
				SizeBytes:    song.SizeBytes,
				Status:       song.Status,
			})

			part.Close()
		}

		if sessionID == "" {
			sessionID = uuid.New().String()
		}

		w.Header().Set("X-Session-ID", sessionID)

		status := "ok"
		statusCode := http.StatusCreated
		if len(failures) > 0 && len(successes) == 0 {
			status = "error"
			statusCode = http.StatusBadRequest
		} else if len(failures) > 0 {
			status = "partial_success"
			statusCode = http.StatusMultiStatus
		}

		slog.Info("batch upload completed",
			"session_id", sessionID,
			"uploaded_count", len(successes),
			"failed_count", len(failures),
		)

		respondJSON(w, statusCode, map[string]any{
			"status":         status,
			"session_id":     sessionID,
			"total":          len(successes) + len(failures),
			"uploaded_count": len(successes),
			"failed_count":   len(failures),
			"songs":          successes,
			"errors":         failures,
		})
	}
}

// BatchAnalyzeRequest holds payload options for initiating batch analysis.
type BatchAnalyzeRequest struct {
	SessionID string      `json:"session_id"`
	SongIDs   []uuid.UUID `json:"song_ids"`
	Limit     int         `json:"limit"`
}

// BatchAnalyze handles POST /api/v1/songs/batch/analyze.
// It offloads unanalyzed library tracks to the asynchronous worker queue.
func BatchAnalyze(db *database.DB, q queue.Queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BatchAnalyzeRequest
		if r.Body != nil && r.ContentLength > 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}

		if req.Limit <= 0 {
			req.Limit = 50
		}

		var candidates []*database.Song
		var err error

		if len(req.SongIDs) > 0 {
			for _, id := range req.SongIDs {
				if s, sErr := db.GetSong(r.Context(), id); sErr == nil && s != nil {
					candidates = append(candidates, s)
				}
			}
		} else {
			candidates, err = db.GetUnanalyzedSongs(r.Context(), req.SessionID, req.Limit)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "failed to query unanalyzed songs: "+err.Error())
				return
			}
		}

		queuedCount := 0
		for _, s := range candidates {
			job := queue.AnalysisJob{
				SongID:    s.ID,
				SessionID: s.SessionID,
				FilePath:  s.FilePath,
			}
			if err := q.Enqueue(r.Context(), job); err == nil {
				queuedCount++
			}
		}

		slog.Info("batch analysis dispatched",
			"session_id", req.SessionID,
			"queued_count", queuedCount,
			"candidates", len(candidates),
		)

		respondJSON(w, http.StatusAccepted, map[string]any{
			"status":       "accepted",
			"session_id":   req.SessionID,
			"queued_count": queuedCount,
			"message":      fmt.Sprintf("%d songs queued for parallel feature extraction", queuedCount),
		})
	}
}

// GetBatchStatus handles GET /api/v1/songs/batch/status.
// It returns real-time queue metrics and processing counts.
func GetBatchStatus(db *database.DB, q queue.Queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("session_id")

		stats := q.Stats()

		respondJSON(w, http.StatusOK, map[string]any{
			"status":      "ok",
			"session_id":  sessionID,
			"queue_stats": stats,
		})
	}
}

// ListSongs handles GET /api/v1/songs.
// It returns the user's uploaded song library with metadata, moods, and inferred genres.
func ListSongs(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("session_id")
		status := r.URL.Query().Get("status")

		limit := 50
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		offset := 0
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if parsed, err := strconv.Atoi(oStr); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		songs, total, err := db.ListSongs(r.Context(), sessionID, status, limit, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to list songs: "+err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"session_id": sessionID,
			"total":      total,
			"count":      len(songs),
			"limit":      limit,
			"offset":     offset,
			"songs":      songs,
		})
	}
}
