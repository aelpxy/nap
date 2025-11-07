package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/internal/utils"
	"github.com/spf13/cobra"
)

var (
	volumeBackupDescription string
	volumeBackupOutput      string
)

var appVolumeBackupCmd = &cobra.Command{
	Use:   "backup [app-name] [volume-name]",
	Short: "Backup a volume to a tarball",
	Long: `Create a compressed backup of a volume's data.

The backup is stored as a compressed tarball (.tar.gz) and can be restored later.

Examples:
  yap app volume backup myapp uploads
  yap app volume backup myapp data --description "pre-deployment"
  yap app volume backup myapp uploads --output /backups/uploads.tar.gz`,
	Args: cobra.ExactArgs(2),
	Run:  runAppVolumeBackup,
}

var appVolumeRestoreCmd = &cobra.Command{
	Use:   "restore [app-name] [volume-name] [backup-file]",
	Short: "Restore a volume from a backup",
	Long: `Restore a volume's data from a backup tarball.

WARNING: This will replace all data in the volume with the backup data.

Examples:
  yap app volume restore myapp uploads backup-20251101-001234.tar.gz
  yap app volume restore myapp data /path/to/backup.tar.gz`,
	Args: cobra.ExactArgs(3),
	Run:  runAppVolumeRestore,
}

var appVolumeBackupsCmd = &cobra.Command{
	Use:   "backups [app-name] [volume-name]",
	Short: "List volume backups",
	Long: `List all volume backups or filter by app/volume.

Examples:
  yap app volume backups              # list all backups
  yap app volume backups myapp        # list backups for app
  yap app volume backups myapp data   # list backups for specific volume`,
	Args: cobra.MaximumNArgs(2),
	Run:  runAppVolumeBackups,
}

var appVolumeBackupDeleteCmd = &cobra.Command{
	Use:   "backup-delete [backup-id]",
	Short: "Delete a volume backup",
	Args:  cobra.ExactArgs(1),
	Run:   runAppVolumeBackupDelete,
}

func init() {
	appVolumeCmd.AddCommand(appVolumeBackupCmd)
	appVolumeCmd.AddCommand(appVolumeRestoreCmd)
	appVolumeCmd.AddCommand(appVolumeBackupsCmd)
	appVolumeCmd.AddCommand(appVolumeBackupDeleteCmd)

	appVolumeBackupCmd.Flags().StringVar(&volumeBackupDescription, "description", "", "Backup description")
	appVolumeBackupCmd.Flags().StringVar(&volumeBackupOutput, "output", "", "Output file path (default: ~/.yap/volume-backups/)")
}

