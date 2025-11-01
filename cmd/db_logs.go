package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
)

var (
	followLogs bool
)

var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Stream database logs",
	Long:  "Stream logs from a database",
	Args:  cobra.ExactArgs(1),
	Run:   runLogs,
}

func runLogs(cmd *cobra.Command, args []string) {
	dbName := args[0]

	registry, err := database.NewRegistryManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize registry: %v", err)))
		os.Exit(1)
	}

	db, err := registry.Get(dbName)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] database not found: %v", err)))
		os.Exit(1)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to initialize database service: %v", err)))
		os.Exit(1)
	}
	defer dockerClient.Close()

	logs, err := dockerClient.GetContainerLogs(db.ContainerID, followLogs)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to get logs: %v", err)))
		os.Exit(1)
	}
	defer logs.Close()

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> logs: %s", dbName)))
	fmt.Println()

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
	logsCmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "Follow log output")
	dbCmd.AddCommand(logsCmd)
}
