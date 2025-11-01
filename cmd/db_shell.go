package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/aelpxy/nap/internal/database"
	"github.com/spf13/cobra"
)

var dbShellCmd = &cobra.Command{
	Use:   "shell [name]",
	Short: "Open an interactive database shell",
	Long: `Open an interactive shell to the database.

For PostgreSQL databases, opens psql.
For Valkey databases, opens redis-cli.

Examples:
  nap db shell mydb           # Open PostgreSQL shell
  nap db shell cache          # Open Valkey shell`,
	Args: cobra.ExactArgs(1),
	Run:  runDBShell,
}

func init() {
	dbCmd.AddCommand(dbShellCmd)
}

func runDBShell(cmd *cobra.Command, args []string) {
	dbName := args[0]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> connecting to database: %s", dbName)))
	fmt.Println()

	registry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	db, err := registry.Get(dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s database not found: %s\n", errorStyle.Render("[error]"), dbName)
		fmt.Println()
		fmt.Println(dimStyle.Render("  check available databases:"))
		fmt.Println(dimStyle.Render("    nap db list"))
		os.Exit(1)
	}

	if db.Status != "running" {
		fmt.Fprintf(os.Stderr, "%s database is not running (status: %s)\n", errorStyle.Render("[error]"), db.Status)
		fmt.Println()
		fmt.Println(dimStyle.Render("  start the database first:"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    nap db start %s", dbName)))
		os.Exit(1)
	}

	var shellCmd *exec.Cmd

	switch db.Type {
	case "postgres":
		fmt.Println(progressStyle.Render("  --> opening postgresql shell..."))
		fmt.Println()
		fmt.Println(dimStyle.Render("  connected to postgresql"))
		fmt.Printf("    database: %s\n", valueStyle.Render(db.DatabaseName))
		fmt.Printf("    user: %s\n", valueStyle.Render(db.Username))
		fmt.Println()
		fmt.Println(dimStyle.Render("  type '\\q' to exit"))
		fmt.Println()

		shellCmd = exec.Command("docker", "exec", "-it",
			db.ContainerName,
			"psql",
			"-U", db.Username,
			"-d", db.DatabaseName,
		)

	case "valkey":
		fmt.Println(progressStyle.Render("  --> opening valkey shell..."))
		fmt.Println()
		fmt.Println(dimStyle.Render("  connected to valkey"))
		fmt.Printf("    port: %s\n", valueStyle.Render(fmt.Sprintf("%d", db.InternalPort)))
		fmt.Println()
		fmt.Println(dimStyle.Render("  type 'exit' or 'quit' to exit"))
		fmt.Println()

		shellCmd = exec.Command("docker", "exec", "-it",
			db.ContainerName,
			"redis-cli",
			"-a", db.Password,
		)

	default:
		fmt.Fprintf(os.Stderr, "%s unsupported database type: %s\n", errorStyle.Render("[error]"), db.Type)
		os.Exit(1)
	}

	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	if err := shellCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s shell exited with error: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  [done] disconnected"))
}
