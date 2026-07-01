package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/models"
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

func (s *RedisStore) Add(ctx context.Context, job *models.Job) {
	data, err := json.Marshal(job)
	if err != nil {
		logutil.Error(ctx, "store: marshal failed",
			"job_id", job.ID,
			"error", err,
		)
		return
	}

	err = s.Client.Set(ctx, jobKey(job.ID), data, jobTTL).Err()
	if err != nil {
		logutil.Error(ctx, "store: add failed",
			"job_id", job.ID,
			"error", err,
		)
		return
	}

	logutil.Info(ctx, "store: job added",
		"job_id", job.ID,
		"status", job.Status,
		"language", job.Language,
	)
}

func (s *RedisStore) Get(ctx context.Context, id string) (*models.Job, bool) {
	data, err := s.Client.Get(ctx, jobKey(id)).Result()
	if err == redis.Nil {
		logutil.Debug(ctx, "store: job not found",
			"job_id", id,
		)
		return nil, false
	}

	if err != nil {
		logutil.Error(ctx, "store: get failed",
			"job_id", id,
			"error", err,
		)
		return nil, false
	}

	var job models.Job
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		logutil.Error(ctx, "store: unmarshal failed",
			"job_id", id,
			"error", err,
			"raw_data_len", len(data),
		)
		return nil, false
	}

	logutil.Debug(ctx, "store: job found",
		"job_id", job.ID,
		"status", job.Status,
	)
	return &job, true
}

func (s *RedisStore) Update(ctx context.Context, job *models.Job) {
	data, err := json.Marshal(job)
	if err != nil {
		logutil.Error(ctx, "store: update marshal failed",
			"job_id", job.ID,
			"error", err,
		)
		return
	}

	err = s.Client.Set(ctx, jobKey(job.ID), data, jobTTL).Err()
	if err != nil {
		logutil.Error(ctx, "store: update failed",
			"job_id", job.ID,
			"error", err,
		)
		return
	}

	logutil.Info(ctx, "store: job updated",
		"job_id", job.ID,
		"status", job.Status,
		"language", job.Language,
	)
}

func (s *RedisStore) Delete(ctx context.Context, id string) {
	err := s.Client.Del(ctx, jobKey(id)).Err()
	if err != nil {
		logutil.Error(ctx, "store: delete failed",
			"job_id", id,
			"error", err,
		)
		return
	}

	logutil.Info(ctx, "store: job deleted",
		"job_id", id,
	)
}

func (s *RedisStore) Cleanup(ctx context.Context, ttl time.Duration) int {
	iter := s.Client.Scan(ctx, 0, "job:*", 100).Iterator()

	removed := 0
	now := time.Now()
	logutil.Debug(ctx, "store: running cleanup",
		"ttl", ttl,
	)

	for iter.Next(ctx) {
		key := iter.Val()

		data, err := s.Client.Get(ctx, key).Result()
		if err != nil {
			logutil.Error(ctx, "store: cleanup get failed",
				"key", key,
				"error", err,
			)
			continue
		}

		var job models.Job
		if err := json.Unmarshal([]byte(data), &job); err != nil {
			logutil.Error(ctx, "store: cleanup unmarshal failed",
				"key", key,
				"error", err,
				"raw_data_len", len(data),
			)
			continue
		}

		if job.CompletedAt.IsZero() {
			logutil.Debug(ctx, "store: cleanup skipping incomplete job",
				"job_id", job.ID,
			)
			continue
		}

		if now.Sub(job.CompletedAt) > ttl {
			if err := s.Client.Del(ctx, key).Err(); err == nil {
				removed++
				logutil.Info(ctx, "store: cleanup removed expired job",
					"job_id", job.ID,
				)
			} else {
				logutil.Error(ctx, "store: cleanup delete failed",
					"job_id", job.ID,
					"error", err,
				)
			}
		} else {
			logutil.Debug(ctx, "store: cleanup skipping unexpired job",
				"job_id", job.ID,
				"completed_at", job.CompletedAt,
			)
		}
	}

	if err := iter.Err(); err != nil {
		logutil.Error(ctx, "store: cleanup scan failed",
			"error", err,
		)
	}
	logutil.Info(ctx, "store: cleanup completed",
		"removed_jobs", removed,
	)

	return removed
}
