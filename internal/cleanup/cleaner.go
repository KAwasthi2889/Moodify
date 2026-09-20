package cleanup

import (
	"context"
	"log/slog"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

// Cleaner periodically scans for and removes expired song files and database records.
type Cleaner struct {
	db       *database.DB
	store    storage.FileStore
	ttl      time.Duration
	interval time.Duration
}

// NewCleaner creates a new Cleaner instance with configured TTL and interval.
func NewCleaner(db *database.DB, store storage.FileStore, ttl, interval time.Duration) *Cleaner {
	return &Cleaner{
		db:       db,
		store:    store,
		ttl:      ttl,
		interval: interval,
	}
}

// Start begins the periodic cleanup loop in a background goroutine.
// It stops when ctx is cancelled.
func (c *Cleaner) Start(ctx context.Context) {
	slog.Info("starting session cleanup worker",
		"ttl", c.ttl,
		"interval", c.interval,
	)

	ticker := time.NewTicker(c.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count, err := c.PurgeExpired(ctx)
				if err != nil {
					slog.Error("session cleanup purge failed", "error", err)
				} else if count > 0 {
					slog.Info("session cleanup purge completed", "deleted_songs", count)
				}
			case <-ctx.Done():
				slog.Info("session cleanup worker stopped")
				return
			}
		}
	}()
}

// PurgeExpired deletes all songs and associated files older than c.ttl.
func (c *Cleaner) PurgeExpired(ctx context.Context) (int, error) {
	if c.db == nil {
		return 0, nil
	}
	cutoff := time.Now().Add(-c.ttl)
	deleted, err := c.db.DeleteExpiredSongs(ctx, cutoff)
	if err != nil {
		return 0, err
	}

	for _, s := range deleted {
		if err := c.store.Delete(ctx, s.FilePath); err != nil {
			slog.Warn("failed to delete expired file from storage",
				"song_id", s.ID,
				"file_path", s.FilePath,
				"error", err,
			)
		}
	}

	return len(deleted), nil
}
