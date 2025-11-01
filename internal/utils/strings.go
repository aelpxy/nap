package utils

import (
	"os"
	"path/filepath"
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

// write to temp file first then rename to prevent corruption
func AtomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	defer func() {
		tmpFile.Close()
		os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}

	return os.Rename(tmpName, filename)
}
