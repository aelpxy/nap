package cmd

import (
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
	Long:  "Provision and manage PostgreSQL and Valkey databases",
}

func init() {
	rootCmd.AddCommand(dbCmd)
}
