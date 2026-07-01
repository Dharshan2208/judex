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
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		logutil.Info(ctx, "no .env file found, using system environment variables")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := goredis.NewClient(&goredis.Options{
		Addr: addr,
	})

	if err := client.Ping(Ctx).Err(); err != nil {
		logutil.Fatal(ctx, "failed to connect to Redis",
			"error", err,
		)
	}

	logutil.Info(ctx, "connected to Redis",
		"addr", addr,
	)

	return client
}
