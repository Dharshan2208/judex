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

func (s *RedisStore) Add(job *models.Job) {
	data, err := json.Marshal(job)
	if err != nil {
		logutil.Error("Redis store add marshal failed: job_id=%s error=%v", job.ID, err)
		return
	}

	ctx := context.Background()
	err = s.Client.Set(ctx, jobKey(job.ID), data, jobTTL).Err()
	if err != nil {
		logutil.Error("redis store add failed: job_id=%s error=%v", job.ID, err)
		return
	}

	logutil.Info("redis store add: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
}

func (s *RedisStore) Get(id string) (*models.Job, bool) {
	ctx := context.Background()
	data, err := s.Client.Get(ctx, jobKey(id)).Result()
	if err == redis.Nil {
		logutil.Debug("redis store get: job_id=%s found=false", id)
		return nil, false
	}

	if err != nil {
		logutil.Error("redis store get failed: job_id=%s error=%v", id, err)
		return nil, false
	}

	var job models.Job
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		logutil.Error("redis store get unmarshal failed: job_id=%s error=%v raw_data_len=%d", id, err, len(data))
		return nil, false
	}

	logutil.Debug("redis store get: job_id=%s status=%s found=true", id, job.Status)
	return &job, true
}

func (s *RedisStore) Update(job *models.Job) {
	data, err := json.Marshal(job)
	if err != nil {
		logutil.Error("redis store update marshal failed: job_id=%s error=%v", job.ID, err)
		return
	}

	ctx := context.Background()
	err = s.Client.Set(ctx, jobKey(job.ID), data, jobTTL).Err()
	if err != nil {
		logutil.Error("redis store update failed: job_id=%s error=%v", job.ID, err)
		return
	}

	logutil.Info("redis store update: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
}

func (s *RedisStore) Delete(id string) {
	ctx := context.Background()
	err := s.Client.Del(ctx, jobKey(id)).Err()
	if err != nil {
		logutil.Error("redis store delete failed: job_id=%s error=%v", id, err)
		return
	}

	logutil.Info("redis store delete: job_id=%s", id)
}

func (s *RedisStore) Cleanup(ttl time.Duration) int {
	ctx := context.Background()
	iter := s.Client.Scan(ctx, 0, "job:*", 100).Iterator()

	removed := 0
	now := time.Now()
	logutil.Debug("running redis store cleanup: ttl=%v", ttl)

	for iter.Next(ctx) {
		key := iter.Val()

		data, err := s.Client.Get(ctx, key).Result()
		if err != nil {
			logutil.Error("redis store cleanup: failed to get job data for key=%s error=%v", key, err)
			continue
		}

		var job models.Job
		if err := json.Unmarshal([]byte(data), &job); err != nil {
			logutil.Error("redis store cleanup: unmarshal failed for key=%s error=%v raw_data_len=%d", key, err, len(data))
			continue
		}

		if job.CompletedAt.IsZero() {
			logutil.Debug("redis store cleanup: job not completed, skipping: job_id=%s", job.ID)
			continue
		}

		if now.Sub(job.CompletedAt) > ttl {
			if err := s.Client.Del(ctx, key).Err(); err == nil {
				removed++
				logutil.Info("redis store cleanup: removed expired job: job_id=%s", job.ID)
			} else {
				logutil.Error("redis store cleanup: failed to delete expired job: job_id=%s error=%v", job.ID, err)
			}
		} else {
			logutil.Debug("redis store cleanup: job not expired, skipping: job_id=%s completed_at=%v", job.ID, job.CompletedAt)
		}
	}

	if err := iter.Err(); err != nil {
		logutil.Error("redis cleanup scan failed: error=%v", err)
	}
	logutil.Info("redis store cleanup completed: removed_jobs=%d", removed)

	return removed
}
