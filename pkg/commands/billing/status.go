package billing

import (
	"fmt"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var StatusCommand = &cli.Command{
	Name:  "status",
	Usage: "Show subscription status and usage",
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		// Get API client
		apiClient, err := utils.NewBillingApiClient(cCtx)
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		// Get subscription details
		subscription, err := apiClient.GetUserSubscription(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get subscription details: %w", err)
		}

		// Display subscription information
		fmt.Println()

		// Subscription status
		statusDisplay := formatStatus(subscription.Status)
		logger.Info("Status: %s", statusDisplay)

		if subscription.Status == "inactive" {
			logger.Info("\nYou don't have an active subscription.")
			logger.Info("Run 'eigenx billing subscribe' to get started.")
			return nil
		}

		// Plan details
		if subscription.PlanPrice > 0 {
			logger.Info("Plan: $%.2f/%s", subscription.PlanPrice, subscription.Currency)
		} else {
			logger.Info("Plan: Standard")
		}
		logger.Info("  • 1 app per network")
		logger.Info("")

		// Network-specific usage
		logger.Info("Network capacity:")

		// Get developer address
		developerAddr, err := utils.GetDeveloperAddress(cCtx)
		if err != nil {
			logger.Warn("  Unable to fetch usage: failed to get developer address")
		} else {
			// Get private key for contract calls
			privateKey, err := utils.GetPrivateKeyOrFail(cCtx)
			if err != nil {
				logger.Warn("  Unable to fetch usage: failed to get private key")
			} else {
				ctx := cCtx.Context

				// Query all networks for active app counts
				results, err := GetActiveAppCountsForAllNetworks(ctx, cCtx, developerAddr, privateKey, false)
				if err != nil {
					logger.Warn("  Unable to fetch usage: %v", err)
				} else {
					// Display results for each network
					for env, info := range results {
						displayName := utils.GetEnvironmentDescription(env, env, false)
						logger.Info("  %s: %d / 1 apps deployed", displayName, info.Count)
					}
				}
			}
		}

		logger.Info("")

		// Billing information
		logger.Info("Billing:")

		// Next billing date and amount
		if subscription.UpcomingInvoice != nil {
			nextBilling := time.Unix(subscription.UpcomingInvoice.Date, 0)
			logger.Info("  Next charge: $%.2f on %s",
				subscription.UpcomingInvoice.Amount,
				nextBilling.Format("January 2, 2006"))
		} else if subscription.CurrentPeriodEnd > 0 {
			nextBilling := time.Unix(subscription.CurrentPeriodEnd, 0)
			logger.Info("  Next billing: %s", nextBilling.Format("January 2, 2006"))
		}

		// Cancellation status
		if subscription.CancelAtPeriodEnd {
			endDate := time.Unix(subscription.CurrentPeriodEnd, 0)
			logger.Info("  ⚠ Scheduled for cancellation on %s", endDate.Format("January 2, 2006"))
		}

		logger.Info("")

		// Billing portal link
		if subscription.PortalURL != "" {
			logger.Info("Manage billing online:")
			logger.Info("  %s", subscription.PortalURL)
			logger.Info("")
		}
		return nil
	},
}

func formatStatus(status string) string {
	switch status {
	case "active":
		return "✓ Active"
	case "past_due":
		return "⚠ Past Due"
	case "canceled":
		return "✗ Canceled"
	case "inactive":
		return "✗ Inactive"
	default:
		return status
	}
}
