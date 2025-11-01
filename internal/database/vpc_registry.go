package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aelpxy/nap/pkg/models"
)

const (
	vpcRegistryFile = "vpcs.json"
	primaryVPCName  = "primary"
)

type VPCRegistryManager struct {
	registryPath string
	mu           sync.RWMutex
}

func NewVPCRegistryManager() (*VPCRegistryManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	napDir := filepath.Join(homeDir, ".nap")
	registryPath := filepath.Join(napDir, vpcRegistryFile)

	return &VPCRegistryManager{
		registryPath: registryPath,
	}, nil
}

func (r *VPCRegistryManager) Initialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := os.Stat(r.registryPath); os.IsNotExist(err) {
		emptyRegistry := models.VPCList{
			VPCs: []models.VPC{},
		}
		if err := r.saveRegistry(&emptyRegistry); err != nil {
			return fmt.Errorf("failed to create VPC registry: %w", err)
		}
	}

	return nil
}

func (r *VPCRegistryManager) loadRegistry() (*models.VPCList, error) {
	data, err := os.ReadFile(r.registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read VPC registry: %w", err)
	}

	var vpcList models.VPCList
	if err := json.Unmarshal(data, &vpcList); err != nil {
		return nil, fmt.Errorf("failed to parse VPC registry: %w", err)
	}

	return &vpcList, nil
}

func (r *VPCRegistryManager) saveRegistry(vpcList *models.VPCList) error {
	data, err := json.MarshalIndent(vpcList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal VPC registry: %w", err)
	}

	if err := os.WriteFile(r.registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write VPC registry: %w", err)
	}

	return nil
}

func (r *VPCRegistryManager) Add(vpc models.VPC) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	vpcList, err := r.loadRegistry()
	if err != nil {
		return err
	}

	for _, existingVPC := range vpcList.VPCs {
		if existingVPC.Name == vpc.Name {
			return fmt.Errorf("VPC with name %s already exists", vpc.Name)
		}
	}

	vpcList.VPCs = append(vpcList.VPCs, vpc)
	return r.saveRegistry(vpcList)
}

func (r *VPCRegistryManager) Get(name string) (*models.VPC, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vpcList, err := r.loadRegistry()
	if err != nil {
		return nil, err
	}

	for _, vpc := range vpcList.VPCs {
		if vpc.Name == name {
			return &vpc, nil
		}
	}

	return nil, fmt.Errorf("VPC %s not found", name)
}

func (r *VPCRegistryManager) List() ([]models.VPC, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vpcList, err := r.loadRegistry()
	if err != nil {
		return nil, err
	}

	return vpcList.VPCs, nil
}

func (r *VPCRegistryManager) Update(vpc models.VPC) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	vpcList, err := r.loadRegistry()
	if err != nil {
		return err
	}

	found := false
	for i, existingVPC := range vpcList.VPCs {
		if existingVPC.Name == vpc.Name {
			vpcList.VPCs[i] = vpc
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("VPC %s not found", vpc.Name)
	}

	return r.saveRegistry(vpcList)
}

func (r *VPCRegistryManager) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	vpcList, err := r.loadRegistry()
	if err != nil {
		return err
	}

	for i, vpc := range vpcList.VPCs {
		if vpc.Name == name {
			if len(vpc.Databases) > 0 || len(vpc.Apps) > 0 {
				return fmt.Errorf("cannot delete VPC %s: %d database(s) and %d app(s) still attached",
					name, len(vpc.Databases), len(vpc.Apps))
			}
			vpcList.VPCs = append(vpcList.VPCs[:i], vpcList.VPCs[i+1:]...)
			return r.saveRegistry(vpcList)
		}
	}

	return fmt.Errorf("VPC %s not found", name)
}

func (r *VPCRegistryManager) AddDatabaseToVPC(vpcName string, dbID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	vpcList, err := r.loadRegistry()
	if err != nil {
		return err
	}

	for i, vpc := range vpcList.VPCs {
		if vpc.Name == vpcName {
			for _, existingDBID := range vpc.Databases {
				if existingDBID == dbID {
					return nil // Already added
				}
			}
			vpcList.VPCs[i].Databases = append(vpcList.VPCs[i].Databases, dbID)
			return r.saveRegistry(vpcList)
		}
	}

	return fmt.Errorf("VPC %s not found", vpcName)
}

func (r *VPCRegistryManager) RemoveDatabaseFromVPC(vpcName string, dbID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	vpcList, err := r.loadRegistry()
	if err != nil {
		return err
	}

	for i, vpc := range vpcList.VPCs {
		if vpc.Name == vpcName {
			for j, existingDBID := range vpc.Databases {
				if existingDBID == dbID {
					vpcList.VPCs[i].Databases = append(vpc.Databases[:j], vpc.Databases[j+1:]...)
					return r.saveRegistry(vpcList)
				}
			}
			return nil // Not found, nothing to remove
		}
	}

	return fmt.Errorf("VPC %s not found", vpcName)
}

func (r *VPCRegistryManager) Exists(name string) (bool, error) {
	_, err := r.Get(name)
	if err != nil {
		if err.Error() == fmt.Sprintf("VPC %s not found", name) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
