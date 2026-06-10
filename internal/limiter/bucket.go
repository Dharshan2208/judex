package limiter

import (
	"sync"
	"time"
)

type Bucket struct {
	Tokens     float64 `json:"tokens"`
	LastRefill int64   `json:"last_refill"`
}

type Manager struct {
	mu sync.Mutex

	buckets map[string]*Bucket

	capacity   float64
	refillRate float64
}

func NewManager(capacity, refillRate float64) *Manager {
	return &Manager{
		buckets:    make(map[string]*Bucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

func (m *Manager) Allow(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UnixMilli()

	// Getting or creating the bucket
	bucket, exists := m.buckets[key]
	if !exists {
		bucket = &Bucket{
			Tokens:     m.capacity,
			LastRefill: now,
		}

		m.buckets[key] = bucket
	}

	elapsed := float64(now-bucket.LastRefill) / 1000.0

	// refilling the token
	bucket.Tokens += elapsed * m.refillRate

	// checking for cap exceeding
	if bucket.Tokens > m.capacity {
		bucket.Tokens = m.capacity
	}

	// updating the refill time
	bucket.LastRefill = now

	// check if the token can be consumed
	if bucket.Tokens >= 1 {
		bucket.Tokens--
		return true
	}

	return false
}
