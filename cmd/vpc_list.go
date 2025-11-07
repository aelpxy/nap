package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/database"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var vpcListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all VPCs",
	Long:  "Display all Virtual Private Clouds (VPCs)",
	Run:   runVPCList,
}

func runVPCList(cmd *cobra.Command, args []string) {
	vpcRegistry, err := database.NewVPCRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize vpc registry: %v", err)))
		os.Exit(1)
	}

	if err := vpcRegistry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize vpc registry: %v", err)))
		os.Exit(1)
	}

	vpcs, err := vpcRegistry.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to list vpcs: %v", err)))
		os.Exit(1)
	}

	if len(vpcs) == 0 {
		fmt.Println(dimStyle.Render("no vpcs found"))
		fmt.Println()
		fmt.Println(dimStyle.Render("create a vpc with: yap vpc create <name>"))
		return
	}

	fmt.Println(titleStyle.Render("==> vpcs"))
	fmt.Println()

	rows := [][]string{}
	for _, vpc := range vpcs {
		dbCount := fmt.Sprintf("%d", len(vpc.Databases))
		appCount := fmt.Sprintf("%d", len(vpc.Apps))

		rows = append(rows, []string{
			vpc.Name,
			vpc.Subnet,
			dbCount,
			appCount,
			vpc.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("238"))).
		Headers("name", "subnet", "databases", "apps", "created").
		Rows(rows...)

	fmt.Println(t)
	fmt.Println()
}

func init() {
	vpcCmd.AddCommand(vpcListCmd)
}
