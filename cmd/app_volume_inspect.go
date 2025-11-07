package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/spf13/cobra"
)

var appVolumeInspectCmd = &cobra.Command{
	Use:   "inspect [app-name] [volume-name]",
	Short: "Inspect a volume",
	Long:  "Show detailed information about a volume including Docker details",
	Args:  cobra.ExactArgs(2),
	Run:   runAppVolumeInspect,
}

func init() {
	appVolumeCmd.AddCommand(appVolumeInspectCmd)
}

func runAppVolumeInspect(cmd *cobra.Command, args []string) {
	appName := args[0]
	volumeName := args[1]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> volume details: %s/%s", appName, volumeName)))
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

	ctx := context.Background()
	volumeInfo, err := volumeManager.InspectVolume(ctx, appName, volumeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  configuration:"))
	fmt.Printf("    name: %s\n", dimStyle.Render(volumeInfo.Volume.Name))
	fmt.Printf("    mount path: %s\n", dimStyle.Render(volumeInfo.Volume.MountPath))
	fmt.Printf("    type: %s\n", dimStyle.Render(volumeInfo.Volume.Type))
	if volumeInfo.Volume.Type == "bind" {
		fmt.Printf("    source: %s\n", dimStyle.Render(volumeInfo.Volume.Source))
	}
	if volumeInfo.Volume.ReadOnly {
		fmt.Printf("    read-only: %s\n", dimStyle.Render("true"))
	}
	if volumeInfo.Volume.Size != "" {
		fmt.Printf("    size limit: %s\n", dimStyle.Render(volumeInfo.Volume.Size))
	}
	fmt.Printf("    created: %s\n", dimStyle.Render(volumeInfo.Volume.CreatedAt.Format("2006-01-02 15:04:05")))
	fmt.Println()

	if volumeInfo.Volume.Type == "volume" && volumeInfo.DockerName != "" {
		fmt.Println(progressStyle.Render("  docker details:"))
		fmt.Printf("    volume name: %s\n", dimStyle.Render(volumeInfo.DockerName))
		if volumeInfo.MountPoint != "" {
			fmt.Printf("    mountpoint: %s\n", dimStyle.Render(volumeInfo.MountPoint))
		}
		if volumeInfo.Driver != "" {
			fmt.Printf("    driver: %s\n", dimStyle.Render(volumeInfo.Driver))
		}
		fmt.Println()
	}

	fmt.Println(progressStyle.Render("  usage:"))
	fmt.Printf("    containers: %s\n", dimStyle.Render(fmt.Sprintf("%d", volumeInfo.UsedBy)))
	fmt.Println()
}
