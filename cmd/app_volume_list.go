package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/spf13/cobra"
)

var appVolumeListCmd = &cobra.Command{
	Use:   "list [app-name]",
	Short: "List volumes for an application",
	Long:  "List all persistent volumes attached to an application",
	Args:  cobra.ExactArgs(1),
	Run:   runAppVolumeList,
}

func init() {
	appVolumeCmd.AddCommand(appVolumeListCmd)
}

func runAppVolumeList(cmd *cobra.Command, args []string) {
	appName := args[0]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> volumes: %s", appName)))
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
	volumes, err := volumeManager.ListVolumes(ctx, appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if len(volumes) == 0 {
		fmt.Println(dimStyle.Render("  no volumes configured"))
		fmt.Println()
		fmt.Println(infoStyle.Render("  [info] the container filesystem is ephemeral. use volumes for persistent data."))
		fmt.Println(dimStyle.Render(fmt.Sprintf("  run 'yap app volume add %s <name> <path>' to add a volume", appName)))
		return
	}

	for _, vol := range volumes {
		fmt.Printf("  %s\n", successStyle.Render(vol.Name))
		fmt.Printf("    mount path: %s\n", dimStyle.Render(vol.MountPath))
		fmt.Printf("    type: %s\n", dimStyle.Render(vol.Type))
		if vol.Type == "bind" {
			fmt.Printf("    source: %s\n", dimStyle.Render(vol.Source))
		}
		if vol.ReadOnly {
			fmt.Printf("    read-only: %s\n", dimStyle.Render("true"))
		}
		if vol.Size != "" {
			fmt.Printf("    size: %s\n", dimStyle.Render(vol.Size))
		}
		fmt.Printf("    created: %s\n", dimStyle.Render(vol.CreatedAt.Format("2006-01-02 15:04:05")))
		fmt.Println()
	}
}
