package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/cleanup"
	"github.com/Dharshan2208/judex/internal/logutil"
)

func main() {
	logutil.Init("worker", slog.LevelInfo)

	application := app.NewWorker()

	cleanup.Start(application.Store, 15*time.Minute)
	application.Queue.StartRecovery(application.Store, 5*time.Minute)
	application.Pool.Start()

	logutil.Info(context.Background(), "worker service running (with warm pool)")
	select {}
}
