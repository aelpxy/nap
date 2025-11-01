package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/app"
	"github.com/spf13/cobra"
)

var appEnvUnsetCmd = &cobra.Command{
	Use:   "unset [app-name] KEY [KEY2...]",
	Short: "Unset environment variables",
	Long:  "Remove one or more environment variables from an application",
	Args:  cobra.MinimumNArgs(2),
	Run:   runAppEnvUnset,
}

func init() {
	appEnvCmd.AddCommand(appEnvUnsetCmd)
}

func runAppEnvUnset(cmd *cobra.Command, args []string) {
	appName := args[0]
	keys := args[1:]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> unsetting environment variables: %s", appName)))
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

	if len(application.EnvVars) == 0 {
		fmt.Println(dimStyle.Render("  no environment variables to unset"))
		return
	}

	removedCount := 0
	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> removing %d variable(s)...", len(keys))))
	for _, key := range keys {
		if _, exists := application.EnvVars[key]; exists {
			delete(application.EnvVars, key)
			fmt.Printf("    %s\n", dimStyle.Render(key))
			removedCount++
		} else {
			fmt.Printf("    %s (not found, skipping)\n", dimStyle.Render(key))
		}
	}

	if removedCount == 0 {
		fmt.Println()
		fmt.Println(infoStyle.Render("  [info] no variables were removed"))
		return
	}

	if err := registry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] removed %d environment variable(s)", removedCount)))
	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  run 'nap app restart %s' to apply changes", appName)))
}
