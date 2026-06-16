package worker

import (
	"log"

	"github.com/Dharshan2208/judex/internal/queue"
	"github.com/Dharshan2208/judex/internal/sandbox"
	"github.com/Dharshan2208/judex/internal/store"
)

type Pool struct {
	Workers []*Worker
}

func NewPool(count int, q *queue.Queue, s *store.RedisStore, stats *queue.Stats, pm *sandbox.PoolManager) *Pool {
	pool := &Pool{}
	log.Printf("creating worker pool: count=%d", count)

	for i := 1; i <= count; i++ {
		pool.Workers = append(
			pool.Workers,
			NewWorker(i, q, s, stats, pm),
		)
	}

	return pool
}

func (p *Pool) Start() {
	for _, worker := range p.Workers {
		log.Printf("starting worker: id=%d", worker.ID)
		go worker.Start()
	}
}
