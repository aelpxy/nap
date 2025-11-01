package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aelpxy/nap/internal/app"
	"github.com/aelpxy/nap/internal/config"
	"github.com/aelpxy/nap/internal/docker"
	"github.com/aelpxy/nap/internal/utils"
	"github.com/spf13/cobra"
)

var (
	publishDomain string
)

var appPublishCmd = &cobra.Command{
	Use:   "publish [app]",
	Short: "make app accessible from the internet with https",
	Long:  "publish an application to make it accessible externally with automatic ssl/tls",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		ctx := context.Background()

		dockerClient, err := docker.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to initialize docker: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		configManager, err := config.NewConfigManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load config: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		registry, err := app.NewRegistryManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}
		publishingMgr := app.NewPublishingManager(dockerClient, registry, configManager)

		fmt.Println()
		fmt.Println(titleStyle.Render("==> publishing application"))
		fmt.Println()

		fmt.Println(progressStyle.Render("  --> configuring external access..."))
		if err := publishingMgr.PublishApp(ctx, appName, publishDomain); err != nil {
			fmt.Println()
			fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		application, err := registry.Get(appName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to get app info: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("  [done]") + " application published")
		fmt.Println()
		fmt.Println("  " + labelStyle.Render("url:") + " " + infoStyle.Render(application.PublishedURL))
		fmt.Println("  " + labelStyle.Render("domain:") + " " + infoStyle.Render(application.PublishedDomain))

		fmt.Println()
		fmt.Println(titleStyle.Render("==> dns configuration required"))
		fmt.Println()

		publicIP, err := utils.GetPublicIP()
		if err != nil {
			fmt.Println("  " + dimStyle.Render("detecting server ip..."))
			fmt.Println("  " + errorStyle.Render(fmt.Sprintf("[warn] could not detect public ip: %v", err)))
			fmt.Println()
			fmt.Println("  " + dimStyle.Render("create an A record in your dns provider:"))
			fmt.Println("  " + infoStyle.Render(fmt.Sprintf("  %s  →  <your server ip>", application.PublishedDomain)))
		} else {
			fmt.Println("  " + dimStyle.Render("server ip detected:") + " " + successStyle.Render(publicIP))
			fmt.Println()
			fmt.Println("  " + dimStyle.Render("create an A record in your dns provider:"))
			fmt.Println("  " + infoStyle.Render(fmt.Sprintf("  %s  →  %s", application.PublishedDomain, publicIP)))
			fmt.Println()
			fmt.Println("  " + dimStyle.Render("example dns record:"))
			fmt.Println("    " + dimStyle.Render("type: A"))
			fmt.Println("    " + dimStyle.Render(fmt.Sprintf("name: %s", application.PublishedDomain)))
			fmt.Println("    " + dimStyle.Render(fmt.Sprintf("value: %s", publicIP)))
			fmt.Println("    " + dimStyle.Render("ttl: 300 (or your provider's default)"))
		}

		fmt.Println()
		fmt.Println(titleStyle.Render("==> ssl/tls"))
		fmt.Println()
		fmt.Println("  " + successStyle.Render("[info]") + " ssl certificate will be generated automatically")
		fmt.Println("  " + dimStyle.Render("on first https access, let's encrypt will issue a certificate"))
		fmt.Println("  " + dimStyle.Render("this may take 10-30 seconds on first request"))
		fmt.Println()
		fmt.Println(titleStyle.Render("==> verification"))
		fmt.Println()
		fmt.Println("  " + dimStyle.Render("wait 1-2 minutes for dns propagation, then test:"))
		fmt.Println("  " + infoStyle.Render(fmt.Sprintf("  curl -I %s", application.PublishedURL)))
		fmt.Println()
		fmt.Println("  " + dimStyle.Render("or visit in browser:"))
		fmt.Println("  " + infoStyle.Render(fmt.Sprintf("  %s", application.PublishedURL)))
	},
}

