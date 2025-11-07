package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/database"
	"github.com/spf13/cobra"
)

var credentialsCmd = &cobra.Command{
	Use:   "credentials [name]",
	Short: "Show database credentials",
	Long:  "Display connection credentials for a database",
	Args:  cobra.ExactArgs(1),
	Run:   runCredentials,
}

func runCredentials(cmd *cobra.Command, args []string) {
	dbName := args[0]

	registry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	db, err := registry.Get(dbName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] database not found: %v", err)))
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> credentials: %s", dbName)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  database information:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("type:"), valueStyle.Render(string(db.Type)))
	fmt.Printf("    %s %s\n", dimStyle.Render("vpc:"), valueStyle.Render(db.VPC))
	if db.Published {
		fmt.Printf("    %s %s\n", dimStyle.Render("status:"), successStyle.Render(fmt.Sprintf("published (port %d)", db.PublishedPort)))
	} else {
		fmt.Printf("    %s %s\n", dimStyle.Render("status:"), dimStyle.Render("private (not published)"))
	}
	fmt.Println()

	fmt.Println(labelStyle.Render("  internal connection (within vpc):"))
	fmt.Printf("    %s %s\n", dimStyle.Render("hostname:"), valueStyle.Render(db.InternalHostname))
	fmt.Printf("    %s %s\n", dimStyle.Render("port:"), valueStyle.Render(fmt.Sprintf("%d", db.InternalPort)))
	fmt.Printf("    %s %s\n", dimStyle.Render("username:"), valueStyle.Render(db.Username))
	fmt.Printf("    %s %s\n", dimStyle.Render("password:"), valueStyle.Render(db.Password))
	if db.DatabaseName != "" {
		fmt.Printf("    %s %s\n", dimStyle.Render("database:"), valueStyle.Render(db.DatabaseName))
	}
	fmt.Println()
	fmt.Printf("    %s\n", dimStyle.Render("connection string:"))
	fmt.Printf("    %s\n", dimStyle.Render(db.ConnectionString))
	fmt.Println()

	if db.Published {
		fmt.Println(labelStyle.Render("  external connection (host machine):"))
		fmt.Printf("    %s %s\n", dimStyle.Render("host:"), valueStyle.Render("localhost"))
		fmt.Printf("    %s %s\n", dimStyle.Render("port:"), valueStyle.Render(fmt.Sprintf("%d", db.PublishedPort)))
		fmt.Println()
		fmt.Printf("    %s\n", dimStyle.Render("connection string:"))
		fmt.Printf("    %s\n", dimStyle.Render(db.PublishedConnectionString))
		fmt.Println()

		if db.Type == "postgres" {
			fmt.Println(dimStyle.Render("  example usage:"))
			fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("psql \"%s\"", db.PublishedConnectionString)))
		} else {
			fmt.Println(dimStyle.Render("  example usage:"))
			fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("valkey-cli -h localhost -p %d -a %s", db.PublishedPort, db.Password)))
		}
		fmt.Println()
	} else {
		fmt.Println(dimStyle.Render("  [info] this database is not published to the host"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("  [info] run 'yap db publish %s --port <port>' to access from your machine", dbName)))
		fmt.Println()
	}
}

func init() {
	dbCmd.AddCommand(credentialsCmd)
}
