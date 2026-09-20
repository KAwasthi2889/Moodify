package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

// PlaylistRequest holds options for generating a curated playlist.
type PlaylistRequest struct {
	SessionID  string     `json:"session_id"`
	SeedSongID *uuid.UUID `json:"seed_song_id"`
	Mood       string     `json:"mood"`
	Genre      string     `json:"genre"`
	Title      string     `json:"title"`
	Limit      int        `json:"limit"`
	Format     string     `json:"format"` // "json" (default) or "m3u8"
}

// PlaylistTrack represents an item in the generated playlist.
type PlaylistTrack struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	Artist        string    `json:"artist"`
	Album         string    `json:"album"`
	InferredGenre string    `json:"inferred_genre"`
	MatchedMoods  []string  `json:"matched_moods"`
	DurationSec   float32   `json:"duration_sec"`
	TempoBPM      float32   `json:"tempo_bpm"`
	DownloadURL   string    `json:"download_url"`
}

// GeneratePlaylist handles POST /api/v1/playlists/generate.
// It curates playlists based on seed song multimodal similarity, mood, or genre with smooth BPM sequencing.
func GeneratePlaylist(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PlaylistRequest
		if r.Body != nil && r.ContentLength > 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}

		if req.Limit <= 0 || req.Limit > 50 {
			req.Limit = 15
		}

		if req.Title == "" {
			if req.Mood != "" {
				req.Title = fmt.Sprintf("Moodify — %s Vibe", req.Mood)
			} else if req.Genre != "" {
				req.Title = fmt.Sprintf("Moodify — %s Mix", req.Genre)
			} else {
				req.Title = "Moodify Curated Playlist"
			}
		}

		var tracks []PlaylistTrack

		// 1. Seed Song Mode: 64-D multimodal cosine similarity
		if req.SeedSongID != nil && *req.SeedSongID != uuid.Nil {
			seedSong, err := db.GetSong(r.Context(), *req.SeedSongID)
			if err == nil && seedSong != nil {
				seedMeta, _, _ := db.GetMetadata(r.Context(), *req.SeedSongID)
				seedFeat, _ := db.GetFeatures(r.Context(), *req.SeedSongID)

				title := seedSong.OriginalName
				artist := "Unknown"
				album := ""
				genre := ""
				var moods []string
				var duration, tempo float32

				if seedMeta != nil {
					if seedMeta.Title != "" {
						title = seedMeta.Title
					}
					if seedMeta.Artist != "" {
						artist = seedMeta.Artist
					}
					album = seedMeta.Album
					genre = seedMeta.InferredGenre
				}
				if seedFeat != nil {
					moods = seedFeat.MatchedMoods
					duration = seedFeat.DurationSec
					tempo = seedFeat.TempoBPM
				}

				tracks = append(tracks, PlaylistTrack{
					ID:            seedSong.ID,
					Title:         title,
					Artist:        artist,
					Album:         album,
					InferredGenre: genre,
					MatchedMoods:  moods,
					DurationSec:   duration,
					TempoBPM:      tempo,
					DownloadURL:   fmt.Sprintf("/api/v1/songs/%s/download", seedSong.ID),
				})
			}

			// Fetch nearest neighbor tracks by multimodal 64-D vector
			sims, err := db.FindSimilarSongs(r.Context(), *req.SeedSongID, 0.0, req.Limit, "multimodal")
			if err == nil {
				for _, sim := range sims {
					if sim.ID == *req.SeedSongID {
						continue
					}
					tracks = append(tracks, PlaylistTrack{
						ID:            sim.ID,
						Title:         sim.Title,
						Artist:        sim.Artist,
						Album:         sim.Album,
						InferredGenre: sim.InferredGenre,
						MatchedMoods:  sim.MatchedMoods,
						DurationSec:   sim.DurationSec,
						TempoBPM:      sim.TempoBPM,
						DownloadURL:   fmt.Sprintf("/api/v1/songs/%s/download", sim.ID),
					})
				}
			}
		} else {
			// 2. Library Mood & Genre Filter Mode
			songs, err := db.GetSongsForPlaylist(r.Context(), req.SessionID, req.Mood, req.Genre, req.Limit)
			if err == nil {
				for _, s := range songs {
					title := s.OriginalName
					if s.Title != "" {
						title = s.Title
					}
					artist := "Unknown"
					if s.Artist != "" {
						artist = s.Artist
					}

					tracks = append(tracks, PlaylistTrack{
						ID:            s.ID,
						Title:         title,
						Artist:        artist,
						Album:         s.Album,
						InferredGenre: s.InferredGenre,
						MatchedMoods:  s.MatchedMoods,
						DurationSec:   s.DurationSec,
						TempoBPM:      s.TempoBPM,
						DownloadURL:   fmt.Sprintf("/api/v1/songs/%s/download", s.ID),
					})
				}
			}
		}

		// Smooth BPM sequencing (unless seed song is leading)
		if len(tracks) > 2 && (req.SeedSongID == nil || *req.SeedSongID == uuid.Nil) {
			sort.Slice(tracks, func(i, j int) bool {
				return tracks[i].TempoBPM < tracks[j].TempoBPM
			})
		}

		// Format output: M3U8 vs JSON
		if strings.EqualFold(req.Format, "m3u8") || r.URL.Query().Get("format") == "m3u8" {
			var m3u strings.Builder
			m3u.WriteString("#EXTM3U\n")
			m3u.WriteString(fmt.Sprintf("#PLAYLIST:%s\n", req.Title))

			baseURL := "http://" + r.Host
			for _, t := range tracks {
				m3u.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", int(t.DurationSec), t.Artist, t.Title))
				m3u.WriteString(fmt.Sprintf("%s%s\n", baseURL, t.DownloadURL))
			}

			w.Header().Set("Content-Type", "audio/x-mpegurl")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", sanitizeFilename(req.Title)+".m3u8"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(m3u.String()))
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"status":      "ok",
			"title":       req.Title,
			"track_count": len(tracks),
			"tracks":      tracks,
		})
	}
}
