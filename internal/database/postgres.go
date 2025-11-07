package database

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

const (
	postgresImage       = "postgres:16-alpine"
	postgresDefaultPort = 5432
	postgresUser        = "postgres"
	postgresDB          = "main"
)

type PostgresProvisioner struct {
	dockerClient *docker.Client
	registry     *RegistryManager
}

func NewPostgresProvisioner(dockerClient *docker.Client, registry *RegistryManager) *PostgresProvisioner {
	return &PostgresProvisioner{
		dockerClient: dockerClient,
		registry:     registry,
	}
}

func (p *PostgresProvisioner) Provision(name string, password string, vpc string) (*models.Database, error) {
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

	exists, _, err := p.dockerClient.VPCNetworkExists(vpc)
	if err != nil {
		return nil, fmt.Errorf("failed to check VPC network: %w", err)
	}

	if !exists {
		vpcModel, err := p.dockerClient.CreateVPC(vpc)
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
			_ = p.dockerClient.DeleteVPC(vpcModel.NetworkID)
			return nil, fmt.Errorf("failed to add VPC to registry: %w", err)
		}
	}

	if err := p.dockerClient.CreateVolume(volumeName, "postgres", name, dbID); err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	if err := p.dockerClient.PullImage(postgresImage, os.Stdout); err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	config := &container.Config{
		Image: postgresImage,
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", postgresUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
			fmt.Sprintf("POSTGRES_DB=%s", postgresDB),
		},
		Labels: map[string]string{
			"yap.managed": "true",
			"yap.type":    "database",
			"yap.db.type": "postgres",
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
				Target: "/var/lib/postgresql/data",
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	networkConfig := p.dockerClient.GetVPCNetworkConfig(vpc)

	containerID, err := p.dockerClient.CreateContainer(config, hostConfig, networkConfig, containerName)
	if err != nil {
		_ = p.dockerClient.DeleteVolume(volumeName)
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	if err := p.dockerClient.StartContainer(containerID); err != nil {
		_ = p.dockerClient.RemoveContainer(containerID)
		_ = p.dockerClient.DeleteVolume(volumeName)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	internalConnectionString := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		postgresUser, password, containerName, postgresDefaultPort, postgresDB)

	db := &models.Database{
		ID:            dbID,
		Name:          name,
		Type:          models.DatabaseTypePostgres,
		ContainerID:   containerID,
		ContainerName: containerName,
		VolumeName:    volumeName,
		InternalPort:  postgresDefaultPort,
		Username:      postgresUser,
		Password:      password,
		DatabaseName:  postgresDB,

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

	if err := p.registry.Add(*db); err != nil {
		_ = p.dockerClient.StopContainer(containerID)
		_ = p.dockerClient.RemoveContainer(containerID)
		_ = p.dockerClient.DeleteVolume(volumeName)
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

func generatePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
