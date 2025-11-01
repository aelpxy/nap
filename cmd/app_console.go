package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aelpxy/nap/internal/app"
	"github.com/spf13/cobra"
)

var (
	consoleInstance int
)

var appConsoleCmd = &cobra.Command{
	Use:   "console [app-name] [command]",
	Short: "Open an interactive shell in an application container",
	Long: `Open an interactive shell or run a command in an application container.

If no command is specified, opens an interactive shell (/bin/sh or /bin/bash).
If a command is specified, runs that command interactively.

Examples:
  nap app console myapp                    # Open default shell
  nap app console myapp /bin/bash          # Open bash shell
  nap app console myapp node               # Open Node.js REPL
  nap app console myapp python             # Open Python REPL
  nap app console myapp --instance 2       # Connect to instance 2`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runAppConsole,
}

func init() {
	appCmd.AddCommand(appConsoleCmd)
	appConsoleCmd.Flags().IntVar(&consoleInstance, "instance", 1, "Instance number to connect to")
}

func runAppConsole(cmd *cobra.Command, args []string) {
	appName := args[0]
	command := "/bin/sh"

	if len(args) > 1 {
		command = args[1]
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> connecting to application: %s", appName)))
	fmt.Println()

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	application, err := registry.Get(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s application not found: %s\n", errorStyle.Render("[error]"), appName)
		fmt.Println()
		fmt.Println(dimStyle.Render("  check available apps:"))
		fmt.Println(dimStyle.Render("    nap app list"))
		os.Exit(1)
	}

	if application.Status != "running" {
		fmt.Fprintf(os.Stderr, "%s application is not running (status: %s)\n", errorStyle.Render("[error]"), application.Status)
		fmt.Println()
		fmt.Println(dimStyle.Render("  start the application first:"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    nap app start %s", appName)))
		os.Exit(1)
	}

	if consoleInstance < 1 || consoleInstance > len(application.ContainerIDs) {
		fmt.Fprintf(os.Stderr, "%s invalid instance number: %d (app has %d instances)\n",
			errorStyle.Render("[error]"), consoleInstance, len(application.ContainerIDs))
		fmt.Println()
		fmt.Println(dimStyle.Render("  available instances:"))
		for i := range application.ContainerIDs {
			fmt.Printf("    %d\n", i+1)
		}
		os.Exit(1)
	}

	containerID := application.ContainerIDs[consoleInstance-1]

	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> connecting to instance %d...", consoleInstance)))
	fmt.Println()
	fmt.Println(dimStyle.Render("  application console"))
	fmt.Printf("    app: %s\n", valueStyle.Render(appName))
	fmt.Printf("    instance: %s\n", valueStyle.Render(fmt.Sprintf("%d/%d", consoleInstance, len(application.ContainerIDs))))
	fmt.Printf("    shell: %s\n", valueStyle.Render(command))
	fmt.Println()

	if strings.HasPrefix(command, "/bin/") {
		fmt.Println(dimStyle.Render("  type 'exit' to disconnect"))
	} else {
		fmt.Println(dimStyle.Render("  interactive mode - type 'exit' or Ctrl+D to disconnect"))
	}
	fmt.Println()

	dockerCmd := exec.Command("docker", "exec", "-it", containerID, command)

	dockerCmd.Stdin = os.Stdin
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	if err := dockerCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 0 || exitErr.ExitCode() == 130 {
				fmt.Println()
				fmt.Println(successStyle.Render("  [done] disconnected"))
				return
			}
		}

		fmt.Fprintf(os.Stderr, "\n%s console exited with error: %v\n", errorStyle.Render("[error]"), err)
		fmt.Println()
		fmt.Println(dimStyle.Render("  troubleshooting:"))
		fmt.Println(dimStyle.Render("    • check if the command exists in the container"))
		fmt.Println(dimStyle.Render("    • try a different shell: /bin/bash or /bin/sh"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    • check container status: nap app status %s", appName)))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  [done] disconnected"))
}
