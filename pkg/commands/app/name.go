package app

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var NameCommand = &cli.Command{
	Name:      "name",
	Usage:     "Set, change, or remove a friendly name for an app",
	ArgsUsage: "<app-id|current-name> [new-name]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		&cli.BoolFlag{
			Name:    "delete",
			Aliases: []string{"d"},
			Usage:   "Delete the app name",
		},
	}...),
	Action: nameAction,
}

func nameAction(cCtx *cli.Context) error {
	argCount := cCtx.Args().Len()
	if argCount < 1 {
		return fmt.Errorf("please provide app ID or current name")
	}

	appIDOrName := cCtx.Args().Get(0)

	// Handle delete flag or new name
	var newName string
	if cCtx.Bool("delete") {
		if argCount > 1 {
			return fmt.Errorf("cannot specify new name when using --delete flag")
		}
		newName = ""
	} else {
		if argCount < 2 {
			return fmt.Errorf("please provide new name (or use --delete to remove name)")
		}
		newName = cCtx.Args().Get(1)

		// Validate the friendly name
		if err := common.ValidateAppName(newName); err != nil {
			return fmt.Errorf("invalid app name: %w", err)
		}
	}

	// Get environment config for context
	environmentConfig, err := utils.GetEnvironmentConfig(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get environment config: %w", err)
	}

	if err := common.SetAppName(environmentConfig.Name, appIDOrName, newName); err != nil {
		return fmt.Errorf("failed to set app name: %w", err)
	}

	logger := common.LoggerFromContext(cCtx)
	if newName == "" {
		logger.Info("App name removed successfully")
	} else {
		logger.Info("App name set to: %s", newName)
	}

	return nil
}
