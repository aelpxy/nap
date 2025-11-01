package cmd

import (
	"github.com/spf13/cobra"
)

var appDeploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Manage application deployments",
	Long:  "Manage deployment strategies, confirmations, and rollbacks",
}

func init() {
	appCmd.AddCommand(appDeploymentCmd)
}
