package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aelpxy/yap/internal/app"
	"github.com/aelpxy/yap/internal/docker"
	"github.com/aelpxy/yap/internal/router"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/spf13/cobra"
)

var (
	rollbackVersion int
)

var appRollbackCmd = &cobra.Command{
	Use:   "rollback [name]",
	Short: "Rollback to previous deployment",
	Long:  "Rollback an application to a previous deployment version",
	Args:  cobra.ExactArgs(1),
	Run:   runAppRollback,
}

func init() {
	appCmd.AddCommand(appRollbackCmd)
	appRollbackCmd.Flags().IntVar(&rollbackVersion, "version", 0, "Deployment version to rollback to (0 = previous)")
}

func runAppRollback(cmd *cobra.Command, args []string) {
	appName := args[0]

	lockManager := app.GetGlobalLockManager()
	if err := lockManager.TryLock(appName, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "%s another operation in progress for %s\n", errorStyle.Render("[error]"), appName)
		fmt.Println(dimStyle.Render("  wait for the current operation to complete or try again in a few seconds"))
		os.Exit(1)
	}
	defer lockManager.Unlock(appName)

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
		fmt.Fprintf(os.Stderr, "%s application '%s' not found\n", errorStyle.Render("[error]"), appName)
		os.Exit(1)
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("==> rolling back: %s", appName)))
	fmt.Println()

	if len(application.DeploymentHistory) < 2 {
		fmt.Fprintf(os.Stderr, "%s no previous deployments to rollback to\n", errorStyle.Render("[error]"))
		fmt.Println(dimStyle.Render("  need at least 2 deployments to rollback"))
		os.Exit(1)
	}

	var targetDeployment *models.DeploymentRecord
	var targetVersion int

	if rollbackVersion == 0 {
		targetVersion = len(application.DeploymentHistory) - 1
		targetDeployment = &application.DeploymentHistory[targetVersion-1]
	} else {
		if rollbackVersion < 1 || rollbackVersion > len(application.DeploymentHistory) {
			fmt.Fprintf(os.Stderr, "%s invalid version: %d\n", errorStyle.Render("[error]"), rollbackVersion)
			fmt.Fprintf(os.Stderr, "  valid versions: 1-%d\n", len(application.DeploymentHistory))
			os.Exit(1)
		}
		targetVersion = rollbackVersion
		targetDeployment = &application.DeploymentHistory[rollbackVersion-1]
	}

	fmt.Printf("  --> rolling back to deployment #%d\n", targetVersion)
	fmt.Printf("    image: %s\n", dimStyle.Render(targetDeployment.ImageID[:min(len(targetDeployment.ImageID), 20)]))
	fmt.Printf("    deployed: %s\n", dimStyle.Render(targetDeployment.DeployedAt.Format("2006-01-02 15:04:05")))
	fmt.Println()

	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to initialize: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	traefik := router.NewTraefikManager(dockerClient)

	traefikLabels := traefik.GenerateLabelsForApp(application)

	fmt.Println()
	fmt.Println(progressStyle.Render("  --> validating image availability..."))
	_, _, err = dockerClient.GetClient().ImageInspectWithRaw(cmd.Context(), targetDeployment.ImageID)
	if err != nil {
		fmt.Println()
		fmt.Fprintf(os.Stderr, "%s target image not found\n", errorStyle.Render("[error]"))
		fmt.Fprintf(os.Stderr, "  image: %s\n", dimStyle.Render(targetDeployment.ImageID))
		fmt.Println()
		fmt.Println(dimStyle.Render("  the image for this deployment has been deleted"))
		fmt.Println(dimStyle.Render("  you may need to rebuild from source or use a different version"))
		fmt.Println()
		os.Exit(1)
	}
	fmt.Println(dimStyle.Render("    image found, proceeding with rollback"))

	deployer, err := app.NewDeployer(application.DeploymentStrategy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to create deployer: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	deployOpts := app.DeploymentOptions{
		App:           application,
		NewImageID:    targetDeployment.ImageID,
		Config:        application.DeploymentConfig,
		VPCName:       application.VPC,
		TraefikLabels: traefikLabels,
		MemoryMB:      application.Memory,
		CPUCores:      application.CPU,
	}

	imageID, err := deployer.Deploy(deployOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s rollback failed: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	application.ImageID = imageID
	application.Status = models.AppStatusRunning
	application.UpdatedAt = time.Now()
	application.LastDeployedAt = time.Now()

	rollbackRecord := models.DeploymentRecord{
		ID:         fmt.Sprintf("dep-%s", time.Now().Format("20060102-150405")),
		ImageID:    imageID,
		Strategy:   application.DeploymentStrategy,
		DeployedAt: time.Now(),
		Status:     "active",
	}

	for i := range application.DeploymentHistory {
		if application.DeploymentHistory[i].Status == "active" {
			application.DeploymentHistory[i].Status = "rolled-back"
		}
	}

	application.DeploymentHistory = append(application.DeploymentHistory, rollbackRecord)

	if err := registry.Update(*application); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to update registry: %v\n", errorStyle.Render("[error]"), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  [done] rolled back to deployment #%d", targetVersion)))
	fmt.Printf("    current image: %s\n", dimStyle.Render(imageID[:min(len(imageID), 20)]))
	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  use 'yap app deployments %s' to view history", appName)))
	fmt.Println()
}
