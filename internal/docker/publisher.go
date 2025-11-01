package docker

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

func (c *Client) RecreateContainerWithPorts(
	containerName string,
	image string,
	env []string,
	cmd []string,
	labels map[string]string,
	volumeName string,
	volumeTarget string,
	networkName string,
	internalPort int,
	hostPort int,
) (string, error) {
	containerPort := nat.Port(fmt.Sprintf("%d/tcp", internalPort))
	portBindings := nat.PortMap{
		containerPort: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", hostPort),
			},
		},
	}

	config := &container.Config{
		Image:  image,
		Env:    env,
		Cmd:    cmd,
		Labels: labels,
		ExposedPorts: nat.PortSet{
			containerPort: struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: volumeTarget,
			},
		},
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}

	resp, err := c.cli.ContainerCreate(c.ctx, config, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

func (c *Client) RemovePortBindings(
	containerName string,
	image string,
	env []string,
	cmd []string,
	labels map[string]string,
	volumeName string,
	volumeTarget string,
	networkName string,
) (string, error) {
	config := &container.Config{
		Image:  image,
		Env:    env,
		Cmd:    cmd,
		Labels: labels,
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: volumeTarget,
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}

	resp, err := c.cli.ContainerCreate(c.ctx, config, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}
