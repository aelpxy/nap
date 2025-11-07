package app

import (
	"fmt"

	"github.com/aelpxy/yap/pkg/models"
)

type Deployer struct {
	strategy DeploymentStrategy
}

func NewDeployer(strategyType models.DeploymentStrategy) (*Deployer, error) {
	var strategy DeploymentStrategy

	switch strategyType {
	case models.DeploymentStrategyRecreate:
		strategy = NewRecreateStrategy()
	case models.DeploymentStrategyRolling:
		strategy = NewRollingStrategy()
	case models.DeploymentStrategyBlueGreen:
		strategy = NewBlueGreenStrategy()
	default:
		return nil, fmt.Errorf("unknown deployment strategy: %s", strategyType)
	}

	return &Deployer{
		strategy: strategy,
	}, nil
}

func (d *Deployer) Deploy(opts DeploymentOptions) (string, error) {
	if err := d.strategy.Validate(opts); err != nil {
		return "", fmt.Errorf("deployment validation failed: %w", err)
	}

	imageID, err := d.strategy.Deploy(opts)
	if err != nil {
		return "", fmt.Errorf("deployment failed: %w", err)
	}

	return imageID, nil
}
