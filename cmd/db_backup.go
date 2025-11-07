package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/backup"
	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/internal/utils"
	"github.com/spf13/cobra"
)

var (
	backupCompress    bool
	backupDescription string
	backupOutput      string
)

var backupCmd = &cobra.Command{
	Use:   "backup [name]",
	Short: "Create a database backup",
	Long:  "Create a backup of a database",
	Args:  cobra.ExactArgs(1),
	Run:   runBackup,
}

func runBackup(cmd *cobra.Command, args []string) {
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

	backupManager, err := backup.NewManager(dockerClient)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize backup manager: %v", err)))
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> creating backup: %s", dbName)))
	fmt.Println()

	fmt.Println(progressStyle.Render("  --> connecting to database..."))

	var backupMethod string
	if db.Type == "postgres" {
		backupMethod = "running pg_dump..."
	} else {
		backupMethod = "creating snapshot..."
	}
	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> %s", backupMethod)))

	if backupCompress {
		fmt.Println(progressStyle.Render("  --> compressing backup..."))
	}

	bkp, err := backupManager.CreateBackup(db, backupCompress, backupDescription)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to create backup: %v", err)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> saving metadata..."))
	fmt.Println()

	fmt.Println(successStyle.Render("  [done] backup created successfully"))
	fmt.Println()

	fmt.Println(labelStyle.Render("  backup details:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("id:"), valueStyle.Render(bkp.ID))

	sizeStr := utils.FormatBytes(bkp.SizeBytes)
	if backupCompress {
		sizeStr += " (compressed)"
	}

	fmt.Printf("    %s %s\n", dimStyle.Render("size:"), valueStyle.Render(sizeStr))
	fmt.Printf("    %s %s\n", dimStyle.Render("location:"), valueStyle.Render(bkp.Path))

	if bkp.Version != "" {
		fmt.Printf("    %s %s\n", dimStyle.Render("version:"), valueStyle.Render(bkp.Version))
	}
	fmt.Println()

	fmt.Println(dimStyle.Render(fmt.Sprintf("  restore with: yap db restore %s %s", dbName, bkp.ID)))
	fmt.Println()
}

func init() {
	backupCmd.Flags().BoolVar(&backupCompress, "compress", true, "compress backup")
	backupCmd.Flags().StringVar(&backupDescription, "description", "", "backup description")
	backupCmd.Flags().StringVar(&backupOutput, "output", "", "custom output directory")
	dbCmd.AddCommand(backupCmd)
}
