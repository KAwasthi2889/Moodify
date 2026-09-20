package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWorkerPoolQueue_Stats(t *testing.T) {
	t.Parallel()

	q := NewWorkerPoolQueue(nil, nil, 2, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q.Start(ctx)

	job := AnalysisJob{
		SongID:    uuid.New(),
		SessionID: "test-session",
		FilePath:  "/tmp/nonexistent.mp3",
	}

	// Stats initial
	stats := q.Stats()
	if stats.EnqueuedCount != 0 {
		t.Errorf("expected 0 enqueued, got %d", stats.EnqueuedCount)
	}

	// SQS wrapper fallback test
	sqs := NewSQSQueue("https://sqs.us-east-1.amazonaws.com/12345/test-queue", q)
	_ = sqs.Enqueue(ctx, job)

	time.Sleep(20 * time.Millisecond)
	sqsStats := sqs.Stats()
	if sqsStats.EnqueuedCount < 1 {
		t.Errorf("expected at least 1 enqueued, got %d", sqsStats.EnqueuedCount)
	}
}
