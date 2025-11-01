package cmd

import (
	"github.com/spf13/cobra"
)

var appEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage application environment variables",
	Long:  "List, set, unset, import, and export environment variables for applications",
}

func init() {
	appCmd.AddCommand(appEnvCmd)
}
