package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/aelpxy/nap/internal/backup"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var (
	deleteForce bool
)

var backupsDeleteCmd = &cobra.Command{
	Use:   "delete [backup-id]",
	Short: "Delete a backup",
	Long:  "Delete a backup by ID",
	Args:  cobra.ExactArgs(1),
	Run:   runBackupsDelete,
}

func runBackupsDelete(cmd *cobra.Command, args []string) {
	backupID := args[0]

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

	if !deleteForce {
		fmt.Println(titleStyle.Render("==> delete backup"))
		fmt.Println()
		fmt.Println(labelStyle.Render("  backup details:"))
		fmt.Printf("    %s %s\n", dimStyle.Render("id:"), valueStyle.Render(bkp.ID))
		fmt.Printf("    %s %s\n", dimStyle.Render("database:"), valueStyle.Render(bkp.DatabaseName))
		fmt.Printf("    %s %s\n", dimStyle.Render("type:"), valueStyle.Render(bkp.DatabaseType))
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

		fmt.Println(errorStyle.Render("[warn]  warning: this backup will be permanently deleted"))
		fmt.Println(labelStyle.Render("   this action cannot be undone"))
		fmt.Println()
		fmt.Print(labelStyle.Render("type 'delete' to confirm: "))

		var confirmation string
		fmt.Scanln(&confirmation)

		if strings.TrimSpace(strings.ToLower(confirmation)) != "delete" {
			fmt.Println(labelStyle.Render("\ndeletion cancelled."))
			return
		}
		fmt.Println()
	}

	fmt.Println(progressStyle.Render("  --> deleting backup..."))

	if err := backupManager.DeleteBackup(backupID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to delete backup: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] backup deleted successfully"))
	fmt.Println()
}

func init() {
	backupsDeleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "skip confirmation")
	backupsCmd.AddCommand(backupsDeleteCmd)
}
