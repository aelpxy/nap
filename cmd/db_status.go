package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/internal/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show database status",
	Long:  "Display detailed status information for a database",
	Args:  cobra.ExactArgs(1),
	Run:   runStatus,
}

func runStatus(cmd *cobra.Command, args []string) {
	dbName := args[0]

	registry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	db, err := registry.Get(dbName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] database not found: %v", err)))
		os.Exit(1)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize database service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	containerStatus, err := dockerClient.GetContainerStatus(db.ContainerID)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to get database status: %v", err)))
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> status: %s", dbName)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  database information:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("name:"), valueStyle.Render(db.Name))
	fmt.Printf("    %s %s\n", dimStyle.Render("type:"), valueStyle.Render(string(db.Type)))
	fmt.Printf("    %s %s\n", dimStyle.Render("id:"), valueStyle.Render(db.ID))
	fmt.Printf("    %s %s\n", dimStyle.Render("vpc:"), valueStyle.Render(db.VPC))

	statusColor := "10"
	if containerStatus != "running" {
		statusColor = "240"
	}
	statusStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(containerStatus)
	fmt.Printf("    %s %s\n", dimStyle.Render("status:"), statusStyled)
	fmt.Println()

	fmt.Println(labelStyle.Render("  system details:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("instance id:"), valueStyle.Render(utils.TruncateID(db.ContainerID, 12)))
	fmt.Printf("    %s %s\n", dimStyle.Render("instance name:"), valueStyle.Render(db.ContainerName))
	fmt.Printf("    %s %s\n", dimStyle.Render("volume:"), valueStyle.Render(db.VolumeName))
	fmt.Printf("    %s %s\n", dimStyle.Render("network:"), valueStyle.Render(db.Network))
	fmt.Println()

	fmt.Println(labelStyle.Render("  connection:"))
	if db.Type == "postgres" {
		fmt.Printf("    %s %s\n", dimStyle.Render("database:"), valueStyle.Render(db.DatabaseName))
		fmt.Printf("    %s %s\n", dimStyle.Render("username:"), valueStyle.Render(db.Username))
		fmt.Printf("    %s %s\n", dimStyle.Render("password:"), dimStyle.Render("••••••••"))
	} else if db.Type == "valkey" {
		fmt.Printf("    %s %s\n", dimStyle.Render("password:"), dimStyle.Render("••••••••"))
		fmt.Printf("    %s %s\n", dimStyle.Render("protocol:"), valueStyle.Render("redis"))
	}
	fmt.Println()

	fmt.Println(labelStyle.Render("  network:"))
	if db.Published {
		fmt.Printf("    %s %s\n", dimStyle.Render("published:"), successStyle.Render("yes"))
		fmt.Printf("    %s %s\n", dimStyle.Render("host port:"), valueStyle.Render(fmt.Sprintf("%d", db.PublishedPort)))
		fmt.Printf("    %s %s\n", dimStyle.Render("internal hostname:"), valueStyle.Render(db.InternalHostname))
		fmt.Printf("    %s %s\n", dimStyle.Render("internal port:"), valueStyle.Render(fmt.Sprintf("%d", db.InternalPort)))
	} else {
		fmt.Printf("    %s %s\n", dimStyle.Render("published:"), dimStyle.Render("no (private)"))
		fmt.Printf("    %s %s\n", dimStyle.Render("internal hostname:"), valueStyle.Render(db.InternalHostname))
		fmt.Printf("    %s %s\n", dimStyle.Render("internal port:"), valueStyle.Render(fmt.Sprintf("%d", db.InternalPort)))
	}
	fmt.Println()

	if len(db.LinkedApps) > 0 {
		fmt.Println(labelStyle.Render("  linked applications:"))
		for _, appName := range db.LinkedApps {
			fmt.Printf("    %s %s\n", dimStyle.Render("•"), valueStyle.Render(appName))
		}
		fmt.Println()
	}

	fmt.Println(labelStyle.Render("  metadata:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("created:"), valueStyle.Render(db.CreatedAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("    %s %s\n", dimStyle.Render("updated:"), valueStyle.Render(db.UpdatedAt.Format("2006-01-02 15:04:05")))
	fmt.Println()

	fmt.Println(dimStyle.Render("  quick actions:"))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap db shell %s         # open database shell", dbName)))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap db logs %s          # view logs", dbName)))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap db credentials %s   # show connection details", dbName)))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap db backup %s        # create backup", dbName)))
	fmt.Println()
}

func init() {
	dbCmd.AddCommand(statusCmd)
}
