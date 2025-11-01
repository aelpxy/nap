package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/aelpxy/nap/internal/config"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/pkg/models"
)

type PublishingManager struct {
	dockerClient  *docker.Client
	registry      *RegistryManager
	configManager *config.ConfigManager
}

func NewPublishingManager(dockerClient *docker.Client, registry *RegistryManager, configManager *config.ConfigManager) *PublishingManager {
	return &PublishingManager{
		dockerClient:  dockerClient,
		registry:      registry,
		configManager: configManager,
	}
}

func (pm *PublishingManager) PublishApp(ctx context.Context, appName, customDomain string) error {
	if err := pm.configManager.ValidatePublishing(); err != nil {
		return fmt.Errorf("publishing not configured: %w\n\nrun 'nap config setup' to configure publishing", err)
	}

	lockMgr := GetGlobalLockManager()
	if err := lockMgr.TryLock(appName, 10); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lockMgr.Unlock(appName)

	app, err := pm.registry.Get(appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	var domain string
	if customDomain != "" {
		domain = customDomain
	} else {
		baseDomain := pm.configManager.GetConfig().Publishing.BaseDomain
		domain = fmt.Sprintf("%s.nap.%s", appName, baseDomain)
	}

	if !isValidDomain(domain) {
		return fmt.Errorf("invalid domain: %s", domain)
	}

	app.Published = true
	app.PublishedDomain = domain
	app.PublishedURL = fmt.Sprintf("https://%s", domain)
	app.SSLEnabled = true
	app.SSLCertIssuer = "letsencrypt"

	if err := pm.registry.Update(*app); err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	if err := pm.recreateContainersWithPublishing(ctx, app); err != nil {
		app.Published = false
		app.PublishedDomain = ""
		app.PublishedURL = ""
		app.SSLEnabled = false
		pm.registry.Update(*app)
		return fmt.Errorf("failed to update containers: %w", err)
	}

	return nil
}

func (pm *PublishingManager) UnpublishApp(ctx context.Context, appName string) error {
	lockMgr := GetGlobalLockManager()
	if err := lockMgr.TryLock(appName, 10); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lockMgr.Unlock(appName)

	app, err := pm.registry.Get(appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	if !app.Published {
		return fmt.Errorf("application is not published")
	}

	app.Published = false
	app.PublishedDomain = ""
	app.PublishedURL = fmt.Sprintf("http://%s.nap.local", appName)
	app.CustomDomains = nil
	app.SSLEnabled = false
	app.SSLCertIssuer = ""

	if err := pm.registry.Update(*app); err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	if err := pm.recreateContainersWithPublishing(ctx, app); err != nil {
		return fmt.Errorf("failed to update containers: %w", err)
	}

	return nil
}

func (pm *PublishingManager) AddCustomDomain(ctx context.Context, appName, domain string) error {
	lockMgr := GetGlobalLockManager()
	if err := lockMgr.TryLock(appName, 10); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lockMgr.Unlock(appName)

	app, err := pm.registry.Get(appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	if !app.Published {
		return fmt.Errorf("application must be published first")
	}

	if !isValidDomain(domain) {
		return fmt.Errorf("invalid domain: %s", domain)
	}

	if domain == app.PublishedDomain {
		return fmt.Errorf("domain already added as primary domain: %s", domain)
	}
	for _, d := range app.CustomDomains {
		if d == domain {
			return fmt.Errorf("domain already added: %s", domain)
		}
	}

	app.CustomDomains = append(app.CustomDomains, domain)

	if err := pm.registry.Update(*app); err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	if err := pm.recreateContainersWithPublishing(ctx, app); err != nil {
		return fmt.Errorf("failed to update containers: %w", err)
	}

	return nil
}

func (pm *PublishingManager) RemoveCustomDomain(ctx context.Context, appName, domain string) error {
	lockMgr := GetGlobalLockManager()
	if err := lockMgr.TryLock(appName, 10); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lockMgr.Unlock(appName)

	app, err := pm.registry.Get(appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	var newDomains []string
	found := false
	for _, d := range app.CustomDomains {
		if d == domain {
			found = true
		} else {
			newDomains = append(newDomains, d)
		}
	}

	if !found {
		return fmt.Errorf("domain not found: %s", domain)
	}

	app.CustomDomains = newDomains

	if err := pm.registry.Update(*app); err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	if err := pm.recreateContainersWithPublishing(ctx, app); err != nil {
		return fmt.Errorf("failed to update containers: %w", err)
	}

	return nil
}

func (pm *PublishingManager) recreateContainersWithPublishing(ctx context.Context, app *models.Application) error {

	vpcNetworkName := fmt.Sprintf("%s.nap-vpc-network", app.VPC)

	for i := 1; i <= app.Instances; i++ {
		if i-1 >= len(app.ContainerIDs) {
			return fmt.Errorf("container ID not found for instance %d", i)
		}

		oldContainerID := app.ContainerIDs[i-1]
		newContainerID, err := RecreateContainer(pm.dockerClient, oldContainerID, app, vpcNetworkName, i)
		if err != nil {
			return fmt.Errorf("failed to recreate instance %d: %w", i, err)
		}

		app.ContainerIDs[i-1] = newContainerID
	}

	if err := pm.registry.Update(*app); err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	return nil
}

func isValidDomain(domain string) bool {
	if domain == "" {
		return false
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
	}

	return true
}
