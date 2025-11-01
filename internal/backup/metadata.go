package backup

import (
	"time"
)

type Backup struct {
	ID           string    `json:"id"`
	DatabaseName string    `json:"database_name"`
	DatabaseType string    `json:"database_type"`
	DatabaseID   string    `json:"database_id"`
	VPC          string    `json:"vpc"`
	CreatedAt    time.Time `json:"created_at"`
	SizeBytes    int64     `json:"size_bytes"`
	Compressed   bool      `json:"compressed"`
	Version      string    `json:"version"`
	Status       string    `json:"status"`
	Description  string    `json:"description,omitempty"`
	Path         string    `json:"path"`
}

const (
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)
