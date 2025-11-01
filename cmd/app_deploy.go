package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/builder"
	"github.com/aelpxy/nap/internal/constants"
	"github.com/aelpxy/nap/internal/database"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/internal/project"
	"github.com/aelpxy/nap/internal/router"
	"github.com/aelpxy/nap/internal/utils"
	"github.com/aelpxy/nap/pkg/models"
	"github.com/spf13/cobra"
)

var appDeployCmd = &cobra.Command{
	Use:   "deploy [name] [path]",
	Short: "Deploy an application",
	Long:  "Build and deploy an application from source code or Dockerfile",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runAppDeploy,
}

var (
	deployVPC            string
	deployPort           int
	deployMemory         int
	deployCPU            float64
	deployInstances      int
	deployHealthPath     string
	deployHealthInterval int
	deployHealthTimeout  int
	deployBuildMethod    string

	deployStrategy        string
	deployMaxSurge        int
	deployRollingInterval int
	deployAutoConfirm     bool
)

func init() {
	appCmd.AddCommand(appDeployCmd)

	appDeployCmd.Flags().StringVar(&deployVPC, "vpc", "primary", "VPC to deploy in")
	appDeployCmd.Flags().IntVar(&deployPort, "port", 0, "Application port (auto-detect if not specified)")
	appDeployCmd.Flags().IntVar(&deployMemory, "memory", 512, "Memory limit in MB")
	appDeployCmd.Flags().Float64Var(&deployCPU, "cpu", 0.5, "CPU limit")
	appDeployCmd.Flags().IntVar(&deployInstances, "instances", 1, "Number of instances")
	appDeployCmd.Flags().StringVar(&deployHealthPath, "health-path", "/health", "Health check endpoint")
	appDeployCmd.Flags().IntVar(&deployHealthInterval, "health-interval", 10, "Health check interval in seconds")
	appDeployCmd.Flags().IntVar(&deployHealthTimeout, "health-timeout", 5, "Health check timeout in seconds")
	appDeployCmd.Flags().StringVar(&deployBuildMethod, "build-method", "auto", "Build method: auto, dockerfile, nixpacks, paketo")

	appDeployCmd.Flags().StringVar(&deployStrategy, "strategy", "recreate", "Deployment strategy: recreate, rolling, blue-green")
	appDeployCmd.Flags().IntVar(&deployMaxSurge, "max-surge", 1, "Rolling: deploy N instances at a time")
	appDeployCmd.Flags().IntVar(&deployRollingInterval, "rolling-interval", 5, "Rolling: seconds between instance deployments")
	appDeployCmd.Flags().BoolVar(&deployAutoConfirm, "auto-confirm", false, "Blue-green: auto-destroy old environment")
}

