package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/pkg/models"
	dockerTypes "github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
)

var appDeploymentRollbackCmd = &cobra.Command{
	Use:   "rollback [name]",
	Short: "Rollback to previous deployment",
	Long:  "Switch traffic back to the standby environment and destroy the current active environment",
	Args:  cobra.ExactArgs(1),
	Run:   runAppDeploymentRollback,
}

func init() {
	appDeploymentCmd.AddCommand(appDeploymentRollbackCmd)
}

func runAppDeploymentRollback(cmd *cobra.Command, args []string) {
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

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> rolling back deployment: %s", appName)))
	fmt.Println()

	if application.DeploymentStrategy != models.DeploymentStrategyBlueGreen {
		fmt.Fprintf(os.Stderr, "%s application is not using blue-green deployment\n", errorStyle.Render("[error]"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("  current strategy: %s", application.DeploymentStrategy)))
		os.Exit(1)
	}

	activeColor := application.DeploymentState.Active
	standbyColor := application.DeploymentState.Standby

	if standbyColor == "" || standbyColor == models.DeploymentColorDefault {
		fmt.Fprintf(os.Stderr, "%s no standby environment to rollback to\n", errorStyle.Render("[error]"))
		fmt.Println(dimStyle.Render("  deployment already confirmed or using different strategy"))
		os.Exit(1)
	}

	var activeEnv, standbyEnv *models.Environment
	if activeColor == models.DeploymentColorBlue {
		activeEnv = application.DeploymentState.Blue
		standbyEnv = application.DeploymentState.Green
	} else {
		activeEnv = application.DeploymentState.Green
		standbyEnv = application.DeploymentState.Blue
	}

	if standbyEnv == nil || len(standbyEnv.ContainerIDs) == 0 {
		fmt.Fprintf(os.Stderr, "%s standby environment is empty\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}

	fmt.Printf("  --> switching traffic from %s to %s...\n", activeColor, standbyColor)
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Printf("  --> updating %s environment (adding traffic routing)...\n", standbyColor)

	application.DeploymentState.Active = standbyColor
	application.DeploymentState.Standby = activeColor

	application.ContainerIDs = standbyEnv.ContainerIDs

	if err := registry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Printf("  [done] traffic switched to %s\n", standbyColor)
	fmt.Println()

	fmt.Printf("  --> destroying %s environment (%d instances)...\n", activeColor, len(activeEnv.ContainerIDs))

	for i, containerID := range activeEnv.ContainerIDs {
		instanceNum := i + 1
		fmt.Printf("    [%d/%d] removing %s-%d...\n", instanceNum, len(activeEnv.ContainerIDs), activeColor, instanceNum)

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

	if activeColor == models.DeploymentColorBlue {
		application.DeploymentState.Blue = nil
	} else {
		application.DeploymentState.Green = nil
	}
	application.DeploymentState.Standby = ""

	if err := registry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Printf("  [done] %s environment destroyed\n", activeColor)
	fmt.Println()
	fmt.Println(successStyle.Render("  [done] rollback completed"))
	fmt.Printf("    active environment: %s\n", valueStyle.Render(string(standbyColor)))
	fmt.Println()
}
