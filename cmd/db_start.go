package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a database",
	Long:  "Start a stopped database",
	Args:  cobra.ExactArgs(1),
	Run:   runStart,
}

func runStart(cmd *cobra.Command, args []string) {
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

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> starting database: %s", dbName)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  --> starting database..."))
	if err := dockerClient.StartContainer(db.ContainerID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to start database: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [ok] database started successfully"))
	fmt.Println()

	fmt.Printf("check status with: %s\n", infoStyle.Render(fmt.Sprintf("yap db status %s", dbName)))
	fmt.Println()
}

func init() {
	dbCmd.AddCommand(startCmd)
}
