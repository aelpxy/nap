package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/app"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all applications",
	Long:  "List all deployed applications",
	Run:   runAppList,
}

func runAppList(cmd *cobra.Command, args []string) {
	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	applications, err := registry.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to list applications: %v", err)))
		os.Exit(1)
	}

	if len(applications) == 0 {
		fmt.Println(dimStyle.Render("no applications found."))
		fmt.Println()
		fmt.Printf("deploy one with: %s\n", dimStyle.Render("nap app deploy myapp ."))
		return
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> applications (%d)", len(applications))))
	fmt.Println()

	rows := [][]string{}
	for _, app := range applications {
		statusColor := "42"
		switch app.Status {
		case "running":
			statusColor = "42"
		case "stopped":
			statusColor = "241"
		case "deploying":
			statusColor = "14"
		case "failed":
			statusColor = "9"
		case "scaling":
			statusColor = "11"
		}

		statusStyled := lipgloss.NewStyle().
			Foreground(lipgloss.Color(statusColor)).
			Render(string(app.Status))

		instancesText := fmt.Sprintf("%d", app.Instances)

		resourcesText := fmt.Sprintf("%dMB / %.1fCPU", app.Memory, app.CPU)

		rows = append(rows, []string{
			app.Name,
			string(app.BuildType),
			app.VPC,
			statusStyled,
			instancesText,
			resourcesText,
			app.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().
					Foreground(lipgloss.Color("86")).
					Bold(true).
					Align(lipgloss.Center)
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		}).
		Headers("name", "build", "vpc", "status", "instances", "resources", "created").
		Rows(rows...)

	fmt.Println(t)
	fmt.Println()

	fmt.Println(dimStyle.Render("  common commands:"))
	fmt.Printf("    %s\n", dimStyle.Render("nap app status <name>   # view details"))
	fmt.Printf("    %s\n", dimStyle.Render("nap app logs <name>     # view logs"))
	fmt.Printf("    %s\n", dimStyle.Render("nap app restart <name>  # restart application"))
	fmt.Println()
}

func init() {
	appCmd.AddCommand(appListCmd)
}
