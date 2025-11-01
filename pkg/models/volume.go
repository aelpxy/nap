package models

import "time"

type VolumeBackup struct {
	ID          string    `json:"id"`
	AppName     string    `json:"app_name"`
	VolumeName  string    `json:"volume_name"`
	FilePath    string    `json:"file_path"`
	Size        int64     `json:"size"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
	Status      string    `json:"status"`
}

type VolumeBackupRegistry struct {
	Backups []VolumeBackup `json:"backups"`
}
