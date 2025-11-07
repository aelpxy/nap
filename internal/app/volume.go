package app

import (
	"context"
	"fmt"
	"time"

	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
)

type VolumeManager struct {
	dockerClient *docker.Client
	registry     *RegistryManager
}

func NewVolumeManager(dockerClient *docker.Client, registry *RegistryManager) *VolumeManager {
	return &VolumeManager{
		dockerClient: dockerClient,
		registry:     registry,
	}
}

func (vm *VolumeManager) AddVolume(ctx context.Context, appName string, vol models.Volume) error {
	if vol.Name == "" {
		return fmt.Errorf("volume name cannot be empty")
	}
	if vol.MountPath == "" {
		return fmt.Errorf("mount path cannot be empty")
	}

	lockMgr := GetGlobalLockManager()
	if err := lockMgr.TryLock(appName, 10*time.Second); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lockMgr.Unlock(appName)

	application, err := vm.registry.Get(appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	for _, existing := range application.Volumes {
		if existing.Name == vol.Name {
			return fmt.Errorf("volume '%s' already exists for this application", vol.Name)
		}
	}

	if vol.Type == "volume" || vol.Type == "" {
		vol.Type = "volume" // default
		dockerVolumeName := getDockerVolumeName(appName, vol.Name)

		volumeCreateOptions := volume.CreateOptions{
			Name: dockerVolumeName,
			Labels: map[string]string{
				"yap.managed":  "true",
				"yap.app.name": appName,
				"yap.vol.name": vol.Name,
			},
		}

		_, err := vm.dockerClient.GetClient().VolumeCreate(ctx, volumeCreateOptions)
		if err != nil {
			return fmt.Errorf("failed to create volume: %w", err)
		}
	}

	vol.CreatedAt = time.Now()
	application.Volumes = append(application.Volumes, vol)

	if err := vm.registry.Update(*application); err != nil {
		if vol.Type == "volume" {
			dockerVolumeName := getDockerVolumeName(appName, vol.Name)
			vm.dockerClient.GetClient().VolumeRemove(ctx, dockerVolumeName, false)
		}
		return fmt.Errorf("failed to update registry: %w", err)
	}

	return nil
}

func (vm *VolumeManager) RemoveVolume(ctx context.Context, appName, volumeName string, deleteData bool) error {
	lockMgr := GetGlobalLockManager()
	if err := lockMgr.TryLock(appName, 10*time.Second); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lockMgr.Unlock(appName)

	application, err := vm.registry.Get(appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	var removedVolume *models.Volume
	newVolumes := make([]models.Volume, 0)
	for _, vol := range application.Volumes {
		if vol.Name == volumeName {
			removedVolume = &vol
		} else {
			newVolumes = append(newVolumes, vol)
		}
	}

	if removedVolume == nil {
		return fmt.Errorf("volume '%s' not found", volumeName)
	}

	application.Volumes = newVolumes
	if err := vm.registry.Update(*application); err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	if deleteData && removedVolume.Type == "volume" {
		dockerVolumeName := getDockerVolumeName(appName, volumeName)
		err := vm.dockerClient.GetClient().VolumeRemove(ctx, dockerVolumeName, false)
		if err != nil {
			return fmt.Errorf("failed to delete volume data: %w", err)
		}
	}

	return nil
}

func (vm *VolumeManager) ListVolumes(ctx context.Context, appName string) ([]models.Volume, error) {
	application, err := vm.registry.Get(appName)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	return application.Volumes, nil
}

func (vm *VolumeManager) InspectVolume(ctx context.Context, appName, volumeName string) (*models.VolumeInfo, error) {
	application, err := vm.registry.Get(appName)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	var vol *models.Volume
	for _, v := range application.Volumes {
		if v.Name == volumeName {
			vol = &v
			break
		}
	}

	if vol == nil {
		return nil, fmt.Errorf("volume '%s' not found", volumeName)
	}

	info := &models.VolumeInfo{
		Volume: *vol,
	}

	if vol.Type == "volume" {
		dockerVolumeName := getDockerVolumeName(appName, volumeName)
		info.DockerName = dockerVolumeName

		dockerVol, err := vm.dockerClient.GetClient().VolumeInspect(ctx, dockerVolumeName)
		if err == nil {
			info.Driver = dockerVol.Driver
			info.MountPoint = dockerVol.Mountpoint

			containers, err := vm.dockerClient.GetClient().ContainerList(ctx, container.ListOptions{
				All: true,
			})
			if err == nil {
				usedBy := 0
				for _, container := range containers {
					for _, mount := range container.Mounts {
						if mount.Name == dockerVolumeName {
							usedBy++
							break
						}
					}
				}
				info.UsedBy = usedBy
			}
		}
	} else if vol.Type == "bind" {
		info.DockerName = vol.Source
		info.MountPoint = vol.Source
	}

	return info, nil
}

func getDockerVolumeName(appName, volumeName string) string {
	return fmt.Sprintf("yap-vol-%s-%s", appName, volumeName)
}

func GetVolumeSource(appName string, vol models.Volume) string {
	if vol.Type == "bind" {
		return vol.Source
	}
	return getDockerVolumeName(appName, vol.Name)
}
