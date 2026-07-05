package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Dharshan2208/judex/internal/metrics"
)

func PrometheusMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		metrics.RequestDuration.WithLabelValues(
			r.URL.Path,
			strconv.Itoa(lrw.statusCode),
		).Observe(time.Since(start).Seconds())
	})
}
