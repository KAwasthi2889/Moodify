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
	"github.com/KAwasthi2889/Moodify/internal/queue"
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

	// Initialize file storage (S3 cloud provider with local fallback).
	localStore, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		slog.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	store := storage.NewS3Store(cfg.S3Bucket, cfg.AWSRegion, localStore)

	// Initialize sidecar runners.
	identifier := metadata.NewIdentifier(cfg.PythonBin, "./python/identify.py")
	embedder := metadata.NewEmbedder(cfg.PythonBin, "./python/embed_tags.py")
	az := analyzer.NewAnalyzer(cfg.PythonBin, "./python/analyze.py")
	lc := lyrics.NewClient(cfg.PythonBin, cfg.LyricsScript, cfg.GeminiAPIKey)

	// Initialize background worker pool and AWS SQS offloading queue.
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	workerPool := queue.NewWorkerPoolQueue(db, az, 2, 256)
	workerPool.Start(workerCtx)
	asyncQueue := queue.NewSQSQueue(cfg.SQSQueueURL, workerPool)

	// Initialize and start background session/file cleaner.
	cleanerCtx, cancelCleaner := context.WithCancel(context.Background())
	defer cancelCleaner()
	cleaner := cleanup.NewCleaner(db, store, cfg.SessionTTL, cfg.CleanupInterval)
	cleaner.Start(cleanerCtx)

	// Create and start HTTP server.
	srv := server.New(cfg.ServerPort, db, store, identifier, embedder, az, lc, cfg.AcoustIDAPIKey, asyncQueue, cfg.MaxFileSizeMB)

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("received shutdown signal", "signal", sig)

		cancelCleaner()
		cancelWorker()

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
