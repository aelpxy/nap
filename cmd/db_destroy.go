package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/spf13/cobra"
)

var (
	forceDestroy bool
)

var destroyCmd = &cobra.Command{
	Use:   "destroy [name]",
	Short: "Destroy a database",
	Long:  "Permanently destroy a database and its data volume",
	Args:  cobra.ExactArgs(1),
	Run:   runDestroy,
}

func runDestroy(cmd *cobra.Command, args []string) {
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

	if !forceDestroy {
		fmt.Println(errorStyle.Render(fmt.Sprintf("[warn]  warning: this will permanently destroy the database '%s'", dbName)))
		fmt.Println(labelStyle.Render("   all data will be lost and cannot be recovered."))
		fmt.Println()
		fmt.Print(labelStyle.Render("type the database name to confirm: "))

		var confirmation string
		fmt.Scanln(&confirmation)

		if strings.TrimSpace(confirmation) != dbName {
			fmt.Println(labelStyle.Render("\ndestruction cancelled."))
			return
		}
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize database service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println()
	fmt.Println(errorStyle.Render(fmt.Sprintf("==> destroying database: %s", dbName)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  --> stopping database..."))
	_ = dockerClient.StopContainer(db.ContainerID)
	fmt.Println(successStyle.Render("  [ok] database stopped"))

	fmt.Println(labelStyle.Render("  --> removing database..."))
	if err := dockerClient.RemoveContainer(db.ContainerID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to remove database: %v", err)))
	} else {
		fmt.Println(successStyle.Render("  [ok] database removed"))
	}

	fmt.Println(labelStyle.Render("  --> removing volume..."))
	if err := dockerClient.DeleteVolume(db.VolumeName); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to remove volume: %v", err)))
	} else {
		fmt.Println(successStyle.Render("  [ok] volume removed"))
	}

	fmt.Println(labelStyle.Render("  --> removing from registry..."))
	if err := registry.Remove(dbName); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to remove from registry: %v", err)))
		os.Exit(1)
	}
	fmt.Println(successStyle.Render("  [ok] removed from registry"))

	fmt.Println()
	fmt.Println(successStyle.Render("  [ok] database destroyed successfully"))
	fmt.Println()
}

func init() {
	destroyCmd.Flags().BoolVarP(&forceDestroy, "force", "f", false, "Force destruction without confirmation")
	dbCmd.AddCommand(destroyCmd)
}
