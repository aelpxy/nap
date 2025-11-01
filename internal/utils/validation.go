package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func IsValidName(name string) bool {
	if len(name) == 0 {
		return false
	}
	if name[0] == '-' || name[len(name)-1] == '-' {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

// checks if path is safe and resolves to a valid directory
func ValidateProjectPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	cleanPath := filepath.Clean(absPath)

	if strings.Contains(path, "..") {
		evalPath, err := filepath.EvalSymlinks(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate path: %w", err)
		}
		cleanPath = evalPath
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", cleanPath)
		}
		return "", fmt.Errorf("failed to access path: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", cleanPath)
	}

	return cleanPath, nil
}
