package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/database"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all databases",
	Long:  "List all nap-managed databases",
	Run:   runList,
}

func runList(cmd *cobra.Command, args []string) {
	registry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	databases, err := registry.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to list databases: %v", err)))
		os.Exit(1)
	}

	if len(databases) == 0 {
		fmt.Println(dimStyle.Render("no databases found."))
		fmt.Println()
		fmt.Printf("create one with: %s\n", dimStyle.Render("nap db create postgres my-db"))
		return
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> databases (%d)", len(databases))))
	fmt.Println()

	rows := [][]string{}
	for _, db := range databases {
		statusColor := "42"
		if db.Status != "running" {
			statusColor = "241"
		}

		statusStyled := lipgloss.NewStyle().
			Foreground(lipgloss.Color(statusColor)).
			Render(string(db.Status))

		var publishedStatus string
		if db.Published {
			publishedStatus = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Render(fmt.Sprintf("yes:%d", db.PublishedPort))
		} else {
			publishedStatus = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Render("no")
		}

		rows = append(rows, []string{
			db.Name,
			string(db.Type),
			db.VPC,
			statusStyled,
			publishedStatus,
			db.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().
					Foreground(lipgloss.Color("86")).
					Bold(true).
					Align(lipgloss.Center)
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		}).
		Headers("name", "type", "vpc", "status", "published", "created").
		Rows(rows...)

	fmt.Println(t)
	fmt.Println()

	fmt.Println(dimStyle.Render("  common commands:"))
	fmt.Printf("    %s\n", dimStyle.Render("nap db status <name>       # view details"))
	fmt.Printf("    %s\n", dimStyle.Render("nap db credentials <name>  # get credentials"))
	fmt.Printf("    %s\n", dimStyle.Render("nap db logs <name>         # view logs"))
	fmt.Println()
}

func init() {
	dbCmd.AddCommand(listCmd)
}
