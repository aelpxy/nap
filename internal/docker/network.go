package docker

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

func (c *Client) EnsureNetwork() error {
	networks, err := c.cli.NetworkList(c.ctx, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == InternalNetwork {
			return nil
		}
	}

	_, err = c.cli.NetworkCreate(c.ctx, InternalNetwork, network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: InternalNetworkSubnet,
				},
			},
		},
		Labels: map[string]string{
			"yap.managed": "true",
			"yap.type":    "network",
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	return nil
}

func (c *Client) GetNetworkConfig() *network.NetworkingConfig {
	return &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			InternalNetwork: {},
		},
	}
}

func GetPortBindings(internalPort int, hostPort int) (nat.PortSet, nat.PortMap) {
	portSet := nat.PortSet{
		nat.Port(fmt.Sprintf("%d/tcp", internalPort)): struct{}{},
	}

	portMap := nat.PortMap{
		nat.Port(fmt.Sprintf("%d/tcp", internalPort)): []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", hostPort),
			},
		},
	}

	return portSet, portMap
}

func GetHostConfig(portMap nat.PortMap) *container.HostConfig {
	return &container.HostConfig{
		PortBindings: portMap,
		AutoRemove:   false,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}
}
