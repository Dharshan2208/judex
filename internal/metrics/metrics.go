package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Buckets tuned for this service:
// - API responses are fast (ms), but queue-full / Redis stalls can push to seconds.
// - Code execution is slow (100ms - 10s, Java worst).
var (
	apiLatencyBuckets  = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	execLatencyBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30}
)

var (
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_request_duration_seconds",
			Help:    "End-to-end HTTP request latency per endpoint and response status.",
			Buckets: apiLatencyBuckets,
		},
		[]string{"endpoint", "status"},
	)

	// HttpRequestsTotal mirrors RequestDuration as a counter so RPS /
	// error-rate panels don't need histogram_count gymnastics.
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "judex_http_requests_total",
			Help: "Total HTTP requests per endpoint, method and response status.",
		},
		[]string{"endpoint", "method", "status"},
	)

	RateLimitedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "judex_rate_limited_total",
			Help: "Total requests rejected by the Redis token-bucket rate limiter.",
		},
		[]string{"endpoint"},
	)

	QueueFullTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "judex_queue_full_total",
			Help: "Total submissions rejected because the queue is at capacity.",
		},
		[]string{"language"},
	)

	ExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_execution_duration_seconds",
			Help:    "Total code execution latency per language.",
			Buckets: execLatencyBuckets,
		},
		[]string{"language"},
	)

	CompileDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_compile_duration_seconds",
			Help:    "Compilation step latency per language.",
			Buckets: execLatencyBuckets,
		},
		[]string{"language"},
	)

	RunDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_run_duration_seconds",
			Help:    "Program execution step latency per language.",
			Buckets: execLatencyBuckets,
		},
		[]string{"language"},
	)

	// QueueWaitDuration measures time from job creation to worker claim.
	// Under load this separates "API is slow" from "queue is backed up".
	QueueWaitDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_queue_wait_seconds",
			Help:    "Time a job spends waiting in the queue before a worker claims it.",
			Buckets: execLatencyBuckets,
		},
		[]string{"language"},
	)

	JobsSubmittedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "judex_jobs_submitted_total",
			Help: "Total jobs accepted into the queue per language.",
		},
		[]string{"language"},
	)

	// JobsFinishedTotal uses status=completed|compile_error|runtime_error|timeout|internal_error|...
	JobsFinishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "judex_jobs_finished_total",
			Help: "Total jobs finished by the worker per language and final status.",
		},
		[]string{"language", "status"},
	)

	QueuePending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "judex_queue_pending",
			Help: "Current number of jobs in the Redis pending queue.",
		},
	)

	QueueProcessing = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "judex_queue_processing",
			Help: "Current number of jobs in the Redis processing queue.",
		},
	)

	WorkerJobsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "judex_worker_jobs_in_flight",
			Help: "Jobs currently being processed by worker goroutines.",
		},
	)
)