func runAppDeploy(cmd *cobra.Command, args []string) {
	appName := args[0]
	projectPath := "."
	if len(args) > 1 {
		projectPath = args[1]
	}

	absPath, err := utils.ValidateProjectPath(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if appName == "" || len(appName) == 0 {
		fmt.Fprintf(os.Stderr, "%s invalid application name: name is required\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}
	if len(appName) > constants.MaxNameLength {
		fmt.Fprintf(os.Stderr, "%s invalid application name: maximum %d characters\n", errorStyle.Render("[error]"), constants.MaxNameLength)
		os.Exit(1)
	}
	if !utils.IsValidName(appName) {
		fmt.Fprintf(os.Stderr, "%s invalid application name: use only lowercase letters, numbers, and dashes\n", errorStyle.Render("[error]"))
		os.Exit(1)
	}
	if deployPort < constants.MinPort || deployPort > constants.MaxPort {
		fmt.Fprintf(os.Stderr, "%s invalid port: must be between %d and %d\n", errorStyle.Render("[error]"), constants.MinPort, constants.MaxPort)
		os.Exit(1)
	}
	if deployMemory < constants.MinMemoryMB {
		fmt.Fprintf(os.Stderr, "%s invalid memory: must be at least %dMB\n", errorStyle.Render("[error]"), constants.MinMemoryMB)
		os.Exit(1)
	}
	if deployMemory > constants.MaxMemoryMB {
		fmt.Fprintf(os.Stderr, "%s invalid memory: maximum %dMB (%dGB)\n", errorStyle.Render("[error]"), constants.MaxMemoryMB, constants.MaxMemoryMB/1024)
		os.Exit(1)
	}
	if deployCPU < constants.MinCPUCores {
		fmt.Fprintf(os.Stderr, "%s invalid cpu: must be at least %d core\n", errorStyle.Render("[error]"), constants.MinCPUCores)
		os.Exit(1)
	}
	if deployCPU > constants.MaxCPUCores {
		fmt.Fprintf(os.Stderr, "%s invalid cpu: maximum %d cores\n", errorStyle.Render("[error]"), constants.MaxCPUCores)
		os.Exit(1)
	}
	if deployInstances < constants.MinInstances {
		fmt.Fprintf(os.Stderr, "%s invalid instance count: minimum %d required\n", errorStyle.Render("[error]"), constants.MinInstances)
		os.Exit(1)
	}
	if deployInstances > constants.MaxInstances {
		fmt.Fprintf(os.Stderr, "%s invalid instance count: maximum %d instances\n", errorStyle.Render("[error]"), constants.MaxInstances)
		os.Exit(1)
	}

	project, err := project.LoadConfigIfExists(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load nap.toml: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if project != nil {
		fmt.Println(infoStyle.Render("  [info] loaded configuration from nap.toml"))

		if project.App.Name != "" && !cmd.Flags().Changed("name") {
			appName = project.App.Name
		}

		if !cmd.Flags().Changed("instances") {
			deployInstances = project.Deploy.Instances
		}
		if !cmd.Flags().Changed("memory") {
			deployMemory = models.ParseMemory(project.Deploy.Memory)
		}
		if !cmd.Flags().Changed("cpu") {
			deployCPU = project.Deploy.CPU
		}
		if !cmd.Flags().Changed("port") && project.Deploy.Port != 0 {
			deployPort = project.Deploy.Port
		}
		if !cmd.Flags().Changed("health-path") && project.Deploy.HealthCheck.Path != "" {
			deployHealthPath = project.Deploy.HealthCheck.Path
		}
		if !cmd.Flags().Changed("health-interval") {
			deployHealthInterval = project.Deploy.HealthCheck.Interval
		}
		if !cmd.Flags().Changed("health-timeout") {
			deployHealthTimeout = project.Deploy.HealthCheck.Timeout
		}
		if !cmd.Flags().Changed("strategy") && project.Deployment.Strategy != "" {
			deployStrategy = project.Deployment.Strategy
		}
		if !cmd.Flags().Changed("max-surge") {
			deployMaxSurge = project.Deployment.MaxSurge
		}
		if !cmd.Flags().Changed("rolling-interval") {
			deployRollingInterval = project.Deployment.RollingInterval
		}
		if !cmd.Flags().Changed("auto-confirm") {
			deployAutoConfirm = project.Deployment.AutoConfirm
		}
	}

	lockManager := app.GetGlobalLockManager()
	if err := lockManager.TryLock(appName, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "%s another operation in progress for %s\n", errorStyle.Render("[error]"), appName)
		fmt.Println(dimStyle.Render("  wait for the current operation to complete or try again in a few seconds"))
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> deploying application: %s", appName)))
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	registry, err := app.NewRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := registry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	existingApp, err := registry.Get(appName)
	isRedeployment := (err == nil && existingApp != nil)

	fmt.Println(progressStyle.Render("  --> building application..."))
	fmt.Println()

	b := builder.NewBuilder(dockerClient)
	buildResult, err := b.BuildWithMethod(absPath, appName, deployBuildMethod, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s build failed: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  [done] build completed"))
	fmt.Printf("    image: %s\n", dimStyle.Render(buildResult.ImageName))
	fmt.Printf("    id: %s\n", dimStyle.Render(utils.TruncateID(buildResult.ImageID, 12)))
	fmt.Println()

	port := deployPort
	if port == 0 {
		port = builder.GetDefaultPort(buildResult.Language)
		fmt.Println(infoStyle.Render(fmt.Sprintf("  [info] detected port: %d", port)))
	}

	fmt.Println(progressStyle.Render("  --> preparing load balancer..."))
	traefik := router.NewTraefikManager(dockerClient)

	running, err := traefik.IsRunning()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to check load balancer: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if !running {
		fmt.Println(progressStyle.Render("  --> starting load balancer..."))
		if err := traefik.Start(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to start load balancer: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}
	} else {
		fmt.Println(dimStyle.Render("    load balancer already running"))
	}

	fmt.Println(progressStyle.Render(fmt.Sprintf("  --> configuring network (vpc: %s)...", deployVPC)))
	vpcRegistry, err := database.NewVPCRegistryManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to load vpc registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	if err := vpcRegistry.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize vpc registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	vpc, err := vpcRegistry.Get(deployVPC)
	if err != nil {
		fmt.Println(progressStyle.Render(fmt.Sprintf("  --> creating vpc: %s...", deployVPC)))
		vpc, err = dockerClient.CreateVPC(deployVPC)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to create vpc: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}
		if err := vpcRegistry.Add(*vpc); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to register vpc: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}
	}

	if err := traefik.ConnectToVPC(vpc.NetworkName); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to connect load balancer to vpc: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println(successStyle.Render("  [done] network configured"))

	var strategy models.DeploymentStrategy
	var application *models.Application

	if isRedeployment {
		application = existingApp

		if cmd.Flags().Changed("strategy") {
			switch deployStrategy {
			case "recreate":
				strategy = models.DeploymentStrategyRecreate
			case "rolling":
				strategy = models.DeploymentStrategyRolling
			case "blue-green":
				strategy = models.DeploymentStrategyBlueGreen
			default:
				fmt.Fprintf(os.Stderr, "%s unknown deployment strategy: %s\n", errorStyle.Render("[error]"), deployStrategy)
				fmt.Println(dimStyle.Render("  valid strategies: recreate, rolling, blue-green"))
				os.Exit(1)
			}
			application.DeploymentStrategy = strategy
		} else {
			strategy = application.DeploymentStrategy
			if strategy == "" {
				strategy = models.DeploymentStrategyRecreate
				application.DeploymentStrategy = strategy
			}
		}

		if cmd.Flags().Changed("instances") {
			application.Instances = deployInstances
		}
		if cmd.Flags().Changed("memory") {
			application.Memory = deployMemory
		}
		if cmd.Flags().Changed("cpu") {
			application.CPU = deployCPU
		}
		if cmd.Flags().Changed("port") {
			application.Port = port
		}

		if cmd.Flags().Changed("max-surge") {
			application.DeploymentConfig.MaxSurge = deployMaxSurge
		}
		if cmd.Flags().Changed("rolling-interval") {
			application.DeploymentConfig.RollingInterval = deployRollingInterval
		}
		if cmd.Flags().Changed("health-timeout") {
			application.DeploymentConfig.HealthTimeout = deployHealthTimeout
		}
		if cmd.Flags().Changed("auto-confirm") {
			application.DeploymentConfig.AutoConfirm = deployAutoConfirm
		}

		fmt.Println(infoStyle.Render(fmt.Sprintf("  [info] re-deploying with strategy: %s", strategy)))
		fmt.Println()
	} else {
		switch deployStrategy {
		case "recreate":
			strategy = models.DeploymentStrategyRecreate
		case "rolling":
			strategy = models.DeploymentStrategyRolling
		case "blue-green":
			strategy = models.DeploymentStrategyBlueGreen
		default:
			fmt.Fprintf(os.Stderr, "%s unknown deployment strategy: %s\n", errorStyle.Render("[error]"), deployStrategy)
			fmt.Println(dimStyle.Render("  valid strategies: recreate, rolling, blue-green"))
			os.Exit(1)
		}
	}

	var appID string

	if !isRedeployment {
		appID = app.GenerateID()

		application = &models.Application{
			ID:   appID,
			Name: appName,
			VPC:  deployVPC,

			Status:    models.AppStatusDeploying,
			Instances: deployInstances,
			BuildType: buildResult.BuildType,
			ImageID:   buildResult.ImageID,

			Memory: deployMemory,
			CPU:    deployCPU,
			Port:   port,

			HealthCheckPath:     deployHealthPath,
			HealthCheckInterval: deployHealthInterval,
			HealthCheckTimeout:  deployHealthTimeout,

			EnvVars: make(map[string]string),

			DeploymentStrategy: strategy,
			DeploymentConfig: models.DeploymentConfig{
				MaxSurge:            deployMaxSurge,
				RollingInterval:     deployRollingInterval,
				HealthTimeout:       deployHealthTimeout,
				AutoConfirm:         deployAutoConfirm,
				ConfirmationTimeout: 300,
			},
			DeploymentState: models.DeploymentState{
				Active: models.DeploymentColorDefault,
			},

			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			LastDeployedAt: time.Now(),
		}
	} else {
		appID = application.ID

		application.BuildType = buildResult.BuildType
		application.Status = models.AppStatusDeploying
		application.UpdatedAt = time.Now()
	}

	if project != nil && len(project.Env) > 0 {
		if application.EnvVars == nil {
			application.EnvVars = make(map[string]string)
		}
		for key, value := range project.Env {
			if _, exists := application.EnvVars[key]; !exists {
				application.EnvVars[key] = value
			}
		}
	}

	traefikLabels := traefik.GenerateLabelsForApp(application)

	deployer, err := app.NewDeployer(strategy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to create deployer: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	deployOpts := app.DeploymentOptions{
		App:           application,
		SourcePath:    absPath,
		NewImageID:    buildResult.ImageName,
		Config:        application.DeploymentConfig,
		VPCName:       deployVPC,
		TraefikLabels: traefikLabels,
		MemoryMB:      deployMemory,
		CPUCores:      deployCPU,
	}

	imageID, err := deployer.Deploy(deployOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s deployment failed: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	application.ImageID = imageID
	application.Status = models.AppStatusRunning

	if isRedeployment {
		fmt.Println(progressStyle.Render("  --> updating application..."))

		deploymentRecord := models.DeploymentRecord{
			ID:         fmt.Sprintf("dep-%s", time.Now().Format("20060102-150405")),
			ImageID:    imageID,
			Strategy:   strategy,
			DeployedAt: time.Now(),
			Status:     "active",
		}

		for i := range application.DeploymentHistory {
			if application.DeploymentHistory[i].Status == "active" {
				application.DeploymentHistory[i].Status = "superseded"
			}
		}

		application.DeploymentHistory = append(application.DeploymentHistory, deploymentRecord)
		application.LastDeployedAt = time.Now()

		if err := registry.Update(*application); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to update application: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		fmt.Println(successStyle.Render("  [done] application updated"))
	} else {
		fmt.Println(progressStyle.Render("  --> registering application..."))

		application.InternalHostname = fmt.Sprintf("nap-app-%s", appName)
		application.Published = false
		application.PublishedURL = fmt.Sprintf("http://%s.nap.local", appName)

		deploymentRecord := models.DeploymentRecord{
			ID:         fmt.Sprintf("dep-%s", time.Now().Format("20060102-150405")),
			ImageID:    imageID,
			Strategy:   strategy,
			DeployedAt: time.Now(),
			Status:     "active",
		}
		application.DeploymentHistory = []models.DeploymentRecord{deploymentRecord}

		if err := registry.Add(*application); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to register application: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		fmt.Println(successStyle.Render("  [done] application registered"))
	}
	fmt.Println()

	// unlock early because volumes like to fight for their own locks (they're rebellious like that)
	lockManager.Unlock(appName)

	if project != nil && len(project.Volumes) > 0 && !isRedeployment {
		fmt.Println(infoStyle.Render("  [info] adding volumes from nap.toml..."))

		volumeManager := app.NewVolumeManager(dockerClient, registry)
		ctx := context.Background()

		for volumeName, mountPath := range project.Volumes {
			vol := models.Volume{
				Name:      volumeName,
				MountPath: mountPath,
				Type:      "volume",
			}

			if err := volumeManager.AddVolume(ctx, appName, vol); err != nil {
				fmt.Fprintf(os.Stderr, "%s failed to add volume %s: %v\n", errorStyle.Render("[error]"), volumeName, err)
				continue
			}
			fmt.Printf("    added volume: %s -> %s\n", dimStyle.Render(volumeName), dimStyle.Render(mountPath))
		}

		if len(project.Volumes) > 0 {
			fmt.Println()
			fmt.Println(infoStyle.Render("  [info] volumes added - redeploy to mount them"))
			fmt.Println(dimStyle.Render(fmt.Sprintf("  run 'nap app deploy %s' to mount volumes", appName)))
			fmt.Println()
		}
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] %s deployed successfully", appName)))
	fmt.Println()
	fmt.Println(titleStyle.Render("  application details:"))
	fmt.Printf("    name: %s\n", valueStyle.Render(appName))
	fmt.Printf("    id: %s\n", dimStyle.Render(appID))
	fmt.Printf("    vpc: %s\n", valueStyle.Render(deployVPC))
	fmt.Printf("    instances: %s\n", valueStyle.Render(fmt.Sprintf("%d", deployInstances)))
	fmt.Printf("    memory: %s\n", valueStyle.Render(fmt.Sprintf("%d MB", deployMemory)))
	fmt.Printf("    cpu: %s\n", valueStyle.Render(fmt.Sprintf("%.1f", deployCPU)))
	fmt.Printf("    strategy: %s\n", valueStyle.Render(string(strategy)))
	fmt.Println()
	fmt.Println(titleStyle.Render("  access:"))
	fmt.Printf("    url: %s\n", valueStyle.Render(fmt.Sprintf("http://%s.nap.local", appName)))
	fmt.Printf("    internal: %s\n", dimStyle.Render(fmt.Sprintf("nap-app-%s:%d", appName, port)))
	fmt.Println()
	fmt.Println(titleStyle.Render("  next steps:"))
	fmt.Println()
	fmt.Println("  " + dimStyle.Render("test your app:"))
	fmt.Println("  " + infoStyle.Render(fmt.Sprintf("    curl -H \"Host: %s.nap.local\" http://localhost", appName)))
	fmt.Println()
	fmt.Println("  " + dimStyle.Render("monitor and debug:"))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app logs %s [-f]", appName)))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app status %s", appName)))
	fmt.Println()
	fmt.Println("  " + dimStyle.Render("scale your app:"))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app scale %s --instances 3", appName)))
	fmt.Println()
	fmt.Println("  " + dimStyle.Render("add environment variables:"))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app env set %s KEY=value", appName)))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app env import %s .env", appName)))
	fmt.Println()
	fmt.Println("  " + dimStyle.Render("link a database:"))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap db create postgres mydb --vpc %s", deployVPC)))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app link %s mydb", appName)))
	fmt.Println()
	fmt.Println("  " + dimStyle.Render("add persistent storage:"))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app volume add %s data /app/data", appName)))
	fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app deploy %s   # redeploy to mount volume", appName)))

	if !application.Published {
		fmt.Println()
		fmt.Println("  " + dimStyle.Render("publish with https:"))
		fmt.Println("  " + dimStyle.Render("    nap config setup"))
		fmt.Println("  " + dimStyle.Render(fmt.Sprintf("    nap app publish %s", appName)))
	}

	if strategy == models.DeploymentStrategyBlueGreen && !deployAutoConfirm {
		fmt.Println()
		fmt.Println(labelStyle.Render("  blue-green deployment:"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    nap app deployment status %s   # view deployment state", appName)))
	}
}
