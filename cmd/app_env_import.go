package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/utils"
	"github.com/spf13/cobra"
)

var appEnvImportCmd = &cobra.Command{
	Use:   "import [app-name] [path-to-env-file]",
	Short: "Import environment variables from .env file",
	Long:  "Import environment variables from a .env file (KEY=VALUE format)",
	Args:  cobra.ExactArgs(2),
	Run:   runAppEnvImport,
}

func init() {
	appEnvCmd.AddCommand(appEnvImportCmd)
}

func runAppEnvImport(cmd *cobra.Command, args []string) {
	appName := args[0]
	envFilePath := args[1]

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> importing environment variables: %s", appName)))
	fmt.Println()

	absPath, err := filepath.Abs(envFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to resolve path: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s file does not exist: %s\n", errorStyle.Render("[error]"), absPath)
		os.Exit(1)
	}

	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> reading %s...", envFilePath)))

	envVars, err := parseEnvFile(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to parse .env file: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if len(envVars) == 0 {
		fmt.Println()
		fmt.Println(infoStyle.Render("  [info] no variables found in file"))
		return
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("  [info] found %d variables", len(envVars))))
	fmt.Println()

	fmt.Println(dimStyle.Render("  preview:"))
	count := 0
	for key, value := range envVars {
		maskedValue := utils.MaskSensitiveEnvValue(key, value)
		fmt.Printf("    %s=%s\n", key, maskedValue)
		count++
		if count >= 10 {
			remaining := len(envVars) - count
			if remaining > 0 {
				fmt.Printf("    ... and %d more\n", remaining)
			}
			break
		}
	}
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
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if application.EnvVars == nil {
		application.EnvVars = make(map[string]string)
	}

	for key, value := range envVars {
		application.EnvVars[key] = value
	}

	if err := registry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] imported %d environment variables", len(envVars))))
	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  run 'yap app restart %s' to apply changes", appName)))
}

func parseEnvFile(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envVars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "%s line %d: invalid format (expected KEY=VALUE), skipping\n",
				infoStyle.Render("[info]"), lineNumber)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			fmt.Fprintf(os.Stderr, "%s line %d: empty key, skipping\n",
				infoStyle.Render("[info]"), lineNumber)
			continue
		}

		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		envVars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envVars, nil
}
