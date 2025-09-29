package auth

import (
	"fmt"
	"strings"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/urfave/cli/v2"
)

var GenerateCommand = &cli.Command{
	Name:    "generate",
	Aliases: []string{"gen", "new"},
	Usage:   "Generate a new private key and optionally store it in OS keyring",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		&cli.BoolFlag{
			Name:    "store",
			Aliases: []string{"s"},
			Usage:   "Automatically store the generated key in OS keyring",
		},
	}...),
	Action: generateAction,
}

func generateAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Generate a new secp256k1 key
	privateKey, addr, err := generatePrivateKey()
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Display the key for backup
	var sb strings.Builder
	sb.WriteString("\nA new private key was generated for you.\n")
	sb.WriteString("IMPORTANT: You MUST backup this key now. It will never be shown again.\n")
	sb.WriteString("Copy it to your password manager or secure storage before continuing.\n\n")
	sb.WriteString(fmt.Sprintf("Address:     %s\n", addr))
	sb.WriteString(fmt.Sprintf("Private key: %s\n", privateKey))
	sb.WriteString("\n")
	sb.WriteString("When finished backing up, press 'q' to exit this view.\n\n")

	displayed, err := showPrivateKey(sb.String())
	if err != nil {
		return fmt.Errorf("failed to display sensitive content: %w", err)
	}
	if !displayed {
		fmt.Println("\nYou chose not to display the key.")
		fmt.Println("The key was generated but not shown for security reasons.")
		if !cCtx.Bool("store") {
			fmt.Println("Use the --store flag if you want to generate and store without displaying.")
			return nil
		}
	}

	// Determine if we should store the key
	shouldStore := cCtx.Bool("store")
	if !shouldStore && displayed {
		confirmed, err := output.ConfirmWithDefault("Store this key in your OS keyring?", true)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		shouldStore = confirmed
	}

	if shouldStore {
		// Determine the key name
		keyName, err := getAuthKeyName(cCtx)
		if err != nil {
			return fmt.Errorf("failed to determine key name: %w", err)
		}

		// Check if key already exists
		if _, err := common.GetPrivateKey(keyName); err == nil {
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
				logger.Info("Key generation completed but not stored - existing key preserved")
				return nil
			}
		}

		// Store in keyring
		if err := common.StorePrivateKey(keyName, privateKey); err != nil {
			return fmt.Errorf("failed to store private key in keyring: %w", err)
		}

		logger.Info("Key generated and stored successfully")
		logger.Info("Address: %s", addr)
		logger.Info("Stored as: %s", keyName)
	} else {
		logger.Info("Key generated successfully (not stored)")
		logger.Info("Address: %s", addr)
	}

	return nil
}
