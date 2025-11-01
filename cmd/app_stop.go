package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/pkg/models"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
)

var appStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop an application",
	Long:  "Stop all instances of an application",
	Args:  cobra.ExactArgs(1),
	Run:   runAppStop,
}

func runAppStop(cmd *cobra.Command, args []string) {
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

	if application.Status == models.AppStatusStopped {
		fmt.Println(dimStyle.Render(fmt.Sprintf("application '%s' is already stopped", appName)))
		return
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> stopping application: %s", appName)))
	fmt.Println()

	ctx := context.Background()
	timeout := 10

	for i, containerID := range application.ContainerIDs {
		fmt.Println(progressStyle.Render(fmt.Sprintf("  --> stopping instance %d/%d...", i+1, len(application.ContainerIDs))))

		if err := dockerClient.GetClient().ContainerStop(ctx, containerID, dockerTypes.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to stop instance: %v", err)))
			os.Exit(1)
		}
	}

	application.Status = models.AppStatusStopped
	application.UpdatedAt = time.Now()
	if err := registry.Update(*application); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to update registry: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] %s stopped successfully", appName)))
	fmt.Println()
}

func init() {
	appCmd.AddCommand(appStopCmd)
}
