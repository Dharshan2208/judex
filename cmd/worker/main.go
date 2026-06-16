package main

import (
	"log"
	"time"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/cleanup"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("[worker] ")

	application := app.NewWorker()

	cleanup.Start(application.Store, 15*time.Minute)
	application.Queue.StartRecovery(application.Store, 5*time.Minute)
	application.Pool.Start()

	log.Println("worker service running(with warm pool)")
	select {}
}
