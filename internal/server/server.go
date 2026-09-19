package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/analyzer"
	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

// Server holds the HTTP server and its dependencies.
type Server struct {
	httpServer  *http.Server
	db          *database.DB
	store       storage.FileStore
	identifier  *metadata.Identifier
	embedder    *metadata.Embedder
	analyzer    *analyzer.Analyzer
	acoustIDKey string
}

// New creates a new Server with the given dependencies.
func New(port int, db *database.DB, store storage.FileStore, identifier *metadata.Identifier, embedder *metadata.Embedder, az *analyzer.Analyzer, acoustIDKey string) *Server {
	s := &Server{
		db:          db,
		store:       store,
		identifier:  identifier,
		embedder:    embedder,
		analyzer:    az,
		acoustIDKey: acoustIDKey,
	}

	router := s.routes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second, // Higher for file uploads.
		IdleTimeout:  120 * time.Second,
	}

	return s
}

// Start begins listening for HTTP requests. This blocks until the server shuts down.
func (s *Server) Start() error {
	slog.Info("starting server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server, waiting for in-flight requests to complete.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down server")
	return s.httpServer.Shutdown(ctx)
}
