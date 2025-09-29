package environment

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var ShowCommand = &cli.Command{
	Name:  "show",
	Usage: "Show active deployment environment",
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		// Get the active deployment environment configuration
		envConfig, err := utils.GetEnvironmentConfig(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get environment config: %w", err)
		}

		// Check if this came from GlobalConfig or is the fallback default
		defaultEnv, err := common.GetDefaultEnvironment()
		if err != nil {
			logger.Debug("Failed to get default environment from global config: %v", err)
		}

		if defaultEnv != "" {
			logger.Info("Active deployment environment: %s", envConfig.Name)
		} else {
			logger.Info("Active deployment environment: %s (fallback default)", envConfig.Name)
			logger.Info("Run 'eigenx environment set <env>' to set your preferred deployment environment")
		}

		logger.Info("Run 'eigenx environment list' to see available deployment environments")

		return nil
	},
}
