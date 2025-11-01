package cmd

import (
	"github.com/spf13/cobra"
)

var vpcCmd = &cobra.Command{
	Use:   "vpc",
	Short: "VPC management commands",
	Long: `Manage Virtual Private Clouds (VPCs) for network isolation.

VPCs are isolated networks that contain your databases and apps.
Resources in the same VPC can communicate privately using service names.`,
}

func init() {
	rootCmd.AddCommand(vpcCmd)
}
