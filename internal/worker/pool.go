package worker

import (
	"context"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/queue"
	"github.com/Dharshan2208/judex/internal/sandbox"
	"github.com/Dharshan2208/judex/internal/store"
)

type Pool struct {
	Workers []*Worker
}

func NewPool(count int, q *queue.Queue, s *store.RedisStore, stats *queue.Stats, pm *sandbox.PoolManager) *Pool {
	pool := &Pool{}
	logutil.Info(context.Background(), "creating worker pool",
		"count", count,
	)

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
		logutil.Info(context.Background(), "starting worker",
			"id", worker.ID,
		)
		go worker.Start()
	}
}
