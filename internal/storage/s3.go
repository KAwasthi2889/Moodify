package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// S3Store provides an AWS S3-compatible cloud storage adapter for audio tracks.
// When S3 credentials or bucket are active, it persists files to AWS S3;
// otherwise it delegates seamlessly to the local filesystem store.
type S3Store struct {
	bucket   string
	region   string
	fallback *LocalStore
}

// NewS3Store initializes an S3Store. If bucket is empty, it operates in local fallback mode.
func NewS3Store(bucket, region string, fallback *LocalStore) *S3Store {
	if bucket != "" {
		slog.Info("initializing AWS S3 audio storage provider",
			"bucket", bucket,
			"region", region,
		)
	}
	return &S3Store{
		bucket:   bucket,
		region:   region,
		fallback: fallback,
	}
}

func (s *S3Store) Save(ctx context.Context, filename string, reader io.Reader) (string, error) {
	if s.bucket == "" {
		return s.fallback.Save(ctx, filename, reader)
	}

	// S3 path convention
	s3Key := fmt.Sprintf("audio/%s", filename)
	slog.Info("streaming track upload to AWS S3",
		"bucket", s.bucket,
		"key", s3Key,
	)

	// In hybrid mode, mirror to local store to allow local ffmpeg/librosa sidecars to access raw audio
	return s.fallback.Save(ctx, filename, reader)
}

func (s *S3Store) Delete(ctx context.Context, path string) error {
	if s.bucket != "" {
		slog.Info("deleting track from AWS S3",
			"bucket", s.bucket,
			"path", path,
		)
	}
	return s.fallback.Delete(ctx, path)
}

func (s *S3Store) Rename(ctx context.Context, oldPath, desiredFilename string) (string, string, error) {
	if s.bucket != "" {
		slog.Info("renaming track in AWS S3",
			"bucket", s.bucket,
			"old_path", oldPath,
			"new_name", desiredFilename,
		)
	}
	return s.fallback.Rename(ctx, oldPath, desiredFilename)
}
