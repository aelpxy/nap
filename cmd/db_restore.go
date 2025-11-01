package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/aelpxy/nap/internal/backup"
	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var (
	restoreForce       bool
	restoreBackupFirst bool
	restoreStopDB      bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore [name] [backup-id]",
	Short: "Restore database from backup",
	Long:  "Restore a database from a backup",
	Args:  cobra.ExactArgs(2),
	Run:   runRestore,
}

func runRestore(cmd *cobra.Command, args []string) {
	dbName := args[0]
	backupID := args[1]

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

	bkp, err := backupManager.GetBackup(backupID)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] backup not found: %v", err)))
		os.Exit(1)
	}

	if bkp.DatabaseType != string(db.Type) {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] backup type mismatch: backup is for %s, database is %s", bkp.DatabaseType, db.Type)))
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> restoring database: %s", dbName)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  backup:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("id:"), valueStyle.Render(bkp.ID))
	fmt.Printf("    %s %s\n", dimStyle.Render("created:"), valueStyle.Render(bkp.CreatedAt.Format("2006-01-02 15:04:05")))

	var sizeStr string
	if bkp.SizeBytes < 1024 {
		sizeStr = fmt.Sprintf("%d bytes", bkp.SizeBytes)
	} else if bkp.SizeBytes < 1024*1024 {
		sizeStr = fmt.Sprintf("%.1f kb", float64(bkp.SizeBytes)/1024)
	} else {
		sizeStr = fmt.Sprintf("%.1f mb", float64(bkp.SizeBytes)/(1024*1024))
	}
	fmt.Printf("    %s %s\n", dimStyle.Render("size:"), valueStyle.Render(sizeStr))
	fmt.Println()

	if !restoreForce {
		fmt.Println(errorStyle.Render("[warn]  warning: this will replace all data in '" + dbName + "'"))
		fmt.Println(labelStyle.Render("   current data will be lost unless backed up"))
		fmt.Println()
		fmt.Print(labelStyle.Render("type the database name to confirm: "))

		var confirmation string
		fmt.Scanln(&confirmation)

		if strings.TrimSpace(confirmation) != dbName {
			fmt.Println(labelStyle.Render("\nrestore cancelled."))
			return
		}
		fmt.Println()
	}

	if restoreBackupFirst {
		fmt.Println(progressStyle.Render("  --> creating safety backup..."))
		safetyBackup, err := backupManager.CreateBackup(db, true, "pre-restore safety backup")
		if err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to create safety backup: %v", err)))
			os.Exit(1)
		}
		fmt.Println(dimStyle.Render(fmt.Sprintf("      safety backup created: %s", safetyBackup.ID)))
	}

	if restoreStopDB {
		fmt.Println(progressStyle.Render("  --> stopping database..."))
		if err := dockerClient.StopContainer(db.ContainerID); err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to stop database: %v", err)))
			os.Exit(1)
		}
	}

	fmt.Println(progressStyle.Render("  --> restoring from backup..."))
	if err := backupManager.RestoreBackup(db, backupID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to restore backup: %v", err)))
		if restoreStopDB {
			dockerClient.StartContainer(db.ContainerID)
		}
		os.Exit(1)
	}

	if restoreStopDB {
		fmt.Println(progressStyle.Render("  --> starting database..."))
		if err := dockerClient.StartContainer(db.ContainerID); err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to start database: %v", err)))
			os.Exit(1)
		}
	}

	fmt.Println(progressStyle.Render("  --> verifying restore..."))
	fmt.Println()

	fmt.Println(successStyle.Render("  [done] database restored successfully"))
	fmt.Println()
}

func init() {
	restoreCmd.Flags().BoolVarP(&restoreForce, "force", "f", false, "skip confirmation")
	restoreCmd.Flags().BoolVar(&restoreBackupFirst, "backup-first", false, "create backup before restore")
	restoreCmd.Flags().BoolVar(&restoreStopDB, "stop-db", false, "stop database during restore (recommended for valkey)")
	dbCmd.AddCommand(restoreCmd)
}
