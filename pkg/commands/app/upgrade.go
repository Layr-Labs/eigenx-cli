package app

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
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
		common.InstanceTypeFlag,
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

	// 7. Get current app's instance type (best-effort, used as default for selection)
	currentInstanceType := getCurrentInstanceType(cCtx, appID)

	// 8. Get instance type selection (defaults to current app's instance type)
	instanceType, err := utils.GetInstanceTypeInteractive(cCtx, currentInstanceType)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// 9. Get log settings from flags or interactive prompt
	logRedirect, publicLogs, err := utils.GetLogSettingsInteractive(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get log settings: %w", err)
	}

	// 10. Prepare the release (includes build/push if needed, with automatic retry on permission errors)
	release, imageRef, err := utils.PrepareReleaseFromContext(cCtx, preflightCtx.EnvironmentConfig, appID, dockerfilePath, imageRef, envFilePath, logRedirect, instanceType, 3)
	if err != nil {
		return err
	}

	// 11. Check current permission state and determine if change is needed
	currentlyPublic, err := utils.CheckAppLogPermission(cCtx, appID)
	if err != nil {
		return fmt.Errorf("failed to check current permission state: %w", err)
	}

	needsPermissionChange := currentlyPublic != publicLogs

	// 12. Upgrade the app
	err = preflightCtx.Caller.UpgradeApp(cCtx.Context, appID, release, publicLogs, needsPermissionChange, imageRef)
	if err != nil {
		return fmt.Errorf("failed to upgrade app: %w", err)
	}

	// 13. Watch until upgrade completes
	return utils.WatchUntilUpgradeComplete(cCtx, appID, common.AppStatusUpgrading)
}

// getCurrentInstanceType attempts to retrieve the current instance type for an app.
// Returns empty string if unable to fetch (API unavailable, app info not ready, etc.).
// This is used as a convenience default for the upgrade flow.
func getCurrentInstanceType(cCtx *cli.Context, appID ethcommon.Address) string {
	userApiClient, err := utils.NewUserApiClient(cCtx)
	if err != nil {
		return "" // API client creation failed, skip default
	}

	infos, err := userApiClient.GetInfos(cCtx, []ethcommon.Address{appID}, 1)
	if err != nil {
		return "" // API call failed, skip default
	}

	if len(infos.Apps) == 0 {
		return "" // No app info available yet
	}

	return infos.Apps[0].MachineType
}
