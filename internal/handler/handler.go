package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/models"
)

func SubmitHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		logutil.Info("request started: method=%s path=%s request_id=%s client_ip=%s", r.Method, r.URL.Path, requestID, r.RemoteAddr)

		defer func(start time.Time) {
			logutil.Info("request finished: method=%s path=%s request_id=%s duration=%v", r.Method, r.URL.Path, requestID, time.Since(start))
		}(time.Now())

		var req models.RunRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logutil.Error("submit request rejected: invalid json: %v request_id=%s client_ip=%s", err, requestID, r.RemoteAddr)
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
			logutil.Warn("submit request rejected: reason=queue_full job_id=%s language=%s request_id=%s", job.ID, job.Language, requestID)
			http.Error(w, "queue is full", http.StatusTooManyRequests)
			return
		}

		application.Stats.IncSubmitted()
		logutil.Info("job submitted: job_id=%s language=%s request_id=%s", job.ID, job.Language, requestID)

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
		requestID := uuid.New().String()
		logutil.Info("request started: method=%s path=%s request_id=%s client_ip=%s", r.Method, r.URL.Path, requestID, r.RemoteAddr)

		defer func(start time.Time) {
			logutil.Info("request finished: method=%s path=%s request_id=%s duration=%v", r.Method, r.URL.Path, requestID, time.Since(start))
		}(time.Now())

		id := strings.TrimPrefix(r.URL.Path, "/result/")
		logutil.Debug("result requested: job_id=%s request_id=%s", id, requestID)

		job, exists := application.Store.Get(id)
		if !exists {
			logutil.Warn("result request failed: job_id=%s reason=job_not_found request_id=%s", id, requestID)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}

		logutil.Info("result returned: job_id=%s status=%s request_id=%s", job.ID, job.Status, requestID)

		response := models.NewJobResponse(job)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func HealthHandler(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		logutil.Debug("request started: method=%s path=%s request_id=%s client_ip=%s", r.Method, r.URL.Path, requestID, r.RemoteAddr)

		defer func(start time.Time) {
			logutil.Debug("request finished: method=%s path=%s request_id=%s duration=%v", r.Method, r.URL.Path, requestID, time.Since(start))
		}(time.Now())

		submitted, completed, failed := application.Stats.Snapshot()

		resp := models.HealthResponse{
			Status: "ok",

			QueueLength: int(application.Queue.Len()),
			QueueCap:    int(application.Queue.Cap()),

			Submitted: submitted,
			Completed: completed,
			Failed:    failed,
		}

		logutil.Info(
			"Health returned: status=%s queue_length=%d queue_capacity=%d submitted=%d completed=%d failed=%d request_id=%s",
			resp.Status,
			resp.QueueLength,
			resp.QueueCap,
			resp.Submitted,
			resp.Completed,
			resp.Failed,
			requestID,
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
