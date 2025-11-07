package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aelpxy/yap/internal/config"
	"github.com/aelpxy/yap/internal/utils"
	"github.com/aelpxy/yap/pkg/models"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "manage yap configuration",
	Long:  "manage global yap configuration settings",
}

var configSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "configure publishing settings",
	Long:  "interactive setup for external publishing and ssl/tls",
	Run: func(cmd *cobra.Command, args []string) {
		configManager, err := config.NewConfigManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load config: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		reader := bufio.NewReader(os.Stdin)

		fmt.Println()
		fmt.Println(titleStyle.Render("==> publishing configuration"))
		fmt.Println()
		fmt.Println("  " + dimStyle.Render("this will configure yap to publish apps with https using let's encrypt"))
		fmt.Println("  " + dimStyle.Render("you'll need:"))
		fmt.Println("    " + dimStyle.Render("• a domain you control"))
		fmt.Println("    " + dimStyle.Render("• dns access to create records"))
		fmt.Println("    " + dimStyle.Render("• ports 80 and 443 open"))
		fmt.Println()

		fmt.Print("  enable external publishing with https? (y/n): ")
		enableInput, _ := reader.ReadString('\n')
		enableInput = strings.TrimSpace(strings.ToLower(enableInput))

		if enableInput != "y" && enableInput != "yes" {
			cfg := configManager.GetConfig()
			cfg.Publishing.Enabled = false
			if err := configManager.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "%s failed to save config: %v\n", errorStyle.Render("[error]"), err)
				os.Exit(1)
			}

			fmt.Println()
			fmt.Println(successStyle.Render("  [done]") + " publishing disabled")
			return
		}

		fmt.Println()
		fmt.Println("  base domain for published apps")
		fmt.Println("  " + dimStyle.Render("apps will be accessible at {app}.yap.{base-domain}"))
		fmt.Print("  enter domain (e.g., example.com): ")
		domainInput, _ := reader.ReadString('\n')
		domainInput = strings.TrimSpace(domainInput)

		if domainInput == "" {
			fmt.Fprintf(os.Stderr, "%s base domain is required\n", errorStyle.Render("[error]"))
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println("  email for let's encrypt ssl certificates")
		fmt.Print("  enter email: ")
		emailInput, _ := reader.ReadString('\n')
		emailInput = strings.TrimSpace(emailInput)

		if emailInput == "" {
			fmt.Fprintf(os.Stderr, "%s email is required for let's encrypt\n", errorStyle.Render("[error]"))
			os.Exit(1)
		}

		cfg := configManager.GetConfig()
		cfg.Publishing = models.PublishingConfig{
			Enabled:    true,
			BaseDomain: domainInput,
			Email:      emailInput,
		}

		if err := configManager.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to save config: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("  [done]") + " publishing configured")
		fmt.Println()
		fmt.Println("  " + dimStyle.Render("apps will be published to:"))
		fmt.Println("  " + infoStyle.Render(fmt.Sprintf("https://{app}.yap.%s", domainInput)))
		fmt.Println()

		fmt.Println(titleStyle.Render("==> next steps"))
		fmt.Println()

		publicIP, err := utils.GetPublicIP()
		if err != nil {
			fmt.Println("  " + dimStyle.Render("1. configure wildcard dns:"))
			fmt.Println("  " + infoStyle.Render(fmt.Sprintf("     *.yap.%s  →  <your server ip>", domainInput)))
		} else {
			fmt.Println("  " + dimStyle.Render("server ip detected:") + " " + successStyle.Render(publicIP))
			fmt.Println()
			fmt.Println("  " + dimStyle.Render("1. configure wildcard dns in your dns provider:"))
			fmt.Println("  " + infoStyle.Render(fmt.Sprintf("     *.yap.%s  →  %s", domainInput, publicIP)))
			fmt.Println()
			fmt.Println("  " + dimStyle.Render("   example dns record:"))
			fmt.Println("    " + dimStyle.Render("   type: A"))
			fmt.Println("    " + dimStyle.Render(fmt.Sprintf("   name: *.yap.%s", domainInput)))
			fmt.Println("    " + dimStyle.Render(fmt.Sprintf("   value: %s", publicIP)))
		}

		fmt.Println()
		fmt.Println("  " + dimStyle.Render("2. ensure ports 80 and 443 are open in your firewall"))
		fmt.Println()
		fmt.Println("  " + dimStyle.Render("3. publish an app:"))
		fmt.Println("  " + infoStyle.Render("     yap app publish <app-name>"))
		fmt.Println()
		fmt.Println("  " + dimStyle.Render("4. ssl certificates will be issued automatically on first access"))
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "display current configuration",
	Long:  "show current yap configuration settings",
	Run: func(cmd *cobra.Command, args []string) {
		configManager, err := config.NewConfigManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed to load config: %v\n", errorStyle.Render("[error]"), err)
			os.Exit(1)
		}

		cfg := configManager.GetConfig()

		fmt.Println()
		fmt.Println(titleStyle.Render("==> yap configuration"))
		fmt.Println()

		fmt.Println("  " + labelStyle.Render("publishing:"))
		if cfg.Publishing.Enabled {
			fmt.Println("    enabled: " + successStyle.Render("true"))
			fmt.Println("    base domain: " + infoStyle.Render(cfg.Publishing.BaseDomain))
			fmt.Println("    email: " + infoStyle.Render(cfg.Publishing.Email))
			fmt.Println()
			fmt.Println("    " + dimStyle.Render("apps publish to:"))
			fmt.Println("    " + dimStyle.Render(fmt.Sprintf("https://{app}.yap.%s", cfg.Publishing.BaseDomain)))
		} else {
			fmt.Println("    enabled: " + dimStyle.Render("false"))
			fmt.Println()
			fmt.Println("    " + dimStyle.Render("run 'yap config setup' to enable publishing"))
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetupCmd)
	configCmd.AddCommand(configShowCmd)
}
