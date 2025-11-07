package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/utils"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/spf13/cobra"
)

var appDeploymentStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show deployment status",
	Long:  "Display current deployment state for blue-green deployments",
	Args:  cobra.ExactArgs(1),
	Run:   runAppDeploymentStatus,
}

func init() {
	appDeploymentCmd.AddCommand(appDeploymentStatusCmd)
}

func runAppDeploymentStatus(cmd *cobra.Command, args []string) {
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

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> deployment status: %s", appName)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  deployment strategy:"))
	strategy := application.DeploymentStrategy
	if strategy == "" {
		strategy = models.DeploymentStrategyRecreate
	}
	fmt.Printf("    %s\n", valueStyle.Render(string(strategy)))
	fmt.Println()

	if strategy == models.DeploymentStrategyBlueGreen {
		fmt.Println(labelStyle.Render("  blue-green state:"))

		active := application.DeploymentState.Active
		if active == "" || active == models.DeploymentColorDefault {
			fmt.Printf("    active: %s\n", dimStyle.Render("none (using recreate strategy)"))
		} else {
			fmt.Printf("    active: %s\n", valueStyle.Render(string(active)))
			fmt.Printf("    standby: %s\n", dimStyle.Render(string(application.DeploymentState.Standby)))
		}
		fmt.Println()

		if application.DeploymentState.Blue != nil {
			fmt.Println(labelStyle.Render("  blue environment:"))
			fmt.Printf("    instances: %s\n", valueStyle.Render(fmt.Sprintf("%d", len(application.DeploymentState.Blue.ContainerIDs))))
			fmt.Printf("    image: %s\n", dimStyle.Render(utils.TruncateID(application.DeploymentState.Blue.ImageID, 12)))
			fmt.Printf("    deployed: %s\n", dimStyle.Render(application.DeploymentState.Blue.DeployedAt.Format("2006-01-02 15:04:05")))
			fmt.Println()
		}

		if application.DeploymentState.Green != nil {
			fmt.Println(labelStyle.Render("  green environment:"))
			fmt.Printf("    instances: %s\n", valueStyle.Render(fmt.Sprintf("%d", len(application.DeploymentState.Green.ContainerIDs))))
			fmt.Printf("    image: %s\n", dimStyle.Render(utils.TruncateID(application.DeploymentState.Green.ImageID, 12)))
			fmt.Printf("    deployed: %s\n", dimStyle.Render(application.DeploymentState.Green.DeployedAt.Format("2006-01-02 15:04:05")))
			fmt.Println()
		}

		if application.DeploymentState.Standby != "" && application.DeploymentState.Standby != models.DeploymentColorDefault {
			fmt.Println(labelStyle.Render("  available actions:"))
			fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap app deployment confirm %s   # destroy %s environment", appName, application.DeploymentState.Standby)))
			fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap app deployment rollback %s  # switch back to %s", appName, application.DeploymentState.Standby)))
			fmt.Println()
		}
	} else {
		fmt.Println(dimStyle.Render(fmt.Sprintf("  using %s strategy - no blue-green state available", strategy)))
		fmt.Println()
	}
}
