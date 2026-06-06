package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/models"
	"github.com/Dharshan2208/code-compiler/internal/store"
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
	log.Printf("creating job queue: size=%d", size)

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
		log.Printf("queue length failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	processingLen, err := q.Client.LLen(ctx, q.Running).Result()
	if err != nil {
		log.Printf("processing queue length failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	if pendingLen+processingLen >= q.Capacity {
		log.Printf("queue full: rejected job_id=%s length=%d capacity=%d", job.ID, pendingLen+processingLen, q.Capacity)
		return false
	}

	data, err := json.Marshal(job)
	if err != nil {
		log.Printf("queue marshal failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	if err := q.Client.LPush(ctx, q.Pending, data).Err(); err != nil {
		log.Printf("queue push failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	log.Printf("queue push: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
	return true
}

func (q *Queue) Claim() *ClaimedJob {
	ctx := context.Background()

	for {
		// right and then left so it'll pop the oldest first like fifo behavior
		raw, err := q.Client.BLMove(ctx, q.Pending, q.Running, "RIGHT", "LEFT", 0*time.Second).Result()
		if err != nil {
			log.Printf("queue claim failed : error=%v", err)
			time.Sleep(time.Second)
			continue
		}

		var job models.Job
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			log.Printf("queue unmarshal failed: error=%v", err)
			q.Client.LRem(ctx, q.Running, 1, raw)
			continue
		}

		log.Printf("queue claimed: job_id=%s language=%s", job.ID, job.Language)

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
		log.Printf("queue ack failed: error=%v", err)
		return
	}

	if removed == 0 {
		log.Printf("queue ack warning: job was not found in processing queue")
	}
}

func (q *Queue) Len() int64 {
	length, err := q.Client.LLen(context.Background(), q.Pending).Result()
	if err != nil {
		log.Printf("queue length failed: error=%v", err)
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
		log.Printf("processing queue length failed: error=%v", err)
		return 0
	}

	return length
}

func (q *Queue) StartRecovery(s *store.RedisStore, timeout time.Duration) {
	go func() {
		//  Tickeris a built-in function used to execute an action repeatedly at regular time intervals.
		// It instantiates and returns a new *time.Ticker struct containing a channel (C) that receives continuous time signals
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			q.recoverStruckjobs(s, timeout)
		}
	}()
}

func (q *Queue) recoverStruckjobs(s *store.RedisStore, timeout time.Duration) {
	ctx := context.Background()

	items, err := q.Client.LRange(ctx, q.Running, 0, -1).Result()
	if err != nil {
		log.Printf("recovery scan failed: error=%v", err)
		return
	}

	now := time.Now()

	for _, raw := range items {
		var queuedJob models.Job

		if err := json.Unmarshal([]byte(raw), &queuedJob); err != nil {
			log.Printf("recovery unmarshal failed: error=%v", err)
			continue
		}

		storedJob, exists := s.Get(queuedJob.ID)
		if !exists {
			q.Client.LRem(ctx, q.Running, 1, raw)
			continue
		}

		if storedJob.Status != "running" {
			continue
		}

		if now.Sub(storedJob.ClaimedAt) < timeout {
			continue
		}

		storedJob.Status = "pending"
		storedJob.ClaimedAt = time.Time{}
		s.Update(storedJob)

		removed, err := q.Client.LRem(ctx, q.Running, 1, raw).Result()
		if err != nil {
			log.Printf("recovery remove failed: job_id=%s error=%v", queuedJob.ID, err)
			continue
		}

		if removed == 0 {
			continue
		}

		if err := q.Client.LPush(ctx, q.Pending, raw).Err(); err != nil {
			log.Printf("recovery requeue failed: job_id=%s error=%v", queuedJob.ID, err)
			continue
		}

		log.Printf("recovered stuck job: job_id=%s", queuedJob.ID)
	}
}
