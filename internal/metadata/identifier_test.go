package metadata_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

func TestIdentifier_NonExistentFile(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/identify.py"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found at relative path")
	}

	id := metadata.NewIdentifier(pythonBin, scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := id.Identify(ctx, "non_existent_audio_file.mp3", "")
	if err == nil {
		t.Fatal("expected error identifying non-existent file, got nil")
	}
}

func TestIdentifier_InvalidBinary(t *testing.T) {
	t.Parallel()

	id := metadata.NewIdentifier("/non/existent/python_binary_xyz", "../../python/identify.py")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := id.Identify(ctx, "some_file.mp3", "")
	if err == nil {
		t.Fatal("expected error with invalid python binary, got nil")
	}
}

func TestIdentifier_ContextCancellation(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/identify.py"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found at relative path")
	}

	id := metadata.NewIdentifier(pythonBin, scriptPath)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := id.Identify(ctx, "some_file.mp3", "")
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}

func TestIdentifier_RealAudioFallback(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/identify.py"
	audioPath := "../../local/Without you.mp3"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found")
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skip("skipping test: test audio file not found")
	}

	id := metadata.NewIdentifier(pythonBin, scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run without API key: should fall back to mutagen tags or return matches if tags exist
	res, err := id.Identify(ctx, audioPath, "")
	if err != nil {
		t.Fatalf("Identify failed on real audio: %v", err)
	}

	if res.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", res.Status)
	}
}
