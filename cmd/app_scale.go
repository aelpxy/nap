package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var (
	scaleInstances int
	scaleAdd       int
	scaleRemove    int
)

var appScaleCmd = &cobra.Command{
	Use:   "scale [app-name]",
	Short: "Scale an application",
	Long:  "Scale an application to a specific number of instances or add/remove instances",
	Args:  cobra.ExactArgs(1),
	Run:   runAppScale,
}

func init() {
	appCmd.AddCommand(appScaleCmd)
	appScaleCmd.Flags().IntVar(&scaleInstances, "instances", 0, "Scale to N instances")
	appScaleCmd.Flags().IntVar(&scaleAdd, "add", 0, "Add N instances")
	appScaleCmd.Flags().IntVar(&scaleRemove, "remove", 0, "Remove N instances")
}

func runAppScale(cmd *cobra.Command, args []string) {
	appName := args[0]

	flagsSet := 0
	instancesSet := cmd.Flags().Changed("instances")
	addSet := cmd.Flags().Changed("add")
	removeSet := cmd.Flags().Changed("remove")

	if instancesSet {
		flagsSet++
	}
	if addSet {
		flagsSet++
	}
	if removeSet {
		flagsSet++
	}

	if flagsSet == 0 {
		fmt.Fprintf(os.Stderr, "%s must specify --instances, --add, or --remove\n", errorStyle.Render("[error]"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  usage examples:"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    nap app scale %s --instances 3   # scale to exactly 3 instances", appName)))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    nap app scale %s --add 2         # add 2 more instances", appName)))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    nap app scale %s --remove 1      # remove 1 instance", appName)))
		os.Exit(1)
	}
	if flagsSet > 1 {
		fmt.Fprintf(os.Stderr, "%s can only use one of --instances, --add, or --remove\n", errorStyle.Render("[error]"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  use only one scaling option at a time"))
		os.Exit(1)
	}

	if instancesSet && scaleInstances < 1 {
		fmt.Fprintf(os.Stderr, "%s invalid instance count: minimum 1 required\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}
	if addSet && scaleAdd < 1 {
		fmt.Fprintf(os.Stderr, "%s invalid add count: must be positive\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}
	if removeSet && scaleRemove < 1 {
		fmt.Fprintf(os.Stderr, "%s invalid remove count: must be positive\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}

	lockManager := app.GetGlobalLockManager()
	if err := lockManager.TryLock(appName, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "%s another operation in progress for %s\n", errorStyle.Render("[error]"), appName)
		fmt.Println(dimStyle.Render("  wait for the current operation to complete or try again in a few seconds"))
		os.Exit(1)
	}
	defer lockManager.Unlock(appName)

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	application, err := registry.Get(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s application not found: %s\n", errorStyle.Render("[error]"), appName)
		fmt.Println()
		fmt.Println(dimStyle.Render("  check available apps:"))
		fmt.Println(dimStyle.Render("    nap app list"))
		os.Exit(1)
	}

	currentInstances := len(application.ContainerIDs)
	targetInstances := currentInstances

	if scaleInstances > 0 {
		targetInstances = scaleInstances
	} else if scaleAdd > 0 {
		targetInstances = currentInstances + scaleAdd
	} else if scaleRemove > 0 {
		targetInstances = currentInstances - scaleRemove
	}

	if targetInstances < 1 {
		fmt.Fprintf(os.Stderr, "%s cannot scale to less than 1 instance\n", errorStyle.Render("[error]"))
		fmt.Println()
		fmt.Printf("  current instances: %s\n", valueStyle.Render(fmt.Sprintf("%d", currentInstances)))
		fmt.Printf("  requested change would result in: %s\n", errorStyle.Render(fmt.Sprintf("%d instances", targetInstances)))
		fmt.Println()
		fmt.Println(dimStyle.Render("  minimum is 1 instance. to remove the app entirely, use:"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    nap app destroy %s", appName)))
		os.Exit(1)
	}

	if targetInstances == currentInstances {
		fmt.Println(infoStyle.Render(fmt.Sprintf("  [info] already at %d instances", currentInstances)))
		return
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> scaling application: %s", appName)))
	fmt.Println()
	fmt.Printf("  current: %s\n", valueStyle.Render(fmt.Sprintf("%d instances", currentInstances)))
	fmt.Printf("  target: %s\n", valueStyle.Render(fmt.Sprintf("%d instances", targetInstances)))
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	vpcRegistry, err := database.NewVPCRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load vpc registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	if err := vpcRegistry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize vpc registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	vpc, err := vpcRegistry.Get(application.VPC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to get vpc: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if targetInstances > currentInstances {
		count := targetInstances - currentInstances
		fmt.Println(progressStyle.Render(fmt.Sprintf("  --> adding %d instance(s)...", count)))

		newIDs, err := app.ScaleUp(
			dockerClient,
			application,
			count,
			vpc.NetworkName,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to scale up: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		application.ContainerIDs = append(application.ContainerIDs, newIDs...)
	} else {
		count := currentInstances - targetInstances
		fmt.Println(progressStyle.Render(fmt.Sprintf("  --> removing %d instance(s)...", count)))

		if err := app.ScaleDown(dockerClient, application, count); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to scale down: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		application.ContainerIDs = application.ContainerIDs[:targetInstances]
	}

	application.Instances = targetInstances

	if err := registry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> updating load balancer..."))
	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] scaled to %d instances", targetInstances)))
	fmt.Println()
	fmt.Println(dimStyle.Render("  traffic is load balanced across all instances"))
}
