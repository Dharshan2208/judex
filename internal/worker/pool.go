package worker

import "github.com/Dharshan2208/code-compiler/internal/queue"

type Pool struct {
	Workers []*Worker
}

func NewPool(count int, q *queue.Queue, s *queue.Store) *Pool {
	pool := &Pool{}

	for i := 1; i <= count; i++ {
		pool.Workers = append(
			pool.Workers,
			NewWorker(i, q, s),
		)
	}

	return pool
}

func (p *Pool) Start() {
	for _, worker := range p.Workers {
		go worker.Start()
	}
}
