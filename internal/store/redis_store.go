package store

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Dharshan2208/code-compiler/internal/models"
)

const jobTTL = 24 * time.Hour

type RedisStore struct {
	Client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		Client: client,
	}
}

func jobKey(id string) string {
	return "job:" + id
}

func (s *RedisStore) Add(job *models.Job) {
	data, err := json.Marshal(job)
	if err != nil {
		log.Printf("Redis store add marshal failed : job_id%s error=%v", job.ID, err)
		return
	}

	ctx := context.Background()
	err = s.Client.Set(ctx, jobKey(job.ID), data, jobTTL).Err()
	if err != nil {
		log.Printf("redis store add failed: job_id=%s error=%v", job.ID, err)
		return
	}

	log.Printf("redis store add: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
}

func (s *RedisStore) Get(id string) (*models.Job, bool) {
	ctx := context.Background()
	data, err := s.Client.Get(ctx, jobKey(id)).Result()
	if err == redis.Nil {
		log.Printf("redis store get: job_id=%s found=false", id)
		return nil, false
	}

	if err != nil {
		log.Printf("redis store get failed: job_id=%s error=%v", id, err)
		return nil, false
	}

	var job models.Job
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		log.Printf("redis store get unmarshal failed: job_id=%s error=%v", id, err)
		return nil, false
	}

	log.Printf("redis store get: job_id=%s status=%s found=true", id, job.Status)
	return &job, true
}

func (s *RedisStore) Update(job *models.Job) {
	data, err := json.Marshal(job)
	if err != nil {
		log.Printf("redis store update marshal failed: job_id=%s error=%v", job.ID, err)
		return
	}

	ctx := context.Background()
	err = s.Client.Set(ctx, jobKey(job.ID), data, jobTTL).Err()
	if err != nil {
		log.Printf("redis store update failed: job_id=%s error=%v", job.ID, err)
		return
	}

	log.Printf("redis store update: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
}

func (s *RedisStore) Delete(id string) {
	ctx := context.Background()
	err := s.Client.Del(ctx, jobKey(id)).Err()
	if err != nil {
		log.Printf("redis store delete failed: job_id=%s error=%v", id, err)
		return
	}

	log.Printf("redis store delete: job_id=%s", id)
}

func (s *RedisStore) Cleanup(ttl time.Duration) int {
	ctx := context.Background()
	iter := s.Client.Scan(ctx, 0, "job:*", 100).Iterator()

	removed := 0
	now := time.Now()

	for iter.Next(ctx) {
		key := iter.Val()

		data, err := s.Client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var job models.Job
		if err := json.Unmarshal([]byte(data), &job); err != nil {
			continue
		}

		if job.CompletedAt.IsZero() {
			continue
		}

		if now.Sub(job.CompletedAt) > ttl {
			if err := s.Client.Del(ctx, key).Err(); err == nil {
				removed++
			}
		}
	}

	if err := iter.Err(); err != nil {
		log.Printf("redis cleanup scan failed: error=%v", err)
	}

	return removed
}
