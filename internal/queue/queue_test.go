package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Dharshan2208/judex/internal/models"
	"github.com/Dharshan2208/judex/internal/store"
	"github.com/Dharshan2208/judex/tests"
)

func TestTryPushCapacity(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	q := NewQueue(client, 1)

	if ok := q.TryPush(tests.NewJob("j1", "go", "pending")); !ok {
		t.Fatalf("expected first push to succeed")
	}
	if ok := q.TryPush(tests.NewJob("j2", "go", "pending")); ok {
		t.Fatalf("expected second push to be rejected")
	}
}

func TestClaimAckFIFO(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	q := NewQueue(client, 10)

	_ = q.TryPush(tests.NewJob("first", "python", "pending"))
	_ = q.TryPush(tests.NewJob("second", "python", "pending"))

	claimed1 := q.Claim()
	claimed2 := q.Claim()
	if claimed1.Job.ID != "first" || claimed2.Job.ID != "second" {
		t.Fatalf("expected fifo order, got %s then %s", claimed1.Job.ID, claimed2.Job.ID)
	}

	q.Ack(claimed1.Raw)
	q.Ack(claimed2.Raw)
	if got := q.ProcessingLen(); got != 0 {
		t.Fatalf("expected empty processing queue, got %d", got)
	}
}

func TestClaimSkipsInvalidPayload(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	q := NewQueue(client, 10)

	if err := client.LPush(context.Background(), q.Pending, `{"id":"good","language":"go","status":"pending","created_at":"2026-01-01T00:00:00Z","claimed_at":"0001-01-01T00:00:00Z","completed_at":"0001-01-01T00:00:00Z","result":{"stdout":"","stderr":"","status":"","language":"","execution_time_ms":0}}`).Err(); err != nil {
		t.Fatalf("seed good payload: %v", err)
	}
	if err := client.LPush(context.Background(), q.Pending, "bad-json").Err(); err != nil {
		t.Fatalf("seed bad payload: %v", err)
	}

	claimed := q.Claim()
	if claimed.Job.ID != "good" {
		t.Fatalf("expected valid payload to be claimed, got %s", claimed.Job.ID)
	}
}

func TestRecoverStuckRequeuesTimedOutRunningJob(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	q := NewQueue(client, 10)
	s := store.NewRedisStore(client)

	job := &models.Job{
		ID:        "stuck-1",
		Language:  "go",
		Code:      "package main",
		Status:    "running",
		CreatedAt: time.Now().Add(-10 * time.Minute),
		ClaimedAt: time.Now().Add(-10 * time.Minute),
	}
	s.Add(job)

	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	if err := client.LPush(context.Background(), q.Running, raw).Err(); err != nil {
		t.Fatalf("seed running queue: %v", err)
	}

	q.recoverStuck(s, 2*time.Minute)

	if got := q.Len(); got != 1 {
		t.Fatalf("expected requeued pending job, got pending=%d", got)
	}
	if got := q.ProcessingLen(); got != 0 {
		t.Fatalf("expected running queue empty, got %d", got)
	}

	updated, ok := s.Get(job.ID)
	if !ok {
		t.Fatalf("expected job to remain in store")
	}
	if updated.Status != "pending" {
		t.Fatalf("expected status pending after recovery, got %s", updated.Status)
	}
}
