package lyrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

type lyricsOutput struct {
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	Source         string `json:"source"`
	IsSynced       bool   `json:"is_synced"`
	IsInstrumental bool   `json:"is_instrumental"`
	Language       string `json:"language"`
	HasTiming      bool   `json:"has_timing"`
	SyncedLyrics   string `json:"synced_lyrics"`
	PlainLyrics    string `json:"plain_lyrics"`
	Emotions       struct {
		TopEmotions   []database.LyricEmotion `json:"top_emotions"`
		EmotionVector []float32               `json:"emotion_vector"`
		EmotionLabels []string                `json:"emotion_labels"`
	} `json:"emotions"`
}

// Client manages lyrics retrieval and sentiment analysis via python/lyrics.py.
type Client struct {
	pythonBin    string
	scriptPath   string
	geminiAPIKey string
}

// NewClient creates a new lyrics retrieval client.
func NewClient(pythonBin, scriptPath, geminiAPIKey string) *Client {
	return &Client{
		pythonBin:    pythonBin,
		scriptPath:   scriptPath,
		geminiAPIKey: geminiAPIKey,
	}
}

// FetchLyrics retrieves synchronized/plain lyrics and 28-D emotion vector from LRCLIB and RoBERTa.
func (c *Client) FetchLyrics(ctx context.Context, track, artist, album string, durationSec int) (*database.SongLyrics, error) {
	args := []string{c.scriptPath}
	if track != "" {
		args = append(args, "--track", track)
	}
	if artist != "" {
		args = append(args, "--artist", artist)
	}
	if album != "" {
		args = append(args, "--album", album)
	}
	if durationSec > 0 {
		args = append(args, "--duration", strconv.Itoa(durationSec))
	}

	return c.run(ctx, args)
}

// FetchLyricsMock deterministic mock runner for offline testing.
func (c *Client) FetchLyricsMock(ctx context.Context, mockMode string) (*database.SongLyrics, error) {
	return c.run(ctx, []string{c.scriptPath, "--mock", mockMode})
}

// AnalyzeLyricsText runs sentiment analysis on raw input lyrics text.
func (c *Client) AnalyzeLyricsText(ctx context.Context, text string) (*database.SongLyrics, error) {
	return c.run(ctx, []string{c.scriptPath, "--text", text})
}

func (c *Client) run(ctx context.Context, args []string) (*database.SongLyrics, error) {
	cmd := exec.CommandContext(ctx, c.pythonBin, args...)

	env := os.Environ()
	if c.geminiAPIKey != "" {
		env = append(env, "GEMINI_API_KEY="+c.geminiAPIKey)
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run lyrics script: %w (stderr: %s)", err, stderr.String())
	}

	var out lyricsOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("decode lyrics output: %w (raw: %s)", err, stdout.String())
	}

	if out.Status == "not_found" {
		return nil, fmt.Errorf("lyrics not found: %s", out.Message)
	}
	if out.Status != "ok" {
		return nil, fmt.Errorf("lyrics error: %s", out.Message)
	}

	l := &database.SongLyrics{
		PlainLyrics:    out.PlainLyrics,
		SyncedLyrics:   out.SyncedLyrics,
		IsSynced:       out.IsSynced,
		IsInstrumental: out.IsInstrumental,
		Language:       out.Language,
		TopEmotions:    out.Emotions.TopEmotions,
		EmotionVector:  out.Emotions.EmotionVector,
	}

	if l.TopEmotions == nil {
		l.TopEmotions = []database.LyricEmotion{}
	}

	return l, nil
}
