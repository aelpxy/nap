package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aelpxy/nap/internal/builder"
	"github.com/spf13/cobra"
)

var (
	initFull bool
	initName string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new nap project",
	Long:  "Create a nap.toml configuration file in the current directory",
	Run:   runInit,
}

func runInit(cmd *cobra.Command, args []string) {
	if _, err := os.Stat("nap.toml"); err == nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[error] nap.toml already exists"))
		fmt.Println(dimStyle.Render("  use 'nap.toml' to configure your deployment"))
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render("==> initializing nap project"))
	fmt.Println()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to get current directory: %v", err)))
		os.Exit(1)
	}

	language, _ := builder.DetectLanguage(cwd)
	wasDetected := language != "unknown"

	if language == "unknown" {
		language = promptForRuntime()
	}

	port := builder.GetDefaultPort(language)

	appName := initName
	if appName == "" {
		appName = filepath.Base(cwd)
	}

	fmt.Println()
	if wasDetected {
		fmt.Println(progressStyle.Render("  --> detected project"))
		fmt.Printf("    %s %s\n", dimStyle.Render("language:"), valueStyle.Render(language))
		fmt.Printf("    %s %s\n", dimStyle.Render("default port:"), valueStyle.Render(fmt.Sprintf("%d", port)))
	} else {
		fmt.Println(progressStyle.Render("  --> configuring project"))
		fmt.Printf("    %s %s\n", dimStyle.Render("runtime:"), valueStyle.Render(language))
		fmt.Printf("    %s %s\n", dimStyle.Render("port:"), valueStyle.Render(fmt.Sprintf("%d", port)))
	}
	fmt.Printf("    %s %s\n", dimStyle.Render("app name:"), valueStyle.Render(appName))
	fmt.Println()

	var config string
	if initFull {
		fmt.Println(progressStyle.Render("  --> creating full configuration..."))
		config = generateFullConfig(appName, language, port)
	} else {
		fmt.Println(progressStyle.Render("  --> creating minimal configuration..."))
		config = generateMinimalConfig(appName, language, port)
	}

	if err := os.WriteFile("nap.toml", []byte(config), 0644); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to write nap.toml: %v", err)))
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] nap.toml created"))
	fmt.Println()
	fmt.Println(labelStyle.Render("  next steps:"))
	fmt.Printf("    %s\n", dimStyle.Render("1. review and customize nap.toml"))
	fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("2. deploy with: nap app deploy %s", appName)))
	fmt.Println()
}

func generateMinimalConfig(name, runtime string, port int) string {
	envVar := "APP_ENV"
	switch runtime {
	case "nodejs":
		envVar = "NODE_ENV"
	case "python":
		envVar = "PYTHON_ENV"
	case "go":
		envVar = "GO_ENV"
	case "rust":
		envVar = "RUST_ENV"
	case "ruby":
		envVar = "RAILS_ENV"
	case "php":
		envVar = "APP_ENV"
	case "java":
		envVar = "SPRING_PROFILES_ACTIVE"
	}

	runtimeValue := runtime
	if runtime == "unknown" {
		runtimeValue = "docker"
	}

	return fmt.Sprintf(`# nap.toml - Minimal configuration

[app]
name = "%s"
region = "us-east-1"
runtime = "%s"
env = "production"

[deployment]
strategy = "recreate"          # recreate (default), rolling, blue-green

[deploy]
instances = 1
memory = "512M"
cpu = 0.5
port = %d

[deploy.health_check]
path = "/health"
interval = 10
timeout = 5

[env]
%s = "production"
`, name, runtimeValue, port, envVar)
}

