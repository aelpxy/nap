package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/utils"
	"github.com/spf13/cobra"
)

var appEnvListCmd = &cobra.Command{
	Use:   "list [app-name]",
	Short: "List environment variables",
	Long:  "Display all environment variables for an application",
	Args:  cobra.ExactArgs(1),
	Run:   runAppEnvList,
}

func init() {
	appEnvCmd.AddCommand(appEnvListCmd)
}

func runAppEnvList(cmd *cobra.Command, args []string) {
	appName := args[0]

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

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> environment variables: %s", appName)))
	fmt.Println()

	if len(application.EnvVars) == 0 {
		fmt.Println(dimStyle.Render("  no environment variables set"))
		fmt.Println()
		fmt.Println(dimStyle.Render(fmt.Sprintf("  use 'nap app env set %s KEY=value' to add variables", appName)))
		return
	}

	keys := make([]string, 0, len(application.EnvVars))
	for k := range application.EnvVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxKeyLen := 0
	for _, key := range keys {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}

	for _, key := range keys {
		value := application.EnvVars[key]
		maskedValue := utils.MaskSensitiveEnvValue(key, value)
		paddedKey := key + strings.Repeat(" ", maxKeyLen-len(key))
		fmt.Printf("  %s  %s\n", dimStyle.Render(paddedKey), maskedValue)
	}

	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  total: %d variables", len(application.EnvVars))))
	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  use 'nap app env set %s KEY=value' to add variables", appName)))
	fmt.Println(dimStyle.Render(fmt.Sprintf("  use 'nap app env unset %s KEY' to remove variables", appName)))
}
