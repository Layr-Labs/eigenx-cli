package app

import (
	"fmt"
	"os"

	"github.com/Layr-Labs/eigenx-cli/config"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var ConfigureTLSCommand = &cli.Command{
	Name:    "configure",
	Aliases: []string{},
	Usage:   "Configure your application",
	Subcommands: []*cli.Command{
		{
			Name:  "tls",
			Usage: "Configure TLS for your application",
			Description: `
Adds TLS configuration to your EigenX application.

This command creates:
- Caddyfile: Reverse proxy configuration for automatic HTTPS
- .env.example.tls: Example environment variables for TLS

TLS certificates are automatically obtained via Let's Encrypt using the tls-keygen tool.`,
			Action: configureTLSAction,
		},
	},
}

func configureTLSAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Write Caddyfile
	caddyfilePath := "Caddyfile"
	if _, err := os.Stat(caddyfilePath); err == nil {
		// Caddyfile already exists
		logger.Warn("Caddyfile already exists. Skipping...")
	} else {
		if err := os.WriteFile(caddyfilePath, []byte(config.CaddyfileTLS), 0644); err != nil {
			return fmt.Errorf("failed to write Caddyfile: %w", err)
		}
		logger.Info("Created Caddyfile")
	}

	// Write .env.example.tls
	envTLSPath := ".env.example.tls"
	if _, err := os.Stat(envTLSPath); err == nil {
		// .env.example.tls already exists
		logger.Warn(".env.example.tls already exists. Skipping...")
	} else {
		if err := os.WriteFile(envTLSPath, []byte(config.EnvExampleTLS), 0644); err != nil {
			return fmt.Errorf("failed to write .env.example.tls: %w", err)
		}
		logger.Info("Created .env.example.tls")
	}

	// Print success message and instructions
	fmt.Println()
	color.Green("TLS configuration added successfully")
	fmt.Println()
	fmt.Println("Created:")
	fmt.Println("  - Caddyfile")
	fmt.Println("  - .env.example.tls")
	fmt.Println()

	fmt.Println("To enable TLS:")
	fmt.Println()
	fmt.Println("1. Add TLS variables to .env:")
	fmt.Println("   cat .env.example.tls >> .env")
	fmt.Println()

	fmt.Println("2. Configure required variables:")
	fmt.Println("   DOMAIN=yourdomain.com")
	fmt.Println("   APP_PORT=3000")
	fmt.Println()
	fmt.Println("   For first deployment (recommended):")
	fmt.Println("   ENABLE_CADDY_LOGS=true")
	fmt.Println("   ACME_STAGING=true")
	fmt.Println()

	fmt.Println("3. Set up DNS A record pointing to instance IP")
	fmt.Println("   Run 'eigenx app info' to get IP address")
	fmt.Println()

	fmt.Println("4. Upgrade:")
	fmt.Println("   eigenx app upgrade")
	fmt.Println()

	fmt.Println("Note: Let's Encrypt rate limit is 5 certificates/week per domain")
	fmt.Println("      To switch staging -> production: set ACME_STAGING=false")
	fmt.Println("      If cert exists, use ACME_FORCE_ISSUE=true once to replace")

	return nil
}
