package queue

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/metrics"
)

// StartMetricsExporter polls Redis queue lengths and exposes them as
// Prometheus gauges. Run it once per process that serves /judex/metrics
// (API is enough; worker can run it too — last write wins, values agree
// because both read the same Redis lists).
func (q *Queue) StartMetricsExporter(interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	logutil.Info(context.Background(), "queue: starting metrics exporter",
		"interval", interval,
	)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			pending, err1 := q.Client.LLen(ctx, q.Pending).Result()
			processing, err2 := q.Client.LLen(ctx, q.Running).Result()
			cancel()
			if err1 == nil {
				metrics.QueuePending.Set(float64(pending))
			}
			if err2 == nil {
				metrics.QueueProcessing.Set(float64(processing))
			}
		}
	}()
}
