package database

import (
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

const (
	valkeyImage       = "valkey/valkey:8-alpine"
	valkeyDefaultPort = 6379
	valkeyUser        = "default"
)

type ValkeyProvisioner struct {
	dockerClient *docker.Client
	registry     *RegistryManager
}

func NewValkeyProvisioner(dockerClient *docker.Client, registry *RegistryManager) *ValkeyProvisioner {
	return &ValkeyProvisioner{
		dockerClient: dockerClient,
		registry:     registry,
	}
}

func (v *ValkeyProvisioner) Provision(name string, password string, vpc string) (*models.Database, error) {
	dbID := GenerateID("db")
	containerName := fmt.Sprintf("yap-db-%s", name)
	volumeName := fmt.Sprintf("yap-vol-%s", dbID)

	if password == "" {
		var err error
		password, err = generatePassword(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate password: %w", err)
		}
	}

	exists, _, err := v.dockerClient.VPCNetworkExists(vpc)
	if err != nil {
		return nil, fmt.Errorf("failed to check VPC network: %w", err)
	}

	if !exists {
		vpcModel, err := v.dockerClient.CreateVPC(vpc)
		if err != nil {
			return nil, fmt.Errorf("failed to create VPC network: %w", err)
		}

		vpcRegistry, err := NewVPCRegistryManager()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize VPC registry: %w", err)
		}
		if err := vpcRegistry.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize VPC registry: %w", err)
		}
		if err := vpcRegistry.Add(*vpcModel); err != nil {
			_ = v.dockerClient.DeleteVPC(vpcModel.NetworkID)
			return nil, fmt.Errorf("failed to add VPC to registry: %w", err)
		}
	}

	if err := v.dockerClient.CreateVolume(volumeName, "valkey", name, dbID); err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	if err := v.dockerClient.PullImage(valkeyImage, os.Stdout); err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	config := &container.Config{
		Image: valkeyImage,
		Cmd: []string{
			"valkey-server",
			"--requirepass", password,
			"--appendonly", "yes",
		},
		Labels: map[string]string{
			"yap.managed": "true",
			"yap.type":    "database",
			"yap.db.type": "valkey",
			"yap.db.name": name,
			"yap.db.id":   dbID,
			"yap.vpc":     vpc,
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/data",
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	networkConfig := v.dockerClient.GetVPCNetworkConfig(vpc)

	containerID, err := v.dockerClient.CreateContainer(config, hostConfig, networkConfig, containerName)
	if err != nil {
		_ = v.dockerClient.DeleteVolume(volumeName)
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	if err := v.dockerClient.StartContainer(containerID); err != nil {
		_ = v.dockerClient.RemoveContainer(containerID)
		_ = v.dockerClient.DeleteVolume(volumeName)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	internalConnectionString := fmt.Sprintf("valkey://%s:%s@%s:%d",
		valkeyUser, password, containerName, valkeyDefaultPort)

	db := &models.Database{
		ID:            dbID,
		Name:          name,
		Type:          models.DatabaseTypeValkey,
		ContainerID:   containerID,
		ContainerName: containerName,
		VolumeName:    volumeName,
		InternalPort:  valkeyDefaultPort,
		Username:      valkeyUser,
		Password:      password,
		DatabaseName:  "", // Valkey doesn't have database names

		VPC:                       vpc,
		Published:                 false,
		PublishedPort:             0,
		InternalHostname:          containerName,
		ConnectionString:          internalConnectionString,
		PublishedConnectionString: "",

		Network: vpc + ".yap-vpc-network",
		Port:    0,
		Host:    "",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    models.DatabaseStatusRunning,
	}

	if err := v.registry.Add(*db); err != nil {
		_ = v.dockerClient.StopContainer(containerID)
		_ = v.dockerClient.RemoveContainer(containerID)
		_ = v.dockerClient.DeleteVolume(volumeName)
		return nil, fmt.Errorf("failed to add to registry: %w", err)
	}

	vpcRegistry, err := NewVPCRegistryManager()
	if err == nil {
		if err := vpcRegistry.Initialize(); err == nil {
			_ = vpcRegistry.AddDatabaseToVPC(vpc, dbID)
		}
	}

	return db, nil
}
