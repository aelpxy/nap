package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("213"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true)
)

var rootCmd = &cobra.Command{
	Use:   "yap",
	Short: "yet another platform - a lightweight self-hosted PaaS",
	Long: titleStyle.Render(`
    __  __ ____ _ ____
   / / / // __ `+"`"+`/ __ \
  / /_/ // /_/ / /_/ /
  \__, / \__,_/ .___/
 /____/      /_/
`) + "\n" + subtitleStyle.Render("yet another platform") + "\n\n" +
		"A barebones PaaS alternative for self-hosting your projects.",
	Version: "0.1.0",
}

func SetVersionInfo(v, bt, gc string) {
	version = v
	buildTime = bt
	gitCommit = gc
	rootCmd.Version = fmt.Sprintf("%s (built: %s, commit: %s)", version, buildTime, gitCommit)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] Error: %v", err)))
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}
