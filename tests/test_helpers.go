package tests

import (
	"context"
	"testing"
	"time"

	"github.com/Dharshan2208/judex/internal/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func NewMiniRedisClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})

	return mr, client
}

func MustSetRaw(t *testing.T, client *redis.Client, key, value string) {
	t.Helper()
	if err := client.Set(context.Background(), key, value, 0).Err(); err != nil {
		t.Fatalf("set key %s: %v", key, err)
	}
}

func NewJob(id, language, status string) *models.Job {
	return &models.Job{
		ID:        id,
		Language:  language,
		Code:      "print('hello')",
		Status:    status,
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}
}
