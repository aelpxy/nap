package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aelpxy/nap/internal/utils"
	"github.com/aelpxy/nap/pkg/models"
)

const (
	napDir       = ".nap"
	registryFile = "apps.json"
)

var (
	mu sync.Mutex
)

type RegistryManager struct {
	path string
}

func NewRegistryManager() (*RegistryManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	napPath := filepath.Join(homeDir, napDir)

	if err := os.MkdirAll(napPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create nap directory: %w", err)
	}

	return &RegistryManager{
		path: filepath.Join(napPath, registryFile),
	}, nil
}

func (r *RegistryManager) Initialize() error {
	mu.Lock()
	defer mu.Unlock()

	if _, err := os.Stat(r.path); os.IsNotExist(err) {
		registry := models.AppRegistry{
			Applications: []models.Application{},
		}

		data, err := json.MarshalIndent(registry, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal registry: %w", err)
		}

		if err := os.WriteFile(r.path, data, 0644); err != nil {
			return fmt.Errorf("failed to write registry: %w", err)
		}
	}

	return nil
}

func (r *RegistryManager) Read() (*models.AppRegistry, error) {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var registry models.AppRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	return &registry, nil
}

func (r *RegistryManager) Write(registry *models.AppRegistry) error {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := utils.AtomicWriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

func (r *RegistryManager) Add(app models.Application) error {
	registry, err := r.Read()
	if err != nil {
		return err
	}

	for _, existing := range registry.Applications {
		if existing.Name == app.Name {
			return fmt.Errorf("application with name %s already exists", app.Name)
		}
	}

	registry.Applications = append(registry.Applications, app)

	return r.Write(registry)
}

func (r *RegistryManager) Get(name string) (*models.Application, error) {
	registry, err := r.Read()
	if err != nil {
		return nil, err
	}

	for _, app := range registry.Applications {
		if app.Name == name {
			return &app, nil
		}
	}

	return nil, fmt.Errorf("application %s not found", name)
}

func (r *RegistryManager) GetByID(id string) (*models.Application, error) {
	registry, err := r.Read()
	if err != nil {
		return nil, err
	}

	for _, app := range registry.Applications {
		if app.ID == id {
			return &app, nil
		}
	}

	return nil, fmt.Errorf("application with id %s not found", id)
}

func (r *RegistryManager) List() ([]models.Application, error) {
	registry, err := r.Read()
	if err != nil {
		return nil, err
	}

	return registry.Applications, nil
}

func (r *RegistryManager) Update(app models.Application) error {
	registry, err := r.Read()
	if err != nil {
		return err
	}

	found := false
	for i, existing := range registry.Applications {
		if existing.ID == app.ID {
			registry.Applications[i] = app
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("application %s not found", app.Name)
	}

	return r.Write(registry)
}

func (r *RegistryManager) Delete(name string) error {
	registry, err := r.Read()
	if err != nil {
		return err
	}

	found := false
	for i, app := range registry.Applications {
		if app.Name == name {
			registry.Applications = append(registry.Applications[:i], registry.Applications[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("application %s not found", name)
	}

	return r.Write(registry)
}

func GenerateID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "app-" + hex.EncodeToString(bytes), nil
}

func (r *RegistryManager) Exists(name string) (bool, error) {
	_, err := r.Get(name)
	if err != nil {
		if err.Error() == fmt.Sprintf("application %s not found", name) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
