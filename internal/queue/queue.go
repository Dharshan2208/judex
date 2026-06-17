package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/models"
	"github.com/Dharshan2208/judex/internal/store"
	"github.com/redis/go-redis/v9"
)

// const queueName = "jobs:queue"
const (
	pendingJobsQueue    = "pending_jobs"
	processingJobsQueue = "processing_jobs"
)

// here raw is the original redis json
// we need raw to remove the excat value from processing_jobs
type ClaimedJob struct {
	Job *models.Job
	Raw string
}

type Queue struct {
	Client   *redis.Client
	Pending  string
	Running  string
	Capacity int64
}

func NewQueue(client *redis.Client, size int64) *Queue {
	logutil.Info("creating job queue: size=%d", size)

	return &Queue{
		Client:   client,
		Pending:  pendingJobsQueue,
		Running:  processingJobsQueue,
		Capacity: size,
	}
}

func (q *Queue) TryPush(job *models.Job) bool {
	ctx := context.Background()

	pendingLen, err := q.Client.LLen(ctx, q.Pending).Result()
	if err != nil {
		logutil.Error("queue pending length failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	processingLen, err := q.Client.LLen(ctx, q.Running).Result()
	if err != nil {
		logutil.Error("queue processing length failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	if pendingLen+processingLen >= q.Capacity {
		logutil.Warn("queue full: rejected job_id=%s length=%d capacity=%d", job.ID, pendingLen+processingLen, q.Capacity)
		return false
	}

	data, err := json.Marshal(job)
	if err != nil {
		logutil.Error("queue marshal failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	if err := q.Client.LPush(ctx, q.Pending, data).Err(); err != nil {
		logutil.Error("queue push failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	logutil.Info("job pushed to queue: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
	return true
}

func (q *Queue) Claim() *ClaimedJob {
	ctx := context.Background()

	for {
		raw, err := q.Client.BLMove(ctx, q.Pending, q.Running, "RIGHT", "LEFT", 0*time.Second).Result()
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				logutil.Debug("queue claim cancelled: error=%v", err)
				return nil
			}
			logutil.Error("queue claim failed: error=%v", err)
			time.Sleep(time.Second) // Addding sleep to avoid tight looping on repeated errors
			continue
		}

		var job models.Job
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			logutil.Error("queue unmarshal failed for raw job data: raw_data_len=%d error=%v", len(raw), err)
			q.Client.LRem(ctx, q.Running, 1, raw)
			continue
		}

		logutil.Info("job claimed from queue: job_id=%s language=%s", job.ID, job.Language)

		return &ClaimedJob{
			Job: &job,
			Raw: raw,
		}
	}
}

func (q *Queue) Ack(raw string) {
	ctx := context.Background()

	removed, err := q.Client.LRem(ctx, q.Running, 1, raw).Result()
	if err != nil {
		logutil.Error("queue ack failed: error=%v", err)
		return
	}

	if removed == 0 {
		logutil.Warn("queue ack warning: job was not found in processing queue for raw data: %s", raw)
	} else {
		logutil.Debug("job acknowledged: raw_data_len=%d removed=%d", len(raw), removed)
	}
}

func (q *Queue) Len() int64 {
	length, err := q.Client.LLen(context.Background(), q.Pending).Result()
	if err != nil {
		logutil.Error("queue pending length failed: error=%v", err)
		return 0
	}
	return length
}

func (q *Queue) Cap() int64 {
	return q.Capacity
}

func (q *Queue) ProcessingLen() int64 {
	length, err := q.Client.LLen(context.Background(), q.Running).Result()
	if err != nil {
		logutil.Error("queue processing length failed: error=%v", err)
		return 0
	}
	return length
}

func (q *Queue) StartRecovery(s *store.RedisStore, timeout time.Duration) {
	logutil.Info("queue recovery started: timeout=%s interval=%s", timeout, time.Minute)

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			q.recoverStuck(s, timeout)
		}
	}()
}

func (q *Queue) recoverStuck(s *store.RedisStore, timeout time.Duration) {
	ctx := context.Background()
	logutil.Debug("running queue stuck job recovery: timeout=%s", timeout)

	items, err := q.Client.LRange(ctx, q.Running, 0, -1).Result()
	if err != nil {
		logutil.Error("queue recovery scan failed: error=%v", err)
		return
	}

	now := time.Now()

	for _, raw := range items {
		var queuedJob models.Job

		if err := json.Unmarshal([]byte(raw), &queuedJob); err != nil {
			logutil.Error("queue recovery unmarshal failed for raw job data: raw_data_len=%d error=%v", len(raw), err)
			// Attempt to remove the malformed raw job from the processing queue to prevent infinite unmarshal errors
			q.Client.LRem(ctx, q.Running, 1, raw)
			continue
		}

		storedJob, exists := s.Get(queuedJob.ID)
		if !exists {
			logutil.Warn("queue recovery: job not found in store, removing from processing queue: job_id=%s", queuedJob.ID)
			q.Client.LRem(ctx, q.Running, 1, raw)
			continue
		}

		if storedJob.Status != "running" {
			logutil.Debug("queue recovery: job not in running status, skipping: job_id=%s current_status=%s", storedJob.ID, storedJob.Status)
			continue
		}

		if now.Sub(storedJob.ClaimedAt) < timeout {
			logutil.Debug("queue recovery: job not yet timed out, skipping: job_id=%s claimed_at=%v timeout=%v", storedJob.ID, storedJob.ClaimedAt, timeout)
			continue
		}

		storedJob.Status = "pending"
		storedJob.ClaimedAt = time.Time{}
		s.Update(storedJob)

		logutil.Warn("recovered stuck job: job_id=%s status_before_requeue=%s", storedJob.ID, storedJob.Status)

		removed, err := q.Client.LRem(ctx, q.Running, 1, raw).Result()
		if err != nil {
			logutil.Error("queue recovery remove failed from processing queue: job_id=%s error=%v", queuedJob.ID, err)
			continue
		}

		if removed == 0 {
			logutil.Warn("queue recovery: failed to remove job from processing queue (already gone?): job_id=%s", queuedJob.ID)
			continue
		}

		if err := q.Client.LPush(ctx, q.Pending, raw).Err(); err != nil {
			logutil.Error("queue recovery requeue failed to pending queue: job_id=%s error=%v", queuedJob.ID, err)
			continue
		}

		logutil.Info("recovered and requeued stuck job: job_id=%s", queuedJob.ID)
	}
}
