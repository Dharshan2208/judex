package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_request_duration_seconds",
			Help:    "End-to-end HTTP request latency per endpoint and response status.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "status"},
	)

	ExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_execution_duration_seconds",
			Help:    "Total code execution latency per language.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"language"},
	)

	CompileDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_compile_duration_seconds",
			Help:    "Compilation step latency per language.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"language"},
	)

	RunDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "judex_run_duration_seconds",
			Help:    "Program execution step latency per language.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"language"},
	)
)