var appUnpublishCmd = &cobra.Command{
	Use:   "unpublish [app]",
	Short: "make app private (vpc-only)",
	Long:  "unpublish an application to make it accessible only within vpc",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		ctx := context.Background()

		dockerClient, err := docker.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to initialize docker: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		configManager, err := config.NewConfigManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load config: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		registry, err := app.NewRegistryManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}
		publishingMgr := app.NewPublishingManager(dockerClient, registry, configManager)

		fmt.Println()
		fmt.Println(titleStyle.Render("==> unpublishing application"))
		fmt.Println()

		fmt.Println(progressStyle.Render("  --> removing external access..."))
		if err := publishingMgr.UnpublishApp(ctx, appName); err != nil {
			fmt.Println()
			fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("  [done]") + " application unpublished")
		fmt.Println()
		fmt.Println("  app is now accessible only within vpc")
		fmt.Println("  " + dimStyle.Render(fmt.Sprintf("internal: http://%s.nap.local", appName)))
	},
}

var appDomainCmd = &cobra.Command{
	Use:   "domain",
	Short: "manage custom domains",
	Long:  "add or remove custom domains for published applications",
}

var appDomainAddCmd = &cobra.Command{
	Use:   "add [app] [domain]",
	Short: "add custom domain to published app",
	Long:  "add an additional custom domain to a published application",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		domain := args[1]
		ctx := context.Background()

		dockerClient, err := docker.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to initialize docker: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		configManager, err := config.NewConfigManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load config: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		registry, err := app.NewRegistryManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}
		publishingMgr := app.NewPublishingManager(dockerClient, registry, configManager)

		fmt.Println()
		fmt.Println(titleStyle.Render("==> adding custom domain"))
		fmt.Println()

		fmt.Println(progressStyle.Render("  --> configuring domain..."))
		if err := publishingMgr.AddCustomDomain(ctx, appName, domain); err != nil {
			fmt.Println()
			fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("  [done]") + " domain added")
		fmt.Println()
		fmt.Println("  " + labelStyle.Render("domain:") + " " + infoStyle.Render(domain))

		fmt.Println()
		fmt.Println(titleStyle.Render("==> dns configuration required"))
		fmt.Println()

		publicIP, err := utils.GetPublicIP()
		if err != nil {
			fmt.Println("  " + dimStyle.Render("create an A record in your dns provider:"))
			fmt.Println("  " + infoStyle.Render(fmt.Sprintf("  %s  →  <your server ip>", domain)))
		} else {
			fmt.Println("  " + dimStyle.Render("server ip:") + " " + successStyle.Render(publicIP))
			fmt.Println()
			fmt.Println("  " + dimStyle.Render("create an A record:"))
			fmt.Println("  " + infoStyle.Render(fmt.Sprintf("  %s  →  %s", domain, publicIP)))
		}

		fmt.Println()
		fmt.Println("  " + successStyle.Render("[info]") + " ssl certificate will be generated on first https access")
		fmt.Println("  " + dimStyle.Render(fmt.Sprintf("test with: curl -I https://%s", domain)))
	},
}

var appDomainRemoveCmd = &cobra.Command{
	Use:   "remove [app] [domain]",
	Short: "remove custom domain from app",
	Long:  "remove a custom domain from a published application",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		domain := args[1]
		ctx := context.Background()

		dockerClient, err := docker.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to initialize docker: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		configManager, err := config.NewConfigManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load config: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		registry, err := app.NewRegistryManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load registry: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}
		publishingMgr := app.NewPublishingManager(dockerClient, registry, configManager)

		fmt.Println()
		fmt.Println(titleStyle.Render("==> removing custom domain"))
		fmt.Println()

		fmt.Println(progressStyle.Render("  --> removing domain..."))
		if err := publishingMgr.RemoveCustomDomain(ctx, appName, domain); err != nil {
			fmt.Println()
			fmt.Fprintf(os.Stderr, "%s %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("  [done]") + " domain removed")
	},
}

func init() {
	appCmd.AddCommand(appPublishCmd)
	appCmd.AddCommand(appUnpublishCmd)
	appCmd.AddCommand(appDomainCmd)
	appDomainCmd.AddCommand(appDomainAddCmd)
	appDomainCmd.AddCommand(appDomainRemoveCmd)

	appPublishCmd.Flags().StringVar(&publishDomain, "domain", "", "custom domain (optional, defaults to {app}.nap.{base-domain})")
}
