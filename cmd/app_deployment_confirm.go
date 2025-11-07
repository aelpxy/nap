package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
)

var appDeploymentConfirmCmd = &cobra.Command{
	Use:   "confirm [name]",
	Short: "Confirm blue-green deployment",
	Long:  "Destroy the standby environment after confirming deployment success",
	Args:  cobra.ExactArgs(1),
	Run:   runAppDeploymentConfirm,
}

func init() {
	appDeploymentCmd.AddCommand(appDeploymentConfirmCmd)
}

func runAppDeploymentConfirm(cmd *cobra.Command, args []string) {
	appName := args[0]

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	application, err := registry.Get(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s application '%s' not found\n", errorStyle.Render("[error]"), appName)
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> confirming deployment: %s", appName)))
	fmt.Println()

	if application.DeploymentStrategy != models.DeploymentStrategyBlueGreen {
		fmt.Fprintf(os.Stderr, "%s application is not using blue-green deployment\n", errorStyle.Render("[error]"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("  current strategy: %s", application.DeploymentStrategy)))
		os.Exit(1)
	}

	standbyColor := application.DeploymentState.Standby
	if standbyColor == "" || standbyColor == models.DeploymentColorDefault {
		fmt.Fprintf(os.Stderr, "%s no standby environment to destroy\n", errorStyle.Render("[error]"))
		fmt.Println(dimStyle.Render("  deployment already confirmed or using different strategy"))
		os.Exit(1)
	}

	var standbyEnv *models.Environment
	if standbyColor == models.DeploymentColorBlue {
		standbyEnv = application.DeploymentState.Blue
	} else {
		standbyEnv = application.DeploymentState.Green
	}

	if standbyEnv == nil || len(standbyEnv.ContainerIDs) == 0 {
		fmt.Fprintf(os.Stderr, "%s standby environment is empty\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}

	fmt.Printf("  --> destroying %s environment (%d instances)...\n", standbyColor, len(standbyEnv.ContainerIDs))

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	ctx := context.Background()

	for i, containerID := range standbyEnv.ContainerIDs {
		instanceNum := i + 1
		fmt.Printf("    [%d/%d] removing %s-%d...\n", instanceNum, len(standbyEnv.ContainerIDs), standbyColor, instanceNum)

		timeout := 10
		if err := dockerClient.GetClient().ContainerStop(ctx, containerID, dockerTypes.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			fmt.Printf("    [warn] failed to stop container: %v\n", err)
		}

		if err := dockerClient.GetClient().ContainerRemove(ctx, containerID, dockerTypes.RemoveOptions{
			Force: true,
		}); err != nil {
			fmt.Printf("    [warn] failed to remove container: %v\n", err)
		}
	}

	fmt.Printf("  [done] %s environment destroyed\n", standbyColor)

	if standbyColor == models.DeploymentColorBlue {
		application.DeploymentState.Blue = nil
	} else {
		application.DeploymentState.Green = nil
	}
	application.DeploymentState.Standby = ""

	if err := registry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  [done] deployment confirmed"))
	fmt.Printf("    active environment: %s\n", valueStyle.Render(string(application.DeploymentState.Active)))
	fmt.Println()
}
