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

		// Get API client
		apiClient, err := utils.NewBillingApiClient(cCtx)
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		// Check current subscription status
		subscription, err := apiClient.GetUserSubscription(cCtx)
		if err != nil {
			return fmt.Errorf("failed to check subscription status: %w", err)
		}

		if subscription.Status != "active" {
			logger.Info("You don't have an active subscription.")
			return nil
		}

		// Get developer address
		developerAddr, err := utils.GetDeveloperAddress(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get developer address: %w", err)
		}

		// Check active apps across all networks
		// Get private key for contract calls
		privateKey, err := utils.GetPrivateKeyOrFail(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get private key: %w", err)
		}

		// Query all networks for active app counts
		results, err := GetActiveAppCountsForAllNetworks(ctx, cCtx, developerAddr, privateKey, true)
		if err != nil {
			return fmt.Errorf("failed to query networks: %w", err)
		}

		// Calculate total active apps
		totalActiveApps := uint32(0)
		for _, info := range results {
			totalActiveApps += info.Count
		}

		// If apps exist, show per-network breakdown and get confirmation
		if totalActiveApps > 0 {
			logger.Info("You have active apps that will be suspended:")
			for env, info := range results {
				if info.Count > 0 {
					displayName := utils.GetEnvironmentDescription(env, env, false)
					logger.Info("  • %s: %d app(s)", displayName, info.Count)
				}
			}
			logger.Info("")

			confirmed, err := output.Confirm("Continue?")
			if err != nil {
				return fmt.Errorf("failed to get confirmation: %w", err)
			}

			if !confirmed {
				logger.Info("Cancellation aborted.")
				return nil
			}

			// Suspend apps on each network that has active apps
			for env, info := range results {
				if info.Count == 0 {
					continue
				}

				caller := info.Caller
				logger.Info("Suspending apps on %s...", env)

				// Get only active apps for this developer on this network
				activeApps, err := getActiveAppsByCreator(ctx, caller, developerAddr)
				if err != nil {
					return fmt.Errorf("failed to get active apps for %s: %w", env, err)
				}

				if len(activeApps) == 0 {
					logger.Info("No active apps to suspend on %s", env)
					continue
				}

				// Suspend only active apps
				err = caller.Suspend(ctx, developerAddr, activeApps)
				if err != nil {
					return fmt.Errorf("failed to suspend apps on %s: %w", env, err)
				}

				logger.Info("✓ Apps suspended on %s", env)
			}
		} else {
			// No active apps, just confirm cancellation
			logger.Warn("Canceling your subscription will prevent you from deploying new apps.")
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

		logger.Info("\n✓ Subscription canceled successfully.")
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
