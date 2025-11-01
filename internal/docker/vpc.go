package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/aelpxy/nap/pkg/models"
	"github.com/docker/docker/api/types/network"
)

const (
	VPCNetworkSuffix = ".nap-vpc-network"
)

func (c *Client) CreateVPC(vpcName string) (*models.VPC, error) {
	ctx, cancel := context.WithTimeout(c.ctx, NetworkOpTimeout)
	defer cancel()

	networkName := vpcName + VPCNetworkSuffix

	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == networkName {
			return nil, fmt.Errorf("VPC network %s already exists", networkName)
		}
	}

	subnet, err := c.allocateSubnet()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate subnet: %w", err)
	}

	resp, err := c.cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: subnet,
				},
			},
		},
		Labels: map[string]string{
			"nap.managed":  "true",
			"nap.type":     "vpc",
			"nap.vpc.name": vpcName,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create VPC network: %w", err)
	}

	return &models.VPC{
		Name:        vpcName,
		NetworkID:   resp.ID,
		NetworkName: networkName,
		Subnet:      subnet,
		CreatedAt:   time.Now(),
		Databases:   []string{},
		Apps:        []string{},
	}, nil
}

func (c *Client) DeleteVPC(networkID string) error {
	ctx, cancel := context.WithTimeout(c.ctx, NetworkOpTimeout)
	defer cancel()

	err := c.cli.NetworkRemove(ctx, networkID)
	if err != nil {
		return fmt.Errorf("failed to delete VPC network %s: %w", networkID, err)
	}
	return nil
}

func (c *Client) GetVPCNetworkConfig(vpcName string) *network.NetworkingConfig {
	networkName := vpcName + VPCNetworkSuffix
	return &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}
}

func (c *Client) VPCNetworkExists(vpcName string) (bool, string, error) {
	networkName := vpcName + VPCNetworkSuffix

	networks, err := c.cli.NetworkList(c.ctx, network.ListOptions{})
	if err != nil {
		return false, "", fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == networkName {
			return true, net.ID, nil
		}
	}

	return false, "", nil
}

func (c *Client) allocateSubnet() (string, error) {
	networks, err := c.cli.NetworkList(c.ctx, network.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	usedOctets := make(map[int]bool)
	for _, net := range networks {
		if len(net.IPAM.Config) > 0 {
			subnet := net.IPAM.Config[0].Subnet
			var octet int
			if _, err := fmt.Sscanf(subnet, "172.%d.0.0/16", &octet); err == nil {
				usedOctets[octet] = true
			}
		}
	}

	for octet := 20; octet <= 254; octet++ {
		if !usedOctets[octet] {
			return fmt.Sprintf("172.%d.0.0/16", octet), nil
		}
	}

	return "", fmt.Errorf("subnet exhausted: too many VPCs")
}
