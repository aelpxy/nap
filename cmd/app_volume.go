package cmd

import (
	"github.com/spf13/cobra"
)

var appVolumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage application volumes",
	Long:  "Add, remove, list, and inspect persistent volumes for applications",
}

func init() {
	appCmd.AddCommand(appVolumeCmd)
}
