package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/cleanup"
	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logutil.Init("worker", slog.LevelInfo)

	application := app.NewWorker()

	cleanup.Start(application.Store, 15*time.Minute)
	application.Queue.StartRecovery(application.Store, 5*time.Minute)
	application.Pool.Start()

	logutil.Info(context.Background(), "worker service running (with warm pool)")

	// exposing the prometheus metrics on port 8081
	http.Handle("/judex/metrics", promhttp.Handler())
	go func() {
		logutil.Info(context.Background(), "worker metrics listening", "addr", ":8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			logutil.Fatal(context.Background(), "worker metrics server failed", "error", err)
		}
	}()

	select {}
}
