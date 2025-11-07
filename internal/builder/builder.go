package builder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
)

type Builder struct {
	dockerClient *docker.Client
}

func NewBuilder(dockerClient *docker.Client) *Builder {
	return &Builder{
		dockerClient: dockerClient,
	}
}

func (b *Builder) BuildDockerfile(projectPath, imageName string, output io.Writer) (string, error) {
	ctx := context.Background()

	buildContext, err := b.createBuildContext(projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to create build context: %w", err)
	}
	defer buildContext.Close()

	buildOptions := types.ImageBuildOptions{
		Tags:           []string{imageName},
		Dockerfile:     "Dockerfile",
		Remove:         true,
		ForceRemove:    true,
		PullParent:     true,
		SuppressOutput: false,
	}

	buildResponse, err := b.dockerClient.GetClient().ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return "", fmt.Errorf("failed to start build: %w", err)
	}
	defer buildResponse.Body.Close()

	imageID, err := b.streamBuildOutput(buildResponse.Body, output)
	if err != nil {
		return "", err
	}

	return imageID, nil
}

func (b *Builder) streamBuildOutput(reader io.Reader, output io.Writer) (string, error) {
	var imageID string
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Bytes()

		var buildEvent struct {
			Stream      string `json:"stream"`
			Status      string `json:"status"`
			Progress    string `json:"progress"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
			Aux struct {
				ID string `json:"ID"`
			} `json:"aux"`
		}

		if err := json.Unmarshal(line, &buildEvent); err != nil {
			fmt.Fprintln(output, string(line))
			continue
		}

		if buildEvent.Error != "" {
			return "", fmt.Errorf("build error: %s", buildEvent.Error)
		}
		if buildEvent.ErrorDetail.Message != "" {
			return "", fmt.Errorf("build error: %s", buildEvent.ErrorDetail.Message)
		}

		if buildEvent.Stream != "" {
			msg := strings.TrimRight(buildEvent.Stream, "\n\r")
			if msg != "" {
				fmt.Fprintln(output, "  "+msg)
			}
		}

		if buildEvent.Status != "" {
			msg := buildEvent.Status
			if buildEvent.Progress != "" {
				msg = fmt.Sprintf("%s %s", msg, buildEvent.Progress)
			}
			fmt.Fprintf(output, "\r  %s", msg)
			if !strings.Contains(buildEvent.Progress, "Download") &&
				!strings.Contains(buildEvent.Progress, "Extract") {
				fmt.Fprintln(output)
			}
		}

		if buildEvent.Aux.ID != "" {
			imageID = buildEvent.Aux.ID
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading build output: %w", err)
	}

	if imageID == "" {
		return "", fmt.Errorf("build completed but no image ID was returned")
	}

	return imageID, nil
}

func (b *Builder) createBuildContext(projectPath string) (io.ReadCloser, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}

	exclusions := []string{
		".git",
		".gitignore",
		"node_modules",
		".env",
		".env.local",
		"*.log",
		".DS_Store",
		"__pycache__",
		"*.pyc",
		".pytest_cache",
		"venv",
		".venv",
		"target",
		"dist",
		"build",
	}

	dockerIgnorePath := filepath.Join(absPath, ".dockerignore")
	if _, err := os.Stat(dockerIgnorePath); err == nil {
		content, err := os.ReadFile(dockerIgnorePath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					exclusions = append(exclusions, line)
				}
			}
		}
	}

	buildContext, err := archive.TarWithOptions(absPath, &archive.TarOptions{
		ExcludePatterns: exclusions,
		Compression:     archive.Gzip,
	})

	return buildContext, err
}

func (b *Builder) BuildWithMethod(projectPath, appName string, buildMethod string, output io.Writer) (*BuildResult, error) {
	var buildType models.BuildType
	var dockerfilePath string
	var err error

	if buildMethod == "auto" {
		buildType, dockerfilePath, err = DetectBuildMethod(projectPath)
		if err != nil {
			return nil, err
		}
	} else {
		switch buildMethod {
		case "dockerfile":
			dockerfilePath = filepath.Join(projectPath, "Dockerfile")
			if _, err := os.Stat(dockerfilePath); err != nil {
				return nil, fmt.Errorf("dockerfile not found at %s", dockerfilePath)
			}
			buildType = models.BuildTypeDockerfile
		case "nixpacks":
			if !IsNixpacksInstalled() {
				return nil, fmt.Errorf("nixpacks is not installed (run: curl -sSL https://nixpacks.com/install.sh | bash)")
			}
			buildType = models.BuildTypeNixpacks
		case "paketo":
			buildType = models.BuildTypePacketo
		default:
			return nil, fmt.Errorf("invalid build method: %s (valid: auto, dockerfile, nixpacks, paketo)", buildMethod)
		}
	}

	return b.buildInternal(projectPath, appName, buildType, dockerfilePath, output)
}

func (b *Builder) Build(projectPath, appName string, output io.Writer) (*BuildResult, error) {
	buildType, dockerfilePath, err := DetectBuildMethod(projectPath)
	if err != nil {
		return nil, err
	}

	return b.buildInternal(projectPath, appName, buildType, dockerfilePath, output)
}

func (b *Builder) buildInternal(projectPath, appName string, buildType models.BuildType, dockerfilePath string, output io.Writer) (*BuildResult, error) {

	language, _ := DetectLanguage(projectPath)

	imageName := fmt.Sprintf("yap/%s:latest", appName)

	fmt.Fprintf(output, "  --> detected build method: %s\n", buildType)
	if language != "unknown" {
		fmt.Fprintf(output, "  --> detected language: %s\n", language)
	}
	fmt.Fprintln(output, "")

	var imageID string
	var err error

	switch buildType {
	case models.BuildTypeDockerfile:
		fmt.Fprintln(output, "  --> building with dockerfile...")
		imageID, err = b.BuildDockerfile(projectPath, imageName, output)
		if err != nil {
			return nil, fmt.Errorf("dockerfile build failed: %w", err)
		}

	case models.BuildTypeNixpacks:
		fmt.Fprintln(output, "  --> building with nixpacks...")
		imageID, err = b.BuildNixpacks(projectPath, imageName, output)
		if err != nil {
			return nil, fmt.Errorf("nixpacks build failed: %w", err)
		}

	case models.BuildTypePacketo:
		return nil, fmt.Errorf("packeto support not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported build type: %s", buildType)
	}

	result := &BuildResult{
		ImageID:        imageID,
		ImageName:      imageName,
		BuildType:      buildType,
		Language:       language,
		DockerfilePath: dockerfilePath,
	}

	return result, nil
}

type BuildResult struct {
	ImageID        string
	ImageName      string
	BuildType      models.BuildType
	Language       string
	DockerfilePath string
}
