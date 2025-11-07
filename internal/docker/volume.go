package docker

import (
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/volume"
)

func (c *Client) CreateVolume(volumeName string, dbType string, dbName string, dbID string) error {
	_, err := c.cli.VolumeCreate(c.ctx, volume.CreateOptions{
		Name:   volumeName,
		Driver: "local",
		Labels: map[string]string{
			"yap.managed": "true",
			"yap.type":    "database",
			"yap.db.type": dbType,
			"yap.db.name": dbName,
			"yap.db.id":   dbID,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	return nil
}

func (c *Client) DeleteVolume(volumeName string) error {
	err := c.cli.VolumeRemove(c.ctx, volumeName, true)
	if err != nil {
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	return nil
}

func (c *Client) VolumeExists(volumeName string) (bool, error) {
	_, err := c.cli.VolumeInspect(c.ctx, volumeName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
