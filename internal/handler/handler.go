package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/metrics"
	"github.com/Dharshan2208/judex/internal/models"
)

func SubmitHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req models.RunRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logutil.Error(ctx, "submit: invalid request body",
				"error", err,
			)
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

		application.Store.Add(ctx, job)

		if ok := application.Queue.TryPush(ctx, job); !ok {
			application.Store.Delete(ctx, job.ID)
			metrics.QueueFullTotal.WithLabelValues(job.Language).Inc()
			logutil.Warn(ctx, "submit: queue full, job rejected",
				"job_id", job.ID,
				"language", job.Language,
			)
			http.Error(w, "queue is full", http.StatusTooManyRequests)
			return
		}

		application.Stats.IncSubmitted()
		metrics.JobsSubmittedTotal.WithLabelValues(job.Language).Inc()
		logutil.Info(ctx, "job submitted",
			"job_id", job.ID,
			"language", job.Language,
		)

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
		ctx := r.Context()

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/judex/result/")

		job, exists := application.Store.Get(ctx, id)
		if !exists {
			logutil.Warn(ctx, "result: job not found",
				"job_id", id,
			)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}

		logutil.Info(ctx, "result returned",
			"job_id", job.ID,
			"status", job.Status,
		)

		response := models.NewJobResponse(job)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func HealthHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		submitted, completed, failed := application.Stats.Snapshot()

		resp := models.HealthResponse{
			Status: "ok",

			QueueLength: int(application.Queue.Len(ctx)),
			QueueCap:    int(application.Queue.Cap()),

			Submitted: submitted,
			Completed: completed,
			Failed:    failed,
		}

		logutil.Info(ctx, "health check",
			"queue_length", resp.QueueLength,
			"queue_capacity", resp.QueueCap,
			"submitted", resp.Submitted,
			"completed", resp.Completed,
			"failed", resp.Failed,
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
