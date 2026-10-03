package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/handler"
	"github.com/Dharshan2208/judex/internal/limiter"
	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/middleware"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func envFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func main() {
	logutil.Init("api", slog.LevelInfo)

	application := app.NewAPI()
	application.Queue.StartMetricsExporter(5 * time.Second)

	// Stress-test knob, no code edits needed:
	//   RATE_LIMIT_BURST=10 RATE_LIMIT_RPS=1     (default, prod-like)
	//   RATE_LIMIT_BURST=1000 RATE_LIMIT_RPS=500 (exec throughput test)
	//   RATE_LIMIT_ENABLED=false                (limiter fully off)
	burst := envFloat("RATE_LIMIT_BURST", 10)
	rps := envFloat("RATE_LIMIT_RPS", 1)
	enabled := os.Getenv("RATE_LIMIT_ENABLED") != "false"
	if burst <= 0 || rps <= 0 {
		enabled = false
	}
	ratelimiter := limiter.NewRedisManager(application.Redis, burst, rps)
	logutil.Info(context.Background(), "rate limiter config",
		"enabled", enabled,
		"burst", burst,
		"rps", rps,
	)

	submit := http.HandlerFunc(handler.SubmitHandler(application))
	var runHandler http.Handler = submit
	if enabled {
		runHandler = middleware.RateLimit(ratelimiter)(submit)
	}

	mux := http.NewServeMux()

	mux.Handle("/judex/metrics", promhttp.Handler())

	mux.Handle(
		"/judex/run",
		middleware.RequestID(
			middleware.Logging(
				middleware.PrometheusMetrics(
					middleware.CORS(
						runHandler,
					),
				),
			),
		),
	)

	mux.Handle(
		"/judex/result/",
		middleware.RequestID(
			middleware.Logging(
				middleware.PrometheusMetrics(
					middleware.CORS(
						http.HandlerFunc(handler.ResultHandler(application)),
					),
				),
			),
		),
	)

	mux.Handle(
		"/health",
		middleware.RequestID(
			middleware.Logging(
				middleware.PrometheusMetrics(
					middleware.CORS(
						http.HandlerFunc(handler.HealthHandler(application)),
					),
				),
			),
		),
	)

	logutil.Info(context.Background(), "api server starting", "addr", ":8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		logutil.Fatal(context.Background(), "http server failed", "error", err)
	}
}
