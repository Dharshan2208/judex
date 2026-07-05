package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/handler"
	"github.com/Dharshan2208/judex/internal/limiter"
	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/middleware"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logutil.Init("api", slog.LevelInfo)

	application := app.NewAPI()
	ratelimiter := limiter.NewRedisManager(application.Redis, 10, 1)

	mux := http.NewServeMux()

	mux.Handle("/judex/metrics", promhttp.Handler())

	mux.Handle("/judex/run",
		middleware.RequestID(
			middleware.Logging(
				middleware.PrometheusMetrics(
					middleware.CORS(
						middleware.RateLimit(ratelimiter)(
							http.HandlerFunc(handler.SubmitHandler(application)),
						),
					),
				),
			),
		),
	)

	mux.Handle("/judex/result/",
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

	mux.Handle("/health",
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
