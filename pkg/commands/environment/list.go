package environment

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var ListCommand = &cli.Command{
	Name:  "list",
	Usage: "List available deployment environments",
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		// Get active deployment environment to mark it
		currentEnvConfig, err := utils.GetEnvironmentConfig(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get active deployment environment: %w", err)
		}

		logger.Info("Available deployment environments:")

		// List all available deployment environments from EnvironmentConfigs
		for name, config := range common.EnvironmentConfigs {
			marker := ""
			if name == currentEnvConfig.Name {
				marker = " (active)"
			}

			// Add description based on environment
			description := utils.GetEnvironmentDescription(name, config.Name, true)

			logger.Info("  • %s %s%s", name, description, marker)
		}

		return nil
	},
}
