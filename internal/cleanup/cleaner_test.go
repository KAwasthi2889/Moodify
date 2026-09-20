package cleanup

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

func testDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "postgres://ever:first_commit@localhost:5432/moods?sslmode=disable"
}

func TestCleaner_PurgeExpired(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping integration test, postgres unavailable: %v", err)
	}
	defer db.Close()

	// 1. Setup temporary local storage
	tempDir := t.TempDir()
	store, err := storage.NewLocalStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}

	// 2. Create physical files in store
	activePath, err := store.Save(ctx, "active.mp3", bytes.NewReader([]byte("active audio data")))
	if err != nil {
		t.Fatalf("failed to save active file: %v", err)
	}
	expiredPath, err := store.Save(ctx, "expired.mp3", bytes.NewReader([]byte("expired audio data")))
	if err != nil {
		t.Fatalf("failed to save expired file: %v", err)
	}

	// 3. Insert active song record (uploaded now)
	activeSong := &database.Song{
		SessionID:    "active-session-" + uuid.New().String(),
		Filename:     "active.mp3",
		OriginalName: "Active Song",
		Format:       "mp3",
		FilePath:     activePath,
		SizeBytes:    int64(len("active audio data")),
		Status:       "ready",
	}
	if err := db.CreateSong(ctx, activeSong); err != nil {
		t.Fatalf("failed to create active song: %v", err)
	}

	// 4. Insert expired song record directly with an older timestamp (2 hours ago)
	expiredID := uuid.New()
	expiredUploadedAt := time.Now().Add(-2 * time.Hour)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO songs (id, session_id, filename, original_name, format, file_path, file_size, status, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		expiredID, "expired-session-"+uuid.New().String(), "expired.mp3", "Expired Song",
		"mp3", expiredPath, int64(len("expired audio data")), "ready", expiredUploadedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert expired song: %v", err)
	}

	// 5. Initialize Cleaner with 1 hour TTL
	cleaner := NewCleaner(db, store, 1*time.Hour, 100*time.Millisecond)

	count, err := cleaner.PurgeExpired(ctx)
	if err != nil {
		t.Fatalf("PurgeExpired failed: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 purged song, got %d", count)
	}

	// 6. Verify expired record is deleted from DB
	_, err = db.GetSong(ctx, expiredID)
	if err == nil {
		t.Errorf("expected expired song to be deleted from database")
	}

	// 7. Verify expired physical file is deleted from disk
	if _, err := os.Stat(expiredPath); !os.IsNotExist(err) {
		t.Errorf("expected expired physical file %s to be deleted, but it still exists", expiredPath)
	}

	// 8. Verify active record and physical file are still present
	loadedActive, err := db.GetSong(ctx, activeSong.ID)
	if err != nil {
		t.Errorf("expected active song to remain in DB, got err: %v", err)
	} else if loadedActive.ID != activeSong.ID {
		t.Errorf("unexpected active song loaded: %+v", loadedActive)
	}

	if _, err := os.Stat(activePath); os.IsNotExist(err) {
		t.Errorf("expected active physical file %s to remain on disk", activePath)
	}

	// Cleanup active song
	_, _ = db.DeleteSong(ctx, activeSong.ID)
}

func TestCleaner_StartAndShutdown(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, _ := storage.NewLocalStore(tempDir)
	cleaner := NewCleaner(nil, store, 1*time.Hour, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cleaner.Start(ctx)

	// Let ticker tick once or twice
	time.Sleep(50 * time.Millisecond)

	// Cancel context and verify clean shutdown without deadlocks
	cancel()
	time.Sleep(50 * time.Millisecond)
}
