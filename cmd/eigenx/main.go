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
	"github.com/urfave/cli/v2"
)

// List of all commands available in the CLI.
var allCommands = []*cli.Command{
	commands.AppCommand,
	commands.AuthCommand,
	commands.BillingCommand,
	commands.EnvironmentCommand,
	commands.UndelegateCommand,
	commands.UpgradeCommand,
	commands.TelemetryCommand,
	version.VersionCommand, // Always include version command
}

// validateBuildEnvironment checks if the required build environment variables are set.
// It uses log.Fatal for immediate exit on configuration failure.
func validateBuildEnvironment() {
	if common.Build == "" {
		// Log a specific error message about the missing environment
		log.Fatal("Build environment is not properly configured. 'common.Build' variable is empty.")
	}
}

// main initializes and runs the EigenX CLI application.
func main() {
	// 1. Mandatory Build Check
	validateBuildEnvironment()

	// 2. Setup Context with Shutdown Signal
	// This context will carry the cancellation signal for graceful shutdown (e.g., on SIGINT).
	ctx := common.WithShutdown(context.Background())

	// 3. Define the CLI Application
	app := &cli.App{
		Name:                   "eigenx",
		Usage:                  "EigenX Development Kit",
		Flags:                  common.GlobalFlags,
		UseShortOptionHandling: true,
		
		// Before hook runs before any command logic.
		Before: func(cCtx *cli.Context) error {
			// A. Load environment variables from .env file
			if err := hooks.LoadEnvFile(cCtx); err != nil {
				return fmt.Errorf("failed to load environment file: %w", err)
			}
			common.WithAppEnvironment(cCtx)

			// B. Initialize Logger and Progress Tracker
			// The logger should automatically respect the 'verbose' flag read by the CLI context.
			logger, tracker := common.GetLoggerFromCLIContext(cCtx)
			cCtx.Context = common.WithLogger(cCtx.Context, logger)
			cCtx.Context = common.WithProgressTracker(cCtx.Context, tracker)

			// C. Handle First-Run Setup
			// Skips setup for commands that don't require configuration (e.g., help, version, telemetry).
			// This check assumes the commands package provides a reliable way to skip this setup.
			isMetaCommand := cCtx.Command.Name == "help" || cCtx.Command.Name == "version" || cCtx.Command.Name == "environment" || cCtx.Command.Name == "telemetry"
			
			if !isMetaCommand {
				if err := hooks.WithFirstRunSetup(cCtx); err != nil {
					// Log as debug/warning, do not fail the command execution
					logger.Debug("First-run setup failed: %v", err)
				}
			}

			// D. Check for CLI updates (non-blocking)
			hooks.InitVersionCheck(cCtx)

			// E. Setup metrics tracking context
			return hooks.WithCommandMetricsContext(cCtx)
		},

		Commands: allCommands,
	}

	// 4. Define and Apply Middleware (Action Chain)
	// These hooks run after the 'Before' hook, right before the command's main Action.
	actionChain := hooks.NewActionChain()
	actionChain.Use(hooks.WithVersionCheck) // Final check/display of version status
	actionChain.Use(hooks.WithMetricEmission) // Emit command metrics after execution

	hooks.ApplyMiddleware(app.Commands, actionChain)

	// 5. Run the application
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
