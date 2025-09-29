package utils

import (
	"context"
	"fmt"
	"io/fs"

	project "github.com/Layr-Labs/eigenx-cli"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

func GetEnvironmentConfig(cCtx *cli.Context) (common.EnvironmentConfig, error) {
	// 1. Explicit environment flag takes precedence
	if environment := cCtx.String(common.EnvironmentFlag.Name); environment != "" {
		return getEnvironmentByName(environment)
	}

	// 2. Try to detect from RPC URL's chain ID
	if rpcURL := cCtx.String(common.RpcUrlFlag.Name); rpcURL != "" {
		if environment, err := detectEnvironmentFromRPC(cCtx.Context, rpcURL); err == nil {
			return getEnvironmentByName(environment)
		}
		// If RPC detection fails, continue to default
	}

	// 3. Check user's preferred default environment from GlobalConfig
	if defaultEnv, err := common.GetDefaultEnvironment(); err == nil && defaultEnv != "" {
		return getEnvironmentByName(defaultEnv)
	}

	// 4. Use fallback environment
	return getEnvironmentByName(common.FallbackEnvironment)
}

// GetEnvironmentDescription returns a human-readable description for an environment
func GetEnvironmentDescription(name, fallback string, withPrefix bool) string {
	var description string
	switch name {
	case "sepolia":
		description = "Ethereum Sepolia testnet"
	case "mainnet-alpha":
		description = "Ethereum mainnet (⚠️  real funds at risk)"
	default:
		description = fallback
	}

	if withPrefix {
		return "- " + description
	}
	return description
}

// getEnvironmentByName retrieves an environment config by name
func getEnvironmentByName(name string) (common.EnvironmentConfig, error) {
	config, exists := common.EnvironmentConfigs[name]
	if !exists {
		return common.EnvironmentConfig{}, fmt.Errorf("unknown environment: %s", name)
	}
	return config, nil
}

// detectEnvironmentFromRPC connects to an RPC endpoint and detects the environment from chain ID
func detectEnvironmentFromRPC(ctx context.Context, rpcURL string) (string, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return "", fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get chain ID: %w", err)
	}

	environment, ok := common.DefaultEnvironmentForChainID[chainID.Uint64()]
	if !ok {
		return "", fmt.Errorf("no default environment for chain ID %s", chainID.String())
	}

	return environment, nil
}

func getKMSKeysForEnvironment(environment string) (encryptionKey []byte, signingKey []byte, err error) {
	encryptionPath := fmt.Sprintf("keys/%s/%s/kms-encryption-public-key.pem", environment, common.Build)
	signingPath := fmt.Sprintf("keys/%s/%s/kms-signing-public-key.pem", environment, common.Build)

	encryptionKey, err = fs.ReadFile(project.KeysFS, encryptionPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read encryption key for environment %s: %w", environment, err)
	}

	signingKey, err = fs.ReadFile(project.KeysFS, signingPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read signing key for environment %s: %w", environment, err)
	}

	return encryptionKey, signingKey, nil
}
