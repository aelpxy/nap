package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/internal/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var appStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show application status",
	Long:  "Display detailed status information for an application",
	Args:  cobra.ExactArgs(1),
	Run:   runAppStatus,
}

func runAppStatus(cmd *cobra.Command, args []string) {
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

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	var containerStatuses []string
	for _, containerID := range application.ContainerIDs {
		status, err := dockerClient.GetContainerStatus(containerID)
		if err != nil {
			containerStatuses = append(containerStatuses, "unknown")
		} else {
			containerStatuses = append(containerStatuses, status)
		}
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> status: %s", appName)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  application information:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("name:"), valueStyle.Render(application.Name))
	fmt.Printf("    %s %s\n", dimStyle.Render("id:"), valueStyle.Render(application.ID))
	fmt.Printf("    %s %s\n", dimStyle.Render("vpc:"), valueStyle.Render(application.VPC))

	statusColor := "10"
	switch application.Status {
	case "running":
		statusColor = "10"
	case "stopped":
		statusColor = "240"
	case "deploying":
		statusColor = "14"
	case "failed":
		statusColor = "9"
	case "scaling":
		statusColor = "11"
	}
	statusStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(string(application.Status))
	fmt.Printf("    %s %s\n", dimStyle.Render("status:"), statusStyled)
	fmt.Println()

	fmt.Println(labelStyle.Render("  deployment:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("build type:"), valueStyle.Render(string(application.BuildType)))
	fmt.Printf("    %s %s\n", dimStyle.Render("image:"), dimStyle.Render(utils.TruncateID(application.ImageID, 12)))
	fmt.Printf("    %s %s\n", dimStyle.Render("instances:"), valueStyle.Render(fmt.Sprintf("%d", application.Instances)))
	fmt.Printf("    %s %s\n", dimStyle.Render("strategy:"), valueStyle.Render(string(application.DeploymentStrategy)))
	fmt.Printf("    %s %s\n", dimStyle.Render("deployed:"), valueStyle.Render(application.LastDeployedAt.Format("2006-01-02 15:04:05")))
	fmt.Println()

	fmt.Println(labelStyle.Render("  resources:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("memory:"), valueStyle.Render(fmt.Sprintf("%d MB", application.Memory)))
	fmt.Printf("    %s %s\n", dimStyle.Render("cpu:"), valueStyle.Render(fmt.Sprintf("%.1f", application.CPU)))
	fmt.Printf("    %s %s\n", dimStyle.Render("port:"), valueStyle.Render(fmt.Sprintf("%d", application.Port)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  network:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("url:"), valueStyle.Render(application.PublishedURL))
	fmt.Printf("    %s %s\n", dimStyle.Render("internal hostname:"), dimStyle.Render(application.InternalHostname))
	if application.Published {
		fmt.Printf("    %s %s\n", dimStyle.Render("published:"), successStyle.Render("yes"))
		if application.SSLEnabled {
			fmt.Printf("    %s %s\n", dimStyle.Render("ssl/tls:"), successStyle.Render("enabled"))
		}
		if len(application.CustomDomains) > 0 {
			fmt.Printf("    %s %s\n", dimStyle.Render("custom domains:"), valueStyle.Render(fmt.Sprintf("%d", len(application.CustomDomains))))
		}
	} else {
		fmt.Printf("    %s %s\n", dimStyle.Render("published:"), dimStyle.Render("no (local only)"))
	}
	fmt.Println()

	fmt.Println(labelStyle.Render("  health checks:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("endpoint:"), valueStyle.Render(application.HealthCheckPath))
	fmt.Printf("    %s %s\n", dimStyle.Render("interval:"), valueStyle.Render(fmt.Sprintf("%ds", application.HealthCheckInterval)))
	fmt.Printf("    %s %s\n", dimStyle.Render("timeout:"), valueStyle.Render(fmt.Sprintf("%ds", application.HealthCheckTimeout)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  instances:"))
	for i, containerID := range application.ContainerIDs {
		status := containerStatuses[i]
		statusColor := "10"
		if status != "running" {
			statusColor = "240"
		}
		statusStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(status)
		fmt.Printf("    %s %s - %s\n", dimStyle.Render(fmt.Sprintf("[%d]", i+1)), dimStyle.Render(utils.TruncateID(containerID, 12)), statusStyled)
	}
	fmt.Println()

	if len(application.LinkedDatabases) > 0 {
		fmt.Println(labelStyle.Render("  linked databases:"))
		for _, dbName := range application.LinkedDatabases {
			fmt.Printf("    %s %s\n", dimStyle.Render("•"), valueStyle.Render(dbName))
		}
		fmt.Println()
	}

	if len(application.Volumes) > 0 {
		fmt.Println(labelStyle.Render("  volumes:"))
		for _, vol := range application.Volumes {
			fmt.Printf("    %s %s → %s\n", dimStyle.Render("•"), valueStyle.Render(vol.Name), dimStyle.Render(vol.MountPath))
		}
		fmt.Println()
	}

	if len(application.EnvVars) > 0 {
		fmt.Println(labelStyle.Render("  environment variables:"))
		for key := range application.EnvVars {
			fmt.Printf("    %s %s\n", dimStyle.Render("•"), valueStyle.Render(key))
		}
		fmt.Println()
	}

	fmt.Println(labelStyle.Render("  metadata:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("created:"), valueStyle.Render(application.CreatedAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("    %s %s\n", dimStyle.Render("updated:"), valueStyle.Render(application.UpdatedAt.Format("2006-01-02 15:04:05")))
	fmt.Println()

	fmt.Println(dimStyle.Render("  quick actions:"))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("nap app logs %s         # view logs", appName)))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("nap app console %s      # open shell", appName)))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("nap app restart %s      # restart", appName)))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("nap app scale %s --add 1   # add instance", appName)))
	fmt.Println()
}

func init() {
	appCmd.AddCommand(appStatusCmd)
}