func runAppVolumeBackup(cmd *cobra.Command, args []string) {
	appName := args[0]
	volumeName := args[1]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> backing up volume: %s/%s", appName, volumeName)))
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize docker client: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	volumeManager := app.NewVolumeManager(dockerClient, registry)
	backupManager := app.NewVolumeBackupManager(dockerClient, volumeManager)

	ctx := context.Background()
	volumeInfo, err := volumeManager.InspectVolume(ctx, appName, volumeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println("  " + dimStyle.Render("backing up:"))
	fmt.Printf("    app: %s\n", valueStyle.Render(appName))
	fmt.Printf("    volume: %s\n", valueStyle.Render(volumeName))
	fmt.Printf("    mount path: %s\n", dimStyle.Render(volumeInfo.MountPath))
	if volumeBackupDescription != "" {
		fmt.Printf("    description: %s\n", dimStyle.Render(volumeBackupDescription))
	}
	fmt.Println()

	fmt.Println(progressStyle.Render("  --> creating backup..."))

	backup, err := backupManager.BackupVolume(ctx, appName, volumeName, volumeBackupDescription, volumeBackupOutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  [done] backup created successfully"))
	fmt.Println()

	fmt.Println(progressStyle.Render("  backup details:"))
	fmt.Printf("    id: %s\n", dimStyle.Render(backup.ID))
	fmt.Printf("    file: %s\n", dimStyle.Render(backup.FilePath))
	fmt.Printf("    size: %s\n", dimStyle.Render(utils.FormatBytes(backup.Size)))
	if backup.Description != "" {
		fmt.Printf("    description: %s\n", dimStyle.Render(backup.Description))
	}
	fmt.Printf("    created: %s\n", dimStyle.Render(backup.CreatedAt.Format("2006-01-02 15:04:05")))
	fmt.Println()
}

func runAppVolumeRestore(cmd *cobra.Command, args []string) {
	appName := args[0]
	volumeName := args[1]
	backupFile := args[2]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> restoring volume: %s/%s", appName, volumeName)))
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize docker client: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	volumeManager := app.NewVolumeManager(dockerClient, registry)
	backupManager := app.NewVolumeBackupManager(dockerClient, volumeManager)

	backupFileInfo, err := os.Stat(backupFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s backup file not found: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	ctx := context.Background()
	volumeInfo, err := volumeManager.InspectVolume(ctx, appName, volumeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render("  restore details:"))
	fmt.Println()
	fmt.Println("  " + dimStyle.Render("backup file:"))
	fmt.Printf("    path: %s\n", dimStyle.Render(backupFile))
	fmt.Printf("    size: %s\n", dimStyle.Render(utils.FormatBytes(backupFileInfo.Size())))
	fmt.Printf("    modified: %s\n", dimStyle.Render(backupFileInfo.ModTime().Format("2006-01-02 15:04:05")))
	fmt.Println()
	fmt.Println("  " + dimStyle.Render("target volume:"))
	fmt.Printf("    app: %s\n", valueStyle.Render(appName))
	fmt.Printf("    volume: %s\n", valueStyle.Render(volumeName))
	fmt.Printf("    mount path: %s\n", dimStyle.Render(volumeInfo.MountPath))
	fmt.Println()

	fmt.Println(errorStyle.Render("  [warn] this will permanently delete all current data in the volume"))
	fmt.Println(errorStyle.Render("  [warn] this action cannot be undone"))
	fmt.Println()
	fmt.Print("  continue? [y/N]: ")

	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println()
		fmt.Println(infoStyle.Render("  [info] cancelled"))
		return
	}
	fmt.Println()

	fmt.Println(progressStyle.Render("  --> restoring from backup..."))

	err = backupManager.RestoreVolume(ctx, appName, volumeName, backupFile, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  [done] volume restored successfully"))
	fmt.Println()

	fmt.Println(infoStyle.Render("  [info] restart app to use restored data"))
	fmt.Println(dimStyle.Render(fmt.Sprintf("  run 'yap app restart %s' to restart", appName)))
}

func runAppVolumeBackups(cmd *cobra.Command, args []string) {
	var appName, volumeName string
	if len(args) > 0 {
		appName = args[0]
	}
	if len(args) > 1 {
		volumeName = args[1]
	}

	if appName != "" && volumeName != "" {
		fmt.Println(titleStyle.Render(fmt.Sprintf("==> backups: %s/%s", appName, volumeName)))
	} else if appName != "" {
		fmt.Println(titleStyle.Render(fmt.Sprintf("==> backups: %s", appName)))
	} else {
		fmt.Println(titleStyle.Render("==> volume backups"))
	}
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize docker client: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	volumeManager := app.NewVolumeManager(dockerClient, registry)
	backupManager := app.NewVolumeBackupManager(dockerClient, volumeManager)

	backups, err := backupManager.ListBackups(appName, volumeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if len(backups) == 0 {
		fmt.Println(dimStyle.Render("  no backups found"))
		return
	}

	for _, backup := range backups {
		fmt.Printf("  %s\n", successStyle.Render(backup.ID))
		fmt.Printf("    app: %s\n", dimStyle.Render(backup.AppName))
		fmt.Printf("    volume: %s\n", dimStyle.Render(backup.VolumeName))
		fmt.Printf("    size: %s\n", dimStyle.Render(utils.FormatBytes(backup.Size)))
		if backup.Description != "" {
			fmt.Printf("    description: %s\n", dimStyle.Render(backup.Description))
		}
		fmt.Printf("    created: %s\n", dimStyle.Render(backup.CreatedAt.Format("2006-01-02 15:04:05")))
		fmt.Printf("    file: %s\n", dimStyle.Render(backup.FilePath))
		fmt.Println()
	}
}

func runAppVolumeBackupDelete(cmd *cobra.Command, args []string) {
	backupID := args[0]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> deleting backup: %s", backupID)))
	fmt.Println()

	fmt.Print("  delete backup file? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println()
		fmt.Println(infoStyle.Render("  [info] cancelled"))
		return
	}
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize docker client: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	volumeManager := app.NewVolumeManager(dockerClient, registry)
	backupManager := app.NewVolumeBackupManager(dockerClient, volumeManager)

	err = backupManager.DeleteBackup(backupID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] backup deleted"))
}

