package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/internal/runtime"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and dependencies",
	Long:  "Verify that all required dependencies are installed and running correctly",
	Run:   runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) {
	fmt.Println(titleStyle.Render("==> checking system health"))
	fmt.Println()

	allGood := true

	allGood = checkRuntime() && allGood
	allGood = checkDirectories() && allGood
	allGood = checkRegistries() && allGood
	allGood = checkProxy() && allGood
	allGood = checkVPCs() && allGood
	allGood = checkGlobalConfig() && allGood

	fmt.Println()
	if allGood {
		fmt.Println(successStyle.Render("  [done] all checks passed"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  your yap installation is healthy and ready to use"))
	} else {
		fmt.Println(errorStyle.Render("  [error] some checks failed"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  fix the issues above before deploying applications"))
		os.Exit(1)
	}
}

func checkRuntime() bool {
	fmt.Println(labelStyle.Render("  runtime"))

	info, err := runtime.DetectRuntime()

	if err != nil {
		fmt.Printf("    %s runtime not detected\n", errorStyle.Render("[✗]"))
		fmt.Printf("      %s\n", dimStyle.Render(err.Error()))
		fmt.Printf("      %s\n", dimStyle.Render("install docker or podman to continue"))
		return false
	}

	fmt.Printf("    %s %s detected\n", successStyle.Render("[✓]"), valueStyle.Render(string(info.Type)))
	fmt.Printf("      %s %s\n", dimStyle.Render("version:"), dimStyle.Render(info.Version))
	fmt.Printf("      %s %s\n", dimStyle.Render("socket:"), dimStyle.Render(info.SocketPath))

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Printf("    %s runtime daemon not responding\n", errorStyle.Render("[✗]"))
		fmt.Printf("      %s\n", dimStyle.Render(err.Error()))
		return false
	}
	defer dockerClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = dockerClient.GetClient().Ping(ctx)
	if err != nil {
		fmt.Printf("    %s runtime daemon not responding\n", errorStyle.Render("[✗]"))
		fmt.Printf("      %s\n", dimStyle.Render(err.Error()))
		return false
	}

	fmt.Printf("    %s daemon running\n", successStyle.Render("[✓]"))
	fmt.Println()

	return true
}

func checkDirectories() bool {
	fmt.Println(labelStyle.Render("  yap directories"))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("    %s cannot determine home directory\n", errorStyle.Render("[✗]"))
		return false
	}

	yapDir := filepath.Join(homeDir, ".yap")

	if _, err := os.Stat(yapDir); os.IsNotExist(err) {
		fmt.Printf("    %s ~/.yap directory missing\n", errorStyle.Render("[✗]"))
		fmt.Printf("      %s\n", dimStyle.Render("run 'yap init' or deploy an app to initialize"))
		return false
	}

	fmt.Printf("    %s %s exists\n", successStyle.Render("[✓]"), dimStyle.Render("~/.yap"))

	info, err := os.Stat(yapDir)
	if err != nil {
		fmt.Printf("    %s cannot access ~/.yap\n", errorStyle.Render("[✗]"))
		return false
	}

	if info.Mode().Perm()&0700 != 0700 {
		fmt.Printf("    %s incorrect permissions on ~/.yap\n", errorStyle.Render("[!]"))
		fmt.Printf("      %s\n", dimStyle.Render("run: chmod 700 ~/.yap"))
	} else {
		fmt.Printf("    %s permissions correct\n", successStyle.Render("[✓]"))
	}

	fmt.Println()
	return true
}

func checkRegistries() bool {
	fmt.Println(labelStyle.Render("  registries"))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	yapDir := filepath.Join(homeDir, ".yap")
	allGood := true

	appsFile := filepath.Join(yapDir, "apps.json")
	if _, err := os.Stat(appsFile); os.IsNotExist(err) {
		fmt.Printf("    %s %s missing\n", errorStyle.Render("[!]"), dimStyle.Render("apps.json"))
		fmt.Printf("      %s\n", dimStyle.Render("will be created on first app deployment"))
	} else {
		appRegistry, err := app.NewRegistryManager()
		if err != nil {
			fmt.Printf("    %s %s corrupted\n", errorStyle.Render("[✗]"), dimStyle.Render("apps.json"))
			allGood = false
		} else {
			apps, err := appRegistry.List()
			if err != nil {
				fmt.Printf("    %s %s corrupted\n", errorStyle.Render("[✗]"), dimStyle.Render("apps.json"))
				allGood = false
			} else {
				fmt.Printf("    %s %s (%d apps)\n", successStyle.Render("[✓]"), dimStyle.Render("apps.json"), len(apps))
			}
		}
	}

	dbsFile := filepath.Join(yapDir, "databases.json")
	if _, err := os.Stat(dbsFile); os.IsNotExist(err) {
		fmt.Printf("    %s %s missing\n", errorStyle.Render("[!]"), dimStyle.Render("databases.json"))
		fmt.Printf("      %s\n", dimStyle.Render("will be created on first database creation"))
	} else {
		dbRegistry, err := database.NewRegistryManager()
		if err != nil {
			fmt.Printf("    %s %s corrupted\n", errorStyle.Render("[✗]"), dimStyle.Render("databases.json"))
			allGood = false
		} else {
			dbs, err := dbRegistry.List()
			if err != nil {
				fmt.Printf("    %s %s corrupted\n", errorStyle.Render("[✗]"), dimStyle.Render("databases.json"))
				allGood = false
			} else {
				fmt.Printf("    %s %s (%d databases)\n", successStyle.Render("[✓]"), dimStyle.Render("databases.json"), len(dbs))
			}
		}
	}

	vpcsFile := filepath.Join(yapDir, "vpcs.json")
	if _, err := os.Stat(vpcsFile); os.IsNotExist(err) {
		fmt.Printf("    %s %s missing\n", errorStyle.Render("[!]"), dimStyle.Render("vpcs.json"))
		fmt.Printf("      %s\n", dimStyle.Render("will be created on first vpc creation"))
	} else {
		vpcRegistry, err := database.NewVPCRegistryManager()
		if err != nil {
			fmt.Printf("    %s %s corrupted\n", errorStyle.Render("[✗]"), dimStyle.Render("vpcs.json"))
			allGood = false
		} else {
			vpcs, err := vpcRegistry.List()
			if err != nil {
				fmt.Printf("    %s %s corrupted\n", errorStyle.Render("[✗]"), dimStyle.Render("vpcs.json"))
				allGood = false
			} else {
				fmt.Printf("    %s %s (%d vpcs)\n", successStyle.Render("[✓]"), dimStyle.Render("vpcs.json"), len(vpcs))
			}
		}
	}

	fmt.Println()
	return allGood
}

func checkProxy() bool {
	fmt.Println(labelStyle.Render("  http proxy"))

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Printf("    %s cannot check proxy status\n", errorStyle.Render("[✗]"))
		return false
	}
	defer dockerClient.Close()

	ctx := context.Background()

	listOpts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("name", "yap-traefik"),
		),
	}

	containers, err := dockerClient.GetClient().ContainerList(ctx, listOpts)
	if err != nil {
		fmt.Printf("    %s cannot list containers\n", errorStyle.Render("[✗]"))
		return false
	}

	var traefik *types.Container
	for _, c := range containers {
		traefik = &c
		break
	}

	if traefik == nil {
		fmt.Printf("    %s proxy not running\n", errorStyle.Render("[✗]"))
		fmt.Printf("      %s\n", dimStyle.Render("proxy will start automatically on first app deployment"))
		fmt.Println()
		return true
	}

	if traefik.State == "running" {
		fmt.Printf("    %s proxy running\n", successStyle.Render("[✓]"))
		fmt.Printf("      %s %s\n", dimStyle.Render("container:"), dimStyle.Render(traefik.ID[:12]))
		fmt.Printf("      %s %s\n", dimStyle.Render("image:"), dimStyle.Render(traefik.Image))
	} else {
		fmt.Printf("    %s proxy not running (state: %s)\n", errorStyle.Render("[✗]"), traefik.State)
		fmt.Printf("      %s\n", dimStyle.Render("run: docker start yap-traefik"))
		fmt.Println()
		return false
	}

	fmt.Println()
	return true
}

