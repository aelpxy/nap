package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/utils"
	"github.com/spf13/cobra"
)

var vpcInspectCmd = &cobra.Command{
	Use:   "inspect [name]",
	Short: "Show VPC details",
	Long:  "Display detailed information about a Virtual Private Cloud (VPC)",
	Args:  cobra.ExactArgs(1),
	Run:   runVPCInspect,
}

func runVPCInspect(cmd *cobra.Command, args []string) {
	vpcName := args[0]

	vpcRegistry, err := database.NewVPCRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize vpc registry: %v", err)))
		os.Exit(1)
	}

	if err := vpcRegistry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize vpc registry: %v", err)))
		os.Exit(1)
	}

	vpc, err := vpcRegistry.Get(vpcName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] vpc not found: %v", err)))
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

	fmt.Println(titleStyle.Render(fmt.Sprintf("  vpc: %s", vpcName)))
	fmt.Println()

	fmt.Println(infoStyle.Render("network details:"))
	fmt.Println()
	fmt.Printf("  %s  %s\n", labelStyle.Render("name:"), valueStyle.Render(vpc.Name))
	fmt.Printf("  %s  %s\n", labelStyle.Render("network name:"), valueStyle.Render(vpc.NetworkName))
	fmt.Printf("  %s  %s\n", labelStyle.Render("network id:"), valueStyle.Render(utils.TruncateID(vpc.NetworkID, 12)))
	fmt.Printf("  %s  %s\n", labelStyle.Render("subnet:"), valueStyle.Render(vpc.Subnet))
	fmt.Printf("  %s  %s\n", labelStyle.Render("created:"), valueStyle.Render(vpc.CreatedAt.Format("2006-01-02 15:04:05")))
	fmt.Println()

	fmt.Println(infoStyle.Render("resources:"))
	fmt.Println()
	fmt.Printf("  %s  %s\n", labelStyle.Render("databases:"), valueStyle.Render(fmt.Sprintf("%d", len(vpc.Databases))))

	if len(vpc.Databases) > 0 {
		for _, dbID := range vpc.Databases {
			dbs, _ := dbRegistry.List()
			for _, db := range dbs {
				if db.ID == dbID {
					fmt.Printf("    - %s (%s)\n", db.Name, db.Type)
					break
				}
			}
		}
	}

	fmt.Printf("  %s  %s\n", labelStyle.Render("apps:"), valueStyle.Render(fmt.Sprintf("%d", len(vpc.Apps))))
	fmt.Println()
}

func init() {
	vpcCmd.AddCommand(vpcInspectCmd)
}
