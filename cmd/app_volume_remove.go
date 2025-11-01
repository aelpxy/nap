package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var deleteVolumeData bool

var appVolumeRemoveCmd = &cobra.Command{
	Use:   "remove [app-name] [volume-name]",
	Short: "Remove a volume from an application",
	Long: `Remove a volume from an application.

By default, the volume data is preserved. Use --delete-data to remove the data.

Examples:
  nap app volume remove myapp data
  nap app volume remove myapp uploads --delete-data`,
	Args: cobra.ExactArgs(2),
	Run:  runAppVolumeRemove,
}

func init() {
	appVolumeCmd.AddCommand(appVolumeRemoveCmd)
	appVolumeRemoveCmd.Flags().BoolVar(&deleteVolumeData, "delete-data", false, "Delete the volume data permanently")
}

func runAppVolumeRemove(cmd *cobra.Command, args []string) {
	appName := args[0]
	volumeName := args[1]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> removing volume from application: %s", appName)))
	fmt.Println()

	if deleteVolumeData {
		fmt.Println(errorStyle.Render(fmt.Sprintf("  [warn] this will permanently delete all data in volume '%s'", volumeName)))
		fmt.Print("  type volume name to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to read input: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		response = strings.TrimSpace(response)
		if response != volumeName {
			fmt.Println()
			fmt.Println(infoStyle.Render("  [info] cancelled"))
			return
		}
		fmt.Println()
	}

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

	ctx := context.Background()
	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> removing volume '%s'...", volumeName)))

	if err := volumeManager.RemoveVolume(ctx, appName, volumeName, deleteVolumeData); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	if deleteVolumeData {
		fmt.Println(successStyle.Render(fmt.Sprintf("  [done] volume '%s' removed and data deleted", volumeName)))
	} else {
		fmt.Println(successStyle.Render(fmt.Sprintf("  [done] volume '%s' removed (data preserved)", volumeName)))
	}
	fmt.Println()

	fmt.Println(infoStyle.Render("  [info] redeploy app to unmount the volume"))
	fmt.Println(dimStyle.Render(fmt.Sprintf("  run 'nap app deploy %s' to apply changes", appName)))
}
