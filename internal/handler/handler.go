package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Dharshan2208/code-compiler/internal/app"
	"github.com/Dharshan2208/code-compiler/internal/models"
)

func SubmitHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.RunRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Printf("submit request rejected: invalid json: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		jobID := uuid.New().String()

		job := &models.Job{
			ID:        jobID,
			Language:  req.Language,
			Code:      req.Code,
			Status:    "pending",
			CreatedAt: time.Now(),
		}

		application.Store.Add(job)

		if ok := application.Queue.TryPush(job); !ok {
			application.Store.Delete(job.ID)
			log.Printf("submit request rejected: reason=queue_full job_id=%s language=%s", job.ID, job.Language)
			http.Error(w, "queue is full", http.StatusTooManyRequests)
			return
		}

		application.Stats.IncSubmitted()
		log.Printf("Job submitted: ID=%s Language=%s", job.ID, job.Language)

		response := models.SubmitResponse{
			JobID:  jobID,
			Status: "pending",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func ResultHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/result/")
		log.Printf("Result requested: ID=%s", id)

		job, exists := application.Store.Get(id)
		if !exists {
			log.Printf("result request failed: id=%s reason=job_not_found", id)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}

		log.Printf("Result returned: ID=%s status=%s", job.ID, job.Status)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	}
}

func HealthHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		submitted, completed, failed := application.Stats.Snapshot()

		resp := models.HealthResponse{
			Status: "ok",

			QueueLength: int(application.Queue.Len()),
			QueueCap:    int(application.Queue.Cap()),

			Submitted: submitted,
			Completed: completed,
			Failed:    failed,
		}

		log.Printf(
			"Health returned: status=%s queue_length=%d queue_capacity=%d submitted=%d completed=%d failed=%d",
			resp.Status,
			resp.QueueLength,
			resp.QueueCap,
			resp.Submitted,
			resp.Completed,
			resp.Failed,
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
