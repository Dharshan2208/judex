package worker

import (
	"log"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/executor"
	"github.com/Dharshan2208/code-compiler/internal/models"
	"github.com/Dharshan2208/code-compiler/internal/queue"
	"github.com/Dharshan2208/code-compiler/internal/store"
	"github.com/Dharshan2208/code-compiler/internal/workspace"
)

type Worker struct {
	ID    int
	Queue *queue.Queue
	Store *store.RedisStore
	Stats *queue.Stats
}

func NewWorker(id int, q *queue.Queue, s *store.RedisStore, stats *queue.Stats) *Worker {
	return &Worker{
		ID:    id,
		Queue: q,
		Store: s,
		Stats: stats,
	}
}

func (w *Worker) Start() {
	log.Printf("worker started: worker_id=%d", w.ID)

	for {
		// job := w.Queue.Pop()
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

	dir, err := workspace.CreateWorkspace()
	if err != nil {
		log.Printf("workspace create failed: worker_id=%d job_id=%s error=%v", w.ID, job.ID, err)
		job.Status = "failed"
		w.Store.Update(job)
		w.Stats.IncFailed()
		return
	}
	log.Printf("workspace created: worker_id=%d job_id=%s dir=%s", w.ID, job.ID, dir)

	defer func() {
		if err := workspace.Cleanup(dir); err != nil {
			log.Printf("workspace cleanup failed: worker_id=%d job_id=%s dir=%s error=%v", w.ID, job.ID, dir, err)
			return
		}

		log.Printf("workspace cleaned: worker_id=%d job_id=%s dir=%s", w.ID, job.ID, dir)
	}()

	var (
		filename string
		execLang executor.Executor
	)

	switch job.Language {

	case "python":
		filename = "main.py"
		execLang = executor.PythonExecutor{}

	case "cpp":
		filename = "main.cpp"
		execLang = executor.CppExecutor{}
	
	case "java":
		filename = "Main.java"
		execLang = executor.JavaExecutor{}
	
	case "go":
		filename = "main.go"
		execLang = executor.GoExecutor{}

	default:
		log.Printf("job failed: worker_id=%d job_id=%s reason=unsupported_language language=%s", w.ID, job.ID, job.Language)
		job.Status = "failed"
		w.Store.Update(job)
		w.Stats.IncFailed()
		return
	}

	file, err := workspace.WriteFile(dir, filename, job.Code)
	if err != nil {
		log.Printf("workspace write failed: worker_id=%d job_id=%s file=%s error=%v", w.ID, job.ID, filename, err)
		job.Status = "failed"
		w.Store.Update(job)
		w.Stats.IncFailed()
		return
	}
	log.Printf("workspace file written: worker_id=%d job_id=%s file=%s", w.ID, job.ID, file)

	result := execLang.Execute(file, dir)

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
