package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RuntimeType string

const (
	RuntimeDocker RuntimeType = "docker"
	RuntimePodman RuntimeType = "podman"
)

type RuntimeInfo struct {
	Type          RuntimeType
	SocketPath    string
	Version       string
	IsRootless    bool
	ServiceActive bool
}

func DetectRuntime() (*RuntimeInfo, error) {
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
		if strings.Contains(dockerHost, "podman") {
			return detectPodman()
		}
		return detectDocker()
	}

	if info, err := detectDocker(); err == nil {
		return info, nil
	}

	if info, err := detectPodman(); err == nil {
		return info, nil
	}

	return nil, fmt.Errorf("no container runtime detected (tried docker, podman)")
}

func detectDocker() (*RuntimeInfo, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker command not found")
	}

	socketPath := "/var/run/docker.sock"
	if _, err := os.Stat(socketPath); err != nil {
		return nil, fmt.Errorf("docker socket not found at %s", socketPath)
	}

	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker version: %w", err)
	}

	version := strings.TrimSpace(string(output))

	return &RuntimeInfo{
		Type:          RuntimeDocker,
		SocketPath:    socketPath,
		Version:       version,
		IsRootless:    false, // docker typically runs aA root
		ServiceActive: true,  // if socket exists then the daemon is running
	}, nil
}

func detectPodman() (*RuntimeInfo, error) {
	if _, err := exec.LookPath("podman"); err != nil {
		return nil, fmt.Errorf("podman command not found")
	}

	isRootless := os.Getuid() != 0

	var socketPath string
	if isRootless {
		uid := os.Getuid()
		socketPath = fmt.Sprintf("/run/user/%d/podman/podman.sock", uid)
	} else {
		socketPath = "/run/podman/podman.sock"
	}

	serviceActive := false
	if _, err := os.Stat(socketPath); err == nil {
		serviceActive = true
	}

	cmd := exec.Command("podman", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("podman", "version", "--format", "{{.Client.Version}}")
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get podman version: %w", err)
		}
	}

	version := strings.TrimSpace(string(output))

	return &RuntimeInfo{
		Type:          RuntimePodman,
		SocketPath:    socketPath,
		Version:       version,
		IsRootless:    isRootless,
		ServiceActive: serviceActive,
	}, nil
}

func (r *RuntimeInfo) GetSocketURI() string {
	return fmt.Sprintf("unix://%s", r.SocketPath)
}

func (r *RuntimeInfo) GetRuntimeName() string {
	name := string(r.Type)
	if r.Type == RuntimePodman && r.IsRootless {
		name += " (rootless)"
	}
	return name
}

func (r *RuntimeInfo) EnsureSocketExists() error {
	if _, err := os.Stat(r.SocketPath); err != nil {
		if r.Type == RuntimePodman {
			return fmt.Errorf("podman socket not found at %s - run 'yap daemon start' to start the service", r.SocketPath)
		}
		return fmt.Errorf("runtime socket not found at %s", r.SocketPath)
	}
	return nil
}

func (r *RuntimeInfo) GetSystemdServiceName() string {
	if r.Type == RuntimePodman {
		if r.IsRootless {
			return "podman.service"
		}
		return "podman.service"
	}
	return "docker.service"
}

func GetPodmanSocketPath() string {
	if os.Getuid() != 0 {
		return filepath.Join("/run/user", fmt.Sprintf("%d", os.Getuid()), "podman", "podman.sock")
	}
	return "/run/podman/podman.sock"
}
