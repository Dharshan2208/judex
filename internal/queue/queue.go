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
	logutil.Info(context.Background(), "creating job queue",
		"size", size,
	)

	return &Queue{
		Client:   client,
		Pending:  pendingJobsQueue,
		Running:  processingJobsQueue,
		Capacity: size,
	}
}

// TryPush attempts to push a job onto the pending queue.
// Returns false if the queue is at capacity.
func (q *Queue) TryPush(ctx context.Context, job *models.Job) bool {
	pendingLen, err := q.Client.LLen(ctx, q.Pending).Result()
	if err != nil {
		logutil.Error(ctx, "queue: failed to get pending length",
			"job_id", job.ID,
			"error", err,
		)
		return false
	}

	processingLen, err := q.Client.LLen(ctx, q.Running).Result()
	if err != nil {
		logutil.Error(ctx, "queue: failed to get processing length",
			"job_id", job.ID,
			"error", err,
		)
		return false
	}

	total := pendingLen + processingLen
	if total >= q.Capacity {
		logutil.Warn(ctx, "queue: full, rejecting job",
			"job_id", job.ID,
			"current_length", total,
			"capacity", q.Capacity,
		)
		return false
	}

	data, err := json.Marshal(job)
	if err != nil {
		logutil.Error(ctx, "queue: marshal failed",
			"job_id", job.ID,
			"error", err,
		)
		return false
	}

	if err := q.Client.LPush(ctx, q.Pending, data).Err(); err != nil {
		logutil.Error(ctx, "queue: push failed",
			"job_id", job.ID,
			"error", err,
		)
		return false
	}

	logutil.Info(ctx, "queue: job pushed",
		"job_id", job.ID,
		"status", job.Status,
		"language", job.Language,
	)
	return true
}

func (q *Queue) Claim() *ClaimedJob {
	ctx := context.Background()

	for {
		raw, err := q.Client.BLMove(ctx, q.Pending, q.Running, "RIGHT", "LEFT", 0*time.Second).Result()
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				logutil.Debug(ctx, "queue: claim cancelled",
					"error", err,
				)
				return nil
			}
			logutil.Error(ctx, "queue: claim failed",
				"error", err,
			)
			time.Sleep(time.Second)
			continue
		}

		var job models.Job
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			logutil.Error(ctx, "queue: unmarshal failed on claimed job",
				"raw_data_len", len(raw),
				"error", err,
			)
			q.Client.LRem(ctx, q.Running, 1, raw)
			continue
		}

		logutil.Info(ctx, "queue: job claimed",
			"job_id", job.ID,
			"language", job.Language,
		)

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
		logutil.Error(ctx, "queue: ack failed",
			"error", err,
		)
		return
	}

	if removed == 0 {
		logutil.Warn(ctx, "queue: ack warning — job not in processing queue",
			"raw_data_len", len(raw),
		)
	} else {
		logutil.Debug(ctx, "queue: job acknowledged",
			"raw_data_len", len(raw),
			"removed", removed,
		)
	}
}

// Len returns the number of jobs in the pending queue.
func (q *Queue) Len(ctx context.Context) int64 {
	length, err := q.Client.LLen(ctx, q.Pending).Result()
	if err != nil {
		logutil.Error(ctx, "queue: failed to get pending length",
			"error", err,
		)
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
		logutil.Error(context.Background(), "queue: failed to get processing length",
			"error", err,
		)
		return 0
	}
	return length
}

// StartRecovery launches a background goroutine that periodically checks for
// stuck jobs (in processing for longer than timeout) and re-queues them.
func (q *Queue) StartRecovery(s *store.RedisStore, timeout time.Duration) {
	logutil.Info(context.Background(), "queue: starting recovery",
		"timeout", timeout,
		"interval", time.Minute,
	)

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
	logutil.Debug(ctx, "queue: running stuck job recovery",
		"timeout", timeout,
	)

	items, err := q.Client.LRange(ctx, q.Running, 0, -1).Result()
	if err != nil {
		logutil.Error(ctx, "queue: recovery scan failed",
			"error", err,
		)
		return
	}

	now := time.Now()

	for _, raw := range items {
		var queuedJob models.Job

		if err := json.Unmarshal([]byte(raw), &queuedJob); err != nil {
			logutil.Error(ctx, "queue: recovery unmarshal failed",
				"raw_data_len", len(raw),
				"error", err,
			)
			q.Client.LRem(ctx, q.Running, 1, raw)
			continue
		}

		storedJob, exists := s.Get(ctx, queuedJob.ID)
		if !exists {
			logutil.Warn(ctx, "queue: recovery — job not in store, removing",
				"job_id", queuedJob.ID,
			)
			q.Client.LRem(ctx, q.Running, 1, raw)
			continue
		}

		if storedJob.Status != "running" {
			logutil.Debug(ctx, "queue: recovery — job not running, skipping",
				"job_id", storedJob.ID,
				"current_status", storedJob.Status,
			)
			continue
		}

		if now.Sub(storedJob.ClaimedAt) < timeout {
			logutil.Debug(ctx, "queue: recovery — job not yet timed out, skipping",
				"job_id", storedJob.ID,
				"claimed_at", storedJob.ClaimedAt,
				"timeout", timeout,
			)
			continue
		}

		storedJob.Status = "pending"
		storedJob.ClaimedAt = time.Time{}
		s.Update(ctx, storedJob)

		logutil.Warn(ctx, "queue: recovered stuck job",
			"job_id", storedJob.ID,
			"status_before", "running",
		)

		removed, err := q.Client.LRem(ctx, q.Running, 1, raw).Result()
		if err != nil {
			logutil.Error(ctx, "queue: recovery remove from processing failed",
				"job_id", queuedJob.ID,
				"error", err,
			)
			continue
		}

		if removed == 0 {
			logutil.Warn(ctx, "queue: recovery — job already gone from processing",
				"job_id", queuedJob.ID,
			)
			continue
		}

		if err := q.Client.LPush(ctx, q.Pending, raw).Err(); err != nil {
			logutil.Error(ctx, "queue: recovery requeue failed",
				"job_id", queuedJob.ID,
				"error", err,
			)
			continue
		}

		logutil.Info(ctx, "queue: recovered and requeued stuck job",
			"job_id", queuedJob.ID,
		)
	}
}
