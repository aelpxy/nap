package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/database"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/spf13/cobra"
)

var (
	vpcDeleteForce bool
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

	appRegistry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize app registry: %v", err)))
		os.Exit(1)
	}
	if err := appRegistry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize app registry: %v", err)))
		os.Exit(1)
	}

	dbRegistry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize database registry: %v", err)))
		os.Exit(1)
	}
	if err := dbRegistry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize database registry: %v", err)))
		os.Exit(1)
	}

	apps, err := appRegistry.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to list apps: %v", err)))
		os.Exit(1)
	}

	databases, err := dbRegistry.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to list databases: %v", err)))
		os.Exit(1)
	}

	appsInVPC := []string{}
	for _, application := range apps {
		if application.VPC == vpcName {
			appsInVPC = append(appsInVPC, application.Name)
		}
	}

	dbsInVPC := []string{}
	for _, db := range databases {
		if db.VPC == vpcName {
			dbsInVPC = append(dbsInVPC, db.Name)
		}
	}

	if (len(appsInVPC) > 0 || len(dbsInVPC) > 0) && !vpcDeleteForce {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] vpc '%s' contains resources", vpcName)))
		fmt.Println()
		if len(appsInVPC) > 0 {
			fmt.Println(labelStyle.Render("  applications:"))
			for _, appName := range appsInVPC {
				fmt.Printf("    - %s\n", appName)
			}
			fmt.Println()
		}
		if len(dbsInVPC) > 0 {
			fmt.Println(labelStyle.Render("  databases:"))
			for _, dbName := range dbsInVPC {
				fmt.Printf("    - %s\n", dbName)
			}
			fmt.Println()
		}
		fmt.Println(dimStyle.Render("  destroy resources first, or use --force to delete vpc anyway"))
		fmt.Println(dimStyle.Render("  warning: --force will disconnect resources from the vpc"))
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
	vpcDeleteCmd.Flags().BoolVarP(&vpcDeleteForce, "force", "f", false, "Force delete VPC even if it contains resources")
	vpcCmd.AddCommand(vpcDeleteCmd)
}
