package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/spf13/cobra"
)

var vpcDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a VPC",
	Long:  "Delete a Virtual Private Cloud (VPC). VPC must be empty (no databases or apps attached).",
	Args:  cobra.ExactArgs(1),
	Run:   runVPCDelete,
}

func runVPCDelete(cmd *cobra.Command, args []string) {
	vpcName := args[0]

	vpcRegistry, err := database.NewVPCRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] Failed to initialize VPC registry: %v", err)))
		os.Exit(1)
	}

	if err := vpcRegistry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] Failed to initialize VPC registry: %v", err)))
		os.Exit(1)
	}

	vpc, err := vpcRegistry.Get(vpcName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] VPC not found: %v", err)))
		os.Exit(1)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize network service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> deleting vpc: %s", vpcName)))
	fmt.Println()

	fmt.Println(progressStyle.Render("  --> removing from registry..."))
	if err := vpcRegistry.Delete(vpcName); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to delete vpc: %v", err)))
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render("  --> removing network..."))
	if err := dockerClient.DeleteVPC(vpc.NetworkID); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to delete vpc network: %v", err)))
		_ = vpcRegistry.Add(*vpc)
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [ok] vpc deleted successfully"))
	fmt.Println()
}

func init() {
	vpcCmd.AddCommand(vpcDeleteCmd)
}
