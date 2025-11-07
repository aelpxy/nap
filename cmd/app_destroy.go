package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
)

var (
	forceDestroyApp bool
)

var appDestroyCmd = &cobra.Command{
	Use:   "destroy [name]",
	Short: "Destroy an application",
	Long:  "Permanently destroy an application and remove all instances",
	Args:  cobra.ExactArgs(1),
	Run:   runAppDestroy,
}

func runAppDestroy(cmd *cobra.Command, args []string) {
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

	if !forceDestroyApp {
		fmt.Println(errorStyle.Render(fmt.Sprintf("[warn]  warning: this will permanently destroy the application '%s'", appName)))
		fmt.Println(labelStyle.Render("   all instances will be removed."))
		fmt.Println()
		fmt.Print(labelStyle.Render("type the application name to confirm: "))

		var confirmation string
		fmt.Scanln(&confirmation)

		if strings.TrimSpace(confirmation) != appName {
			fmt.Println(labelStyle.Render("\ndestruction cancelled."))
			return
		}
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println()
	fmt.Println(errorStyle.Render(fmt.Sprintf("==> destroying application: %s", appName)))
	fmt.Println()

	ctx := context.Background()

	for i, containerID := range application.ContainerIDs {
		fmt.Println(labelStyle.Render(fmt.Sprintf("  --> removing instance %d/%d...", i+1, len(application.ContainerIDs))))

		_ = dockerClient.StopContainer(containerID)

		if err := dockerClient.GetClient().ContainerRemove(ctx, containerID, dockerTypes.RemoveOptions{
			Force: true,
		}); err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to remove instance: %v", err)))
			os.Exit(1)
		}

		fmt.Println(successStyle.Render(fmt.Sprintf("  [ok] instance %d removed", i+1)))
	}

	fmt.Println(labelStyle.Render("  --> removing from registry..."))
	if err := registry.Delete(appName); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to remove from registry: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] %s destroyed successfully", appName)))
	fmt.Println()
}

func init() {
	appDestroyCmd.Flags().BoolVarP(&forceDestroyApp, "force", "f", false, "Skip confirmation")
	appCmd.AddCommand(appDestroyCmd)
}
