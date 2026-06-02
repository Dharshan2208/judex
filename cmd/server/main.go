package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Dharshan2208/code-compiler/internal/app"
	"github.com/Dharshan2208/code-compiler/internal/models"
)

func submitHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.RunRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		jobID := uuid.New().String()

		job := &models.Job{
			ID:       jobID,
			Language: req.Language,
			Code:     req.Code,
			Status:   "pending",
		}

		application.Store.Add(job)
		application.Queue.Push(job)

		response := models.SubmitResponse{
			JobID:  jobID,
			Status: "pending",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func resultHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/result/")

		job, exists := application.Store.Get(id)
		if !exists {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	}
}

func main() {
	application := app.New()
	application.Pool.Start()

	http.HandleFunc("/run", submitHandler(application))
	http.HandleFunc("/result/", resultHandler(application))

	log.Println("server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
