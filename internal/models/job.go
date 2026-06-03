package models

import "time"

type Job struct {
	ID       string `json:"id"`
	Language string `json:"language"`
	Code     string `json:"-"`
	Status   string `json:"status"`

	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`

	Result RunResponse `json:"result"`
}
