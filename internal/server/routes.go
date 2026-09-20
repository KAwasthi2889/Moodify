package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/KAwasthi2889/Moodify/internal/handler"
)

// routes builds and returns the chi router with all middleware and route registrations.
func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	// ── Middleware ──────────────────────────────────
	r.Use(middleware.RequestID)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// ── Routes ─────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handler.Health(s.db))

		// Song endpoints
		r.Post("/songs/upload", handler.Upload(s.db, s.store))
		r.Post("/songs/{id}/identify", handler.Identify(s.db, s.identifier, s.analyzer, s.acoustIDKey))
		r.Put("/songs/{id}/metadata", handler.SaveMetadata(s.db))
		r.Get("/songs/{id}/metadata", handler.GetMetadata(s.db))
		r.Post("/songs/{id}/embed", handler.EmbedTags(s.db, s.embedder))
		r.Post("/songs/{id}/rename", handler.RenameSong(s.db, s.store))
		r.Post("/songs/{id}/analyze", handler.AnalyzeSong(s.db, s.analyzer))
		r.Get("/songs/{id}/features", handler.GetSongFeatures(s.db))
		r.Post("/songs/{id}/lyrics/sync", handler.SyncLyrics(s.db, s.lyricsClient))
		r.Get("/songs/{id}/lyrics", handler.GetSongLyrics(s.db))
		r.Get("/songs/clusters", handler.GetMoodClusters(s.db))
		r.Get("/songs/{id}/similar", handler.GetSimilarSongs(s.db))
		r.Get("/songs", handler.ListSongs(s.db))
		r.Post("/songs/batch/upload", handler.BatchUpload(s.db, s.store, s.maxFileSizeMB))
		r.Post("/songs/batch/analyze", handler.BatchAnalyze(s.db, s.queue))
		r.Get("/songs/batch/status", handler.GetBatchStatus(s.db, s.queue))
		r.Delete("/songs/{id}", handler.DeleteSong(s.db, s.store))

		// Session endpoints
		r.Delete("/sessions/{id}", handler.DeleteSession(s.db, s.store))

		// Curated Playlist endpoints
		r.Post("/playlists/generate", handler.GeneratePlaylist(s.db))
	})

	return r
}

// requestLogger is a middleware that logs each HTTP request using slog.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			slog.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}
