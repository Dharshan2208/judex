package models

type SubmitResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}