func generateFullConfig(name, runtime string, port int) string {
	envVar := "APP_ENV"
	switch runtime {
	case "nodejs":
		envVar = "NODE_ENV"
	case "python":
		envVar = "PYTHON_ENV"
	case "go":
		envVar = "GO_ENV"
	case "rust":
		envVar = "RUST_ENV"
	case "ruby":
		envVar = "RAILS_ENV"
	case "php":
		envVar = "APP_ENV"
	case "java":
		envVar = "SPRING_PROFILES_ACTIVE"
	}

	runtimeValue := runtime
	if runtime == "unknown" {
		runtimeValue = "docker"
	}

	return fmt.Sprintf(`# nap.toml - Full configuration

[app]
name = "%s"
region = "us-east-1"
runtime = "%s"  # nodejs, python, go, rust, ruby, php, java, docker
env = "production"  # production, staging, development

[build]
# Build configuration
dockerfile = "Dockerfile"  # Optional: custom Dockerfile path
buildpacks = false         # Use buildpacks instead of Dockerfile
build_args = []

[deployment]
# Deployment strategy configuration
strategy = "recreate"          # recreate (default), rolling, blue-green
max_surge = 1                  # Rolling: deploy N instances at a time
rolling_interval = 5           # Rolling: seconds to wait between instance deployments
health_timeout = 30            # Rolling/Blue-Green: seconds to wait for health check
auto_confirm = false           # Blue-Green: auto-destroy old version after switch
confirmation_timeout = 300     # Blue-Green: seconds to wait for manual confirmation

[deploy]
# Deployment settings
instances = 1              # Number of instances to run
memory = "512M"            # Memory allocation per instance
cpu = 0.5                  # CPU allocation (vCPU)
port = %d                  # Internal application port
auto_scaling = false       # Enable auto-scaling

[deploy.health_check]
# Health check configuration
path = "/health"
interval = 10              # seconds
timeout = 5                # seconds
retries = 3

[deploy.resources]
# Resource limits
memory_limit = "1G"
cpu_limit = 1.0

[network]
# Network configuration
ssl = false                # Auto-provision SSL certificate
domain = "%s.nap.local"    # Custom domain
internal_only = false      # Only accessible internally

[env]
# Environment variables
%s = "production"
LOG_LEVEL = "info"

[database]
# Database connections (optional)
# These are auto-populated when using 'nap app link'
# postgres = "db-abc123"
# valkey = "cache-xyz789"

[volumes]
# Persistent volumes (optional)
# data = "/app/data"
# uploads = "/app/uploads"

[hooks]
# Lifecycle hooks
# prebuild = "npm run prebuild"
# postbuild = "npm run postbuild"
# predeploy = "npm run migrate"
# postdeploy = "npm run seed"

[monitoring]
# Monitoring and observability
metrics = false
logs_retention = 7         # days
alerts = false

[scaling]
# Auto-scaling configuration (when auto_scaling = true)
min_instances = 1
max_instances = 10
cpu_threshold = 70         # percentage
memory_threshold = 80      # percentage
scale_up_delay = 60        # seconds
scale_down_delay = 300     # seconds
`, name, runtimeValue, port, name, envVar)
}

func promptForRuntime() string {
	fmt.Println()
	fmt.Println(infoStyle.Render("  [info] could not detect project language"))
	fmt.Println()
	fmt.Println(labelStyle.Render("  select runtime:"))
	fmt.Println(dimStyle.Render("    1. nodejs"))
	fmt.Println(dimStyle.Render("    2. python"))
	fmt.Println(dimStyle.Render("    3. go"))
	fmt.Println(dimStyle.Render("    4. rust"))
	fmt.Println(dimStyle.Render("    5. ruby"))
	fmt.Println(dimStyle.Render("    6. php"))
	fmt.Println(dimStyle.Render("    7. java"))
	fmt.Println()
	fmt.Print("  [1-7]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("[error] failed to read input: %v", err)))
		os.Exit(1)
	}

	choice := strings.TrimSpace(input)

	runtimes := map[string]string{
		"1": "nodejs",
		"2": "python",
		"3": "go",
		"4": "rust",
		"5": "ruby",
		"6": "php",
		"7": "java",
	}

	runtime, ok := runtimes[choice]
	if !ok {
		fmt.Println(dimStyle.Render("  invalid choice, using generic configuration"))
		return "unknown"
	}

	return runtime
}

func init() {
	initCmd.Flags().BoolVar(&initFull, "full", false, "Create full configuration with all options")
	initCmd.Flags().StringVar(&initName, "name", "", "Application name (defaults to directory name)")
	rootCmd.AddCommand(initCmd)
}
