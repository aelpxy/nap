package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var appEnvSetCmd = &cobra.Command{
	Use:   "set [app-name] KEY=value [KEY2=value2...]",
	Short: "Set environment variables",
	Long:  "Set one or more environment variables for an application",
	Args:  cobra.MinimumNArgs(2),
	Run:   runAppEnvSet,
}

func init() {
	appEnvCmd.AddCommand(appEnvSetCmd)
}

func runAppEnvSet(cmd *cobra.Command, args []string) {
	appName := args[0]
	envPairs := args[1:]

	envVars := make(map[string]string)
	for _, pair := range envPairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "%s invalid format: %s (expected KEY=value)\n", errorStyle.Render("[error]"), pair)
			os.Exit(1)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			fmt.Fprintf(os.Stderr, "%s empty key in: %s\n", errorStyle.Render("[error]"), pair)
			os.Exit(1)
		}

		envVars[key] = value
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> setting environment variables: %s", appName)))
	fmt.Println()

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
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if application.EnvVars == nil {
		application.EnvVars = make(map[string]string)
	}

	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> setting %d variable(s)...", len(envVars))))
	for key, value := range envVars {
		maskedValue := maskSensitiveValue(key, value)
		fmt.Printf("    %s = %s\n", dimStyle.Render(key), dimStyle.Render(maskedValue))
		application.EnvVars[key] = value
	}

	if err := registry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] set %d environment variable(s)", len(envVars))))
	fmt.Println()

	fmt.Println(infoStyle.Render("  [info] restart app to apply changes?"))
	fmt.Print("  [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to read input: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "" || response == "y" || response == "yes" {
		fmt.Println()
		fmt.Println(progressStyle.Render("  --> restarting application..."))

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

		newContainerIDs := make([]string, 0, len(application.ContainerIDs))
		for i, containerID := range application.ContainerIDs {
			instanceNum := i + 1

			newID, err := app.RecreateContainer(
				dockerClient,
				containerID,
				application,
				vpc.NetworkName,
				instanceNum,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s failed to recreate instance: %v\n", errorStyle.Render("[error]"), err)
				os.Exit(1)
			}
			newContainerIDs = append(newContainerIDs, newID)
		}

		application.ContainerIDs = newContainerIDs
		if err := registry.Update(*application); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		fmt.Println(successStyle.Render("  [done] application restarted"))
	} else {
		fmt.Println()
		fmt.Println(dimStyle.Render(fmt.Sprintf("  run 'nap app restart %s' to apply changes", appName)))
	}
}