func checkVPCs() bool {
	fmt.Println(labelStyle.Render("  vpcs"))

	vpcRegistry, err := database.NewVPCRegistryManager()
	if err != nil {
		fmt.Printf("    %s cannot load vpc registry\n", errorStyle.Render("[✗]"))
		return false
	}

	vpcs, err := vpcRegistry.List()
	if err != nil {
		fmt.Printf("    %s cannot read vpc registry\n", errorStyle.Render("[✗]"))
		return false
	}

	if len(vpcs) == 0 {
		fmt.Printf("    %s no vpcs configured\n", errorStyle.Render("[!]"))
		fmt.Printf("      %s\n", dimStyle.Render("create one with: yap vpc create primary"))
		fmt.Println()
		return true
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Printf("    %s cannot check vpc networks\n", errorStyle.Render("[✗]"))
		return false
	}
	defer dockerClient.Close()

	ctx := context.Background()
	networks, err := dockerClient.GetClient().NetworkList(ctx, network.ListOptions{})
	if err != nil {
		fmt.Printf("    %s cannot list networks\n", errorStyle.Render("[✗]"))
		return false
	}

	allGood := true
	for _, vpc := range vpcs {
		found := false
		for _, network := range networks {
			if network.ID == vpc.NetworkID {
				found = true
				break
			}
		}

		if found {
			fmt.Printf("    %s vpc %s\n", successStyle.Render("[✓]"), valueStyle.Render(vpc.Name))
		} else {
			fmt.Printf("    %s vpc %s (network missing)\n", errorStyle.Render("[✗]"), valueStyle.Render(vpc.Name))
			fmt.Printf("      %s\n", dimStyle.Render("delete and recreate vpc"))
			allGood = false
		}
	}

	fmt.Println()
	return allGood
}

func checkGlobalConfig() bool {
	fmt.Println(labelStyle.Render("  global configuration"))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configFile := filepath.Join(homeDir, ".yap", "config.toml")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Printf("    %s %s missing\n", errorStyle.Render("[!]"), dimStyle.Render("config.toml"))
		fmt.Printf("      %s\n", dimStyle.Render("run 'yap config setup' to configure publishing"))
		fmt.Println()
		return true
	}

	fmt.Printf("    %s %s exists\n", successStyle.Render("[✓]"), dimStyle.Render("config.toml"))
	fmt.Println()

	return true
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
