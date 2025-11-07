package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/spf13/cobra"
)

var (
	volumeType     string
	volumeSource   string
	volumeReadOnly bool
	volumeSize     string
)

var appVolumeAddCmd = &cobra.Command{
	Use:   "add [app-name] [volume-name] [mount-path]",
	Short: "Add a volume to an application",
	Long: `Add a persistent volume to an application.

Examples:
  yap app volume add myapp data /app/data
  yap app volume add myapp uploads /app/uploads --readonly
  yap app volume add myapp config /etc/app --type bind --source /host/config`,
	Args: cobra.ExactArgs(3),
	Run:  runAppVolumeAdd,
}

func init() {
	appVolumeCmd.AddCommand(appVolumeAddCmd)
	appVolumeAddCmd.Flags().StringVar(&volumeType, "type", "volume", "Volume type: 'volume' or 'bind'")
	appVolumeAddCmd.Flags().StringVar(&volumeSource, "source", "", "Source path (required for bind mounts)")
	appVolumeAddCmd.Flags().BoolVar(&volumeReadOnly, "readonly", false, "Mount volume as read-only")
	appVolumeAddCmd.Flags().StringVar(&volumeSize, "size", "", "Volume size limit (e.g. '10GB')")
}

func runAppVolumeAdd(cmd *cobra.Command, args []string) {
	appName := args[0]
	volumeName := args[1]
	mountPath := args[2]

	if volumeType != "volume" && volumeType != "bind" {
		fmt.Fprintf(os.Stderr, "%s volume type must be 'volume' or 'bind'\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}

	if volumeType == "bind" && volumeSource == "" {
		fmt.Fprintf(os.Stderr, "%s bind mounts require --source flag\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> adding volume to application: %s", appName)))
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

	vol := models.Volume{
		Name:      volumeName,
		MountPath: mountPath,
		Type:      volumeType,
		Source:    volumeSource,
		ReadOnly:  volumeReadOnly,
		Size:      volumeSize,
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> adding volume '%s'...", volumeName)))
	fmt.Printf("    type: %s\n", dimStyle.Render(volumeType))
	fmt.Printf("    mount path: %s\n", dimStyle.Render(mountPath))
	if volumeType == "bind" {
		fmt.Printf("    source: %s\n", dimStyle.Render(volumeSource))
	}
	if volumeReadOnly {
		fmt.Printf("    read-only: %s\n", dimStyle.Render("true"))
	}
	if volumeSize != "" {
		fmt.Printf("    size: %s\n", dimStyle.Render(volumeSize))
	}

	if err := volumeManager.AddVolume(ctx, appName, vol); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] volume '%s' added", volumeName)))
	fmt.Println()

	fmt.Println(infoStyle.Render("  [info] redeploy app to mount the volume"))
	fmt.Println(dimStyle.Render(fmt.Sprintf("  run 'yap app deploy %s' to apply changes", appName)))
}
