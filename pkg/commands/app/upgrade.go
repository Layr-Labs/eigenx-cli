package app

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var UpgradeCommand = &cli.Command{
	Name:      "upgrade",
	Usage:     "Upgrade existing deployment",
	ArgsUsage: "<app-id|name> <image_ref>",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		common.PrivateKeyFlag,
		common.EnvFlag,
		common.FileFlag,
		common.LogVisibilityFlag,
	}...),
	Action: upgradeAction,
}

func upgradeAction(cCtx *cli.Context) error {
	// 1. Do preflight checks (auth, network, etc.) first
	preflightCtx, err := utils.DoPreflightChecks(cCtx)
	if err != nil {
		return err
	}

	// 2. Check if docker is running, else try to start it
	err = common.EnsureDockerIsRunning(cCtx)
	if err != nil {
		return err
	}

	// 3. Get app ID from args or interactive selection
	appID, err := utils.GetAppIDInteractive(cCtx, 0, "upgrade")
	if err != nil {
		return fmt.Errorf("failed to get app id: %w", err)
	}

	// 4. Check for Dockerfile before asking for image reference
	dockerfilePath, err := utils.GetDockerfileInteractive(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get dockerfile path: %w", err)
	}
	buildFromDockerfile := dockerfilePath != ""

	// 5. Get image reference (context-aware based on Dockerfile decision)
	imageRef, err := utils.GetImageReferenceInteractive(cCtx, 1, buildFromDockerfile)
	if err != nil {
		return fmt.Errorf("failed to get image reference: %w", err)
	}

	// 6. Get environment file configuration
	envFilePath, err := utils.GetEnvFileInteractive(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get env file path: %w", err)
	}

	// 7. Get log settings from flags or interactive prompt
	logRedirect, publicLogs, err := utils.GetLogSettingsInteractive(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get log settings: %w", err)
	}

	// 8. Prepare the release (includes build/push if needed, with automatic retry on permission errors)
	release, imageRef, err := utils.PrepareReleaseFromContext(cCtx, preflightCtx.EnvironmentConfig, appID, dockerfilePath, imageRef, envFilePath, logRedirect, 3)
	if err != nil {
		return err
	}

	// 9. Check current permission state and determine if change is needed
	currentlyPublic, err := utils.CheckAppLogPermission(cCtx, appID)
	if err != nil {
		return fmt.Errorf("failed to check current permission state: %w", err)
	}

	needsPermissionChange := currentlyPublic != publicLogs

	// 10. Upgrade the app
	err = preflightCtx.Caller.UpgradeApp(cCtx.Context, appID, release, publicLogs, needsPermissionChange, imageRef)
	if err != nil {
		return fmt.Errorf("failed to upgrade app: %w", err)
	}

	// 11. Watch until app is running
	return utils.WatchUntilRunning(cCtx, appID, common.AppStatusUpgrading)
}
