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
	log.Println("Initializing application...")

	q := queue.NewQueue(100)

	redisClient := redisclient.New()
	s := store.NewRedisStore(redisClient)
	stats := &queue.Stats{}

	p := worker.NewPool(4, q, s, stats)

	log.Println("Application initialized with queue_size=100 worker_count=4")

	return &App{
		Queue: q,
		Store: s,
		Pool:  p,
		Stats: stats,
	}
}
