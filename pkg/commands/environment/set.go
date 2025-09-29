package environment

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var SetCommand = &cli.Command{
	Name:      "set",
	Usage:     "Set deployment environment",
	ArgsUsage: "<environment>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "yes",
			Usage: "Skip confirmation prompts (for automation)",
		},
	},
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		if cCtx.NArg() > 1 {
			return fmt.Errorf("too many arguments - expected at most one environment name")
		}

		newEnv, err := utils.GetEnvironmentInteractive(cCtx, 0)
		if err != nil {
			return fmt.Errorf("failed to get environment: %w", err)
		}

		// Validate that the environment exists
		if _, exists := common.EnvironmentConfigs[newEnv]; !exists {
			return fmt.Errorf("unknown environment: %s\nRun 'eigenx environment list' to see available environments", newEnv)
		}

		// Check if this is mainnet and requires confirmation
		if common.IsMainnetEnvironment(newEnv) && !cCtx.Bool("yes") {
			if err := utils.ConfirmMainnetEnvironment(newEnv); err != nil {
				return err
			}
		}

		// Set the deployment environment
		if err := common.SetDefaultEnvironment(newEnv); err != nil {
			return fmt.Errorf("failed to set deployment environment: %w", err)
		}

		logger.Info("✅ Deployment environment set to %s", newEnv)
		return nil
	},
}
