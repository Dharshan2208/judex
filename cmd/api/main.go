package main

import (
	"log"
	"net/http"

	"github.com/Dharshan2208/code-compiler/internal/app"
	"github.com/Dharshan2208/code-compiler/internal/handler"
	"github.com/Dharshan2208/code-compiler/internal/limiter"
	"github.com/Dharshan2208/code-compiler/internal/middleware"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("[api] ")

	application := app.NewAPI()

	ratelimiter := limiter.NewManager(10, 1)

	http.Handle(
		"/run",
		middleware.RateLimit(ratelimiter)(
			http.HandlerFunc(handler.SubmitHandler(application)),
		),
	)

	http.HandleFunc("/result/", handler.ResultHandler(application))
	http.HandleFunc("/health", handler.HealthHandler(application))

	log.Println("api server starting: addr=:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
