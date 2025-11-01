package cmd

import (
	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Application management commands",
	Long:  "Deploy and manage applications with automatic builds, load balancing, and health checks",
}

func init() {
	rootCmd.AddCommand(appCmd)
}
