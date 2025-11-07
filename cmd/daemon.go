package cmd

import (
	"fmt"
	"os"

	"github.com/aelpxy/yap/internal/runtime"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage container runtime daemon",
	Long:  "Manage the container runtime daemon (Podman)",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the container runtime daemon",
	Long: `Start the container runtime daemon (Podman API service).

This command starts the Podman API service, which allows yap to communicate with Podman
using the Docker-compatible API. The service runs in the background and persists across
system restarts (if systemd is available).

For rootless Podman, the service runs as your user without requiring root privileges.

Examples:
  yap daemon start              # Start Podman service
  yap daemon start --enable-linger  # Enable systemd user services at boot`,
	Run: runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the container runtime daemon",
	Long: `Stop the container runtime daemon (Podman API service).

This stops the Podman API service. All running containers will continue to run,
but yap commands will not work until the service is restarted.

Examples:
  yap daemon stop`,
	Run: runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long: `Show the status of the container runtime daemon.

Displays information about the detected runtime (Docker or Podman), version,
socket location, and whether the service is running.

Examples:
  yap daemon status`,
	Run: runDaemonStatus,
}

var (
	enableLinger bool
)

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)

	daemonStartCmd.Flags().BoolVar(&enableLinger, "enable-linger", false, "Enable systemd user services at boot")
}

func runDaemonStart(cmd *cobra.Command, args []string) {
	fmt.Println(titleStyle.Render("==> starting container runtime daemon"))
	fmt.Println()

	runtimeInfo, err := runtime.DetectRuntime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to detect runtime: %v\n", errorStyle.Render("[error]"), err)
		fmt.Println()
		fmt.Println(dimStyle.Render("  please install docker or podman:"))
		fmt.Println(dimStyle.Render("    • docker: https://docs.docker.com/get-docker/"))
		fmt.Println(dimStyle.Render("    • podman: https://podman.io/getting-started/installation"))
		os.Exit(1)
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("  [info] detected runtime: %s", runtimeInfo.GetRuntimeName())))
	fmt.Println()

	if runtimeInfo.Type == runtime.RuntimeDocker {
		fmt.Println(successStyle.Render("  [done] docker daemon is already running"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  docker daemon is managed by:"))
		fmt.Println(dimStyle.Render("    • systemctl (linux)"))
		fmt.Println(dimStyle.Render("    • docker desktop (macos/windows)"))
		return
	}

	daemonMgr := runtime.NewDaemonManager(runtimeInfo)

	if enableLinger {
		fmt.Println(progressStyle.Render("  --> enabling systemd user services at boot..."))
		if err := daemonMgr.EnableSystemdUserService(); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to enable linger: %v\n", errorStyle.Render("[error]"), err)
			fmt.Println()
			fmt.Println(dimStyle.Render("  you may need to run manually:"))
			fmt.Println(dimStyle.Render("    loginctl enable-linger $USER"))
			fmt.Println()
		} else {
			fmt.Println(successStyle.Render("  [done] user services enabled at boot"))
			fmt.Println()
		}
	}

	fmt.Println(progressStyle.Render("  --> starting podman api service..."))
	if err := daemonMgr.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to start daemon: %v\n", errorStyle.Render("[error]"), err)
		fmt.Println()
		fmt.Println(dimStyle.Render("  troubleshooting:"))
		fmt.Println(dimStyle.Render("    • check if podman is installed: podman --version"))
		fmt.Println(dimStyle.Render("    • try starting manually: systemctl --user start podman.socket"))
		fmt.Println(dimStyle.Render("    • check logs: journalctl --user -u podman.socket"))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] podman daemon started"))
	fmt.Println()
	fmt.Println(labelStyle.Render("  service information:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("socket:"), valueStyle.Render(runtimeInfo.SocketPath))
	fmt.Printf("    %s %s\n", dimStyle.Render("version:"), valueStyle.Render(runtimeInfo.Version))
	if runtimeInfo.IsRootless {
		fmt.Printf("    %s %s\n", dimStyle.Render("mode:"), successStyle.Render("rootless"))
	}
	fmt.Println()
	fmt.Println(dimStyle.Render("  the daemon will restart automatically on reboot (if systemd is enabled)"))
	fmt.Println()
}

func runDaemonStop(cmd *cobra.Command, args []string) {
	fmt.Println(titleStyle.Render("==> stopping container runtime daemon"))
	fmt.Println()

	runtimeInfo, err := runtime.DetectRuntime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to detect runtime: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("  [info] detected runtime: %s", runtimeInfo.GetRuntimeName())))
	fmt.Println()

	if runtimeInfo.Type == runtime.RuntimeDocker {
		fmt.Println(dimStyle.Render("  docker daemon cannot be stopped via yap"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  use systemctl or docker desktop to manage the docker daemon"))
		return
	}

	daemonMgr := runtime.NewDaemonManager(runtimeInfo)

	fmt.Println(progressStyle.Render("  --> stopping podman api service..."))
	if err := daemonMgr.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to stop daemon: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] podman daemon stopped"))
	fmt.Println()
	fmt.Println(dimStyle.Render("  note: all containers continue running, but yap commands won't work"))
	fmt.Println(dimStyle.Render("  run 'yap daemon start' to restart the service"))
	fmt.Println()
}

func runDaemonStatus(cmd *cobra.Command, args []string) {
	fmt.Println(titleStyle.Render("==> container runtime status"))
	fmt.Println()

	runtimeInfo, err := runtime.DetectRuntime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s no container runtime detected\n", errorStyle.Render("[error]"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  please install docker or podman:"))
		fmt.Println(dimStyle.Render("    • docker: https://docs.docker.com/get-docker/"))
		fmt.Println(dimStyle.Render("    • podman: https://podman.io/getting-started/installation"))
		os.Exit(1)
	}

	daemonMgr := runtime.NewDaemonManager(runtimeInfo)

	status, _ := daemonMgr.Status()
	isRunning := daemonMgr.IsRunning()

	fmt.Println(labelStyle.Render("  runtime information:"))
	fmt.Printf("    %s %s\n", dimStyle.Render("type:"), valueStyle.Render(string(runtimeInfo.Type)))
	fmt.Printf("    %s %s\n", dimStyle.Render("version:"), valueStyle.Render(runtimeInfo.Version))
	if runtimeInfo.Type == runtime.RuntimePodman && runtimeInfo.IsRootless {
		fmt.Printf("    %s %s\n", dimStyle.Render("mode:"), successStyle.Render("rootless (secure)"))
	}
	fmt.Println()

	fmt.Println(labelStyle.Render("  service status:"))
	if isRunning {
		fmt.Printf("    %s %s\n", dimStyle.Render("status:"), successStyle.Render("running"))
	} else {
		fmt.Printf("    %s %s\n", dimStyle.Render("status:"), errorStyle.Render("stopped"))
	}
	fmt.Printf("    %s %s\n", dimStyle.Render("socket:"), valueStyle.Render(runtimeInfo.SocketPath))
	if status != "" && status != "running" && status != "stopped" {
		fmt.Printf("    %s %s\n", dimStyle.Render("details:"), dimStyle.Render(status))
	}
	fmt.Println()

	if !isRunning && runtimeInfo.Type == runtime.RuntimePodman {
		fmt.Println(dimStyle.Render("  start the daemon:"))
		fmt.Println(dimStyle.Render("    yap daemon start"))
		fmt.Println()
	}

	if isRunning {
		fmt.Println(successStyle.Render("  [ready] yap is ready to use"))
		fmt.Println()
	}
}
