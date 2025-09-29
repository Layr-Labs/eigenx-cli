package auth

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/urfave/cli/v2"
)

var LoginCommand = &cli.Command{
	Name:  "login",
	Usage: "Store an existing private key in OS keyring",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
	}...),
	Action: loginAction,
}

func loginAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Determine the key name
	keyName, err := getAuthKeyName(cCtx)
	if err != nil {
		return fmt.Errorf("failed to determine key name: %w", err)
	}

	// Check if key already exists
	if _, err := common.GetPrivateKey(keyName); err == nil {
		// Key exists, ask for confirmation to overwrite with strong warning
		fmt.Printf("\n⚠️  WARNING: A private key for '%s' already exists in your keyring!\n", keyName)
		fmt.Println("⚠️  If you continue, the existing key will be PERMANENTLY REPLACED and CANNOT BE RECOVERED.")
		fmt.Println("⚠️  Unless you have a backup of the existing key, it will be LOST FOREVER!")
		fmt.Println("⚠️  This could result in permanent loss of access to funds or applications.")
		fmt.Println()

		confirmed, err := output.Confirm("Are you absolutely sure you want to overwrite the existing key?")
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			logger.Info("Login cancelled - existing key preserved")
			return nil
		}
	}

	// Prompt for private key with hidden input
	fmt.Println("Enter your private key. Input will be hidden for security.")
	privateKey, err := output.InputHiddenString(
		"Private key:",
		"Your private key for signing transactions (input will be hidden)",
		common.ValidatePrivateKey,
	)
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}

	// Get address from private key for display
	address, err := common.GetAddressFromPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to validate private key: %w", err)
	}

	// Store in keyring
	if err := common.StorePrivateKey(keyName, privateKey); err != nil {
		return fmt.Errorf("failed to store private key in keyring: %w", err)
	}

	logger.Info("Successfully logged in")
	logger.Info("Address: %s", address)
	logger.Info("Stored as: %s", keyName)

	return nil
}
