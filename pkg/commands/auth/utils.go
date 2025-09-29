package auth

import (
	"encoding/hex"
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
)

// getAuthKeyName determines the keyring storage name based on environment
func getAuthKeyName(cCtx *cli.Context) (string, error) {
	// 1. Try to detect current environment
	if environmentConfig, err := utils.GetEnvironmentConfig(cCtx); err == nil {
		return environmentConfig.Name, nil
	}

	// 2. Default to default environment
	return common.FallbackEnvironment, nil
}

// generatePrivateKey creates a new secp256k1 private key and derives its address.
// Returns hex-encoded private key with 0x prefix and the checksummed address.
func generatePrivateKey() (string, string, error) {
	// Generate secp256k1 key
	ecdsaKey, err := crypto.GenerateKey()
	if err != nil {
		return "", "", err
	}
	// Encode private key to hex
	privBytes := crypto.FromECDSA(ecdsaKey)
	privHex := "0x" + hex.EncodeToString(privBytes)

	// Derive address
	address := crypto.PubkeyToAddress(ecdsaKey.PublicKey).Hex()
	return privHex, address, nil
}

// showPrivateKey attempts to display private key content via a pager (less/more).
// Returns displayed=true if content was shown; if no pager is available, prompts user to
// abort (recommended) or print-and-clear. displayed=false indicates user aborted display.
func showPrivateKey(content string) (displayed bool, err error) {
	if pager := output.DetectPager(); pager != "" {
		if err := output.RunPager(pager, content); err != nil {
			return false, fmt.Errorf("pager error: %w", err)
		}
		return true, nil
	}

	// No pager available: give guarded choices using survey for cross-platform input
	fmt.Println("\nNo pager (less/more) found on PATH.")
	fmt.Println("For security, avoid printing private keys to the terminal.")

	choice, err := output.SelectString(
		"Choose an option:",
		[]string{
			"Abort (recommended)",
			"Print and clear screen",
		},
	)
	if err != nil {
		return false, fmt.Errorf("failed to get selection: %w", err)
	}

	if choice == "Print and clear screen" {
		fmt.Println(content)
		// Use survey for cross-platform Enter key handling
		_, _ = output.InputString(
			"Press Enter after you have securely saved the key. The screen will be cleared...",
			"",
			"",
			nil,
		)
		output.ClearTerminal()
		return true, nil
	}
	// default abort
	return false, nil
}
