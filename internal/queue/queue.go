package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/models"
	"github.com/redis/go-redis/v9"
)

const queueName = "jobs:queue"

type Queue struct {
	Client   *redis.Client
	Name     string
	Capacity int64
}

func NewQueue(client *redis.Client, size int64) *Queue {
	log.Printf("creating job queue: size=%d", size)

	return &Queue{
		Client:   client,
		Name:     queueName,
		Capacity: size,
	}
}

func (q *Queue) Push(job *models.Job) {
	log.Printf("queue push: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
	q.TryPush(job)
}

func (q *Queue) TryPush(job *models.Job) bool {
	ctx := context.Background()

	length, err := q.Client.LLen(ctx, q.Name).Result()
	if err != nil {
		log.Printf("queue length failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	if length >= q.Capacity {
		log.Printf("queue full: rejected job_id=%s language=%s length=%d capacity=%d", job.ID, job.Language, length, q.Capacity)
		return false
	}

	data, err := json.Marshal(job)
	if err != nil {
		log.Printf("queue marshal failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	if err := q.Client.LPush(ctx, q.Name, data).Err(); err != nil {
		log.Printf("queue push failed: job_id=%s error=%v", job.ID, err)
		return false
	}

	log.Printf("queue push: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
	return true
}

func (q *Queue) Pop() *models.Job {
	ctx := context.Background()

	for {
		result, err := q.Client.BRPop(ctx, 0*time.Second, q.Name).Result()
		if err != nil {
			log.Printf("queue pop failed: error=%v", err)
			time.Sleep(time.Second)
			continue
		}

		if len(result) != 2 {
			log.Printf("queue pop returned unexpected result: length=%d", len(result))
			continue
		}

		var job models.Job
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			log.Printf("queue unmarshal failed: error=%v", err)
			continue
		}

		log.Printf("queue pop: job_id=%s status=%s language=%s", job.ID, job.Status, job.Language)
		return &job
	}
}

func (q *Queue) Len() int64 {
	length, err := q.Client.LLen(context.Background(), q.Name).Result()
	if err != nil {
		log.Printf("queue length failed: error=%v", err)
		return 0
	}

	return length
}

func (q *Queue) Cap() int64 {
	return q.Capacity
}
