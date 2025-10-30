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

		// Show historical details if canceled
		if subscription.Status == utils.StatusCanceled {
			logger.Info("\nYour subscription has been canceled.")
			if subscription.CurrentPeriodEnd != nil && *subscription.CurrentPeriodEnd > 0 {
				endDate := time.Unix(*subscription.CurrentPeriodEnd, 0)
				logger.Info("Access ended on %s.", endDate.Format("January 2, 2006"))
			}
			logger.Info("Run 'eigenx billing subscribe' to resubscribe.")

			// Show portal link for viewing invoices
			if subscription.PortalURL != nil && *subscription.PortalURL != "" {
				logger.Info("\nView invoices and billing history:")
				logger.Info("  %s", *subscription.PortalURL)
			}

			return nil
		}

		// Handle payment issues (past_due, unpaid) - show portal to fix
		if subscription.Status == utils.StatusPastDue || subscription.Status == utils.StatusUnpaid {
			logger.Warn("\nYour subscription has a payment issue.")
			logger.Info("Please update your payment method to restore access.")

			if subscription.PortalURL != nil && *subscription.PortalURL != "" {
				logger.Info("\nUpdate payment method:")
				logger.Info("  %s", *subscription.PortalURL)
			}

			return nil
		}

		// Handle all other inactive statuses (incomplete, expired, paused, inactive)
		if !isSubscriptionActive(subscription.Status) {
			logger.Info("\nYou don't have an active subscription on %s.", envName)
			logger.Info("Run 'eigenx billing subscribe' to get started.")
			return nil
		}

		// Plan details
		if subscription.PlanPrice != nil && subscription.Currency != nil && *subscription.PlanPrice > 0 {
			logger.Info("Plan: $%.2f/%s", *subscription.PlanPrice, *subscription.Currency)
		} else {
			logger.Info("Plan: Standard")
		}
		logger.Info("  - 1 app deployed on %s", envName)
		logger.Info("")

		// Current environment usage
		logger.Info("Usage:")

		// Get contract caller for current environment
		caller, err := utils.GetContractCaller(cCtx)
		if err != nil {
			logger.Warn("  Unable to fetch usage: %v", err)
			logger.Info("")
		} else if developerAddr, err := utils.GetDeveloperAddress(cCtx); err != nil {
			logger.Warn("  Unable to fetch usage: failed to get developer address")
			logger.Info("")
		} else if count, err := caller.GetActiveAppCount(cCtx.Context, developerAddr); err != nil {
			logger.Warn("  Unable to fetch usage: %v", err)
			logger.Info("")
		} else {
			logger.Info("  %d / 1 apps deployed on %s", count, envName)
			logger.Info("")
		}

		// Billing information
		logger.Info("Billing:")

		// Next billing date and amount
		if subscription.UpcomingInvoice != nil && subscription.UpcomingInvoice.Date > 0 {
			nextBilling := time.Unix(subscription.UpcomingInvoice.Date, 0)
			logger.Info("  Next charge: $%.2f on %s",
				subscription.UpcomingInvoice.Amount,
				nextBilling.Format("January 2, 2006"))
		} else if subscription.CurrentPeriodEnd != nil && *subscription.CurrentPeriodEnd > 0 {
			nextBilling := time.Unix(*subscription.CurrentPeriodEnd, 0)
			logger.Info("  Next billing: %s", nextBilling.Format("January 2, 2006"))
		}

		// Cancellation status
		if subscription.CancelAtPeriodEnd != nil && *subscription.CancelAtPeriodEnd {
			if subscription.CurrentPeriodEnd != nil {
				endDate := time.Unix(*subscription.CurrentPeriodEnd, 0)
				logger.Info("  ⚠ Scheduled for cancellation on %s", endDate.Format("January 2, 2006"))
			}
		}

		logger.Info("")

		// Billing portal link
		if subscription.PortalURL != nil && *subscription.PortalURL != "" {
			logger.Info("Manage billing online:")
			logger.Info("  %s", *subscription.PortalURL)
			logger.Info("")
		}
		return nil
	},
}

// isSubscriptionActive returns true if the subscription status allows deploying apps
func isSubscriptionActive(status utils.SubscriptionStatus) bool {
	return status == utils.StatusActive || status == utils.StatusTrialing
}

func formatStatus(status utils.SubscriptionStatus) string {
	switch status {
	case utils.StatusActive:
		return "✓ Active"
	case utils.StatusTrialing:
		return "✓ Trial"
	case utils.StatusPastDue:
		return "⚠ Past Due"
	case utils.StatusCanceled:
		return "✗ Canceled"
	case utils.StatusInactive:
		return "✗ Inactive"
	case utils.StatusIncomplete:
		return "⚠ Incomplete"
	case utils.StatusIncompleteExpired:
		return "✗ Expired"
	case utils.StatusUnpaid:
		return "⚠ Unpaid"
	case utils.StatusPaused:
		return "⚠ Paused"
	default:
		return string(status)
	}
}
