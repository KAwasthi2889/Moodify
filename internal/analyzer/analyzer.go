package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/KAwasthi2889/Moodify/internal/audio"
	"github.com/KAwasthi2889/Moodify/internal/database"
)

type analyzeOutput struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	File        string                 `json:"file"`
	DurationSec float32                `json:"duration_sec"`
	Features    audio.AcousticFeatures `json:"features"`
	MoodVector  []float32              `json:"mood_vector"`
}

// Analyzer runs DSP audio feature extraction via the Python sidecar.
type Analyzer struct {
	pythonBin  string
	scriptPath string
}

// NewAnalyzer creates a new Analyzer runner.
func NewAnalyzer(pythonBin, scriptPath string) *Analyzer {
	return &Analyzer{
		pythonBin:  pythonBin,
		scriptPath: scriptPath,
	}
}

// Analyze executes the feature extraction script on the specified audio file.
func (a *Analyzer) Analyze(ctx context.Context, filePath string) (*database.SongFeatures, error) {
	cmd := exec.CommandContext(ctx, a.pythonBin, a.scriptPath, filePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run analyze script: %w (stderr: %s)", err, stderr.String())
	}

	var output analyzeOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("decode analyze output: %w (raw: %s)", err, stdout.String())
	}

	if output.Status != "ok" {
		return nil, fmt.Errorf("analyze failed: %s", output.Message)
	}

	feat := &database.SongFeatures{
		AcousticFeatures: output.Features,
		MoodVector:       output.MoodVector,
	}
	feat.DurationSec = output.DurationSec

	return feat, nil
}
