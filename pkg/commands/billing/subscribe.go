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
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		// Check if already subscribed
		client, err := utils.NewBillingApiClient(cCtx)
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		subscription, err := client.GetUserSubscription(cCtx)
		if err != nil {
			return fmt.Errorf("failed to check subscription status: %w", err)
		}

		if subscription.Status == "active" {
			logger.Info("You're already subscribed. Run 'eigenx billing status' for details.")
			return nil
		}

		// Create checkout session
		logger.Info("Creating checkout session...")
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

				if subscription.Status == "active" {
					logger.Info("\n✓ Subscription activated successfully!")
					logger.Info("\nYou now have access to:")
					logger.Info("  • 1 app on testnet (sepolia)")
					logger.Info("  • 1 app on mainnet (mainnet-alpha)")
					logger.Info("\nStart deploying with: eigenx app deploy")
					return nil
				}
			}
		}
	},
}
