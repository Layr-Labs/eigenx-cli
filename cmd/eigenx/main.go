package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands"
	"github.com/Layr-Labs/eigenx-cli/pkg/commands/version"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/hooks"
	"github.com/Layr-Labs/eigenx-cli/pkg/versioncheck"
	"github.com/urfave/cli/v2"
)

func validateBuildEnvironment() {
	if common.Build == "" {
		log.Fatal("Build environment not properly configured")
	}
}

func main() {
	validateBuildEnvironment()

	ctx := common.WithShutdown(context.Background())

	app := &cli.App{
		Name:  "eigenx",
		Usage: "EigenX Development Kit",
		Flags: common.GlobalFlags,
		Before: func(cCtx *cli.Context) error {
			err := hooks.LoadEnvFile(cCtx)
			if err != nil {
				return err
			}
			common.WithAppEnvironment(cCtx)

			// Parse verbose flags from raw argv to capture from subcommand flags
			verbose := common.PeelBoolFromFlags(os.Args[1:], "--verbose", "-v")
			// Set verbose directly if it appears in subcommand flags
			if verbose {
				err := cCtx.Set("verbose", "true")
				if err != nil {
					return fmt.Errorf("failed to set verbose flag globally: %w", err)
				}
			}

			// Get logger based on CLI context (handles verbosity internally)
			logger, tracker := common.GetLoggerFromCLIContext(cCtx)

			// Store logger and tracker in the context
			cCtx.Context = common.WithLogger(cCtx.Context, logger)
			cCtx.Context = common.WithProgressTracker(cCtx.Context, tracker)

			// Handle first-run setup (environment + telemetry)
			if cCtx.Command.Name != "help" && cCtx.Command.Name != "version" && cCtx.Command.Name != "environment" && cCtx.Command.Name != "telemetry" {
				if err := hooks.WithFirstRunSetup(cCtx); err != nil {
					// Log error but don't fail the command
					logger.Debug("First-run setup failed: %v", err)
				}
			}

			// Check for updates (only for prod builds, skip for upgrade, version, and help commands)
			if common.Build == "prod" && cCtx.Command.Name != "upgrade" && cCtx.Command.Name != "version" && cCtx.Command.Name != "help" {
				// Run version check asynchronously to avoid blocking
				go func() {
					updateInfo, err := versioncheck.CheckForUpdate(logger)
					if err == nil && updateInfo != nil && updateInfo.Available {
						versioncheck.PrintUpdateNotification(updateInfo)
					}
				}()
			}

			return hooks.WithCommandMetricsContext(cCtx)
		},
		Commands: []*cli.Command{
			commands.AppCommand,
			commands.AuthCommand,
			commands.EnvironmentCommand,
			version.VersionCommand,
			commands.UndelegateCommand,
			commands.UpgradeCommand,
			commands.TelemetryCommand,
		},
		UseShortOptionHandling: true,
	}

	actionChain := hooks.NewActionChain()
	actionChain.Use(hooks.WithMetricEmission)

	hooks.ApplyMiddleware(app.Commands, actionChain)

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
