package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/app"
	"github.com/spf13/cobra"
)

var appDeploymentsCmd = &cobra.Command{
	Use:   "deployments [name]",
	Short: "List deployment history",
	Long:  "Display the deployment history for an application showing all previous deployments",
	Args:  cobra.ExactArgs(1),
	Run:   runAppDeployments,
}

func init() {
	appCmd.AddCommand(appDeploymentsCmd)
}

func runAppDeployments(cmd *cobra.Command, args []string) {
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

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> deployment history: %s", appName)))
	fmt.Println()

	if len(application.DeploymentHistory) == 0 {
		fmt.Println(dimStyle.Render("  no deployment history available"))
		fmt.Println()
		return
	}

	for i := len(application.DeploymentHistory) - 1; i >= 0; i-- {
		deployment := application.DeploymentHistory[i]

		var statusStr string
		switch deployment.Status {
		case "active":
			statusStr = successStyle.Render("active")
		case "superseded":
			statusStr = dimStyle.Render("superseded")
		case "rolled-back":
			statusStr = errorStyle.Render("rolled-back")
		default:
			statusStr = deployment.Status
		}

		fmt.Println(labelStyle.Render(fmt.Sprintf("  deployment #%d:", len(application.DeploymentHistory)-i)))
		fmt.Printf("    id: %s\n", valueStyle.Render(deployment.ID))
		fmt.Printf("    image: %s\n", dimStyle.Render(deployment.ImageID[:min(len(deployment.ImageID), 20)]))
		fmt.Printf("    strategy: %s\n", valueStyle.Render(string(deployment.Strategy)))
		fmt.Printf("    status: %s\n", statusStr)
		fmt.Printf("    deployed: %s\n", dimStyle.Render(deployment.DeployedAt.Format("2006-01-02 15:04:05")))
		fmt.Println()
	}

	fmt.Println(labelStyle.Render("  rollback:"))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap app rollback %s --version N", appName)))
	fmt.Println()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
