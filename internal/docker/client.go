package docker

import (
	"context"
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/runtime"
	"github.com/docker/docker/client"
)

const (
	InternalNetwork       = "yap-net-internal"
	InternalNetworkSubnet = "172.20.0.0/16"
)

type Client struct {
	cli         *client.Client
	ctx         context.Context
	runtimeInfo *runtime.RuntimeInfo
}

func NewClient() (*Client, error) {
	runtimeInfo, err := runtime.DetectRuntime()
	if err != nil {
		return nil, fmt.Errorf("failed to detect container runtime: %w\nplease install docker or podman", err)
	}

	if err := runtimeInfo.EnsureSocketExists(); err != nil {
		return nil, err
	}

	if os.Getenv("DOCKER_HOST") == "" {
		os.Setenv("DOCKER_HOST", runtimeInfo.GetSocketURI())
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create container runtime client: %w", err)
	}

	return &Client{
		cli:         cli,
		ctx:         context.Background(),
		runtimeInfo: runtimeInfo,
	}, nil
}

func NewClientWithRuntime(runtimeInfo *runtime.RuntimeInfo) (*Client, error) {
	if err := runtimeInfo.EnsureSocketExists(); err != nil {
		return nil, err
	}

	os.Setenv("DOCKER_HOST", runtimeInfo.GetSocketURI())

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create container runtime client: %w", err)
	}

	return &Client{
		cli:         cli,
		ctx:         context.Background(),
		runtimeInfo: runtimeInfo,
	}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) GetContext() context.Context {
	return c.ctx
}

func (c *Client) GetClient() *client.Client {
	return c.cli
}

func (c *Client) GetRuntimeInfo() *runtime.RuntimeInfo {
	return c.runtimeInfo
}
