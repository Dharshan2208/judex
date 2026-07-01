package middleware

import (
	"net/http"
	"time"

	"github.com/Dharshan2208/judex/internal/logutil"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		defer func() {
			logutil.Info(r.Context(), "request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", lrw.statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"client_ip", getClientIP(r),
			)
		}()

		next.ServeHTTP(lrw, r)
	})
}
