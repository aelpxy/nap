package app

import (
	"context"
	"fmt"

	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/internal/router"
	"github.com/aelpxy/yap/pkg/models"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
)

func prepareVolumeMounts(appName string, volumes []models.Volume) []mount.Mount {
	var mounts []mount.Mount
	for _, vol := range volumes {
		volMount := mount.Mount{
			Type:     mount.Type(vol.Type),
			Source:   GetVolumeSource(appName, vol),
			Target:   vol.MountPath,
			ReadOnly: vol.ReadOnly,
		}
		mounts = append(mounts, volMount)
	}
	return mounts
}

func RecreateContainer(
	dockerClient *docker.Client,
	oldContainerID string,
	app *models.Application,
	vpcNetworkName string,
	instanceNum int,
) (string, error) {
	ctx := context.Background()

	containerInfo, err := dockerClient.GetClient().ContainerInspect(ctx, oldContainerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	timeout := 10
	if err := dockerClient.GetClient().ContainerStop(ctx, oldContainerID, dockerTypes.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return "", fmt.Errorf("failed to stop container: %w", err)
	}

	if err := dockerClient.GetClient().ContainerRemove(ctx, oldContainerID, dockerTypes.RemoveOptions{
		Force: true,
	}); err != nil {
		return "", fmt.Errorf("failed to remove container: %w", err)
	}

	envVars := make(map[string]string)
	for k, v := range app.EnvVars {
		envVars[k] = v
	}
	InjectMetadata(envVars, app.ID, instanceNum, "local")
	envArray := BuildEnvArray(envVars)

	traefik := router.NewTraefikManager(dockerClient)
	traefikLabels := traefik.GenerateLabelsForApp(app)

	labels := map[string]string{
		"yap.managed":      "true",
		"yap.type":         "app",
		"yap.app.name":     app.Name,
		"yap.app.id":       app.ID,
		"yap.vpc":          app.VPC,
		"yap.app.instance": fmt.Sprintf("%d", instanceNum),
	}
	for k, v := range traefikLabels {
		labels[k] = v
	}

	containerConfig := &dockerTypes.Config{
		Image:  containerInfo.Config.Image,
		Labels: labels,
		Env:    envArray, // NEW env vars here!
	}

	mounts := prepareVolumeMounts(app.Name, app.Volumes)

	hostConfig := &dockerTypes.HostConfig{
		RestartPolicy: dockerTypes.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: dockerTypes.Resources{
			Memory:   int64(app.Memory) * 1024 * 1024, // convert MB to bytes
			NanoCPUs: int64(app.CPU * 1e9),            // convert CPUs to nano CPUs
		},
		Mounts: mounts,
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			vpcNetworkName: {},
		},
	}

	containerName := containerInfo.Name
	if len(containerName) > 0 && containerName[0] == '/' {
		containerName = containerName[1:]
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
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := dockerClient.GetClient().ContainerStart(ctx, resp.ID, dockerTypes.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

func BuildEnvArray(envVars map[string]string) []string {
	env := make([]string, 0, len(envVars))
	for key, value := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

func InjectMetadata(envVars map[string]string, appID string, instanceNum int, region string) {
	envVars["YAP_APP_ID"] = appID
	envVars["YAP_INSTANCE_ID"] = fmt.Sprintf("%d", instanceNum)
	envVars["YAP_REGION"] = region
}
