package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/database"
	"github.com/spf13/cobra"
)

var dbAppsCmd = &cobra.Command{
	Use:   "apps [db-name]",
	Short: "List applications using a database",
	Long:  "Show all applications that are linked to a database",
	Args:  cobra.ExactArgs(1),
	Run:   runDbApps,
}

func init() {
	dbCmd.AddCommand(dbAppsCmd)
}

func runDbApps(cmd *cobra.Command, args []string) {
	dbName := args[0]

	dbRegistry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load database registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	if err := dbRegistry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize database registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	db, err := dbRegistry.Get(dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> applications using: %s", dbName)))
	fmt.Println()

	if len(db.LinkedApps) == 0 {
		fmt.Println(dimStyle.Render("  no applications linked"))
		fmt.Println()
		fmt.Println(dimStyle.Render(fmt.Sprintf("  use 'yap app link <app-name> %s' to link an application", dbName)))
		return
	}

	for i, appName := range db.LinkedApps {
		fmt.Printf("  %d. %s\n", i+1, valueStyle.Render(appName))
	}

	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  total: %d application(s)", len(db.LinkedApps))))
}
