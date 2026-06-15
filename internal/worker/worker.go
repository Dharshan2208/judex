package worker

import (
	"context"
	"log"
	"time"

	"github.com/Dharshan2208/judex/internal/executor"
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
	log.Printf("worker started: worker_id=%d", w.ID)

	for {
		claimed := w.Queue.Claim()
		w.Process(claimed.Job)
		w.Queue.Ack(claimed.Raw)
	}
}

func (w *Worker) Process(job *models.Job) {
	log.Printf("job processing started: worker_id=%d job_id=%s language=%s", w.ID, job.ID, job.Language)

	job.Status = "running"
	job.ClaimedAt = time.Now()
	w.Store.Update(job)

	// creating a context for the entier lifecycle (maybe 20secs max)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// step1 acquiring a warm container
	warmContainer, err := w.PoolManager.Acquire(ctx, job.Language)
	if err != nil {
		log.Printf("failed to acquire container: worker_id=%d job_id=%s error=%v", w.ID, job.ID, err)
		w.failJob(job, "internal_error")
		return
	}

	// santising container (ensuring that the container cleaned and returnd after completion)
	defer w.PoolManager.Release(context.Background(), warmContainer)

	// wrapping container in sanbox internface
	sb := &sandbox.Sandbox{
		Container: warmContainer,
		Manager:   w.PoolManager,
	}

	filename, execLang := w.getExecutor(job.Language)
	if execLang == nil {
		log.Printf("unsupported language : %s", job.Language)
		w.failJob(job, "unsupported language")
		return
	}

	// uploading the file(basically streaming code directly into containers memory)
	if err := sb.UploadCode(ctx, filename, job.Code); err != nil {
		log.Printf("failed to upload code : worker_id=%d job_id=%s error=%v", w.ID, job.ID, err)
		w.failJob(job, "internal_error")
		return
	}

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
	w.Store.Update(job)
	log.Printf("job finished: worker_id=%d job_id=%s status=%s language=%s", w.ID, job.ID, job.Status, job.Language)
}

func (w *Worker) failJob(job *models.Job, status string) {
	job.Status = status
	w.Store.Update(job)
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
