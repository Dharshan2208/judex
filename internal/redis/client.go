package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

func New() *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
	})
}
