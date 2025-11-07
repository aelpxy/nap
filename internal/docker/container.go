package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
)

type PullProgress struct {
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	Progress string `json:"progress"`
	ID       string `json:"id"`
}

func (c *Client) PullImage(imageName string, progressWriter io.Writer) error {
	ctx, cancel := context.WithTimeout(c.ctx, ImagePullTimeout)
	defer cancel()

	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	var lastStatus string
	layerProgress := make(map[string]int64)

	for scanner.Scan() {
		var progress PullProgress
		if err := json.Unmarshal(scanner.Bytes(), &progress); err != nil {
			continue
		}

		if progress.ID != "" && progress.ProgressDetail.Total > 0 {
			layerProgress[progress.ID] = progress.ProgressDetail.Current
		}

		if progress.Status != lastStatus && progress.ID == "" {
			if progressWriter != nil {
				statusMsg := progress.Status
				if strings.Contains(statusMsg, "Digest:") || strings.Contains(statusMsg, "Status:") {
					continue // skip
				}
				fmt.Fprintf(progressWriter, "  %s\n", statusMsg)
			}
			lastStatus = progress.Status
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	return nil
}

func (c *Client) CreateContainer(config *container.Config, hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig, containerName string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, ContainerOpTimeout)
	defer cancel()

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container %s: %w", containerName, err)
	}

	return resp.ID, nil
}

func (c *Client) StartContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(c.ctx, ContainerOpTimeout)
	defer cancel()

	err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}

	return nil
}

func (c *Client) StopContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(c.ctx, ContainerOpTimeout)
	defer cancel()

	timeout := 10
	err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}

	return nil
}

func (c *Client) RemoveContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(c.ctx, ContainerOpTimeout)
	defer cancel()

	err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: false,
	})
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

func (c *Client) GetContainerStatus(containerID string) (string, error) {
	inspect, err := c.cli.ContainerInspect(c.ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return "not found", nil
		}
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	return inspect.State.Status, nil
}

func (c *Client) GetContainerLogs(containerID string, follow bool) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: true,
	}

	logs, err := c.cli.ContainerLogs(c.ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	return logs, nil
}

func (c *Client) ListManagedContainers() ([]types.Container, error) {
	allContainers, err := c.cli.ContainerList(c.ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	managedContainers := []types.Container{}
	for _, cont := range allContainers {
		if managed, ok := cont.Labels["yap.managed"]; ok && managed == "true" {
			managedContainers = append(managedContainers, cont)
		}
	}

	return managedContainers, nil
}

func (c *Client) ContainerExists(containerID string) (bool, error) {
	_, err := c.cli.ContainerInspect(c.ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
