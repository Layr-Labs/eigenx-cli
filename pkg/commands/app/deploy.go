package app

import (
	"crypto/rand"
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/urfave/cli/v2"
)

var DeployCommand = &cli.Command{
	Name:      "deploy",
	Usage:     "Build, push, and deploy app to TEE",
	ArgsUsage: "[image_ref]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		common.PrivateKeyFlag,
		common.EnvFlag,
		common.FileFlag,
		common.NameFlag,
		common.LogVisibilityFlag,
		common.InstanceTypeFlag,
	}...),
	Action: deployAction,
}

func deployAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// 1. Do preflight checks (auth, network, etc.) first
	preflightCtx, err := utils.DoPreflightChecks(cCtx)
	if err != nil {
		return err
	}

	// 2. Check quota availability
	if err := checkQuotaAvailable(cCtx, preflightCtx); err != nil {
		return err
	}

	// 3. Check if docker is running, else try to start it
	err = common.EnsureDockerIsRunning(cCtx)
	if err != nil {
		return err
	}

	// 4. Check for Dockerfile before asking for image reference
	dockerfilePath, err := utils.GetDockerfileInteractive(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get dockerfile path: %w", err)
	}
	buildFromDockerfile := dockerfilePath != ""

	// 5. Get image reference (context-aware based on Dockerfile decision)
	imageRef, err := utils.GetImageReferenceInteractive(cCtx, 0, buildFromDockerfile)
	if err != nil {
		return fmt.Errorf("failed to get image reference: %w", err)
	}

	// 6. Get app name upfront (before any expensive operations)
	environment := preflightCtx.EnvironmentConfig.Name
	name, err := utils.GetOrPromptAppName(cCtx, environment, imageRef)
	if err != nil {
		return fmt.Errorf("failed to get app name: %w", err)
	}

	// 7. Get environment file configuration
	envFilePath, err := utils.GetEnvFileInteractive(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get env file path: %w", err)
	}

	// 8. Get instance type selection (uses first from backend as default for new apps)
	instanceType, err := utils.GetInstanceTypeInteractive(cCtx, "")
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// 9. Get log settings from flags or interactive prompt
	logRedirect, publicLogs, err := utils.GetLogSettingsInteractive(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get log settings: %w", err)
	}

	// 10. Generate random salt
	salt := [32]byte{}
	_, err = rand.Read(salt[:])
	if err != nil {
		return fmt.Errorf("failed to generate random salt: %w", err)
	}

	// 11. Get app ID
	_, appController, err := utils.GetAppControllerBinding(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get app controller binding: %w", err)
	}
	appIDToBeDeployed, err := appController.CalculateAppId(&bind.CallOpts{Context: cCtx.Context}, preflightCtx.Caller.SelfAddress, salt)
	if err != nil {
		return fmt.Errorf("failed to get app id: %w", err)
	}

	// 12. Prepare the release (includes build/push if needed, with automatic retry on permission errors)
	release, imageRef, err := utils.PrepareReleaseFromContext(cCtx, preflightCtx.EnvironmentConfig, appIDToBeDeployed, dockerfilePath, imageRef, envFilePath, logRedirect, instanceType, 3)
	if err != nil {
		return err
	}

	// 13. Deploy the app
	appID, err := preflightCtx.Caller.DeployApp(cCtx.Context, salt, release, publicLogs, imageRef)
	if err != nil {
		return fmt.Errorf("failed to deploy app: %w", err)
	}

	// 14. Save the app name mapping
	if err := common.SetAppName(environment, appID.Hex(), name); err != nil {
		logger.Warn("Failed to save app name: %s", err.Error())
	} else {
		logger.Info("App saved with name: %s", name)
	}

	// 15. Watch until deployment completes
	return utils.WatchUntilTransitionComplete(cCtx, appID, common.AppStatusDeploying)
}

// checkQuotaAvailable verifies that the user has deployment quota available
// by checking their allowlist status on the contract
func checkQuotaAvailable(cCtx *cli.Context, preflightCtx *utils.PreflightContext) error {
	ctx := cCtx.Context

	// Check user's quota limit from contract
	maxQuota, err := preflightCtx.Caller.GetMaxActiveAppsPerUser(ctx, preflightCtx.Caller.SelfAddress)
	if err != nil {
		return fmt.Errorf("failed to get quota limit: %w", err)
	}

	// If quota is 0, user needs to subscribe
	if maxQuota == 0 {
		return fmt.Errorf("no app quota available. Run 'eigenx billing subscribe' to enable app deployment")
	}

	// Check current active app count from contract
	activeCount, err := preflightCtx.Caller.GetActiveAppCount(ctx, preflightCtx.Caller.SelfAddress)
	if err != nil {
		return fmt.Errorf("failed to get active app count: %w", err)
	}

	// Check if quota is reached
	if activeCount >= maxQuota {
		return fmt.Errorf("app quota reached for %s (%d/%d). Please contact the Eigen team at eigencloud_support@eigenlabs.org for additional capacity", preflightCtx.EnvironmentConfig.Name, activeCount, maxQuota)
	}

	return nil
}
