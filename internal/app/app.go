package app

import (
	"log"

	"github.com/Dharshan2208/judex/internal/queue"
	redisclient "github.com/Dharshan2208/judex/internal/redis"
	"github.com/Dharshan2208/judex/internal/sandbox"
	"github.com/Dharshan2208/judex/internal/store"
	"github.com/Dharshan2208/judex/internal/worker"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Redis       *redis.Client
	Queue       *queue.Queue
	Store       *store.RedisStore
	Pool        *worker.Pool
	PoolManager *sandbox.PoolManager

	Stats *queue.Stats
}

func New() *App {
	return NewWorker()
}

func NewAPI() *App {
	return newApp("api", 0)
}

func NewWorker() *App {
	return newApp("worker", 4)
}

func newApp(role string, workerCount int) *App {
	log.Printf("application initializing: role=%s", role)

	redisClient := redisclient.New()
	q := queue.NewQueue(redisClient, 100)
	s := store.NewRedisStore(redisClient)
	stats := &queue.Stats{}

	var p *worker.Pool
	var pm *sandbox.PoolManager
	if workerCount > 0 {

		languages := map[string]string{
			"go":     "compiler-go",
			"python": "compiler-python",
			"cpp":    "compiler-cpp",
			"c":      "compiler-c",
			"java":   "compiler-java",
		}

		// initialising the poolmamager
		var err error
		pm, err = sandbox.NewPoolManager(workerCount, languages)
		if err != nil {
			log.Fatalf("failed to initialise the pool mamager : %v", err)
		}

		p = worker.NewPool(workerCount, q, s, stats, pm)
	}

	log.Printf("application initialized: role=%s queue_size=100 worker_count=%d", role, workerCount)

	return &App{
		Redis:       redisClient,
		Queue:       q,
		Store:       s,
		Pool:        p,
		PoolManager: pm,
		Stats:       stats,
	}
}
