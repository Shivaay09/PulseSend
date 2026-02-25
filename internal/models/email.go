package models

import "time"

type EmailStatus string

const (
	StatusPending    EmailStatus = "pending"
	StatusProcessing EmailStatus = "processing"
	StatusSent       EmailStatus = "sent"
	StatusFailed     EmailStatus = "failed"
)

type EmailJob struct {
	ID       int64                  `json:"id"`
	To       string                 `json:"to"`
	Subject  string                 `json:"subject"`
	Template string                 `json:"template"`
	Data     map[string]interface{} `json:"data"`

	Status   EmailStatus `json:"status"`
	Retries  int         `json:"retries"`
	ErrorMsg string      `json:"error_msg,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
