package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/aelpxy/nap/internal/utils"
)

type BackupRegistry struct {
	Backups []Backup `json:"backups"`
	path    string
}

func NewBackupRegistry() (*BackupRegistry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	registryPath := filepath.Join(homeDir, ".nap", "backups", "registry.json")

	return &BackupRegistry{
		Backups: []Backup{},
		path:    registryPath,
	}, nil
}

func (r *BackupRegistry) Initialize() error {
	backupsDir := filepath.Dir(r.path)
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		return fmt.Errorf("failed to create backups directory: %w", err)
	}

	if _, err := os.Stat(r.path); err == nil {
		data, err := os.ReadFile(r.path)
		if err != nil {
			return fmt.Errorf("failed to read registry: %w", err)
		}

		if err := json.Unmarshal(data, r); err != nil {
			return fmt.Errorf("failed to parse registry: %w", err)
		}
	}

	return nil
}

func (r *BackupRegistry) Save() error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := utils.AtomicWriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

func (r *BackupRegistry) Add(backup Backup) error {
	r.Backups = append(r.Backups, backup)
	return r.Save()
}

func (r *BackupRegistry) Get(id string) (*Backup, error) {
	for i := range r.Backups {
		if r.Backups[i].ID == id {
			return &r.Backups[i], nil
		}
	}
	return nil, fmt.Errorf("backup not found: %s", id)
}

func (r *BackupRegistry) List(databaseName string) []Backup {
	if databaseName == "" {
		backups := make([]Backup, len(r.Backups))
		copy(backups, r.Backups)
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].CreatedAt.After(backups[j].CreatedAt)
		})
		return backups
	}

	var filtered []Backup
	for _, backup := range r.Backups {
		if backup.DatabaseName == databaseName {
			filtered = append(filtered, backup)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	return filtered
}

func (r *BackupRegistry) Delete(id string) error {
	for i, backup := range r.Backups {
		if backup.ID == id {
			r.Backups = append(r.Backups[:i], r.Backups[i+1:]...)
			return r.Save()
		}
	}
	return fmt.Errorf("backup not found: %s", id)
}

func (r *BackupRegistry) Update(backup Backup) error {
	for i := range r.Backups {
		if r.Backups[i].ID == backup.ID {
			r.Backups[i] = backup
			return r.Save()
		}
	}
	return fmt.Errorf("backup not found: %s", backup.ID)
}
