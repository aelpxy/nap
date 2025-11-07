package app

import (
	"context"
	"fmt"
	"time"

	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

type BlueGreenStrategy struct {
	dockerClient *docker.Client
}

func NewBlueGreenStrategy() *BlueGreenStrategy {
	return &BlueGreenStrategy{}
}

func (s *BlueGreenStrategy) Validate(opts DeploymentOptions) error {
	if opts.App == nil {
		return fmt.Errorf("application configuration is required")
	}
	if opts.NewImageID == "" {
		return fmt.Errorf("image ID is required")
	}
	if opts.App.HealthCheckPath == "" {
		return fmt.Errorf("blue-green deployment requires health check endpoint")
	}
	return nil
}

func (s *BlueGreenStrategy) Deploy(opts DeploymentOptions) (string, error) {
	var err error
	s.dockerClient, err = docker.NewClient()
	if err != nil {
		return "", fmt.Errorf("failed to initialize docker client: %w", err)
	}

	ctx := context.Background()

	currentColor := opts.App.DeploymentState.Active
	if currentColor == "" || currentColor == models.DeploymentColorDefault {
		currentColor = models.DeploymentColorBlue
	}

	var newColor models.DeploymentColor
	if currentColor == models.DeploymentColorBlue {
		newColor = models.DeploymentColorGreen
	} else {
		newColor = models.DeploymentColorBlue
	}

	fmt.Printf("  --> blue-green deployment\n")
	fmt.Printf("    current active: %s\n", currentColor)
	fmt.Printf("    deploying: %s environment (%d instances)\n", newColor, opts.App.Instances)
	fmt.Println()

	healthTimeout := time.Duration(opts.Config.HealthTimeout) * time.Second
	if healthTimeout == 0 {
		healthTimeout = 30 * time.Second
	}

	fmt.Printf("  --> deploying %s environment...\n", newColor)
	newContainerIDs := make([]string, 0, opts.App.Instances)

	for i := 1; i <= opts.App.Instances; i++ {
		fmt.Printf("    [%d/%d] deploying %s-%d...\n", i, opts.App.Instances, newColor, i)

		containerID, err := s.createInstance(ctx, opts, i, newColor, false)
		if err != nil {
			s.cleanup(ctx, newContainerIDs)
			return "", fmt.Errorf("failed to create %s instance %d: %w", newColor, i, err)
		}
		newContainerIDs = append(newContainerIDs, containerID)
	}

	fmt.Println()

	fmt.Println("  --> waiting for health checks...")
	for i, containerID := range newContainerIDs {
		instanceNum := i + 1
		fmt.Printf("    [%d/%d] checking %s-%d (timeout: %ds)...\n", instanceNum, opts.App.Instances, newColor, instanceNum, int(healthTimeout.Seconds()))

		if err := s.waitForHealthy(ctx, containerID, healthTimeout); err != nil {
			fmt.Printf("    [error] %s-%d health check failed: %v\n", newColor, instanceNum, err)
			s.cleanup(ctx, newContainerIDs)
			return "", fmt.Errorf("health check failed for %s-%d: %w", newColor, instanceNum, err)
		}

		fmt.Printf("    [%d/%d] %s-%d healthy\n", instanceNum, opts.App.Instances, newColor, instanceNum)
	}

	fmt.Println()

	fmt.Printf("  --> switching traffic to %s...\n", newColor)

	for i, containerID := range newContainerIDs {
		instanceNum := i + 1
		if err := s.updateContainerLabels(ctx, containerID, opts, instanceNum, newColor, true); err != nil {
			fmt.Printf("    [warn] failed to update labels for %s-%d: %v\n", newColor, instanceNum, err)
		}
	}

	oldEnv := s.getEnvironment(opts.App, currentColor)
	if oldEnv != nil && len(oldEnv.ContainerIDs) > 0 {
		for i, containerID := range oldEnv.ContainerIDs {
			instanceNum := i + 1
			if err := s.updateContainerLabels(ctx, containerID, opts, instanceNum, currentColor, false); err != nil {
				fmt.Printf("    [warn] failed to remove labels from %s-%d: %v\n", currentColor, instanceNum, err)
			}
		}
	}

	fmt.Printf("  [done] traffic switched to %s\n", newColor)
	fmt.Println()

	newEnv := &models.Environment{
		ContainerIDs: newContainerIDs,
		ImageID:      opts.NewImageID,
		DeployedAt:   time.Now(),
	}

	if newColor == models.DeploymentColorBlue {
		opts.App.DeploymentState.Blue = newEnv
		opts.App.DeploymentState.Active = models.DeploymentColorBlue
		opts.App.DeploymentState.Standby = models.DeploymentColorGreen
	} else {
		opts.App.DeploymentState.Green = newEnv
		opts.App.DeploymentState.Active = models.DeploymentColorGreen
		opts.App.DeploymentState.Standby = models.DeploymentColorBlue
	}

	opts.App.ContainerIDs = newContainerIDs

	if opts.Config.AutoConfirm {
		fmt.Printf("  --> auto-confirm enabled, destroying %s environment...\n", currentColor)
		if oldEnv != nil && len(oldEnv.ContainerIDs) > 0 {
			s.cleanup(ctx, oldEnv.ContainerIDs)
			if currentColor == models.DeploymentColorBlue {
				opts.App.DeploymentState.Blue = nil
			} else {
				opts.App.DeploymentState.Green = nil
			}
			opts.App.DeploymentState.Standby = ""
		}
		fmt.Printf("  [done] %s environment destroyed\n", currentColor)
	} else {
		fmt.Printf("  %s environment kept for rollback\n", currentColor)
		fmt.Println()
		fmt.Println("  to complete deployment:")
		fmt.Printf("    yap app deployment confirm %s\n", opts.App.Name)
		fmt.Println()
		fmt.Println("  to rollback:")
		fmt.Printf("    yap app deployment rollback %s\n", opts.App.Name)
	}

	fmt.Println()
	fmt.Printf("  [done] blue-green deployment completed\n")

	return opts.NewImageID, nil
}

func (s *BlueGreenStrategy) createInstance(ctx context.Context, opts DeploymentOptions, instanceNum int, color models.DeploymentColor, withTraefikLabels bool) (string, error) {
	containerName := fmt.Sprintf("yap-app-%s-%s-%d", opts.App.Name, color, instanceNum)

	labels := map[string]string{
		"yap.managed":      "true",
		"yap.type":         "app",
		"yap.app.name":     opts.App.Name,
		"yap.app.id":       opts.App.ID,
		"yap.vpc":          opts.VPCName,
		"yap.app.instance": fmt.Sprintf("%d", instanceNum),
		"yap.app.color":    string(color),
	}

	if withTraefikLabels {
		for k, v := range opts.TraefikLabels {
			labels[k] = v
		}
	}

	envVars := make(map[string]string)
	for k, v := range opts.App.EnvVars {
		envVars[k] = v
	}
	InjectMetadata(envVars, opts.App.ID, instanceNum, "local")
	envArray := BuildEnvArray(envVars)

	containerConfig := &dockerTypes.Config{
		Image:  opts.NewImageID,
		Labels: labels,
		Env:    envArray,
	}

	mounts := prepareVolumeMounts(opts.App.Name, opts.App.Volumes)

	hostConfig := &dockerTypes.HostConfig{
		RestartPolicy: dockerTypes.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: dockerTypes.Resources{
			Memory:   int64(opts.MemoryMB) * 1024 * 1024,
			NanoCPUs: int64(opts.CPUCores * 1e9),
		},
		Mounts: mounts,
	}

	vpcNetworkName := fmt.Sprintf("%s.yap-vpc-network", opts.VPCName)
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			vpcNetworkName: {},
		},
	}

	resp, err := s.dockerClient.GetClient().ContainerCreate(
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

	if err := s.dockerClient.GetClient().ContainerStart(ctx, resp.ID, dockerTypes.StartOptions{}); err != nil {
		s.dockerClient.GetClient().ContainerRemove(ctx, resp.ID, dockerTypes.RemoveOptions{Force: true})
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

func (s *BlueGreenStrategy) updateContainerLabels(ctx context.Context, containerID string, opts DeploymentOptions, instanceNum int, color models.DeploymentColor, addTraefikLabels bool) error {
	inspect, err := s.dockerClient.GetClient().ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	labels := map[string]string{
		"yap.managed":      "true",
		"yap.type":         "app",
		"yap.app.name":     opts.App.Name,
		"yap.app.id":       opts.App.ID,
		"yap.vpc":          opts.VPCName,
		"yap.app.instance": fmt.Sprintf("%d", instanceNum),
		"yap.app.color":    string(color),
	}

	if addTraefikLabels {
		for k, v := range opts.TraefikLabels {
			labels[k] = v
		}
	}

	timeout := 10
	if err := s.dockerClient.GetClient().ContainerStop(ctx, containerID, dockerTypes.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if err := s.dockerClient.GetClient().ContainerRemove(ctx, containerID, dockerTypes.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	containerName := fmt.Sprintf("yap-app-%s-%s-%d", opts.App.Name, color, instanceNum)

	containerConfig := &dockerTypes.Config{
		Image:  inspect.Config.Image,
		Labels: labels,
		Env:    inspect.Config.Env,
	}

	hostConfig := inspect.HostConfig

	vpcNetworkName := fmt.Sprintf("%s.yap-vpc-network", opts.VPCName)
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			vpcNetworkName: {},
		},
	}

	resp, err := s.dockerClient.GetClient().ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName,
	)
	if err != nil {
		return fmt.Errorf("failed to recreate container: %w", err)
	}

	if err := s.dockerClient.GetClient().ContainerStart(ctx, resp.ID, dockerTypes.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (s *BlueGreenStrategy) getEnvironment(app *models.Application, color models.DeploymentColor) *models.Environment {
	if color == models.DeploymentColorBlue {
		return app.DeploymentState.Blue
	} else if color == models.DeploymentColorGreen {
		return app.DeploymentState.Green
	}
	return nil
}

func (s *BlueGreenStrategy) waitForHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			inspect, err := s.dockerClient.GetClient().ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			if !inspect.State.Running {
				return fmt.Errorf("container stopped unexpectedly")
			}

			startedAt, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
			if err != nil {
				if inspect.State.Running {
					return nil
				}
			}

			if inspect.State.Running && time.Since(startedAt) > 5*time.Second {
				return nil
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("health check timeout exceeded")
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *BlueGreenStrategy) cleanup(ctx context.Context, containerIDs []string) {
	for _, id := range containerIDs {
		s.dockerClient.GetClient().ContainerRemove(ctx, id, dockerTypes.RemoveOptions{
			Force: true,
		})
	}
}
