package database

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aelpxy/nap/pkg/models"
)

const (
	napDir       = ".nap"
	registryFile = "databases.json"
)

var (
	mu sync.Mutex // mutex to protect concurrent access to the registry!!
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
		registry := models.Registry{
			Databases: []models.Database{},
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

func (r *RegistryManager) Read() (*models.Registry, error) {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var registry models.Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	return &registry, nil
}

func (r *RegistryManager) Write(registry *models.Registry) error {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

func (r *RegistryManager) Add(db models.Database) error {
	registry, err := r.Read()
	if err != nil {
		return err
	}

	for _, existing := range registry.Databases {
		if existing.Name == db.Name {
			return fmt.Errorf("database with name '%s' already exists", db.Name)
		}
	}

	registry.Databases = append(registry.Databases, db)

	return r.Write(registry)
}

func (r *RegistryManager) Remove(name string) error {
	registry, err := r.Read()
	if err != nil {
		return err
	}

	found := false
	newDatabases := []models.Database{}

	for _, db := range registry.Databases {
		if db.Name != name {
			newDatabases = append(newDatabases, db)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("database '%s' not found in registry", name)
	}

	registry.Databases = newDatabases

	return r.Write(registry)
}

func (r *RegistryManager) Update(db models.Database) error {
	registry, err := r.Read()
	if err != nil {
		return err
	}

	found := false
	for i, existing := range registry.Databases {
		if existing.Name == db.Name {
			registry.Databases[i] = db
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("database '%s' not found in registry", db.Name)
	}

	return r.Write(registry)
}

func (r *RegistryManager) Get(name string) (*models.Database, error) {
	registry, err := r.Read()
	if err != nil {
		return nil, err
	}

	for _, db := range registry.Databases {
		if db.Name == name {
			return &db, nil
		}
	}

	return nil, fmt.Errorf("database '%s' not found", name)
}

func (r *RegistryManager) GetByID(id string) (*models.Database, error) {
	registry, err := r.Read()
	if err != nil {
		return nil, err
	}

	for _, db := range registry.Databases {
		if db.ID == id {
			return &db, nil
		}
	}

	return nil, fmt.Errorf("database with ID '%s' not found", id)
}

func (r *RegistryManager) List() ([]models.Database, error) {
	registry, err := r.Read()
	if err != nil {
		return nil, err
	}

	return registry.Databases, nil
}

func GenerateID(prefix string) string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, os.Getpid())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(bytes))
}
