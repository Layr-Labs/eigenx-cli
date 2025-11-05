package billing

import (
	"fmt"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/pkg/browser"
	"github.com/urfave/cli/v2"
)

var SubscribeCommand = &cli.Command{
	Name:  "subscribe",
	Usage: "Subscribe to start deploying apps",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
	}...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)
		environmentConfig, err := utils.GetEnvironmentConfig(cCtx)
		if err != nil {
			return fmt.Errorf("failed to get environment config: %w", err)
		}
		envName := environmentConfig.Name

		// Check authentication early to provide clear error message
		if _, err := utils.GetPrivateKeyOrFail(cCtx); err != nil {
			return err
		}

		// Check if already subscribed
		client, err := utils.NewUserApiClient(cCtx)
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		subscription, err := client.GetUserSubscription(cCtx)
		if err != nil {
			return fmt.Errorf("failed to check subscription status: %w", err)
		}

		if isSubscriptionActive(subscription.Status) {
			logger.Info("You're already subscribed to %s. Run 'eigenx billing status' for details.", envName)
			return nil
		}

		// Handle payment issues - direct to portal instead of creating new subscription
		if subscription.Status == utils.StatusPastDue || subscription.Status == utils.StatusUnpaid {
			logger.Info("You already have a subscription on %s, but it has a payment issue.", envName)
			logger.Info("Please update your payment method to restore access.")

			if subscription.PortalURL != nil && *subscription.PortalURL != "" {
				logger.Info("\nUpdate payment method:")
				logger.Info("  %s", *subscription.PortalURL)
			}

			return nil
		}

		// Create checkout session
		logger.Info("Creating checkout session for %s...", envName)
		session, err := client.CreateCheckoutSession(cCtx)
		if err != nil {
			return fmt.Errorf("failed to create checkout session: %w", err)
		}

		// Open checkout URL in browser
		logger.Info("Opening payment page in your browser...")
		if err := browser.OpenURL(session.CheckoutURL); err != nil {
			logger.Warn("Failed to open browser automatically: %v", err)
			logger.Info("\nPlease open this URL in your browser:")
			logger.Info(session.CheckoutURL)
		}

		// Poll for subscription activation
		logger.Info("\nWaiting for payment completion...")
		timeout := time.After(5 * time.Minute)
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return fmt.Errorf("payment confirmation timed out after 5 minutes. If you completed payment, run 'eigenx billing status' to check status")
			case <-ticker.C:
				subscription, err := client.GetUserSubscription(cCtx)
				if err != nil {
					logger.Debug("Failed to check subscription status: %v", err)
					continue
				}

				if isSubscriptionActive(subscription.Status) {
					logger.Info("\n✓ Subscription activated successfully for %s!", envName)
					logger.Info("\nYou now have access to deploy 1 app on %s", envName)
					logger.Info("\nStart deploying with: eigenx app deploy")
					return nil
				}
			}
		}
	},
}
