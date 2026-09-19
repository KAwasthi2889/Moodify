package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileStore defines operations for persisting audio files.
type FileStore interface {
	Save(ctx context.Context, filename string, reader io.Reader) (string, error)
	Delete(ctx context.Context, path string) error
	Rename(ctx context.Context, oldPath, desiredFilename string) (finalPath string, finalFilename string, err error)
}

// LocalStore implements FileStore on the local filesystem.
type LocalStore struct {
	baseDir string
}

// NewLocalStore initializes LocalStore rooted at baseDir, creating it if needed.
func NewLocalStore(baseDir string) (*LocalStore, error) {
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve storage dir: %w", err)
	}

	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir %s: %w", absDir, err)
	}

	return &LocalStore{baseDir: absDir}, nil
}

func (s *LocalStore) Save(_ context.Context, filename string, reader io.Reader) (string, error) {
	destPath := filepath.Join(s.baseDir, filename)

	// Guard against path traversal at trust boundary
	rel, err := filepath.Rel(s.baseDir, destPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid path: file escapes storage directory")
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", fmt.Errorf("create parent dir: %w", err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		os.Remove(destPath)
		return "", fmt.Errorf("write file: %w", err)
	}

	return destPath, nil
}

func (s *LocalStore) Delete(_ context.Context, path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

// Rename moves an existing stored file to a new filename within baseDir.
// If a file with desiredFilename already exists from another upload, it appends
// an incremental counter (e.g. "Song (1).m4a") to prevent overwriting existing files.
func (s *LocalStore) Rename(_ context.Context, oldPath, desiredFilename string) (string, string, error) {
	destPath := filepath.Join(s.baseDir, desiredFilename)

	// Guard against path traversal at trust boundary
	rel, err := filepath.Rel(s.baseDir, destPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", "", fmt.Errorf("invalid path: file escapes storage directory")
	}

	// If destination already equals source, nothing to do
	if destPath == oldPath {
		return destPath, desiredFilename, nil
	}

	// If target file already exists, find next available counter
	finalFilename := desiredFilename
	finalPath := destPath
	if _, err := os.Stat(finalPath); err == nil {
		ext := filepath.Ext(desiredFilename)
		base := strings.TrimSuffix(desiredFilename, ext)
		for counter := 1; ; counter++ {
			candidateName := fmt.Sprintf("%s (%d)%s", base, counter, ext)
			candidatePath := filepath.Join(s.baseDir, candidateName)
			if _, err := os.Stat(candidatePath); os.IsNotExist(err) {
				finalFilename = candidateName
				finalPath = candidatePath
				break
			}
		}
	}

	if err := os.Rename(oldPath, finalPath); err != nil {
		return "", "", fmt.Errorf("rename file %s to %s: %w", oldPath, finalPath, err)
	}

	return finalPath, finalFilename, nil
}
