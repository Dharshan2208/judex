package main

import (
	"log"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/app"
	"github.com/Dharshan2208/code-compiler/internal/cleanup"
)

func main() {
	application := app.NewWorker()

	cleanup.Start(application.Store, 15*time.Minute)
	application.Queue.StartRecovery(application.Store, 5*time.Minute)
	application.Pool.Start()

	log.Println("Worker running")
	select {}
}
