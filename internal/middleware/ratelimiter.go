package middleware

import (
	"net/http"

	"github.com/Dharshan2208/judex/internal/limiter"
	"github.com/Dharshan2208/judex/internal/logutil"
)

func RateLimit(limiter *limiter.RedisManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if !limiter.Allow(ip) {
				logutil.Warn("rate limit exceeded: client_ip=%s path=%s method=%s", ip, r.URL.Path, r.Method)
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
