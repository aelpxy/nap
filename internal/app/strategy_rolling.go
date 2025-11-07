package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aelpxy/yap/internal/docker"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

type RollingStrategy struct {
	dockerClient *docker.Client
}

func NewRollingStrategy() *RollingStrategy {
	return &RollingStrategy{}
}

func (s *RollingStrategy) Validate(opts DeploymentOptions) error {
	if opts.App == nil {
		return fmt.Errorf("application configuration is required")
	}
	if opts.NewImageID == "" {
		return fmt.Errorf("image ID is required")
	}
	if opts.App.HealthCheckPath == "" {
		return fmt.Errorf("rolling deployment requires health check endpoint")
	}
	if opts.Config.MaxSurge < 1 {
		return fmt.Errorf("max_surge must be at least 1")
	}
	return nil
}

func (s *RollingStrategy) Deploy(opts DeploymentOptions) (string, error) {
	var err error
	s.dockerClient, err = docker.NewClient()
	if err != nil {
		return "", fmt.Errorf("failed to initialize docker client: %w", err)
	}

	ctx := context.Background()

	currentInstances := opts.App.ContainerIDs
	targetInstances := opts.App.Instances
	maxSurge := opts.Config.MaxSurge
	if maxSurge < 1 {
		maxSurge = 1
	}

	fmt.Printf("  --> rolling deployment (%d instances, max surge: %d)\n", targetInstances, maxSurge)
	fmt.Println()

	rollingInterval := time.Duration(opts.Config.RollingInterval) * time.Second
	if rollingInterval == 0 {
		rollingInterval = 5 * time.Second
	}

	healthTimeout := time.Duration(opts.Config.HealthTimeout) * time.Second
	if healthTimeout == 0 {
		healthTimeout = 30 * time.Second
	}

	newContainerIDs := make([]string, 0, targetInstances)

	for i := 0; i < targetInstances; i += maxSurge {
		batchSize := maxSurge
		if i+batchSize > targetInstances {
			batchSize = targetInstances - i
		}

		batchContainerIDs := make([]string, 0, batchSize)
		for j := 0; j < batchSize; j++ {
			instanceNum := i + j + 1
			fmt.Printf("    [%d/%d] deploying instance %d...\n", instanceNum, targetInstances, instanceNum)

			containerID, err := s.createInstance(ctx, opts, instanceNum)
			if err != nil {
				s.cleanup(ctx, newContainerIDs)
				s.cleanup(ctx, batchContainerIDs)
				return "", fmt.Errorf("failed to create instance %d: %w", instanceNum, err)
			}
			batchContainerIDs = append(batchContainerIDs, containerID)
		}

		for j, containerID := range batchContainerIDs {
			instanceNum := i + j + 1
			fmt.Printf("    [%d/%d] waiting for health check (timeout: %ds)...\n", instanceNum, targetInstances, int(healthTimeout.Seconds()))

			if err := s.waitForHealthy(ctx, containerID, opts, healthTimeout); err != nil {
				fmt.Printf("    [error] instance %d health check failed: %v\n", instanceNum, err)
				s.cleanup(ctx, newContainerIDs)
				s.cleanup(ctx, batchContainerIDs)
				return "", fmt.Errorf("health check failed for instance %d: %w", instanceNum, err)
			}

			if i+j < len(currentInstances) {
				fmt.Printf("    [%d/%d] healthy! replacing old instance %d\n", instanceNum, targetInstances, instanceNum)
			} else {
				fmt.Printf("    [%d/%d] healthy! (new instance)\n", instanceNum, targetInstances)
			}
		}

		if len(currentInstances) > 0 {
			for j := 0; j < batchSize; j++ {
				if i+j < len(currentInstances) {
					instanceNum := i + j + 1
					oldContainerID := currentInstances[i+j]

					timeout := 10
					if err := s.dockerClient.GetClient().ContainerStop(ctx, oldContainerID, dockerTypes.StopOptions{
						Timeout: &timeout,
					}); err != nil {
						fmt.Printf("    [warn] failed to stop old instance %d: %v\n", instanceNum, err)
					}

					if err := s.dockerClient.GetClient().ContainerRemove(ctx, oldContainerID, dockerTypes.RemoveOptions{
						Force: true,
					}); err != nil {
						fmt.Printf("    [warn] failed to remove old instance %d: %v\n", instanceNum, err)
					}

					fmt.Printf("    [%d/%d] removed old instance %d\n", instanceNum, targetInstances, instanceNum)
				}
			}
		}

		newContainerIDs = append(newContainerIDs, batchContainerIDs...)

		if i+batchSize < targetInstances {
			fmt.Printf("\n    waiting %ds before next batch...\n\n", int(rollingInterval.Seconds()))
			time.Sleep(rollingInterval)
		}
	}

	opts.App.ContainerIDs = newContainerIDs

	fmt.Println()
	fmt.Println("  [done] rolling deployment completed")

	return opts.NewImageID, nil
}

func (s *RollingStrategy) createInstance(ctx context.Context, opts DeploymentOptions, instanceNum int) (string, error) {
	var containerName string
	if len(opts.App.ContainerIDs) > 0 {
		containerName = fmt.Sprintf("yap-app-%s-%d-%d", opts.App.Name, instanceNum, time.Now().Unix())
	} else {
		containerName = fmt.Sprintf("yap-app-%s-%d", opts.App.Name, instanceNum)
	}

	labels := map[string]string{
		"yap.managed":      "true",
		"yap.type":         "app",
		"yap.app.name":     opts.App.Name,
		"yap.app.id":       opts.App.ID,
		"yap.vpc":          opts.VPCName,
		"yap.app.instance": fmt.Sprintf("%d", instanceNum),
	}
	for k, v := range opts.TraefikLabels {
		labels[k] = v
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
			Memory:   int64(opts.MemoryMB) * 1024 * 1024, // Convert MB to bytes
			NanoCPUs: int64(opts.CPUCores * 1e9),         // Convert to nano CPUs
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

func (s *RollingStrategy) waitForHealthy(ctx context.Context, containerID string, opts DeploymentOptions, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}

	vpcNetworkName := fmt.Sprintf("%s.yap-vpc-network", opts.VPCName)

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

			var containerIP string
			if networkSettings, ok := inspect.NetworkSettings.Networks[vpcNetworkName]; ok {
				containerIP = networkSettings.IPAddress
			}

			if containerIP == "" {
				continue
			}

			healthURL := fmt.Sprintf("http://%s:%d%s", containerIP, opts.App.Port, opts.App.HealthCheckPath)

			resp, err := httpClient.Get(healthURL)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return nil // Healthy!
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("health check timeout exceeded (last error: %v)", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *RollingStrategy) cleanup(ctx context.Context, containerIDs []string) {
	for _, id := range containerIDs {
		s.dockerClient.GetClient().ContainerRemove(ctx, id, dockerTypes.RemoveOptions{
			Force: true,
		})
	}
}
