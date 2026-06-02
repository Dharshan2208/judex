package app

import (
	"github.com/Dharshan2208/code-compiler/internal/queue"
	"github.com/Dharshan2208/code-compiler/internal/worker"
)

type App struct {
	Queue *queue.Queue
	Store *queue.Store
	Pool  *worker.Pool
}

func New() *App {
	q := queue.NewQueue(100)
	s := queue.NewStore()
	p := worker.NewPool(4, q, s)

	return &App{
		Queue: q,
		Store: s,
		Pool:  p,
	}
}
