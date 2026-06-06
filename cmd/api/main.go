package main

import (
	"log"
	"net/http"

	"github.com/Dharshan2208/code-compiler/internal/app"
	"github.com/Dharshan2208/code-compiler/internal/handler"
)

func main() {
	application := app.NewAPI()

	http.HandleFunc("/run", handler.SubmitHandler(application))
	http.HandleFunc("/result/", handler.ResultHandler(application))
	http.HandleFunc("/health", handler.HealthHandler(application))

	log.Println("api running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
