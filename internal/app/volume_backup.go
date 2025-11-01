package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/pkg/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

type VolumeBackupManager struct {
	dockerClient  *docker.Client
	volumeManager *VolumeManager
	backupDir     string
	registryPath  string
}

func NewVolumeBackupManager(dockerClient *docker.Client, volumeManager *VolumeManager) *VolumeBackupManager {
	homeDir, _ := os.UserHomeDir()
	backupDir := filepath.Join(homeDir, ".nap", "volume-backups")
	os.MkdirAll(backupDir, 0755)

	return &VolumeBackupManager{
		dockerClient:  dockerClient,
		volumeManager: volumeManager,
		backupDir:     backupDir,
		registryPath:  filepath.Join(homeDir, ".nap", "volume-registry.json"),
	}
}

func (vbm *VolumeBackupManager) BackupVolume(ctx context.Context, appName, volumeName, description string, outputPath string) (*models.VolumeBackup, error) {
	volumeInfo, err := vbm.volumeManager.InspectVolume(ctx, appName, volumeName)
	if err != nil {
		return nil, fmt.Errorf("volume not found: %w", err)
	}

	if volumeInfo.Volume.Type != "volume" {
		return nil, fmt.Errorf("cannot backup bind mount volumes, only named volumes")
	}

	backupID := fmt.Sprintf("backup-%s", time.Now().Format("20060102-150405"))

	var backupPath string
	if outputPath != "" {
		backupPath = outputPath
	} else {
		backupPath = filepath.Join(vbm.backupDir, backupID+".tar.gz")
	}

	os.MkdirAll(filepath.Dir(backupPath), 0755)

	dockerVolumeName := GetVolumeSource(appName, volumeInfo.Volume)

	err = vbm.createBackupContainer(ctx, dockerVolumeName, backupPath)
	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	backup := &models.VolumeBackup{
		ID:          backupID,
		AppName:     appName,
		VolumeName:  volumeName,
		FilePath:    backupPath,
		Size:        fileInfo.Size(),
		Description: description,
		CreatedAt:   time.Now(),
		CompletedAt: time.Now(),
		Status:      "completed",
	}

	if err := vbm.saveBackupMetadata(backup); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	return backup, nil
}

func (vbm *VolumeBackupManager) createBackupContainer(ctx context.Context, volumeName, backupPath string) error {
	backupDir := filepath.Dir(backupPath)
	backupFile := filepath.Base(backupPath)

	config := &container.Config{
		Image: "alpine:latest",
		Cmd: []string{
			"sh", "-c",
			fmt.Sprintf("tar czf /backup/%s -C /volume-data .", backupFile),
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeVolume,
				Source:   volumeName,
				Target:   "/volume-data",
				ReadOnly: true,
			},
			{
				Type:   mount.TypeBind,
				Source: backupDir,
				Target: "/backup",
			},
		},
		AutoRemove: true,
	}

	resp, err := vbm.dockerClient.GetClient().ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create backup container: %w", err)
	}

	if err := vbm.dockerClient.GetClient().ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start backup container: %w", err)
	}

	statusCh, errCh := vbm.dockerClient.GetClient().ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error waiting for backup container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("backup container exited with code %d", status.StatusCode)
		}
	}

	return nil
}

func (vbm *VolumeBackupManager) RestoreVolume(ctx context.Context, appName, volumeName, backupPath string, stopApp bool) error {
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	volumeInfo, err := vbm.volumeManager.InspectVolume(ctx, appName, volumeName)
	if err != nil {
		return fmt.Errorf("volume not found: %w", err)
	}

	if volumeInfo.Volume.Type != "volume" {
		return fmt.Errorf("cannot restore to bind mount volumes, only named volumes")
	}

	if volumeInfo.UsedBy > 0 && !stopApp {
		return fmt.Errorf("volume is in use by %d container(s). stop the application first: nap app stop %s", volumeInfo.UsedBy, appName)
	}

	dockerVolumeName := GetVolumeSource(appName, volumeInfo.Volume)

	err = vbm.createRestoreContainer(ctx, dockerVolumeName, backupPath)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	return nil
}

func (vbm *VolumeBackupManager) createRestoreContainer(ctx context.Context, volumeName, backupPath string) error {
	backupDir := filepath.Dir(backupPath)
	backupFile := filepath.Base(backupPath)

	config := &container.Config{
		Image: "alpine:latest",
		Cmd: []string{
			"sh", "-c",
			fmt.Sprintf("rm -rf /volume-data/* /volume-data/..?* /volume-data/.[!.]* 2>/dev/null || true && tar xzf /backup/%s -C /volume-data", backupFile),
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/volume-data",
			},
			{
				Type:     mount.TypeBind,
				Source:   backupDir,
				Target:   "/backup",
				ReadOnly: true,
			},
		},
		AutoRemove: true,
	}

	resp, err := vbm.dockerClient.GetClient().ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create restore container: %w", err)
	}

	if err := vbm.dockerClient.GetClient().ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start restore container: %w", err)
	}

	statusCh, errCh := vbm.dockerClient.GetClient().ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error waiting for restore container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("restore container exited with code %d", status.StatusCode)
		}
	}

	return nil
}

func (vbm *VolumeBackupManager) ListBackups(appName, volumeName string) ([]models.VolumeBackup, error) {
	registry, err := vbm.loadBackupRegistry()
	if err != nil {
		return nil, err
	}

	var filtered []models.VolumeBackup
	for _, backup := range registry.Backups {
		if appName != "" && backup.AppName != appName {
			continue
		}
		if volumeName != "" && backup.VolumeName != volumeName {
			continue
		}
		filtered = append(filtered, backup)
	}

	return filtered, nil
}

func (vbm *VolumeBackupManager) DeleteBackup(backupID string) error {
	registry, err := vbm.loadBackupRegistry()
	if err != nil {
		return err
	}

	var backup *models.VolumeBackup
	var newBackups []models.VolumeBackup

	for _, b := range registry.Backups {
		if b.ID == backupID {
			backup = &b
		} else {
			newBackups = append(newBackups, b)
		}
	}

	if backup == nil {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete backup file: %w", err)
	}

	registry.Backups = newBackups
	return vbm.saveBackupRegistry(registry)
}

func (vbm *VolumeBackupManager) saveBackupMetadata(backup *models.VolumeBackup) error {
	registry, err := vbm.loadBackupRegistry()
	if err != nil {
		registry = &models.VolumeBackupRegistry{
			Backups: []models.VolumeBackup{},
		}
	}

	registry.Backups = append(registry.Backups, *backup)
	return vbm.saveBackupRegistry(registry)
}

func (vbm *VolumeBackupManager) loadBackupRegistry() (*models.VolumeBackupRegistry, error) {
	data, err := os.ReadFile(vbm.registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.VolumeBackupRegistry{Backups: []models.VolumeBackup{}}, nil
		}
		return nil, err
	}

	var registry models.VolumeBackupRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}

	return &registry, nil
}

func (vbm *VolumeBackupManager) saveBackupRegistry(registry *models.VolumeBackupRegistry) error {
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vbm.registryPath, data, 0644)
}
