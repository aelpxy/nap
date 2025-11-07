package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aelpxy/yap/pkg/models"
)

func DetectBuildMethod(projectPath string) (models.BuildType, string, error) {
	dockerfilePath := filepath.Join(projectPath, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		return models.BuildTypeDockerfile, dockerfilePath, nil
	}

	if IsNixpacksInstalled() {
		plan, err := GetNixpacksPlan(projectPath)
		if err == nil && plan.IsValid() {
			return models.BuildTypeNixpacks, "", nil
		}
	}

	return "", "", fmt.Errorf("no supported build method detected (no Dockerfile found, nixpacks not available)")
}

func DetectLanguage(projectPath string) (string, error) {
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); err == nil {
		return "nodejs", nil
	}

	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		return "python", nil
	}
	if _, err := os.Stat(filepath.Join(projectPath, "Pipfile")); err == nil {
		return "python", nil
	}

	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		return "go", nil
	}

	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		return "rust", nil
	}

	if _, err := os.Stat(filepath.Join(projectPath, "Gemfile")); err == nil {
		return "ruby", nil
	}

	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); err == nil {
		return "java", nil
	}

	return "unknown", nil
}

func GetDefaultPort(language string) int {
	return 8080
}
