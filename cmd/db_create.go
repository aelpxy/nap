package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/constants"
	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/internal/utils"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/spf13/cobra"
)

var (
	createPassword string
	createVPC      string
)

var createCmd = &cobra.Command{
	Use:   "create [postgres|valkey] [name]",
	Short: "Create a new database",
	Long:  "Provision a new PostgreSQL or Valkey database",
	Args:  cobra.ExactArgs(2),
	Run:   runCreate,
}

func runCreate(cmd *cobra.Command, args []string) {
	dbType := args[0]
	dbName := args[1]

	if dbType != "postgres" && dbType != "valkey" {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[error] database type must be 'postgres' or 'valkey'"))
		os.Exit(1)
	}

	if dbName == "" || len(dbName) == 0 {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[error] database name is required"))
		os.Exit(1)
	}
	if len(dbName) > constants.MaxNameLength {
		fmt.Fprintf(os.Stderr, "%s database name: maximum %d characters\n", errorStyle.Render("[error]"), constants.MaxNameLength)
		os.Exit(1)
	}
	if !utils.IsValidName(dbName) {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[error] database name: use only lowercase letters, numbers, and dashes"))
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> creating %s database: %s", dbType, dbName)))
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to initialize database service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	registry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> provisioning %s database", dbType)))
	fmt.Println()

	var db interface{}
	var provisionErr error

	if createVPC == "" {
		createVPC = "primary"
	}

	if dbType == "postgres" {
		fmt.Println(dimStyle.Render("      pulling postgresql image..."))
		provisioner := database.NewPostgresProvisioner(dockerClient, registry)
		db, provisionErr = provisioner.Provision(dbName, createPassword, createVPC)
	} else {
		fmt.Println(dimStyle.Render("      pulling valkey image..."))
		provisioner := database.NewValkeyProvisioner(dockerClient, registry)
		db, provisionErr = provisioner.Provision(dbName, createPassword, createVPC)
	}

	if provisionErr != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to provision database: %v", provisionErr)))
		os.Exit(1)
	}

	fmt.Println(dimStyle.Render("      creating volume..."))
	fmt.Println(dimStyle.Render("      configuring database..."))
	fmt.Println(dimStyle.Render("      starting database..."))
	fmt.Println()

	fmt.Println(successStyle.Render("  [done] database created successfully"))
	fmt.Println()

	dbModel := db.(*models.Database)

	fmt.Println(labelStyle.Render("  database information:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("name:"), valueStyle.Render(dbModel.Name))
	fmt.Printf("    %s %s\n", dimStyle.Render("type:"), valueStyle.Render(string(dbModel.Type)))
	fmt.Printf("    %s %s\n", dimStyle.Render("id:"), valueStyle.Render(dbModel.ID))
	fmt.Printf("    %s %s\n", dimStyle.Render("vpc:"), valueStyle.Render(dbModel.VPC))
	fmt.Printf("    %s %s\n", dimStyle.Render("status:"), successStyle.Render(string(dbModel.Status)))
	fmt.Println()

	fmt.Println(labelStyle.Render("  internal connection (within vpc):"))
	fmt.Printf("    %s %s\n", dimStyle.Render("hostname:"), valueStyle.Render(dbModel.InternalHostname))
	fmt.Printf("    %s %s\n", dimStyle.Render("port:"), valueStyle.Render(fmt.Sprintf("%d", dbModel.InternalPort)))
	fmt.Printf("    %s %s\n", dimStyle.Render("username:"), valueStyle.Render(dbModel.Username))
	maskedPassword := dbModel.Password
	if len(dbModel.Password) > 16 {
		maskedPassword = dbModel.Password[:16] + "..."
	}
	fmt.Printf("    %s %s\n", dimStyle.Render("password:"), valueStyle.Render(maskedPassword))
	if dbModel.DatabaseName != "" {
		fmt.Printf("    %s %s\n", dimStyle.Render("database:"), valueStyle.Render(dbModel.DatabaseName))
	}
	fmt.Println()
	fmt.Printf("    %s\n", dimStyle.Render("connection string:"))
	fmt.Printf("    %s\n", dimStyle.Render(dbModel.ConnectionString))
	fmt.Println()

	fmt.Println(dimStyle.Render("  [info] this database is private and only accessible within the vpc"))
	fmt.Println(dimStyle.Render(fmt.Sprintf("  [info] to expose to host: yap db publish %s --port <port>", dbName)))
	fmt.Println()

	fmt.Println(dimStyle.Render("  next steps:"))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap db credentials %s  # view credentials", dbName)))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap db logs %s          # view logs", dbName)))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("yap db publish %s --port 5432  # publish to host", dbName)))
	fmt.Println()
}

func init() {
	createCmd.Flags().StringVarP(&createPassword, "password", "p", "", "Database password (auto-generated if not provided)")
	createCmd.Flags().StringVar(&createVPC, "vpc", "primary", "VPC to create database in (auto-created if doesn't exist)")
	dbCmd.AddCommand(createCmd)
}
