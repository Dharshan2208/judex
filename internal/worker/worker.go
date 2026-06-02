package worker

import (
	"github.com/Dharshan2208/code-compiler/internal/executor"
	"github.com/Dharshan2208/code-compiler/internal/models"
	"github.com/Dharshan2208/code-compiler/internal/queue"
	"github.com/Dharshan2208/code-compiler/internal/workspace"
)

type Worker struct {
	ID    int
	Queue *queue.Queue
	Store *queue.Store
}

func NewWorker(id int, q *queue.Queue, s *queue.Store) *Worker {
	return &Worker{
		ID:    id,
		Queue: q,
		Store: s,
	}
}

func (w *Worker) Start() {
	for {
		job := w.Queue.Pop()
		w.Process(job)
	}
}

func (w *Worker) Process(job *models.Job) {
	job.Status = "running"

	w.Store.Update(job)

	dir, err := workspace.CreateWorkspace()
	if err != nil {
		job.Status = "failed"
		w.Store.Update(job)
		return
	}

	defer workspace.Cleanup(dir)

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
		job.Status = "failed"
		w.Store.Update(job)
		return
	}

	file, err := workspace.WriteFile(dir, filename, job.Code)
	if err != nil {
		job.Status = "failed"
		w.Store.Update(job)
		return
	}

	result := execLang.Execute(file)

	job.Result = models.RunResponse{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		Status:   result.Status,
		Language: job.Language,
	}

	if result.Status == "success" {
		job.Status = "completed"
	} else {
		job.Status = result.Status
	}

	w.Store.Update(job)
}
