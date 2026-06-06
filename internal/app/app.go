package app

import (
	"log"

	"github.com/Dharshan2208/code-compiler/internal/queue"
	redisclient "github.com/Dharshan2208/code-compiler/internal/redis"
	"github.com/Dharshan2208/code-compiler/internal/store"
	"github.com/Dharshan2208/code-compiler/internal/worker"
)

type App struct {
	Queue *queue.Queue
	Store *store.RedisStore
	Pool  *worker.Pool

	Stats *queue.Stats
}

func New() *App {
	return NewWorker()
}

func NewAPI() *App {
	return newApp(0)
}

func NewWorker() *App {
	return newApp(4)
}

func newApp(workerCount int) *App {
	log.Println("Initializing application...")

	redisClient := redisclient.New()
	q := queue.NewQueue(redisClient, 100)
	s := store.NewRedisStore(redisClient)
	stats := &queue.Stats{}

	var p *worker.Pool
	if workerCount > 0 {
		p = worker.NewPool(workerCount, q, s, stats)
	}

	log.Printf("Application initialized with queue_size=100 worker_count=%d", workerCount)

	return &App{
		Queue: q,
		Store: s,
		Pool:  p,
		Stats: stats,
	}
}
