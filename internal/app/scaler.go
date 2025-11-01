package app

import (
	"context"
	"fmt"

	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/internal/router"
	"github.com/aelpxy/nap/pkg/models"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

func ScaleUp(
	dockerClient *docker.Client,
	app *models.Application,
	count int,
	vpcNetworkName string,
) ([]string, error) {
	ctx := context.Background()
	newContainerIDs := make([]string, 0, count)

	currentCount := len(app.ContainerIDs)

	traefik := router.NewTraefikManager(dockerClient)

	for i := 0; i < count; i++ {
		instanceNum := currentCount + i + 1
		containerName := fmt.Sprintf("nap-app-%s-%d", app.Name, instanceNum)

		traefikLabels := traefik.GenerateLabelsForApp(app)

		labels := map[string]string{
			"nap.managed":      "true",
			"nap.type":         "app",
			"nap.app.name":     app.Name,
			"nap.app.id":       app.ID,
			"nap.vpc":          app.VPC,
			"nap.app.instance": fmt.Sprintf("%d", instanceNum),
		}
		for k, v := range traefikLabels {
			labels[k] = v
		}

		envVars := make(map[string]string)
		for k, v := range app.EnvVars {
			envVars[k] = v
		}
		InjectNapMetadata(envVars, app.ID, instanceNum, "local")
		envArray := BuildEnvArray(envVars)

		containerConfig := &dockerTypes.Config{
			Image:  app.ImageID,
			Labels: labels,
			Env:    envArray,
		}

		mounts := prepareVolumeMounts(app.Name, app.Volumes)

		hostConfig := &dockerTypes.HostConfig{
			RestartPolicy: dockerTypes.RestartPolicy{
				Name: "unless-stopped",
			},
			Resources: dockerTypes.Resources{
				Memory:   int64(app.Memory) * 1024 * 1024,
				NanoCPUs: int64(app.CPU * 1e9),
			},
			Mounts: mounts,
		}

		networkConfig := &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				vpcNetworkName: {},
			},
		}

		resp, err := dockerClient.GetClient().ContainerCreate(
			ctx,
			containerConfig,
			hostConfig,
			networkConfig,
			nil,
			containerName,
		)
		if err != nil {
			return newContainerIDs, fmt.Errorf("failed to create instance %d: %w", instanceNum, err)
		}

		if err := dockerClient.GetClient().ContainerStart(ctx, resp.ID, dockerTypes.StartOptions{}); err != nil {
			return newContainerIDs, fmt.Errorf("failed to start instance %d: %w", instanceNum, err)
		}

		newContainerIDs = append(newContainerIDs, resp.ID)
	}

	return newContainerIDs, nil
}

func ScaleDown(
	dockerClient *docker.Client,
	app *models.Application,
	count int,
) error {
	ctx := context.Background()

	if count >= len(app.ContainerIDs) {
		return fmt.Errorf("cannot remove %d instances (only %d running, must keep at least 1)", count, len(app.ContainerIDs))
	}

	toRemove := app.ContainerIDs[len(app.ContainerIDs)-count:]

	for _, containerID := range toRemove {
		timeout := 10
		if err := dockerClient.GetClient().ContainerStop(ctx, containerID, dockerTypes.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}

		if err := dockerClient.GetClient().ContainerRemove(ctx, containerID, dockerTypes.RemoveOptions{
			Force: true,
		}); err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	return nil
}
