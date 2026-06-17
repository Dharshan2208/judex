package main

import (
	"net/http"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/handler"
	"github.com/Dharshan2208/judex/internal/limiter"
	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/middleware"
)

func main() {
	logutil.Init("API")

	application := app.NewAPI()

	ratelimiter := limiter.NewRedisManager(application.Redis, 10, 1)

	http.Handle(
		"/run",
		middleware.RateLimit(ratelimiter)(
			http.HandlerFunc(handler.SubmitHandler(application)),
		),
	)

	http.HandleFunc("/result/", handler.ResultHandler(application))
	http.HandleFunc("/health", handler.HealthHandler(application))

	logutil.Info("api server starting: addr=:8080")
	logutil.Fatal("http server failed: %v", http.ListenAndServe(":8080", nil))
}
