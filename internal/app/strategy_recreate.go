package app

import (
	"context"
	"fmt"
	"time"

	"github.com/aelpxy/yap/internal/docker"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

type RecreateStrategy struct {
	dockerClient *docker.Client
}

func NewRecreateStrategy() *RecreateStrategy {
	return &RecreateStrategy{}
}

func (s *RecreateStrategy) Validate(opts DeploymentOptions) error {
	if opts.App == nil {
		return fmt.Errorf("application configuration is required")
	}
	if opts.NewImageID == "" {
		return fmt.Errorf("image ID is required")
	}
	return nil
}

func (s *RecreateStrategy) Deploy(opts DeploymentOptions) (string, error) {
	var err error
	s.dockerClient, err = docker.NewClient()
	if err != nil {
		return "", fmt.Errorf("failed to initialize docker client: %w", err)
	}

	ctx := context.Background()

	if len(opts.App.ContainerIDs) > 0 {
		fmt.Println("  --> stopping old instances...")
		for _, containerID := range opts.App.ContainerIDs {
			timeout := 10
			if err := s.dockerClient.GetClient().ContainerStop(ctx, containerID, dockerTypes.StopOptions{
				Timeout: &timeout,
			}); err != nil {
				fmt.Printf("    [warn] failed to stop container %s: %v\n", containerID[:12], err)
			}

			if err := s.dockerClient.GetClient().ContainerRemove(ctx, containerID, dockerTypes.RemoveOptions{
				Force: true,
			}); err != nil {
				fmt.Printf("    [warn] failed to remove container %s: %v\n", containerID[:12], err)
			}
		}
		fmt.Println("  [done] old instances stopped")
	}

	fmt.Println("  --> deploying instances...")

	containerIDs := make([]string, 0, opts.App.Instances)

	for i := 1; i <= opts.App.Instances; i++ {
		containerID, err := s.createInstance(ctx, opts, i)
		if err != nil {
			s.cleanup(ctx, containerIDs)
			return "", fmt.Errorf("failed to create instance %d: %w", i, err)
		}
		containerIDs = append(containerIDs, containerID)
	}

	opts.App.ContainerIDs = containerIDs

	fmt.Println("  [done] instances deployed")

	time.Sleep(2 * time.Second)

	return opts.NewImageID, nil
}

func (s *RecreateStrategy) createInstance(ctx context.Context, opts DeploymentOptions, instanceNum int) (string, error) {
	containerName := fmt.Sprintf("yap-app-%s-%d", opts.App.Name, instanceNum)

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

func (s *RecreateStrategy) cleanup(ctx context.Context, containerIDs []string) {
	for _, id := range containerIDs {
		s.dockerClient.GetClient().ContainerRemove(ctx, id, dockerTypes.RemoveOptions{
			Force: true,
		})
	}
}
