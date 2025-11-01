package router

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aelpxy/nap/internal/config"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/pkg/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

const (
	traefikContainerName = "nap-traefik"
	traefikImage         = "traefik:v3.5"
)

type TraefikManager struct {
	dockerClient   *docker.Client
	letsencryptDir string
}

func NewTraefikManager(dockerClient *docker.Client) *TraefikManager {
	return &TraefikManager{
		dockerClient: dockerClient,
	}
}

func (t *TraefikManager) IsRunning() (bool, error) {
	ctx := context.Background()
	containers, err := t.dockerClient.GetClient().ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return false, err
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+traefikContainerName || name == traefikContainerName {
				return c.State == "running", nil
			}
		}
	}

	return false, nil
}

func (t *TraefikManager) Start(output io.Writer) error {
	ctx := context.Background()

	running, err := t.IsRunning()
	if err != nil {
		return fmt.Errorf("failed to check if traefik is running: %w", err)
	}

	if running {
		fmt.Fprintln(output, "  --> traefik already running")
		return nil
	}

	containerID, err := t.getContainerID()
	if err == nil && containerID != "" {
		fmt.Fprintln(output, "  --> starting existing traefik instance...")
		if err := t.dockerClient.GetClient().ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			return fmt.Errorf("failed to start traefik: %w", err)
		}
		fmt.Fprintln(output, "  [done] traefik started")
		return nil
	}

	fmt.Fprintln(output, "  --> creating traefik load balancer...")

	configPath, err := t.generateConfig()
	if err != nil {
		return fmt.Errorf("failed to generate traefik config: %w", err)
	}

	fmt.Fprintln(output, "  --> pulling traefik image...")
	if err := t.pullImage(ctx); err != nil {
		return fmt.Errorf("failed to pull traefik image: %w", err)
	}

	containerConfig := &container.Config{
		Image: traefikImage,
		Labels: map[string]string{
			"nap.managed": "true",
			"nap.type":    "traefik",
		},
		ExposedPorts: nat.PortSet{
			"80/tcp":   struct{}{},
			"443/tcp":  struct{}{},
			"8080/tcp": struct{}{},
		},
	}

	socketPath := "/var/run/docker.sock"
	if t.dockerClient.GetRuntimeInfo() != nil {
		socketPath = t.dockerClient.GetRuntimeInfo().SocketPath
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		PortBindings: nat.PortMap{
			"80/tcp":   []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "80"}},
			"443/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "443"}},
			"8080/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: socketPath,
				// traefik thinks its talking to docker we let it believe that (podman plays along :P)
				Target: "/var/run/docker.sock",
			},
			{
				Type:   mount.TypeBind,
				Source: configPath,
				Target: "/etc/traefik/traefik.yml",
			},
			{
				Type:   mount.TypeBind,
				Source: t.letsencryptDir,
				Target: "/letsencrypt",
			},
		},
	}

	resp, err := t.dockerClient.GetClient().ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		&network.NetworkingConfig{},
		nil,
		traefikContainerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create traefik container: %w", err)
	}

	if err := t.dockerClient.GetClient().ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start traefik container: %w", err)
	}

	fmt.Fprintln(output, "  [done] traefik load balancer started")
	fmt.Fprintln(output, "")
	fmt.Fprintln(output, "  traefik dashboard: http://localhost:8080")

	return nil
}

func (t *TraefikManager) getContainerID() (string, error) {
	ctx := context.Background()
	containers, err := t.dockerClient.GetClient().ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return "", err
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+traefikContainerName || name == traefikContainerName {
				return c.ID, nil
			}
		}
	}

	return "", fmt.Errorf("traefik container not found")
}

func (t *TraefikManager) pullImage(ctx context.Context) error {
	reader, err := t.dockerClient.GetClient().ImagePull(ctx, traefikImage, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	io.Copy(io.Discard, reader)

	return nil
}

func (t *TraefikManager) generateConfig() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	napDir := filepath.Join(homeDir, ".nap")
	if err := os.MkdirAll(napDir, 0755); err != nil {
		return "", err
	}

	letsencryptDir := filepath.Join(napDir, "letsencrypt")
	if err := os.MkdirAll(letsencryptDir, 0755); err != nil {
		return "", err
	}

	configPath := filepath.Join(napDir, "traefik.yml")

	t.letsencryptDir = letsencryptDir

	acmeEmail := "nap@localhost"
	configManager, err := loadGlobalConfig()
	if err == nil && configManager != nil {
		config := configManager.GetConfig()
		if config != nil && config.Publishing.Enabled && config.Publishing.Email != "" {
			acmeEmail = config.Publishing.Email
		}
	}

	config := fmt.Sprintf(`# Traefik configuration for nap
entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"

certificatesResolvers:
  letsencrypt:
    acme:
      email: %s
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    watch: true

api:
  dashboard: true
  insecure: true

log:
  level: INFO

accessLog:
  format: common
`, acmeEmail)

	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return "", err
	}

	return configPath, nil
}

func (t *TraefikManager) ConnectToVPC(vpcNetworkName string) error {
	ctx := context.Background()

	containerID, err := t.getContainerID()
	if err != nil {
		return err
	}

	containerInfo, err := t.dockerClient.GetClient().ContainerInspect(ctx, containerID)
	if err != nil {
		return err
	}

	for networkName := range containerInfo.NetworkSettings.Networks {
		if networkName == vpcNetworkName {
			return nil
		}
	}

	if err := t.dockerClient.GetClient().NetworkConnect(ctx, vpcNetworkName, containerID, nil); err != nil {
		return fmt.Errorf("failed to connect traefik to vpc: %w", err)
	}

	return nil
}

func (t *TraefikManager) GenerateLabelsForApp(app *models.Application) map[string]string {
	labels := map[string]string{
		"traefik.enable": "true",

		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", app.Name): fmt.Sprintf("%d", app.Port),

		fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.path", app.Name):     app.HealthCheckPath,
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.interval", app.Name): fmt.Sprintf("%ds", app.HealthCheckInterval),
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.timeout", app.Name):  fmt.Sprintf("%ds", app.HealthCheckTimeout),
	}

	if app.Published {
		var hostRules []string
		hostRules = append(hostRules, fmt.Sprintf("Host(`%s`)", app.PublishedDomain))
		for _, domain := range app.CustomDomains {
			hostRules = append(hostRules, fmt.Sprintf("Host(`%s`)", domain))
		}
		hostRule := strings.Join(hostRules, " || ")

		labels[fmt.Sprintf("traefik.http.routers.%s-secure.rule", app.Name)] = hostRule
		labels[fmt.Sprintf("traefik.http.routers.%s-secure.entrypoints", app.Name)] = "websecure"
		labels[fmt.Sprintf("traefik.http.routers.%s-secure.tls", app.Name)] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s-secure.tls.certresolver", app.Name)] = "letsencrypt"

		labels[fmt.Sprintf("traefik.http.routers.%s.rule", app.Name)] = hostRule
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", app.Name)] = "web"
		labels[fmt.Sprintf("traefik.http.routers.%s.middlewares", app.Name)] = "redirect-to-https"

		labels["traefik.http.middlewares.redirect-to-https.redirectscheme.scheme"] = "https"
		labels["traefik.http.middlewares.redirect-to-https.redirectscheme.permanent"] = "true"
	} else {
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", app.Name)] = fmt.Sprintf("Host(`%s.nap.local`)", app.Name)
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", app.Name)] = "web"
	}

	return labels
}

func loadGlobalConfig() (*config.ConfigManager, error) {
	cm, err := config.NewConfigManager()
	if err != nil {
		return nil, err
	}
	return cm, nil
}
