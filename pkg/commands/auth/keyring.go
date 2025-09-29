package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// ListStoredKeys returns all stored key environments
func ListStoredKeys() (map[string]string, error) {
	result := make(map[string]string)

	// go-keyring doesn't provide a list function
	// Instead, we check all configured environments
	for envName := range common.EnvironmentConfigs {
		if privateKey, err := common.GetPrivateKey(envName); err == nil {
			// Validate the key and get address
			if addr, err := common.GetAddressFromPrivateKey(privateKey); err == nil {
				result[envName] = addr
			}
		}
	}

	return result, nil
}

// GetPrivateKeyWithSource tries to get private key from multiple sources and returns the source
func GetPrivateKeyWithSource(cCtx *cli.Context) (string, string, error) {
	// 1. Check flag first
	if privateKey := cCtx.String(common.PrivateKeyFlag.Name); privateKey != "" {
		return privateKey, "command flag", nil
	}

	// 2. Check environment variable
	if privateKey := os.Getenv("PRIVATE_KEY"); privateKey != "" {
		return privateKey, "environment variable", nil
	}

	// 3. Try current environment
	var keyringErrors []error
	if environmentConfig, err := utils.GetEnvironmentConfig(cCtx); err == nil {
		if privateKey, err := common.GetPrivateKey(environmentConfig.Name); err == nil {
			return privateKey, fmt.Sprintf("stored credentials (%s)", environmentConfig.Name), nil
		} else if !errors.Is(err, common.ErrKeyNotFound) {
			keyringErrors = append(keyringErrors, fmt.Errorf("keyring error for %s: %w", environmentConfig.Name, err))
		}
	}

	// If we have keyring errors (not just "key not found"), include them for debugging
	baseMsg := `Private key required. Please provide it via:
  • Keyring: eigenx auth login
  • Flag: --private-key YOUR_KEY
  • Environment: export PRIVATE_KEY=YOUR_KEY`

	if len(keyringErrors) > 0 {
		return "", "", fmt.Errorf("%s\n\nKeyring issues detected:\n%v", baseMsg, keyringErrors)
	}

	return "", "", fmt.Errorf("%s", baseMsg)
}
