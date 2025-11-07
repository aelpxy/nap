package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
)

var (
	followAppLogs bool
)

var appLogsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Stream application logs",
	Long:  "Stream logs from an application",
	Args:  cobra.ExactArgs(1),
	Run:   runAppLogs,
}

func runAppLogs(cmd *cobra.Command, args []string) {
	appName := args[0]

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	application, err := registry.Get(appName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] application not found: %v", err)))
		os.Exit(1)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> logs: %s", appName)))
	fmt.Println()

	if len(application.ContainerIDs) == 0 {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[error] no instances found"))
		os.Exit(1)
	}

	containerID := application.ContainerIDs[0]

	logs, err := dockerClient.GetContainerLogs(containerID, followAppLogs)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to get logs: %v", err)))
		os.Exit(1)
	}
	defer logs.Close()

	stdoutPipe, stdoutWriter := io.Pipe()
	stderrPipe, stderrWriter := io.Pipe()

	go func() {
		defer stdoutWriter.Close()
		defer stderrWriter.Close()
		_, err := stdcopy.StdCopy(stdoutWriter, stderrWriter, logs)
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] error demultiplexing logs: %v", err)))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] error reading logs: %v", err)))
		os.Exit(1)
	}
}

func init() {
	appLogsCmd.Flags().BoolVarP(&followAppLogs, "follow", "f", false, "Follow log output")
	appCmd.AddCommand(appLogsCmd)
}
