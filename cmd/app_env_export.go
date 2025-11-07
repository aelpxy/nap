package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aelpxy/yap/internal/app"
	"github.com/spf13/cobra"
)

var appEnvExportCmd = &cobra.Command{
	Use:   "export [app-name]",
	Short: "Export environment variables",
	Long:  "Export environment variables as shell export statements",
	Args:  cobra.ExactArgs(1),
	Run:   runAppEnvExport,
}

func init() {
	appEnvCmd.AddCommand(appEnvExportCmd)
}

func runAppEnvExport(cmd *cobra.Command, args []string) {
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

	if len(application.EnvVars) == 0 {
		fmt.Fprintf(os.Stderr, "%s no environment variables set\n", infoStyle.Render("[info]"))
		return
	}

	keys := make([]string, 0, len(application.EnvVars))
	for k := range application.EnvVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := application.EnvVars[key]
		escapedValue := strings.ReplaceAll(value, "'", "'\\''")
		fmt.Printf("export %s='%s'\n", key, escapedValue)
	}
}
