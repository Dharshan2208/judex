package redis

import (
	"context"
	"os"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

func New() *goredis.Client {
	if err := godotenv.Load(); err != nil {
		logutil.Info("No .env file found, using system environment variables")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := goredis.NewClient(&goredis.Options{
		Addr: addr,
	})

	if err := client.Ping(Ctx).Err(); err != nil {
		logutil.Fatal("Failed to connect to Redis: %v", err)
	}

	logutil.Info("Connected to Redis at %s", addr)

	return client
}
