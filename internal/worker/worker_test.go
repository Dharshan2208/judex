package worker

import (
	"context"
	"testing"

	"github.com/Dharshan2208/judex/internal/queue"
	"github.com/Dharshan2208/judex/internal/sandbox"
	"github.com/Dharshan2208/judex/internal/store"
	"github.com/Dharshan2208/judex/tests"
	"github.com/agiledragon/gomonkey/v2"
)

func TestWorkerProcessSuccess(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	q := queue.NewQueue(client, 10)
	s := store.NewRedisStore(client)
	stats := &queue.Stats{}
	pm := &sandbox.PoolManager{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethod((*sandbox.PoolManager)(nil), "Acquire", func(*sandbox.PoolManager, context.Context, string) (*sandbox.WarmContainer, error) {
		return &sandbox.WarmContainer{ID: "c1", Language: "python", Image: "compiler-python"}, nil
	})
	patches.ApplyMethod((*sandbox.PoolManager)(nil), "Release", func(*sandbox.PoolManager, context.Context, *sandbox.WarmContainer) {})
	patches.ApplyMethod((*sandbox.Sandbox)(nil), "UploadCode", func(*sandbox.Sandbox, context.Context, string, string) error { return nil })
	patches.ApplyMethod((*sandbox.Sandbox)(nil), "Execute", func(*sandbox.Sandbox, context.Context, []string) sandbox.Result {
		return sandbox.Result{Stdout: "done\n", Status: "success"}
	})

	w := NewWorker(1, q, s, stats, pm)
	job := tests.NewJob("job-success", "python", "pending")
	s.Add(job)

	w.Process(job)

	got, ok := s.Get(job.ID)
	if !ok {
		t.Fatalf("job missing after process")
	}
	if got.Status != "completed" {
		t.Fatalf("expected completed status, got %s", got.Status)
	}
	if got.Result.Stdout != "done\n" {
		t.Fatalf("unexpected result stdout: %q", got.Result.Stdout)
	}
	_, completed, failed := stats.Snapshot()
	if completed != 1 || failed != 0 {
		t.Fatalf("unexpected stats completed=%d failed=%d", completed, failed)
	}
}

func TestWorkerProcessUnsupportedLanguage(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	q := queue.NewQueue(client, 10)
	s := store.NewRedisStore(client)
	stats := &queue.Stats{}
	pm := &sandbox.PoolManager{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethod((*sandbox.PoolManager)(nil), "Acquire", func(*sandbox.PoolManager, context.Context, string) (*sandbox.WarmContainer, error) {
		return nil, context.DeadlineExceeded
	})

	w := NewWorker(1, q, s, stats, pm)
	job := tests.NewJob("job-unsupported", "rust", "pending")
	s.Add(job)

	w.Process(job)

	got, _ := s.Get(job.ID)
	if got.Status != "internal_error" {
		t.Fatalf("expected internal_error due acquire failure, got %s", got.Status)
	}
	_, _, failed := stats.Snapshot()
	if failed != 1 {
		t.Fatalf("expected failed=1, got %d", failed)
	}
}
