package billing

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/urfave/cli/v2"
)

var CancelCommand = &cli.Command{
	Name:  "cancel",
	Usage: "Cancel subscription",
	Action: func(cCtx *cli.Context) error {
		ctx := cCtx.Context
		logger := common.LoggerFromContext(cCtx)

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

		// Extract results into separate maps
		networkAppCounts := make(map[string]uint32)
		networkCallers := make(map[string]*common.ContractCaller)
		totalActiveApps := uint32(0)

		for env, info := range results {
			if info.Count > 0 {
				networkAppCounts[env] = info.Count
				totalActiveApps += info.Count
			}
			if info.Caller != nil {
				networkCallers[env] = info.Caller
			}
		}

		// If apps exist, show per-network breakdown and get confirmation
		if totalActiveApps > 0 {
			logger.Info("You have active apps that will be suspended:")
			for env, count := range networkAppCounts {
				if count > 0 {
					displayName := utils.GetEnvironmentDescription(env, env, false)
					logger.Info("  • %s: %d app(s)", displayName, count)
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
			for env, count := range networkAppCounts {
				if count == 0 {
					continue
				}

				caller := networkCallers[env]
				logger.Info("Suspending apps on %s...", env)

				// Get all apps for this developer on this network
				apps, _, err := caller.GetAppsByCreator(ctx, developerAddr, 0, 1_000)
				if err != nil {
					return fmt.Errorf("failed to get apps for %s: %w", env, err)
				}

				// Call suspend with all apps
				// Note: The contract will filter to only active apps (STARTED/STOPPED)
				err = caller.Suspend(ctx, developerAddr, apps)
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
