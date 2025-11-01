package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aelpxy/nap/pkg/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

func (m *Manager) backupValkey(db *models.Database, backupPath string, compress bool) (version string, size int64, err error) {
	versionCmd := []string{"valkey-cli", "-a", db.Password, "INFO", "server"}
	execConfig := container.ExecOptions{
		Cmd:          versionCmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := m.dockerClient.GetClient().ContainerExecCreate(m.dockerClient.GetContext(), db.ContainerID, execConfig)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create version exec: %w", err)
	}

	attachResp, err := m.dockerClient.GetClient().ContainerExecAttach(m.dockerClient.GetContext(), execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("failed to attach to version exec: %w", err)
	}
	defer attachResp.Close()

	var versionBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&versionBuf, io.Discard, attachResp.Reader); err != nil {
		return "", 0, fmt.Errorf("failed to read version: %w", err)
	}

	versionStr := versionBuf.String()
	for _, line := range strings.Split(versionStr, "\n") {
		if strings.HasPrefix(line, "redis_version:") || strings.HasPrefix(line, "valkey_version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				version = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	saveCmd := []string{"valkey-cli", "-a", db.Password, "SAVE"}
	execConfig = container.ExecOptions{
		Cmd:          saveCmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err = m.dockerClient.GetClient().ContainerExecCreate(m.dockerClient.GetContext(), db.ContainerID, execConfig)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create save exec: %w", err)
	}

	attachResp, err = m.dockerClient.GetClient().ContainerExecAttach(m.dockerClient.GetContext(), execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("failed to attach to save exec: %w", err)
	}
	defer attachResp.Close()

	var saveBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&saveBuf, io.Discard, attachResp.Reader); err != nil {
		return "", 0, fmt.Errorf("failed to execute save: %w", err)
	}

	reader, _, err := m.dockerClient.GetClient().CopyFromContainer(
		m.dockerClient.GetContext(),
		db.ContainerID,
		"/data/dump.rdb",
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to copy RDB from container: %w", err)
	}
	defer reader.Close()

	backupFile := filepath.Join(backupPath, "dump.rdb")
	outFile, err := os.Create(backupFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create backup file: %w", err)
	}

	var writeErr error
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil && writeErr == nil {
			writeErr = closeErr
		}
	}()

	written, err := io.Copy(outFile, reader)
	if err != nil {
		writeErr = err
		return "", 0, fmt.Errorf("failed to write backup: %w", err)
	}

	if err := outFile.Close(); err != nil {
		writeErr = err
		return "", 0, fmt.Errorf("failed to close backup file: %w", err)
	}

	return version, written, nil
}

func (m *Manager) restoreValkey(db *models.Database, backup *Backup) error {
	backupFile := filepath.Join(backup.Path, "dump.rdb")

	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	timeout := 10
	if err := m.dockerClient.GetClient().ContainerStop(m.dockerClient.GetContext(), db.ContainerID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if err := m.dockerClient.GetClient().ContainerStart(m.dockerClient.GetContext(), db.ContainerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := m.dockerClient.GetClient().ContainerStop(m.dockerClient.GetContext(), db.ContainerID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return fmt.Errorf("failed to stop container for restore: %w", err)
	}

	resp, err := m.dockerClient.GetClient().ContainerCreate(
		context.Background(),
		&container.Config{
			Image:     "alpine:latest",
			Cmd:       []string{"sh", "-c", "sleep 5"}, // Keep container alive
			Tty:       false,
			StdinOnce: true,
			OpenStdin: true,
		},
		&container.HostConfig{
			Binds: []string{fmt.Sprintf("%s:/data", db.VolumeName)},
		},
		nil,
		nil,
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to create restore container: %w", err)
	}
	defer m.dockerClient.GetClient().ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	if err := m.dockerClient.GetClient().ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start restore container: %w", err)
	}

	execConfig := container.ExecOptions{
		Cmd:          []string{"sh", "-c", "cat > /data/dump.rdb"},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := m.dockerClient.GetClient().ContainerExecCreate(context.Background(), resp.ID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec for restore: %w", err)
	}

	attachResp, err := m.dockerClient.GetClient().ContainerExecAttach(context.Background(), execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach to restore exec: %w", err)
	}
	defer attachResp.Close()

	if _, err := attachResp.Conn.Write(backupData); err != nil {
		return fmt.Errorf("failed to write restore data: %w", err)
	}
	attachResp.CloseWrite()

	io.Copy(io.Discard, attachResp.Reader)

	if err := m.dockerClient.GetClient().ContainerStart(m.dockerClient.GetContext(), db.ContainerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start database container: %w", err)
	}

	return nil
}
