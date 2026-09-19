package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Match represents a single song identification match from AcoustID/MusicBrainz or mutagen fallback.
type Match struct {
	SongMetadata
	SuggestedFilename string `json:"suggested_filename,omitempty"`
}

// IdentifyResult represents the output from the Python identification script.
type IdentifyResult struct {
	Status              string  `json:"status"`
	FingerprintDuration float64 `json:"fingerprint_duration"`
	Matches             []Match `json:"matches"`
	Warning             string  `json:"warning,omitempty"`
	Note                string  `json:"note,omitempty"`
	Message             string  `json:"message,omitempty"`
}

// Identifier runs acoustic fingerprinting and metadata lookup via the python sidecar.
type Identifier struct {
	pythonBin  string
	scriptPath string
}

// NewIdentifier creates a new Identifier.
func NewIdentifier(pythonBin, scriptPath string) *Identifier {
	return &Identifier{
		pythonBin:  pythonBin,
		scriptPath: scriptPath,
	}
}

// Identify executes the identification script on the specified audio file.
func (i *Identifier) Identify(ctx context.Context, filePath, apiKey string) (*IdentifyResult, error) {
	args := []string{i.scriptPath, filePath}
	if apiKey != "" {
		args = append(args, "--api-key", apiKey)
	}

	cmd := exec.CommandContext(ctx, i.pythonBin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run identification script: %w (stderr: %s)", err, stderr.String())
	}

	var result IdentifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("decode identification output: %w (raw: %s)", err, stdout.String())
	}

	if result.Status == "error" {
		return nil, fmt.Errorf("identification failed: %s", result.Message)
	}

	return &result, nil
}
