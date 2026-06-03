package worker

import (
	"log"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/executor"
	"github.com/Dharshan2208/code-compiler/internal/models"
	"github.com/Dharshan2208/code-compiler/internal/queue"
	"github.com/Dharshan2208/code-compiler/internal/workspace"
)

type Worker struct {
	ID    int
	Queue *queue.Queue
	Store *queue.Store
	Stats *queue.Stats
}

func NewWorker(id int, q *queue.Queue, s *queue.Store, stats *queue.Stats) *Worker {
	return &Worker{
		ID:    id,
		Queue: q,
		Store: s,
		Stats: stats,
	}
}

func (w *Worker) Start() {
	log.Printf("Worker started: id=%d", w.ID)

	for {
		job := w.Queue.Pop()
		w.Process(job)
	}
}

func (w *Worker) Process(job *models.Job) {
	log.Printf("Worker processing job: worker_id=%d job_id=%s language=%s", w.ID, job.ID, job.Language)

	job.Status = "running"

	w.Store.Update(job)

	dir, err := workspace.CreateWorkspace()
	if err != nil {
		log.Printf("workspace create failed: worker_id=%d job_id=%s error=%v", w.ID, job.ID, err)
		job.Status = "failed"
		w.Store.Update(job)
		w.Stats.IncFailed()
		return
	}
	log.Printf("Workspace created: worker_id=%d job_id=%s dir=%s", w.ID, job.ID, dir)

	defer func() {
		if err := workspace.Cleanup(dir); err != nil {
			log.Printf("workspace cleanup failed: worker_id=%d job_id=%s dir=%s error=%v", w.ID, job.ID, dir, err)
			return
		}

		log.Printf("Workspace cleaned: worker_id=%d job_id=%s dir=%s", w.ID, job.ID, dir)
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

	default:
		log.Printf("Job failed: worker_id=%d job_id=%s reason=unsupported_language language=%s", w.ID, job.ID, job.Language)
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
	log.Printf("Workspace file written: worker_id=%d job_id=%s file=%s", w.ID, job.ID, file)

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
	log.Printf("Job finished: worker_id=%d job_id=%s status=%s language=%s", w.ID, job.ID, job.Status, job.Language)
}
