package backup

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aelpxy/nap/pkg/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

func (m *Manager) backupPostgres(db *models.Database, backupPath string, compress bool) (version string, size int64, err error) {
	versionCmd := []string{"psql", "-U", db.Username, "-d", db.DatabaseName, "-t", "-c", "SELECT version();"}
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

	versionStr := strings.TrimSpace(versionBuf.String())
	if strings.Contains(versionStr, "PostgreSQL") {
		parts := strings.Fields(versionStr)
		if len(parts) >= 2 {
			version = parts[1]
		}
	}

	dumpCmd := []string{
		"pg_dump",
		"-U", db.Username,
		"-d", db.DatabaseName,
		"--clean",
		"--if-exists",
		"--no-owner",
		"--no-acl",
	}

	execConfig = container.ExecOptions{
		Cmd:          dumpCmd,
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{fmt.Sprintf("PGPASSWORD=%s", db.Password)},
	}

	execID, err = m.dockerClient.GetClient().ContainerExecCreate(m.dockerClient.GetContext(), db.ContainerID, execConfig)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create dump exec: %w", err)
	}

	attachResp, err = m.dockerClient.GetClient().ContainerExecAttach(m.dockerClient.GetContext(), execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("failed to attach to dump exec: %w", err)
	}
	defer attachResp.Close()

	backupFile := filepath.Join(backupPath, "backup.sql")
	if compress {
		backupFile += ".gz"
	}

	outFile, err := os.Create(backupFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	var writer io.Writer = outFile
	var gzWriter *gzip.Writer

	if compress {
		gzWriter = gzip.NewWriter(outFile)
		writer = gzWriter
		defer gzWriter.Close()
	}

	var dumpBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&dumpBuf, io.Discard, attachResp.Reader); err != nil {
		return "", 0, fmt.Errorf("failed to read dump: %w", err)
	}

	if _, err := io.Copy(writer, &dumpBuf); err != nil {
		return "", 0, fmt.Errorf("failed to write backup: %w", err)
	}

	if compress && gzWriter != nil {
		if err := gzWriter.Close(); err != nil {
			return "", 0, fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}

	fileInfo, err := os.Stat(backupFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to stat backup file: %w", err)
	}

	return version, fileInfo.Size(), nil
}

func (m *Manager) restorePostgres(db *models.Database, backup *Backup) error {
	backupFile := filepath.Join(backup.Path, "backup.sql")
	if backup.Compressed {
		backupFile += ".gz"
	}

	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	if backup.Compressed {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	sqlData, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	restoreCmd := []string{
		"psql",
		"-U", db.Username,
		"-d", db.DatabaseName,
	}

	execConfig := container.ExecOptions{
		Cmd:          restoreCmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{fmt.Sprintf("PGPASSWORD=%s", db.Password)},
	}

	execID, err := m.dockerClient.GetClient().ContainerExecCreate(m.dockerClient.GetContext(), db.ContainerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create restore exec: %w", err)
	}

	attachResp, err := m.dockerClient.GetClient().ContainerExecAttach(m.dockerClient.GetContext(), execID.ID, container.ExecAttachOptions{
		Tty: false,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to restore exec: %w", err)
	}
	defer attachResp.Close()

	if _, err := attachResp.Conn.Write(sqlData); err != nil {
		return fmt.Errorf("failed to write restore data: %w", err)
	}
	attachResp.CloseWrite()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader); err != nil {
		return fmt.Errorf("failed to read restore output: %w", err)
	}

	if stderr.Len() > 0 {
		stderrStr := stderr.String()
		if strings.Contains(strings.ToLower(stderrStr), "error") {
			return fmt.Errorf("restore failed: %s", stderrStr)
		}
	}

	return nil
}
