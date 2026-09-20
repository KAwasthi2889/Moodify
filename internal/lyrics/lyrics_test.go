package lyrics_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/lyrics"
)

func TestLyricsMock_SadHopeful(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/lyrics.py"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("python venv not found; skipping integration test")
	}

	client := lyrics.NewClient(pythonBin, scriptPath, "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	l, err := client.FetchLyricsMock(ctx, "sad_hopeful")
	if err != nil {
		t.Fatalf("FetchLyricsMock failed: %v", err)
	}

	if !l.IsSynced {
		t.Errorf("expected IsSynced to be true, got %v", l.IsSynced)
	}
	if l.SyncedLyrics == "" {
		t.Error("expected non-empty SyncedLyrics")
	}
	if l.PlainLyrics == "" {
		t.Error("expected non-empty PlainLyrics")
	}
	if len(l.EmotionVector) != 28 {
		t.Errorf("expected 28-D emotion vector, got %d", len(l.EmotionVector))
	}
	if len(l.TopEmotions) == 0 {
		t.Error("expected top emotions to be populated")
	}

	// In sad_hopeful mock, sadness and optimism should be in top emotions
	foundSadness := false
	foundOptimism := false
	for _, em := range l.TopEmotions {
		if em.Label == "sadness" {
			foundSadness = true
		}
		if em.Label == "optimism" {
			foundOptimism = true
		}
	}
	if !foundSadness || !foundOptimism {
		t.Errorf("expected sadness and optimism in top emotions, got %+v", l.TopEmotions)
	}
}

func TestLyricsMock_InvalidScript(t *testing.T) {
	t.Parallel()

	client := lyrics.NewClient("python3", "non_existent_script.py", "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.FetchLyricsMock(ctx, "sad_hopeful")
	if err == nil {
		t.Error("expected error for non-existent script, got nil")
	}
}
