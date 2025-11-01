package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/aelpxy/nap/pkg/models"
)

type ConfigManager struct {
	configPath string
	config     *models.GlobalConfig
}

func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".nap", "config.toml")

	cm := &ConfigManager{
		configPath: configPath,
	}

	if err := cm.Load(); err != nil {
		if os.IsNotExist(err) {
			cm.config = &models.GlobalConfig{
				Publishing: models.PublishingConfig{
					Enabled: false,
				},
			}
			return cm, nil
		}
		return nil, err
	}

	return cm, nil
}

func (cm *ConfigManager) Load() error {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return err
	}

	var config models.GlobalConfig
	if _, err := toml.DecodeFile(cm.configPath, &config); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	cm.config = &config
	return nil
}

func (cm *ConfigManager) Save() error {
	napDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(napDir, 0755); err != nil {
		return fmt.Errorf("failed to create .nap directory: %w", err)
	}

	f, err := os.Create(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cm.config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

func (cm *ConfigManager) GetConfig() *models.GlobalConfig {
	return cm.config
}

func (cm *ConfigManager) ValidatePublishing() error {
	if !cm.config.Publishing.Enabled {
		return fmt.Errorf("publishing is not enabled")
	}

	if cm.config.Publishing.BaseDomain == "" {
		return fmt.Errorf("base_domain not configured")
	}

	if cm.config.Publishing.Email == "" {
		return fmt.Errorf("email not configured")
	}

	return nil
}
