package worker

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/executor"
	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/models"
	"github.com/Dharshan2208/judex/internal/queue"
	"github.com/Dharshan2208/judex/internal/sandbox"
	"github.com/Dharshan2208/judex/internal/store"
)

type Worker struct {
	ID          int
	Queue       *queue.Queue
	Store       *store.RedisStore
	Stats       *queue.Stats
	PoolManager *sandbox.PoolManager
}

func NewWorker(id int, q *queue.Queue, s *store.RedisStore, stats *queue.Stats, pm *sandbox.PoolManager) *Worker {
	return &Worker{
		ID:          id,
		Queue:       q,
		Store:       s,
		Stats:       stats,
		PoolManager: pm,
	}
}

func (w *Worker) Start() {
	logutil.Info(context.Background(), "worker started",
		"worker_id", w.ID,
	)

	for {
		claimed := w.Queue.Claim()
		if claimed == nil {
			logutil.Debug(context.Background(), "worker: no job claimed, retrying",
				"worker_id", w.ID,
			)
			continue
		}
		logutil.Debug(context.Background(), "worker: job claimed",
			"worker_id", w.ID,
			"job_id", claimed.Job.ID,
			"language", claimed.Job.Language,
		)

		w.Process(claimed.Job)
		w.Queue.Ack(claimed.Raw)
		logutil.Debug(context.Background(), "worker: job acknowledged",
			"worker_id", w.ID,
			"job_id", claimed.Job.ID,
		)
	}
}

func (w *Worker) Process(job *models.Job) {
	processingStartTime := time.Now()

	logutil.Info(context.Background(), "worker: job processing started",
		"worker_id", w.ID,
		"job_id", job.ID,
		"language", job.Language,
	)

	job.Status = "running"
	job.ClaimedAt = time.Now()
	w.Store.Update(context.Background(), job)

	// Context for the entire job lifecycle (max 30 s).
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: acquire a warm container.
	logutil.Debug(ctx, "worker: acquiring warm container",
		"worker_id", w.ID,
		"job_id", job.ID,
		"language", job.Language,
	)
	warmContainer, err := w.PoolManager.Acquire(ctx, job.Language)
	if err != nil {
		logutil.Error(ctx, "worker: failed to acquire container",
			"worker_id", w.ID,
			"job_id", job.ID,
			"error", err,
		)
		w.failJob(ctx, job, "internal_error")
		return
	}
	logutil.Info(ctx, "worker: container acquired",
		"worker_id", w.ID,
		"job_id", job.ID,
		"container_id", warmContainer.ID,
		"language", job.Language,
	)

	// Ensure the container is cleaned up and returned to the pool.
	defer func() {
		logutil.Debug(ctx, "worker: releasing container",
			"worker_id", w.ID,
			"job_id", job.ID,
			"container_id", warmContainer.ID,
		)
		w.PoolManager.Release(context.Background(), warmContainer)
		logutil.Info(ctx, "worker: container released",
			"worker_id", w.ID,
			"job_id", job.ID,
			"container_id", warmContainer.ID,
		)
	}()

	// Wrap container in sandbox interface.
	sb := &sandbox.Sandbox{
		Container: warmContainer,
		Manager:   w.PoolManager,
	}

	filename, execLang := w.getExecutor(job.Language)
	if execLang == nil {
		logutil.Warn(ctx, "worker: unsupported language",
			"language", job.Language,
			"job_id", job.ID,
		)
		w.failJob(ctx, job, "unsupported language")
		return
	}

	// Upload code to the container.
	logutil.Debug(ctx, "worker: uploading code",
		"worker_id", w.ID,
		"job_id", job.ID,
		"container_id", warmContainer.ID,
		"filename", filename,
	)
	if err := sb.UploadCode(ctx, filename, job.Code); err != nil {
		logutil.Error(ctx, "worker: code upload failed",
			"worker_id", w.ID,
			"job_id", job.ID,
			"error", err,
		)
		w.failJob(ctx, job, "internal_error")
		return
	}
	logutil.Debug(ctx, "worker: code uploaded",
		"worker_id", w.ID,
		"job_id", job.ID,
		"container_id", warmContainer.ID,
	)

	result := execLang.Execute(ctx, sb)

	job.Result = models.RunResponse{
		Stdout:        result.Stdout,
		Stderr:        result.Stderr,
		Status:        result.Status,
		Language:      job.Language,
		ExecutionTime: result.ExecutionTime,
	}

	if result.Status == "success" {
		job.Status = "completed"
		w.Stats.IncCompleted()
	} else {
		job.Status = result.Status
		w.Stats.IncFailed()
	}

	job.CompletedAt = time.Now()
	w.Store.Update(ctx, job)

	logutil.Info(ctx, "worker: job finished",
		"worker_id", w.ID,
		"job_id", job.ID,
		"status", job.Status,
		"language", job.Language,
		"container_id", warmContainer.ID,
		"total_duration", time.Since(processingStartTime).Milliseconds(),
		"exec_time_ms", result.ExecutionTime,
	)
}

func (w *Worker) failJob(ctx context.Context, job *models.Job, status string) {
	logutil.Error(ctx, "worker: job failed",
		"worker_id", w.ID,
		"job_id", job.ID,
		"language", job.Language,
		"status", status,
	)
	job.Status = status
	w.Store.Update(ctx, job)
	w.Stats.IncFailed()
}

func (w *Worker) getExecutor(lang string) (string, executor.Executor) {
	switch lang {
	case "python":
		return "main.py", executor.PythonExecutor{}
	case "java":
		return "Main.java", executor.JavaExecutor{}
	case "go":
		return "main.go", executor.GoExecutor{}
	case "cpp":
		return "main.cpp", executor.CppExecutor{}
	case "c":
		return "main.c", executor.CExecutor{}
	default:
		return "", nil
	}
}
