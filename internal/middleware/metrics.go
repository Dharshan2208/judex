package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Dharshan2208/judex/internal/metrics"
)

func PrometheusMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		endpoint := normalizeEndpoint(r.URL.Path)
		status := strconv.Itoa(lrw.statusCode)

		metrics.RequestDuration.WithLabelValues(
			endpoint,
			status,
		).Observe(time.Since(start).Seconds())
		metrics.HttpRequestsTotal.WithLabelValues(
			endpoint,
			r.Method,
			status,
		).Inc()
	})
}

// normalizeEndpoint keeps Prometheus label cardinality bounded.
// Without this, /judex/result/{job_id} creates one series per job
// and kills Prometheus during a load test.
func normalizeEndpoint(path string) string {
	if strings.HasPrefix(path, "/judex/result/") {
		return "/judex/result/{id}"
	}
	return path
}
