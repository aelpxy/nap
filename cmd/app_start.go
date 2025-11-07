package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
)

var appStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a stopped application",
	Long:  "Start all instances of a stopped application",
	Args:  cobra.ExactArgs(1),
	Run:   runAppStart,
}

func runAppStart(cmd *cobra.Command, args []string) {
	appName := args[0]

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	application, err := registry.Get(appName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] application not found: %v", err)))
		os.Exit(1)
	}

	if application.Status == models.AppStatusRunning {
		fmt.Println(dimStyle.Render(fmt.Sprintf("application '%s' is already running", appName)))
		return
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> starting application: %s", appName)))
	fmt.Println()

	ctx := context.Background()

	for i, containerID := range application.ContainerIDs {
		fmt.Println(progressStyle.Render(fmt.Sprintf("  --> starting instance %d/%d...", i+1, len(application.ContainerIDs))))

		if err := dockerClient.GetClient().ContainerStart(ctx, containerID, dockerTypes.StartOptions{}); err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to start instance: %v", err)))
			os.Exit(1)
		}
	}

	application.Status = models.AppStatusRunning
	application.UpdatedAt = time.Now()
	if err := registry.Update(*application); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to update registry: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] %s started successfully", appName)))
	fmt.Println()
}

func init() {
	appCmd.AddCommand(appStartCmd)
}
