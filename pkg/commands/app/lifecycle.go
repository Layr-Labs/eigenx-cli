package app

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var StartCommand = &cli.Command{
	Name:      "start",
	Usage:     "Start stopped app (start GCP instance)",
	ArgsUsage: "[app-id|name]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		common.PrivateKeyFlag,
	}...),
	Action: startAction,
}

var StopCommand = &cli.Command{
	Name:      "stop",
	Usage:     "Stop running app (stop GCP instance)",
	ArgsUsage: "[app-id|name]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		common.PrivateKeyFlag,
	}...),
	Action: stopAction,
}

var TerminateCommand = &cli.Command{
	Name:      "terminate",
	Usage:     "Terminate app (terminate GCP instance) permanently",
	ArgsUsage: "[app-id|name]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		common.PrivateKeyFlag,
		common.ForceFlagWithUsage("Force termination without confirmation"),
	}...),
	Action: terminateAction,
}

func startAction(cCtx *cli.Context) error {
	ctx := cCtx.Context
	logger := common.LoggerFromContext(cCtx)

	// Do preflight checks first
	preflightCtx, err := utils.DoPreflightChecks(cCtx)
	if err != nil {
		return err
	}

	// Get app address from args or interactive selection
	appID, err := utils.GetAppIDInteractive(cCtx, 0, "start")
	if err != nil {
		return fmt.Errorf("failed to get app address: %w", err)
	}

	profileName := utils.GetAppProfileName(cCtx, appID)
	formattedApp := common.FormatAppDisplay(preflightCtx.EnvironmentConfig.Name, appID, profileName)

	// Call AppController.StartApp
	err = preflightCtx.Caller.StartApp(ctx, appID)
	if err != nil {
		infoErr := utils.GetAndPrintAppInfo(cCtx, appID)
		if infoErr != nil {
			return fmt.Errorf("failed to get app info: %w", err)
		}
		return err
	}

	logger.Info("App %s started successfully", formattedApp)

	return utils.WatchUntilTransitionComplete(cCtx, appID, common.AppStatusResuming)
}

func stopAction(cCtx *cli.Context) error {
	ctx := cCtx.Context
	logger := common.LoggerFromContext(cCtx)

	// Do preflight checks first
	preflightCtx, err := utils.DoPreflightChecks(cCtx)
	if err != nil {
		return err
	}

	// Get app address from args or interactive selection
	appID, err := utils.GetAppIDInteractive(cCtx, 0, "stop")
	if err != nil {
		return fmt.Errorf("failed to get app address: %w", err)
	}

	profileName := utils.GetAppProfileName(cCtx, appID)
	formattedApp := common.FormatAppDisplay(preflightCtx.EnvironmentConfig.Name, appID, profileName)

	// Call AppController.StopApp
	err = preflightCtx.Caller.StopApp(ctx, appID)
	if err != nil {
		infoErr := utils.GetAndPrintAppInfo(cCtx, appID)
		if infoErr != nil {
			return fmt.Errorf("failed to get app info: %w", err)
		}
		return err
	}

	logger.Info("App %s stopped successfully", formattedApp)

	return utils.GetAndPrintAppInfo(cCtx, appID, common.AppStatusStopping)
}

func terminateAction(cCtx *cli.Context) error {
	ctx := cCtx.Context
	logger := common.LoggerFromContext(cCtx)

	// Do preflight checks first
	preflightCtx, err := utils.DoPreflightChecks(cCtx)
	if err != nil {
		return err
	}

	// Get app address from args or interactive selection
	appID, err := utils.GetAppIDInteractive(cCtx, 0, "terminate")
	if err != nil {
		return fmt.Errorf("failed to get app address: %w", err)
	}

	// Call AppController.TerminateApp
	err = preflightCtx.Caller.TerminateApp(ctx, appID, cCtx.Bool(common.ForceFlag.Name))
	if err != nil {
		infoErr := utils.GetAndPrintAppInfo(cCtx, appID)
		if infoErr != nil {
			return fmt.Errorf("failed to get app info: %w", err)
		}
		return err
	}

	profileName := utils.GetAppProfileName(cCtx, appID)
	logger.Info("App %s terminated successfully", common.FormatAppDisplay(preflightCtx.EnvironmentConfig.Name, appID, profileName))

	return utils.GetAndPrintAppInfo(cCtx, appID, common.AppStatusTerminating)
}
