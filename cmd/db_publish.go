package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/spf13/cobra"
)

var publishPort int

var publishCmd = &cobra.Command{
	Use:   "publish [name]",
	Short: "Publish database to host",
	Long:  "Expose a private database to the host machine by binding to a port",
	Args:  cobra.ExactArgs(1),
	Run:   runPublish,
}

func runPublish(cmd *cobra.Command, args []string) {
	dbName := args[0]

	if publishPort == 0 {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[error] --port flag is required"))
		fmt.Fprintln(os.Stderr, dimStyle.Render("example: yap db publish mydb --port 5432"))
		os.Exit(1)
	}

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

	if db.Published {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] database %s is already published on port %d", dbName, db.PublishedPort)))
		fmt.Fprintln(os.Stderr, dimStyle.Render(fmt.Sprintf(" use 'yap db unpublish %s' first if you want to change the port", dbName)))
		os.Exit(1)
	}

	originalPort := publishPort
	publishPort = findAvailablePort(publishPort)

	if publishPort != originalPort {
		fmt.Println(dimStyle.Render(fmt.Sprintf("[warn]  port %d is already in use", originalPort)))
		fmt.Println(dimStyle.Render(fmt.Sprintf(" next available port: %d", publishPort)))
		fmt.Println()
		fmt.Print(labelStyle.Render("continue with port ") + valueStyle.Render(fmt.Sprintf("%d", publishPort)) + labelStyle.Render("? (y/n): "))

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" {
			fmt.Println(dimStyle.Render("cancelled."))
			os.Exit(0)
		}
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize database service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> publishing database: %s", dbName)))
	fmt.Println()

	fmt.Println(progressStyle.Render("  --> stopping database..."))
	if err := dockerClient.StopContainer(db.ContainerID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to stop database: %v", err)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> reconfiguring with port binding..."))
	if err := dockerClient.RemoveContainer(db.ContainerID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to reconfigure database: %v", err)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> applying new configuration..."))
	newContainerID, err := recreateContainerWithPort(dockerClient, db, publishPort)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to apply configuration: %v", err)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> starting database..."))
	if err := dockerClient.StartContainer(newContainerID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to start database: %v", err)))
		os.Exit(1)
	}

	db.ContainerID = newContainerID
	db.Published = true
	db.PublishedPort = publishPort

	var publishedConnString string
	if db.Type == "postgres" {
		publishedConnString = fmt.Sprintf("postgresql://%s:%s@localhost:%d/%s",
			db.Username, db.Password, publishPort, db.DatabaseName)
	} else if db.Type == "valkey" {
		publishedConnString = fmt.Sprintf("valkey://%s:%s@localhost:%d",
			db.Username, db.Password, publishPort)
	}
	db.PublishedConnectionString = publishedConnString

	if err := registry.Update(*db); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to update registry: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] database published successfully"))
	fmt.Println()
	fmt.Println(labelStyle.Render("  external connection:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("host:"), valueStyle.Render("localhost"))
	fmt.Printf("    %s %s\n", dimStyle.Render("port:"), valueStyle.Render(fmt.Sprintf("%d", publishPort)))
	fmt.Println()
	fmt.Printf("    %s\n", dimStyle.Render("connection string:"))
	fmt.Printf("    %s\n", dimStyle.Render(publishedConnString))
	fmt.Println()
}

func findAvailablePort(startPort int) int {
	port := startPort
	for port < 65535 {
		if isPortAvailable(port) {
			return port
		}
		port++
	}
	return startPort
}

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func recreateContainerWithPort(dockerClient *docker.Client, db *models.Database, port int) (string, error) {
	var env []string
	var cmd []string
	var volumeTarget string

	if db.Type == "postgres" {
		env = []string{
			fmt.Sprintf("POSTGRES_USER=%s", db.Username),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", db.Password),
			fmt.Sprintf("POSTGRES_DB=%s", db.DatabaseName),
		}
		cmd = nil
		volumeTarget = "/var/lib/postgresql/data"
	} else if db.Type == "valkey" {
		env = []string{}
		cmd = []string{
			"valkey-server",
			"--requirepass", db.Password,
			"--appendonly", "yes",
		}
		volumeTarget = "/data"
	}

	labels := map[string]string{
		"yap.managed": "true",
		"yap.type":    "database",
		"yap.db.type": string(db.Type),
		"yap.db.name": db.Name,
		"yap.db.id":   db.ID,
		"yap.vpc":     db.VPC,
	}

	var image string
	if db.Type == "postgres" {
		image = "postgres:16-alpine"
	} else if db.Type == "valkey" {
		image = "valkey/valkey:8-alpine"
	}

	newContainerID, err := dockerClient.RecreateContainerWithPorts(
		db.ContainerName,
		image,
		env,
		cmd,
		labels,
		db.VolumeName,
		volumeTarget,
		db.Network,
		db.InternalPort,
		port,
	)

	if err != nil {
		return "", fmt.Errorf("failed to recreate container: %w", err)
	}

	return newContainerID, nil
}

func init() {
	publishCmd.Flags().IntVarP(&publishPort, "port", "p", 0, "Port to publish database on")
	publishCmd.MarkFlagRequired("port")
	dbCmd.AddCommand(publishCmd)
}
