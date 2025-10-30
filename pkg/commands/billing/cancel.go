package billing

import (
	"context"
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var CancelCommand = &cli.Command{
	Name:  "cancel",
	Usage: "Cancel subscription",
	Action: func(cCtx *cli.Context) error {
		ctx := cCtx.Context
		logger := common.LoggerFromContext(cCtx)
		environmentConfig, err := utils.GetEnvironmentConfig(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get environment config: %w", err)
		}
		envName := environmentConfig.Name

		// Get API client
		apiClient, err := utils.NewUserApiClient(cCtx)
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		// Check current subscription status
		subscription, err := apiClient.GetUserSubscription(cCtx)
		if err != nil {
			return fmt.Errorf("failed to check subscription status: %w", err)
		}

		if !isSubscriptionActive(subscription.Status) {
			logger.Info("You don't have an active subscription on %s.", envName)
			return nil
		}

		// Get contract caller for current environment
		caller, err := utils.GetContractCaller(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get contract caller: %w", err)
		}

		// Get developer address
		developerAddr, err := utils.GetDeveloperAddress(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get developer address: %w", err)
		}

		// Check active apps on current environment
		activeAppCount, err := caller.GetActiveAppCount(ctx, developerAddr)
		if err != nil {
			return fmt.Errorf("failed to get active app count: %w", err)
		}

		// If apps exist, show warning and get confirmation
		if activeAppCount > 0 {
			logger.Info("You have %d active app(s) on %s that will be suspended.", activeAppCount, envName)
			logger.Info("")

			confirmed, err := output.Confirm("Continue?")
			if err != nil {
				return fmt.Errorf("failed to get confirmation: %w", err)
			}

			if !confirmed {
				logger.Info("Cancellation aborted.")
				return nil
			}

			// Get only active apps for this developer
			activeApps, err := getActiveAppsByCreator(ctx, caller, developerAddr)
			if err != nil {
				return fmt.Errorf("failed to get active apps: %w", err)
			}

			if len(activeApps) > 0 {
				logger.Info("Suspending apps...")

				// Suspend only active apps
				err = caller.Suspend(ctx, developerAddr, activeApps)
				if err != nil {
					return fmt.Errorf("failed to suspend apps: %w", err)
				}

				logger.Info("✓ Apps suspended")
			}
		} else {
			// No active apps, just confirm cancellation
			logger.Warn("Canceling your subscription on %s will prevent you from deploying new apps.", envName)
			confirmed, err := output.Confirm("Are you sure you want to cancel your subscription?")
			if err != nil {
				return fmt.Errorf("failed to get confirmation: %w", err)
			}

			if !confirmed {
				logger.Info("Cancellation aborted.")
				return nil
			}
		}

		// Cancel subscription via API
		logger.Info("Canceling subscription...")
		if err := apiClient.CancelSubscription(cCtx); err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}

		logger.Info("\n✓ Subscription canceled successfully for %s.", envName)
		return nil
	},
}

// getActiveAppsByCreator retrieves only the active apps (STARTED/STOPPED) for a creator
func getActiveAppsByCreator(ctx context.Context, caller *common.ContractCaller, creator ethcommon.Address) ([]ethcommon.Address, error) {
	// Get all apps for this creator on this network
	allApps, appConfigs, err := caller.GetAppsByCreator(ctx, creator, 0, 1_000)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps by creator: %w", err)
	}

	// Filter to only active apps (STARTED/STOPPED)
	var activeApps []ethcommon.Address
	for i, app := range allApps {
		config := appConfigs[i]
		status := common.AppStatus(config.Status)
		if status == common.ContractAppStatusStarted || status == common.ContractAppStatusStopped {
			activeApps = append(activeApps, app)
		}
	}
	return activeApps, nil
}
