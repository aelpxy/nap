package cmd

import (
	"github.com/aelpxy/yap/internal/utils"
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/backup"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var backupsListCmd = &cobra.Command{
	Use:   "list [name]",
	Short: "List backups",
	Long:  "List all backups or backups for a specific database",
	Args:  cobra.MaximumNArgs(1),
	Run:   runBackupsList,
}

func runBackupsList(cmd *cobra.Command, args []string) {
	var dbName string
	if len(args) > 0 {
		dbName = args[0]
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

	backups := backupManager.ListBackups(dbName)

	if len(backups) == 0 {
		if dbName != "" {
			fmt.Println(dimStyle.Render(fmt.Sprintf("no backups found for database: %s", dbName)))
		} else {
			fmt.Println(dimStyle.Render("no backups found"))
		}
		fmt.Println()
		fmt.Println(dimStyle.Render("create a backup with: yap db backup <name>"))
		return
	}

	if dbName != "" {
		fmt.Println(titleStyle.Render(fmt.Sprintf("==> backups for: %s (%d)", dbName, len(backups))))
	} else {
		fmt.Println(titleStyle.Render(fmt.Sprintf("==> all backups (%d)", len(backups))))
	}
	fmt.Println()

	rows := [][]string{}
	var totalSize int64

	for _, bkp := range backups {
		sizeStr := utils.FormatBytes(bkp.SizeBytes)
		totalSize += bkp.SizeBytes

		statusColor := "10"
		if bkp.Status == "failed" {
			statusColor = "9"
		} else if bkp.Status == "in_progress" {
			statusColor = "14"
		}

		statusStyled := lipgloss.NewStyle().
			Foreground(lipgloss.Color(statusColor)).
			Render(bkp.Status)

		rows = append(rows, []string{
			bkp.ID,
			bkp.DatabaseName,
			bkp.DatabaseType,
			statusStyled,
			bkp.CreatedAt.Format("2006-01-02 15:04"),
			sizeStr,
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
		Headers("id", "database", "type", "status", "created", "size").
		Rows(rows...)

	fmt.Println(t)
	fmt.Println()

	var totalStr string
	if totalSize < 1024 {
		totalStr = fmt.Sprintf("%d bytes", totalSize)
	} else if totalSize < 1024*1024 {
		totalStr = fmt.Sprintf("%.1f kb", float64(totalSize)/1024)
	} else if totalSize < 1024*1024*1024 {
		totalStr = fmt.Sprintf("%.1f mb", float64(totalSize)/(1024*1024))
	} else {
		totalStr = fmt.Sprintf("%.2f gb", float64(totalSize)/(1024*1024*1024))
	}

	fmt.Println(dimStyle.Render(fmt.Sprintf("  total: %s", totalStr)))
	fmt.Println()

	fmt.Println(dimStyle.Render("  commands:"))
	fmt.Printf("    %s\n", dimStyle.Render("yap db restore <name> <id>      # restore backup"))
	fmt.Printf("    %s\n", dimStyle.Render("yap db backups delete <id>      # delete backup"))
	fmt.Println()
}

func init() {
	backupsCmd.AddCommand(backupsListCmd)
}
