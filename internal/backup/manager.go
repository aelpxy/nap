package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/pkg/models"
)

type Manager struct {
	registry     *BackupRegistry
	dockerClient *docker.Client
	backupsDir   string
}

func NewManager(dockerClient *docker.Client) (*Manager, error) {
	registry, err := NewBackupRegistry()
	if err != nil {
		return nil, err
	}

	if err := registry.Initialize(); err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	backupsDir := filepath.Join(homeDir, ".nap", "backups")

	return &Manager{
		registry:     registry,
		dockerClient: dockerClient,
		backupsDir:   backupsDir,
	}, nil
}

func (m *Manager) CreateBackup(db *models.Database, compress bool, description string) (*Backup, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupID := fmt.Sprintf("%s-%s", db.Name, timestamp)

	backupPath := filepath.Join(m.backupsDir, backupID)
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	backup := &Backup{
		ID:           backupID,
		DatabaseName: db.Name,
		DatabaseType: string(db.Type),
		DatabaseID:   db.ID,
		VPC:          db.VPC,
		CreatedAt:    time.Now(),
		Compressed:   compress,
		Status:       StatusInProgress,
		Description:  description,
		Path:         backupPath,
	}

	if err := m.registry.Add(*backup); err != nil {
		return nil, fmt.Errorf("failed to add backup to registry: %w", err)
	}

	var err error
	var version string
	var size int64

	switch db.Type {
	case "postgres":
		version, size, err = m.backupPostgres(db, backupPath, compress)
	case "valkey":
		version, size, err = m.backupValkey(db, backupPath, compress)
	default:
		err = fmt.Errorf("unsupported database type: %s", db.Type)
	}

	backup.Version = version
	backup.SizeBytes = size

	if err != nil {
		backup.Status = StatusFailed
		m.registry.Update(*backup)
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	backup.Status = StatusCompleted
	if err := m.registry.Update(*backup); err != nil {
		return nil, fmt.Errorf("failed to update backup: %w", err)
	}

	return backup, nil
}

func (m *Manager) RestoreBackup(db *models.Database, backupID string) error {
	backup, err := m.registry.Get(backupID)
	if err != nil {
		return err
	}

	if backup.DatabaseType != string(db.Type) {
		return fmt.Errorf("backup type mismatch: backup is for %s, database is %s", backup.DatabaseType, db.Type)
	}

	switch db.Type {
	case "postgres":
		return m.restorePostgres(db, backup)
	case "valkey":
		return m.restoreValkey(db, backup)
	default:
		return fmt.Errorf("unsupported database type: %s", db.Type)
	}
}

func (m *Manager) ListBackups(databaseName string) []Backup {
	return m.registry.List(databaseName)
}

func (m *Manager) GetBackup(id string) (*Backup, error) {
	return m.registry.Get(id)
}

func (m *Manager) DeleteBackup(id string) error {
	backup, err := m.registry.Get(id)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(backup.Path); err != nil {
		return fmt.Errorf("failed to delete backup directory: %w", err)
	}

	return m.registry.Delete(id)
}
