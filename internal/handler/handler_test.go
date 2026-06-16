package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dharshan2208/judex/internal/app"
	"github.com/Dharshan2208/judex/internal/models"
	"github.com/Dharshan2208/judex/internal/queue"
	"github.com/Dharshan2208/judex/internal/store"
	"github.com/Dharshan2208/judex/tests"
)

func testApp(t *testing.T, cap int64) *app.App {
	t.Helper()
	_, client := tests.NewMiniRedisClient(t)
	return &app.App{
		Redis: client,
		Queue: queue.NewQueue(client, cap),
		Store: store.NewRedisStore(client),
		Stats: &queue.Stats{},
	}
}

func TestSubmitHandler(t *testing.T) {
	t.Run("invalid_json", func(t *testing.T) {
		application := testApp(t, 10)
		req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString("{invalid"))
		rec := httptest.NewRecorder()

		SubmitHandler(application).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("queue_full", func(t *testing.T) {
		application := testApp(t, 0)
		body, _ := json.Marshal(models.RunRequest{Language: "python", Code: "print(1)"})
		req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		SubmitHandler(application).ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		application := testApp(t, 10)
		body, _ := json.Marshal(models.RunRequest{Language: "python", Code: "print(1)"})
		req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		SubmitHandler(application).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp models.SubmitResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.JobID == "" || resp.Status != "pending" {
			t.Fatalf("unexpected submit response: %+v", resp)
		}
	})
}

func TestResultHandlerAndHealth(t *testing.T) {
	application := testApp(t, 10)
	job := tests.NewJob("job-1", "go", "completed")
	application.Store.Add(job)
	application.Stats.IncSubmitted()
	application.Stats.IncCompleted()

	t.Run("result_not_found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/result/missing", nil)
		rec := httptest.NewRecorder()
		ResultHandler(application).ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("result_found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/result/job-1", nil)
		rec := httptest.NewRecorder()
		ResultHandler(application).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		HealthHandler(application).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp models.HealthResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode health response: %v", err)
		}
		if resp.Submitted != 1 || resp.Completed != 1 {
			t.Fatalf("unexpected health counts: %+v", resp)
		}
	})
}
