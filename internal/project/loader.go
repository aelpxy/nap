package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/aelpxy/nap/pkg/models"
)

func LoadConfig(projectPath string) (*models.ProjectConfig, error) {
	configPath := filepath.Join(projectPath, "nap.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("nap.toml not found in %s", projectPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read nap.toml: %w", err)
	}

	var config models.ProjectConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse nap.toml: %w", err)
	}

	if err := validateAndSetDefaults(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func LoadConfigIfExists(projectPath string) (*models.ProjectConfig, error) {
	configPath := filepath.Join(projectPath, "nap.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // Not an error, just doesn't exist
	}

	return LoadConfig(projectPath)
}

func validateAndSetDefaults(config *models.ProjectConfig) error {
	if config.Deploy.Instances == 0 {
		config.Deploy.Instances = 1
	}
	if config.Deploy.Memory == "" {
		config.Deploy.Memory = "512M"
	}
	if config.Deploy.CPU == 0 {
		config.Deploy.CPU = 0.5
	}
	if config.Deploy.Port == 0 {
		config.Deploy.Port = 3000 // default port
	}

	if config.Deploy.HealthCheck.Path == "" {
		config.Deploy.HealthCheck.Path = "/health"
	}
	if config.Deploy.HealthCheck.Interval == 0 {
		config.Deploy.HealthCheck.Interval = 10
	}
	if config.Deploy.HealthCheck.Timeout == 0 {
		config.Deploy.HealthCheck.Timeout = 5
	}
	if config.Deploy.HealthCheck.Retries == 0 {
		config.Deploy.HealthCheck.Retries = 3
	}

	if config.Deployment.Strategy == "" {
		config.Deployment.Strategy = "recreate"
	}
	if config.Deployment.MaxSurge == 0 {
		config.Deployment.MaxSurge = 1
	}
	if config.Deployment.RollingInterval == 0 {
		config.Deployment.RollingInterval = 5
	}
	if config.Deployment.HealthTimeout == 0 {
		config.Deployment.HealthTimeout = 30
	}
	if config.Deployment.ConfirmationTimeout == 0 {
		config.Deployment.ConfirmationTimeout = 300
	}

	validStrategies := map[string]bool{
		"recreate":   true,
		"rolling":    true,
		"blue-green": true,
	}
	if !validStrategies[config.Deployment.Strategy] {
		return fmt.Errorf("invalid deployment strategy: %s (must be recreate, rolling, or blue-green)", config.Deployment.Strategy)
	}

	if config.Deploy.Instances < 1 {
		return fmt.Errorf("instances must be at least 1, got: %d", config.Deploy.Instances)
	}

	if config.Deploy.AutoScaling {
		if config.Scaling.MinInstances == 0 {
			config.Scaling.MinInstances = 1
		}
		if config.Scaling.MaxInstances == 0 {
			config.Scaling.MaxInstances = 10
		}
		if config.Scaling.MinInstances > config.Scaling.MaxInstances {
			return fmt.Errorf("scaling: min_instances (%d) cannot be greater than max_instances (%d)",
				config.Scaling.MinInstances, config.Scaling.MaxInstances)
		}
	}

	return nil
}

func MergeWithFlags(config *models.ProjectConfig, flagsProvided map[string]bool, flagValues map[string]interface{}) {

	if flagsProvided["instances"] {
		config.Deploy.Instances = flagValues["instances"].(int)
	}
	if flagsProvided["memory"] {
		config.Deploy.Memory = fmt.Sprintf("%dM", flagValues["memory"].(int))
	}
	if flagsProvided["cpu"] {
		config.Deploy.CPU = flagValues["cpu"].(float64)
	}
	if flagsProvided["port"] {
		config.Deploy.Port = flagValues["port"].(int)
	}
	if flagsProvided["strategy"] {
		config.Deployment.Strategy = flagValues["strategy"].(string)
	}
	if flagsProvided["health-path"] {
		config.Deploy.HealthCheck.Path = flagValues["health-path"].(string)
	}
	if flagsProvided["health-interval"] {
		config.Deploy.HealthCheck.Interval = flagValues["health-interval"].(int)
	}
	if flagsProvided["health-timeout"] {
		config.Deploy.HealthCheck.Timeout = flagValues["health-timeout"].(int)
	}
}
