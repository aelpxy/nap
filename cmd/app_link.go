package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var appLinkCmd = &cobra.Command{
	Use:   "link [app-name] [db-name]",
	Short: "Link a database to an application",
	Long:  "Link a database to an application by injecting connection credentials as environment variables",
	Args:  cobra.ExactArgs(2),
	Run:   runAppLink,
}

func init() {
	appCmd.AddCommand(appLinkCmd)
}

func runAppLink(cmd *cobra.Command, args []string) {
	appName := args[0]
	dbName := args[1]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> linking database: %s → %s", dbName, appName)))
	fmt.Println()

	appRegistry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load app registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	if err := appRegistry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize app registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
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

	application, err := appRegistry.Get(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s application not found: %s\n", errorStyle.Render("[error]"), appName)
		fmt.Println()
		fmt.Println(dimStyle.Render("  try one of these:"))
		fmt.Println(dimStyle.Render("    nap app list              # see all applications"))
		fmt.Println(dimStyle.Render("    nap app deploy myapp .    # deploy new application"))
		os.Exit(1)
	}

	db, err := dbRegistry.Get(dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s database not found: %s\n", errorStyle.Render("[error]"), dbName)
		fmt.Println()
		fmt.Println(dimStyle.Render("  try one of these:"))
		fmt.Println(dimStyle.Render("    nap db list                           # see all databases"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    nap db create postgres %s --vpc %s   # create new database", dbName, application.VPC)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> validating..."))
	fmt.Println(dimStyle.Render(fmt.Sprintf("    checking vpc compatibility (both in '%s')...", application.VPC)))

	if err := app.LinkDatabase(application, db); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to link database: %v\n", errorStyle.Render("[error]"), err)

		if application.VPC != db.VPC {
			fmt.Println()
			fmt.Println(dimStyle.Render("  app and database are in different vpcs:"))
			fmt.Printf("    app '%s' is in vpc: %s\n", valueStyle.Render(appName), errorStyle.Render(application.VPC))
			fmt.Printf("    database '%s' is in vpc: %s\n", valueStyle.Render(dbName), errorStyle.Render(db.VPC))
			fmt.Println()
			fmt.Println(dimStyle.Render("  to fix this, either:"))
			fmt.Println(dimStyle.Render(fmt.Sprintf("    1. redeploy app to same vpc: nap app deploy %s . --vpc %s", appName, db.VPC)))
			fmt.Println(dimStyle.Render(fmt.Sprintf("    2. create new database in app vpc: nap db create %s newdb --vpc %s", db.Type, application.VPC)))
		}
		os.Exit(1)
	}

	db.LinkedApps = append(db.LinkedApps, appName)

	if err := appRegistry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update app registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	if err := dbRegistry.Update(*db); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update database registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

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
			fmt.Fprintf(os.Stderr, "%s failed to recreate container: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}
		newContainerIDs = append(newContainerIDs, newID)
	}

	application.ContainerIDs = newContainerIDs
	if err := appRegistry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update app registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  [done] database linked successfully"))
	fmt.Println()
	fmt.Println(titleStyle.Render("  connection details:"))
	fmt.Printf("    database: %s (%s)\n", valueStyle.Render(dbName), db.Type)
	fmt.Printf("    app: %s\n", valueStyle.Render(appName))
	fmt.Printf("    vpc: %s\n", valueStyle.Render(application.VPC))
	fmt.Println()
	fmt.Println(titleStyle.Render("  injected environment variables:"))

	if db.Type == "postgres" {
		fmt.Println(dimStyle.Render("    DATABASE_URL"))
		fmt.Println(dimStyle.Render("    POSTGRES_HOST"))
		fmt.Println(dimStyle.Render("    POSTGRES_PORT"))
		fmt.Println(dimStyle.Render("    POSTGRES_USER"))
		fmt.Println(dimStyle.Render("    POSTGRES_PASSWORD"))
		fmt.Println(dimStyle.Render("    POSTGRES_DATABASE"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  your application can now connect using:"))
		fmt.Println(dimStyle.Render("    const db = new Client(process.env.DATABASE_URL)"))
	} else {
		fmt.Println(dimStyle.Render("    REDIS_URL"))
		fmt.Println(dimStyle.Render("    VALKEY_URL"))
		fmt.Println(dimStyle.Render("    VALKEY_HOST"))
		fmt.Println(dimStyle.Render("    VALKEY_PORT"))
		fmt.Println(dimStyle.Render("    VALKEY_PASSWORD"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  your application can now connect using:"))
		fmt.Println(dimStyle.Render("    const redis = new Redis(process.env.REDIS_URL)"))
	}
}
