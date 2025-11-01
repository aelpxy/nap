package builder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type NixpacksPlan struct {
	Providers  []string                 `json:"providers"`
	BuildImage string                   `json:"buildImage"`
	Variables  map[string]string        `json:"variables"`
	Phases     map[string]NixpacksPhase `json:"phases"`
	Start      NixpacksStart            `json:"start"`
}

type NixpacksPhase struct {
	DependsOn        []string `json:"dependsOn"`
	Commands         []string `json:"cmds"`
	CacheDirectories []string `json:"cacheDirectories"`
	Paths            []string `json:"paths"`
	NixPkgs          []string `json:"nixPkgs"`
	NixOverlays      []string `json:"nixOverlays"`
	NixpkgsArchive   string   `json:"nixpkgsArchive"`
}

type NixpacksStart struct {
	Cmd string `json:"cmd"`
}

func (p *NixpacksPlan) IsValid() bool {
	return p.Start.Cmd != "" && len(p.Phases) > 0
}

func IsNixpacksInstalled() bool {
	cmd := exec.Command("nixpacks", "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func GetNixpacksPlan(projectPath string) (*NixpacksPlan, error) {
	cmd := exec.Command("nixpacks", "plan", projectPath, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("nixpacks plan failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run nixpacks plan: %w", err)
	}

	var plan NixpacksPlan
	if err := json.Unmarshal(output, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse nixpacks plan: %w", err)
	}

	return &plan, nil
}

func (b *Builder) BuildNixpacks(projectPath, imageName string, output io.Writer) (string, error) {
	plan, err := GetNixpacksPlan(projectPath)
	if err != nil {
		return "", err
	}

	if !plan.IsValid() {
		return "", fmt.Errorf("nixpacks generated invalid plan (no start command)")
	}

	fmt.Fprintln(output, "")
	fmt.Fprintf(output, "  --> running nixpacks build...\n")

	cmd := exec.Command("nixpacks", "build", projectPath, "--name", imageName)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start nixpacks build: %w", err)
	}

	go streamOutput(stdout, output, "  ")
	go streamOutput(stderr, output, "  ")

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("nixpacks build failed: %w", err)
	}

	fmt.Fprintln(output, "")
	fmt.Fprintf(output, "  --> build completed successfully\n")

	imageID, err := b.getImageID(imageName)
	if err != nil {
		return "", fmt.Errorf("failed to get image ID: %w", err)
	}

	return imageID, nil
}

func streamOutput(reader io.Reader, writer io.Writer, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			fmt.Fprintf(writer, "%s%s\n", prefix, line)
		}
	}
}

func (b *Builder) getImageID(imageName string) (string, error) {
	cmd := exec.Command("docker", "images", "-q", imageName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get image ID: %w", err)
	}

	imageID := strings.TrimSpace(string(output))
	if imageID == "" {
		return "", fmt.Errorf("no image found with name %s", imageName)
	}

	if !strings.HasPrefix(imageID, "sha256:") {
		imageID = "sha256:" + imageID
	}

	return imageID, nil
}
