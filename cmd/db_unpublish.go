package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/spf13/cobra"
)

var unpublishCmd = &cobra.Command{
	Use:   "unpublish [name]",
	Short: "Unpublish database from host",
	Long:  "Make a published database private again by removing port binding",
	Args:  cobra.ExactArgs(1),
	Run:   runUnpublish,
}

func runUnpublish(cmd *cobra.Command, args []string) {
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

	if !db.Published {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] database %s is not published", dbName)))
		fmt.Fprintln(os.Stderr, dimStyle.Render(" this database is already private"))
		os.Exit(1)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize database service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> unpublishing database: %s", dbName)))
	fmt.Println()

	fmt.Println(progressStyle.Render("  --> stopping database..."))
	if err := dockerClient.StopContainer(db.ContainerID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to stop database: %v", err)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> reconfiguring database..."))
	if err := dockerClient.RemoveContainer(db.ContainerID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to reconfigure database: %v", err)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> applying new configuration..."))
	newContainerID, err := recreateContainerWithoutPort(dockerClient, db)
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
	db.Published = false
	db.PublishedPort = 0
	db.PublishedConnectionString = ""

	if err := registry.Update(*db); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to update registry: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] database unpublished successfully"))
	fmt.Println()
	fmt.Println(dimStyle.Render("  [info] database is now private and only accessible within the vpc"))
	fmt.Println(dimStyle.Render(fmt.Sprintf("  [info] access using: %s", db.ConnectionString)))
	fmt.Println()
}

func recreateContainerWithoutPort(dockerClient *docker.Client, db *models.Database) (string, error) {
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

	newContainerID, err := dockerClient.RemovePortBindings(
		db.ContainerName,
		image,
		env,
		cmd,
		labels,
		db.VolumeName,
		volumeTarget,
		db.Network,
	)

	if err != nil {
		return "", fmt.Errorf("failed to recreate container: %w", err)
	}

	return newContainerID, nil
}

func init() {
	dbCmd.AddCommand(unpublishCmd)
}
