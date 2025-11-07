package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/constants"
	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/internal/utils"
	"github.com/spf13/cobra"
)

var vpcCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new VPC",
	Long:  "Create a new Virtual Private Cloud (VPC) for network isolation",
	Args:  cobra.ExactArgs(1),
	Run:   runVPCCreate,
}

func runVPCCreate(cmd *cobra.Command, args []string) {
	vpcName := args[0]

	if vpcName == "" || len(vpcName) == 0 {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[error] vpc name is required"))
		os.Exit(1)
	}
	if len(vpcName) > constants.MaxNameLength {
		fmt.Fprintf(os.Stderr, "%s vpc name: maximum %d characters\n", errorStyle.Render("[error]"), constants.MaxNameLength)
		os.Exit(1)
	}
	if !utils.IsValidName(vpcName) {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[error] vpc name: use only lowercase letters, numbers, and dashes"))
		os.Exit(1)
	}

	vpcRegistry, err := database.NewVPCRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize vpc registry: %v", err)))
		os.Exit(1)
	}

	if err := vpcRegistry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize vpc registry: %v", err)))
		os.Exit(1)
	}

	exists, err := vpcRegistry.Exists(vpcName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to check vpc: %v", err)))
		os.Exit(1)
	}

	if exists {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] vpc %s already exists", vpcName)))
		os.Exit(1)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize network service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> creating vpc: %s", vpcName)))
	fmt.Println()

	fmt.Println(progressStyle.Render("  --> creating network..."))
	vpc, err := dockerClient.CreateVPC(vpcName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to create vpc network: %v", err)))
		os.Exit(1)
	}

	if err := vpcRegistry.Add(*vpc); err != nil {
		_ = dockerClient.DeleteVPC(vpc.NetworkID)
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("  [error] failed to add vpc to registry: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] vpc created successfully"))
	fmt.Println()
	fmt.Println(labelStyle.Render("  vpc details:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("name:"), valueStyle.Render(vpc.Name))
	fmt.Printf("    %s %s\n", dimStyle.Render("network:"), valueStyle.Render(vpc.NetworkName))
	fmt.Printf("    %s %s\n", dimStyle.Render("subnet:"), valueStyle.Render(vpc.Subnet))
	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  use 'yap db create <type> <name> --vpc %s' to create databases in this vpc", vpcName)))
}

func init() {
	vpcCmd.AddCommand(vpcCreateCmd)
}
