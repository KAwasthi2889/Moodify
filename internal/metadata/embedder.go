package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Embedder runs the python script to embed metadata tags into physical audio files.
type Embedder struct {
	pythonBin  string
	scriptPath string
}

// NewEmbedder creates a new Embedder instance.
func NewEmbedder(pythonBin, scriptPath string) *Embedder {
	return &Embedder{
		pythonBin:  pythonBin,
		scriptPath: scriptPath,
	}
}

// EmbedTags embeds the given metadata into the audio file at filePath.
func (e *Embedder) EmbedTags(ctx context.Context, filePath string, meta *SongMetadata) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata payload: %w", err)
	}

	cmd := exec.CommandContext(ctx, e.pythonBin, e.scriptPath, filePath, "--json", string(metaJSON))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute tag embedder: %w (stderr: %s)", err, stderr.String())
	}

	var resp struct {
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return fmt.Errorf("decode embedder output: %w (raw: %s)", err, stdout.String())
	}

	if resp.Status != "ok" {
		return fmt.Errorf("tag embedding failed: %s", resp.Message)
	}

	return nil
}
