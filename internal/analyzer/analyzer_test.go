package analyzer_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/KAwasthi2889/Moodify/internal/analyzer"
)

func TestAnalyzer_NonExistentFile(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/analyze.py"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found at relative path")
	}

	az := analyzer.NewAnalyzer(pythonBin, scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := az.Analyze(ctx, "non_existent_file.mp3")
	if err == nil {
		t.Fatal("expected error analyzing non-existent file, got nil")
	}
}

func TestAnalyzer_RealAudioFile(t *testing.T) {
	t.Parallel()

	pythonBin := "../../python/.venv/bin/python"
	scriptPath := "../../python/analyze.py"
	audioPath := "../../local/Without you.mp3"

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		t.Skip("skipping test: python venv not found")
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skip("skipping test: test audio file not found")
	}

	az := analyzer.NewAnalyzer(pythonBin, scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	feat, err := az.Analyze(ctx, audioPath)
	if err != nil {
		t.Fatalf("Analyze failed on real audio: %v", err)
	}

	if feat.DurationSec <= 0 {
		t.Errorf("expected positive duration, got %f", feat.DurationSec)
	}
	if feat.TempoBPM <= 0 {
		t.Errorf("expected positive tempo, got %f", feat.TempoBPM)
	}
	if len(feat.MoodVector) != 36 {
		t.Errorf("expected 36 dimensions in MoodVector, got %d", len(feat.MoodVector))
	}
}
