package models

type RunRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type RunResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Status string `json:"status"`

	Language string `json:"language"`
}
