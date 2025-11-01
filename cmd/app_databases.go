package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/database"
	"github.com/spf13/cobra"
)

var appDatabasesCmd = &cobra.Command{
	Use:   "databases [app-name]",
	Short: "List databases linked to an application",
	Long:  "Show all databases linked to an application with connection details",
	Args:  cobra.ExactArgs(1),
	Run:   runAppDatabases,
}

func init() {
	appCmd.AddCommand(appDatabasesCmd)
}

func runAppDatabases(cmd *cobra.Command, args []string) {
	appName := args[0]

	appRegistry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load app registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	if err := appRegistry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize app registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	application, err := appRegistry.Get(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> databases linked to: %s", appName)))
	fmt.Println()

	if len(application.LinkedDatabases) == 0 {
		fmt.Println(dimStyle.Render("  no databases linked"))
		fmt.Println()
		fmt.Println(dimStyle.Render(fmt.Sprintf("  use 'nap app link %s <db-name>' to link a database", appName)))
		return
	}

	dbRegistry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load database registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	if err := dbRegistry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize database registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	for i, dbName := range application.LinkedDatabases {
		db, err := dbRegistry.Get(dbName)
		if err != nil {
			fmt.Printf("  %d. %s %s\n", i+1, dbName, errorStyle.Render("(not found)"))
			continue
		}

		fmt.Printf("  %d. %s (%s)\n", i+1, valueStyle.Render(db.Name), db.Type)
		fmt.Printf("     vpc: %s\n", dimStyle.Render(db.VPC))
		fmt.Printf("     hostname: %s\n", dimStyle.Render(db.ContainerName))

		if db.Type == "postgres" {
			fmt.Printf("     env vars: %s\n", dimStyle.Render("DATABASE_URL, POSTGRES_*"))
		} else {
			fmt.Printf("     env vars: %s\n", dimStyle.Render("REDIS_URL, VALKEY_*"))
		}

		if i < len(application.LinkedDatabases)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  total: %d database(s)", len(application.LinkedDatabases))))
	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  use 'nap app env list %s' to view all environment variables", appName)))
}
