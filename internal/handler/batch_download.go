package handler

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

// BatchDownloadRequest holds the song IDs to be packaged and downloaded.
type BatchDownloadRequest struct {
	SongIDs []string `json:"song_ids"`
}

// BatchDownload handles POST /api/v1/songs/batch/download and GET /api/v1/songs/batch/download?ids=...
// It packages the requested songs into a zip archive named moodify_songs.zip and streams it to the client.
func BatchDownload(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var songIDs []string

		if r.Method == http.MethodPost {
			var req BatchDownloadRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
				return
			}
			songIDs = req.SongIDs
		} else {
			idsParam := r.URL.Query().Get("ids")
			if idsParam != "" {
				songIDs = strings.Split(idsParam, ",")
			}
		}

		if len(songIDs) == 0 {
			respondError(w, http.StatusBadRequest, "no song_ids provided")
			return
		}

		// Enforce safety ceiling of 700 files per batch
		if len(songIDs) > 700 {
			respondError(w, http.StatusBadRequest, "maximum 700 files per batch download")
			return
		}

		// Set headers for streaming zip archive
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="moodify_songs.zip"`)
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		zw := zip.NewWriter(w)
		defer zw.Close()

		usedFilenames := make(map[string]int)

		for _, idStr := range songIDs {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}

			parsedID, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}

			song, err := db.GetSong(r.Context(), parsedID)
			if err != nil || song == nil {
				continue
			}

			// Stat file to make sure it's readable
			fi, err := os.Stat(song.FilePath)
			if err != nil || fi.IsDir() {
				continue
			}

			file, err := os.Open(song.FilePath)
			if err != nil {
				continue
			}

			// Query metadata for clean filename
			meta, _, _ := db.GetMetadata(r.Context(), song.ID)
			cleanFilename := ResolveDownloadFilename(song, meta)

			// Handle duplicate filenames in zip
			count := usedFilenames[cleanFilename]
			usedFilenames[cleanFilename] = count + 1
			if count > 0 {
				ext := filepath.Ext(cleanFilename)
				base := strings.TrimSuffix(cleanFilename, ext)
				cleanFilename = fmt.Sprintf("%s (%d)%s", base, count, ext)
			}

			// Create zip entry header with file modification time
			header, err := zip.FileInfoHeader(fi)
			if err != nil {
				header = &zip.FileHeader{Name: cleanFilename}
			}
			header.Name = cleanFilename
			header.Method = zip.Deflate

			entryWriter, err := zw.CreateHeader(header)
			if err != nil {
				file.Close()
				slog.Error("failed to create zip entry", "filename", cleanFilename, "error", err)
				continue
			}

			_, copyErr := io.Copy(entryWriter, file)
			file.Close()
			if copyErr != nil {
				slog.Error("failed to write song to zip", "filename", cleanFilename, "error", copyErr)
				return
			}
		}
	}
}

// cleanTitlePart removes outer whitespace and enclosing parentheses from title parts.
func cleanTitlePart(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") && len(s) >= 2 {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

// hasNonLatin checks if the string contains Cyrillic, CJK, Hangul, or Arabic characters.
func hasNonLatin(s string) bool {
	for _, r := range s {
		if (r >= 0x0400 && r <= 0x04FF) ||
			(r >= 0x4E00 && r <= 0x9FFF) ||
			(r >= 0x3040 && r <= 0x30FF) ||
			(r >= 0xAC00 && r <= 0xD7AF) ||
			(r >= 0x0600 && r <= 0x06FF) {
			return true
		}
	}
	return false
}

// ResolveDownloadFilename returns a clean, human-readable audio filename
// matching the [title | english.ext] or [Title.ext] naming convention.
func ResolveDownloadFilename(song *database.Song, meta *metadata.SongMetadata) string {
	format := song.Format
	if format == "" {
		format = strings.TrimPrefix(filepath.Ext(song.FilePath), ".")
	}
	if format == "" {
		format = "mp3"
	}

	var baseName string
	if meta != nil {
		title := cleanTitlePart(meta.Title)
		eng := cleanTitlePart(meta.EnglishTitle)
		if eng != "" && title != "" && !strings.EqualFold(eng, title) {
			baseName = fmt.Sprintf("%s | %s", title, eng)
		} else if title != "" {
			if strings.Contains(title, "|") {
				parts := strings.SplitN(title, "|", 2)
				t0 := cleanTitlePart(parts[0])
				t1 := cleanTitlePart(parts[1])
				if t0 != "" && t1 != "" && !strings.EqualFold(t0, t1) {
					baseName = fmt.Sprintf("%s | %s", t0, t1)
				} else {
					baseName = title
				}
			} else if hasNonLatin(title) && song.OriginalName != "" {
				cleanOrig := strings.TrimSuffix(song.OriginalName, filepath.Ext(song.OriginalName))
				cleanOrig = regexp.MustCompile(`_\d{10,}$`).ReplaceAllString(cleanOrig, "")
				cleanOrig = strings.ReplaceAll(cleanOrig, "_", " ")
				cleanOrig = cleanTitlePart(cleanOrig)
				if cleanOrig != "" && !hasNonLatin(cleanOrig) && !strings.EqualFold(cleanOrig, title) {
					baseName = fmt.Sprintf("%s | %s", title, cleanOrig)
				} else {
					baseName = title
				}
			} else {
				baseName = title
			}
		}
	}

	if baseName == "" && song.OriginalName != "" {
		raw := strings.TrimSuffix(song.OriginalName, filepath.Ext(song.OriginalName))
		if strings.Contains(raw, "|") {
			parts := strings.SplitN(raw, "|", 2)
			t0 := cleanTitlePart(parts[0])
			t1 := cleanTitlePart(parts[1])
			if t0 != "" && t1 != "" && !strings.EqualFold(t0, t1) {
				baseName = fmt.Sprintf("%s | %s", t0, t1)
			} else {
				baseName = raw
			}
		} else {
			baseName = raw
		}
	}

	if baseName == "" && song.Filename != "" {
		raw := strings.TrimSuffix(song.Filename, filepath.Ext(song.Filename))
		// Strip server timestamp suffixes if present (e.g. _1726859384938)
		re := regexp.MustCompile(`_\d{10,}$`)
		raw = re.ReplaceAllString(raw, "")
		if strings.Contains(raw, "|") {
			parts := strings.SplitN(raw, "|", 2)
			t0 := cleanTitlePart(parts[0])
			t1 := cleanTitlePart(parts[1])
			if t0 != "" && t1 != "" && !strings.EqualFold(t0, t1) {
				baseName = fmt.Sprintf("%s | %s", t0, t1)
			} else {
				baseName = raw
			}
		} else {
			baseName = raw
		}
	}

	if baseName == "" {
		baseName = "track"
	}

	baseName = sanitizeFilename(baseName)
	return fmt.Sprintf("%s.%s", baseName, format)
}
