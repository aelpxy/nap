package app

import (
	"github.com/aelpxy/nap/pkg/models"
)

type DeploymentStrategy interface {
	Deploy(opts DeploymentOptions) (string, error)

	Validate(opts DeploymentOptions) error
}

type DeploymentOptions struct {
	App *models.Application

	SourcePath string
	NewImageID string

	Config models.DeploymentConfig

	VPCName       string
	TraefikLabels map[string]string

	MemoryMB int
	CPUCores float64
}
