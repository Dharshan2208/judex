package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/app"
	"github.com/Dharshan2208/code-compiler/internal/cleanup"
	"github.com/Dharshan2208/code-compiler/internal/handler"
)

func main() {
	application := app.New()
	cleanup.Start(application.Store, 15*time.Minute)
	application.Pool.Start()

	application.Queue.StartRecovery(application.Store, 5*time.Minute)

	http.HandleFunc("/run", handler.SubmitHandler(application))
	http.HandleFunc("/result/", handler.ResultHandler(application))
	http.HandleFunc("/health", handler.HealthHandler(application))

	log.Println("server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
