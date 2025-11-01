package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func TruncateID(id string, length int) string {
	if length < 0 {
		length = 0
	}
	if len(id) <= length {
		return id
	}
	return id[:length]
}

func TruncateString(s string, max int) string {
	if max < 3 {
		max = 3
	}
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func MaskSensitive(value string, showChars int) string {
	if showChars < 0 {
		showChars = 0
	}
	if len(value) <= showChars {
		return "****"
	}
	return value[:showChars] + "****"
}

// masks environment variable values if key contains sensitive keywords
func MaskSensitiveEnvValue(key, value string) string {
	sensitiveKeywords := []string{"password", "secret", "token", "key", "api_key", "private"}

	keyLower := strings.ToLower(key)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(keyLower, keyword) {
			if len(value) <= 4 {
				return "****"
			}
			return value[:4] + "****"
		}
	}

	return value
}

// formats bytes into human-readable format (B, KB, MB, GB, TB, PB, EB)
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// write to temp file first then rename to prevent corruption
func AtomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	var writeErr error
	defer func() {
		if closeErr := tmpFile.Close(); closeErr != nil && writeErr == nil {
			writeErr = closeErr
		}
		if writeErr != nil {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		writeErr = err
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		writeErr = err
		return err
	}

	if err := tmpFile.Close(); err != nil {
		writeErr = err
		return err
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}

	return os.Rename(tmpName, filename)
}
