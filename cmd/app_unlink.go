package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/spf13/cobra"
)

var appUnlinkCmd = &cobra.Command{
	Use:   "unlink [app-name] [db-name]",
	Short: "Unlink a database from an application",
	Long:  "Unlink a database from an application by removing connection credentials",
	Args:  cobra.ExactArgs(2),
	Run:   runAppUnlink,
}

func init() {
	appCmd.AddCommand(appUnlinkCmd)
}

func runAppUnlink(cmd *cobra.Command, args []string) {
	appName := args[0]
	dbName := args[1]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> unlinking database: %s → %s", dbName, appName)))
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
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	db, err := dbRegistry.Get(dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s database not found: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(errorStyle.Render("  [warn] this will remove database credentials from the application"))
	fmt.Println()
	fmt.Print("  continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to read input: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		fmt.Println()
		fmt.Println(dimStyle.Render("  unlink cancelled"))
		return
	}

	fmt.Println()
	fmt.Println(progressStyle.Render("  --> removing environment variables..."))

	if err := app.UnlinkDatabase(application, db); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to unlink database: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	for i, linkedApp := range db.LinkedApps {
		if linkedApp == appName {
			db.LinkedApps = append(db.LinkedApps[:i], db.LinkedApps[i+1:]...)
			break
		}
	}

	fmt.Println(progressStyle.Render("  --> updating registries..."))
	if err := appRegistry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update app registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}
	if err := dbRegistry.Update(*db); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update database registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

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
	fmt.Println(successStyle.Render("  [done] database unlinked successfully"))
}
