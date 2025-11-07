package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type DaemonManager struct {
	runtime *RuntimeInfo
}

func NewDaemonManager(runtime *RuntimeInfo) *DaemonManager {
	return &DaemonManager{
		runtime: runtime,
	}
}

func (dm *DaemonManager) Start() error {
	if dm.runtime.Type == RuntimeDocker {
		return fmt.Errorf("docker daemon management not supported - use systemctl or docker desktop")
	}

	return dm.startPodman()
}

func (dm *DaemonManager) Stop() error {
	if dm.runtime.Type == RuntimeDocker {
		return fmt.Errorf("docker daemon management not supported - use systemctl or docker desktop")
	}

	return dm.stopPodman()
}

func (dm *DaemonManager) Status() (string, error) {
	if dm.runtime.Type == RuntimeDocker {
		return dm.statusDocker()
	}
	return dm.statusPodman()
}

func (dm *DaemonManager) IsRunning() bool {
	if _, err := os.Stat(dm.runtime.SocketPath); err != nil {
		return false
	}

	if dm.runtime.Type == RuntimePodman {
		cmd := exec.Command("podman", "info")
		if err := cmd.Run(); err != nil {
			return false
		}
	} else {
		cmd := exec.Command("docker", "info")
		if err := cmd.Run(); err != nil {
			return false
		}
	}

	return true
}

func (dm *DaemonManager) startPodman() error {
	if dm.IsRunning() {
		return fmt.Errorf("podman service is already running")
	}

	socketDir := filepath.Dir(dm.runtime.SocketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	if dm.hasSystemd() {
		return dm.startPodmanSystemd()
	}

	return dm.startPodmanManual()
}

func (dm *DaemonManager) stopPodman() error {
	if !dm.IsRunning() {
		return fmt.Errorf("podman service is not running")
	}

	if dm.hasSystemd() {
		return dm.stopPodmanSystemd()
	}

	return dm.stopPodmanManual()
}

func (dm *DaemonManager) statusPodman() (string, error) {
	if dm.IsRunning() {
		if dm.hasSystemd() {
			cmd := exec.Command("systemctl", "--user", "status", "podman.socket", "--no-pager")
			if output, err := cmd.Output(); err == nil {
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					if strings.Contains(line, "Active:") {
						return strings.TrimSpace(line), nil
					}
				}
			}
		}
		return "running", nil
	}
	return "stopped", nil
}

func (dm *DaemonManager) statusDocker() (string, error) {
	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	if output, err := cmd.Output(); err == nil {
		version := strings.TrimSpace(string(output))
		return fmt.Sprintf("running (version %s)", version), nil
	}
	return "stopped or unavailable", nil
}

func (dm *DaemonManager) startPodmanSystemd() error {
	cmd := exec.Command("systemctl", "--user", "enable", "--now", "podman.socket")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start podman.socket via systemd: %w\noutput: %s", err, string(output))
	}

	for i := 0; i < 10; i++ {
		if _, err := os.Stat(dm.runtime.SocketPath); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("podman socket did not become available after starting service")
}

func (dm *DaemonManager) stopPodmanSystemd() error {
	cmd := exec.Command("systemctl", "--user", "stop", "podman.socket", "podman.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop podman via systemd: %w\noutput: %s", err, string(output))
	}
	return nil
}

func (dm *DaemonManager) startPodmanManual() error {
	cmd := exec.Command("podman", "system", "service", "--time=0", fmt.Sprintf("unix://%s", dm.runtime.SocketPath))

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start podman service: %w", err)
	}

	pidFile := filepath.Join(os.TempDir(), "yap-podman-service.pid")
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	for i := 0; i < 20; i++ {
		if _, err := os.Stat(dm.runtime.SocketPath); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("podman socket did not become available after starting service")
}

func (dm *DaemonManager) stopPodmanManual() error {
	pidFile := filepath.Join(os.TempDir(), "yap-podman-service.pid")

	data, err := os.ReadFile(pidFile)
	if err != nil {
		cmd := exec.Command("pkill", "-f", "podman system service")
		_ = cmd.Run()
		return nil
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		_ = os.Remove(pidFile)
		return nil
	}

	time.Sleep(2 * time.Second)

	_ = process.Signal(syscall.SIGKILL)

	_ = os.Remove(pidFile)

	return nil
}

func (dm *DaemonManager) hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func (dm *DaemonManager) EnableSystemdUserService() error {
	if !dm.hasSystemd() {
		return fmt.Errorf("systemd not available")
	}

	cmd := exec.Command("loginctl", "enable-linger", os.Getenv("USER"))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable linger: %w\noutput: %s", err, string(output))
	}

	return nil
}
