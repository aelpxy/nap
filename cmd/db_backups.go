package cmd

import (
	"github.com/spf13/cobra"
)

var backupsCmd = &cobra.Command{
	Use:   "backups",
	Short: "Manage database backups",
	Long:  "Manage database backups for nap databases",
}

func init() {
	dbCmd.AddCommand(backupsCmd)
}
