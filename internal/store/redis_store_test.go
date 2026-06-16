package store

import (
	"testing"
	"time"

	"github.com/Dharshan2208/judex/internal/models"
	"github.com/Dharshan2208/judex/tests"
)

func TestRedisStoreAddGetUpdateDelete(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	s := NewRedisStore(client)

	job := tests.NewJob("job-1", "python", "pending")
	s.Add(job)

	stored, ok := s.Get(job.ID)
	if !ok {
		t.Fatalf("expected job in store")
	}
	if stored.Status != "pending" {
		t.Fatalf("unexpected status: %s", stored.Status)
	}

	job.Status = "completed"
	s.Update(job)

	stored, ok = s.Get(job.ID)
	if !ok || stored.Status != "completed" {
		t.Fatalf("expected updated job, got ok=%v status=%v", ok, stored.Status)
	}

	s.Delete(job.ID)
	if _, ok := s.Get(job.ID); ok {
		t.Fatalf("expected deleted job to be missing")
	}
}

func TestRedisStoreGetInvalidPayload(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	s := NewRedisStore(client)

	tests.MustSetRaw(t, client, "job:bad", "{not-json")
	if _, ok := s.Get("bad"); ok {
		t.Fatalf("expected invalid payload lookup to fail")
	}
}

func TestRedisStoreCleanup(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	s := NewRedisStore(client)

	now := time.Now()
	jobs := []*models.Job{
		{ID: "old-done", Status: "completed", CreatedAt: now.Add(-2 * time.Hour), CompletedAt: now.Add(-2 * time.Hour)},
		{ID: "new-done", Status: "completed", CreatedAt: now.Add(-2 * time.Hour), CompletedAt: now.Add(-1 * time.Minute)},
		{ID: "running", Status: "running", CreatedAt: now.Add(-2 * time.Hour)},
	}
	for _, j := range jobs {
		s.Add(j)
	}
	tests.MustSetRaw(t, client, "job:invalid", "{invalid")

	removed := s.Cleanup(30 * time.Minute)
	if removed != 1 {
		t.Fatalf("expected 1 cleanup removal, got %d", removed)
	}

	if _, ok := s.Get("old-done"); ok {
		t.Fatalf("expected old completed job to be removed")
	}
	if _, ok := s.Get("new-done"); !ok {
		t.Fatalf("expected recently completed job to remain")
	}
	if _, ok := s.Get("running"); !ok {
		t.Fatalf("expected running job to remain")
	}
}
