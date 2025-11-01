package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var appRestartCmd = &cobra.Command{
	Use:   "restart [name]",
	Short: "Restart an application",
	Long:  "Restart all instances of an application",
	Args:  cobra.ExactArgs(1),
	Run:   runAppRestart,
}

func runAppRestart(cmd *cobra.Command, args []string) {
	appName := args[0]

	lockManager := app.GetGlobalLockManager()
	if err := lockManager.TryLock(appName, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "%s another operation in progress for %s\n", errorStyle.Render("[error]"), appName)
		fmt.Println(dimStyle.Render("  wait for the current operation to complete or try again in a few seconds"))
		os.Exit(1)
	}
	defer lockManager.Unlock(appName)

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

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> restarting application: %s", appName)))
	fmt.Println()

	vpcRegistry, err := database.NewVPCRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to load vpc registry: %v", err)))
		os.Exit(1)
	}
	if err := vpcRegistry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize vpc registry: %v", err)))
		os.Exit(1)
	}

	vpc, err := vpcRegistry.Get(application.VPC)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to get vpc: %v", err)))
		os.Exit(1)
	}

	newContainerIDs := make([]string, 0, len(application.ContainerIDs))
	for i, containerID := range application.ContainerIDs {
		instanceNum := i + 1
		fmt.Println(progressStyle.Render(fmt.Sprintf("  --> recreating instance %d/%d...", instanceNum, len(application.ContainerIDs))))

		newID, err := app.RecreateContainer(
			dockerClient,
			containerID,
			application,
			vpc.NetworkName,
			instanceNum,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to recreate instance: %v", err)))
			os.Exit(1)
		}
		newContainerIDs = append(newContainerIDs, newID)
	}

	application.ContainerIDs = newContainerIDs
	if err := registry.Update(*application); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to update registry: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] %s restarted successfully", appName)))
	fmt.Println()
	fmt.Println(dimStyle.Render("  containers recreated with updated configuration"))
}

func init() {
	appCmd.AddCommand(appRestartCmd)
}
