package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/analyzer"
	"github.com/KAwasthi2889/Moodify/internal/cleanup"
	"github.com/KAwasthi2889/Moodify/internal/config"
	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/logging"
	"github.com/KAwasthi2889/Moodify/internal/lyrics"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
	"github.com/KAwasthi2889/Moodify/internal/server"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

func main() {
	// Initialize structured logging.
	logging.Setup()

	// Load configuration from environment.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL and run migrations.
	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.DSN())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize file storage.
	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		slog.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}

	// Initialize and start background session/file cleaner.
	cleanerCtx, cancelCleaner := context.WithCancel(context.Background())
	defer cancelCleaner()
	cleaner := cleanup.NewCleaner(db, store, cfg.SessionTTL, cfg.CleanupInterval)
	cleaner.Start(cleanerCtx)

	// Initialize sidecar runners.
	identifier := metadata.NewIdentifier(cfg.PythonBin, "./python/identify.py")
	embedder := metadata.NewEmbedder(cfg.PythonBin, "./python/embed_tags.py")
	az := analyzer.NewAnalyzer(cfg.PythonBin, "./python/analyze.py")
	lc := lyrics.NewClient(cfg.PythonBin, cfg.LyricsScript, cfg.GeminiAPIKey)

	// Create and start HTTP server.
	srv := server.New(cfg.ServerPort, db, store, identifier, embedder, az, lc, cfg.AcoustIDAPIKey)

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("received shutdown signal", "signal", sig)

		cancelCleaner()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	if err := srv.Start(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
