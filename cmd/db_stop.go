package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a database",
	Long:  "Stop a running database",
	Args:  cobra.ExactArgs(1),
	Run:   runStop,
}

func runStop(cmd *cobra.Command, args []string) {
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

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> stopping database: %s", dbName)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  --> stopping database..."))
	if err := dockerClient.StopContainer(db.ContainerID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to stop database: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [ok] database stopped successfully"))
	fmt.Println()

	fmt.Printf("start it again with: %s\n", infoStyle.Render(fmt.Sprintf("nap db start %s", dbName)))
	fmt.Println()
}

func init() {
	dbCmd.AddCommand(stopCmd)
}
