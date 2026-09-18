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
