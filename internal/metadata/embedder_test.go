package metadata_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

func TestEmbedder_NonExistentFile(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/embed_tags.py"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found at relative path")
	}

	emb := metadata.NewEmbedder(pythonBin, scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meta := &metadata.SongMetadata{Title: "Test Title", Artist: "Test Artist"}
	err := emb.EmbedTags(ctx, "non_existent_file.mp3", meta)
	if err == nil {
		t.Fatal("expected error embedding tags in non-existent file, got nil")
	}
}

func TestEmbedder_InvalidBinary(t *testing.T) {
	t.Parallel()

	emb := metadata.NewEmbedder("/non/existent/python_binary_xyz", "../../python/embed_tags.py")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	meta := &metadata.SongMetadata{Title: "Test"}
	err := emb.EmbedTags(ctx, "some_file.mp3", meta)
	if err == nil {
		t.Fatal("expected error with invalid python binary, got nil")
	}
}

func TestEmbedder_ContextCancellation(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/embed_tags.py"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found at relative path")
	}

	emb := metadata.NewEmbedder(pythonBin, scriptPath)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	meta := &metadata.SongMetadata{Title: "Test"}
	err := emb.EmbedTags(ctx, "some_file.mp3", meta)
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}

func TestEmbedder_RealCopyEmbed(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/embed_tags.py"
	audioPath := "../../local/Without you.mp3"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found")
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skip("skipping test: test audio file not found")
	}

	// Copy audio file to a temp directory so we don't mutate original test fixture
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_embed.mp3")

	src, err := os.Open(audioPath)
	if err != nil {
		t.Fatalf("failed to open source audio: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		t.Fatalf("failed to copy audio data: %v", err)
	}
	dst.Close()

	emb := metadata.NewEmbedder(pythonBin, scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	meta := &metadata.SongMetadata{
		Title:         "Unit Test Song",
		Artist:        "Unit Test Artist",
		Album:         "Unit Test Album",
		ReleaseYear:   2026,
		Genre:         "Electronic",
		TrackNumber:   1,
		MusicbrainzID: "mb-uuid-12345",
	}

	if err := emb.EmbedTags(ctx, tmpFile, meta); err != nil {
		t.Fatalf("EmbedTags failed: %v", err)
	}
}
