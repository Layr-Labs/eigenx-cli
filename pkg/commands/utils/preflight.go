package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

// PreflightContext contains validated context needed for contract operations
type PreflightContext struct {
	Caller            *common.ContractCaller
	EnvironmentConfig *common.EnvironmentConfig
	Client            *ethclient.Client
	PrivateKey        string
}

// DoPreflightChecks performs early validation of authentication and network connectivity
// This should be called at the beginning of any command that requires contract interaction
func DoPreflightChecks(cCtx *cli.Context) (*PreflightContext, error) {
	logger := common.LoggerFromContext(cCtx)

	// 1. Get and validate private key first (fail fast)
	logger.Debug("Checking authentication...")
	privateKey, err := GetPrivateKeyOrFail(cCtx)
	if err != nil {
		return nil, err
	}

	// 2. Get environment configuration
	logger.Debug("Determining environment...")
	environmentConfig, err := GetEnvironmentConfig(cCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment config: %w", err)
	}

	// 3. Get RPC URL
	rpcURL, err := getRPCURL(cCtx, &environmentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get RPC URL: %w", err)
	}

	// 4. Test network connectivity
	logger.Debug("Testing network connectivity...")
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s RPC at %s: %w", environmentConfig.Name, rpcURL, err)
	}

	// 5. Get chain ID
	chainID, err := client.ChainID(cCtx.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from %s: %w", rpcURL, err)
	}

	// 6. Create contract caller
	contractCaller, err := common.NewContractCaller(
		privateKey,
		chainID,
		environmentConfig,
		client,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract caller: %w", err)
	}

	return &PreflightContext{
		Caller:            contractCaller,
		EnvironmentConfig: &environmentConfig,
		Client:            client,
		PrivateKey:        privateKey,
	}, nil
}

// GetPrivateKeyOrFail gets the private key from flag, environment, or keyring, failing with clear instructions if not found
func GetPrivateKeyOrFail(cCtx *cli.Context) (string, error) {
	// Check flag first
	if privateKey := cCtx.String(common.PrivateKeyFlag.Name); privateKey != "" {
		// Validate the key format
		if err := common.ValidatePrivateKey(privateKey); err != nil {
			return "", fmt.Errorf("invalid private key format: %w", err)
		}
		return privateKey, nil
	}

	// Check environment variable
	if privateKey := os.Getenv("PRIVATE_KEY"); privateKey != "" {
		// Validate the key format
		if err := common.ValidatePrivateKey(privateKey); err != nil {
			return "", fmt.Errorf("invalid private key in PRIVATE_KEY environment variable: %w", err)
		}
		return privateKey, nil
	}

	// Check keyring - try current environment first, then default
	if environmentConfig, err := GetEnvironmentConfig(cCtx); err == nil {
		if privateKey, err := common.GetPrivateKey(environmentConfig.Name); err == nil {
			// Validate the key format
			if err := common.ValidatePrivateKey(privateKey); err != nil {
				return "", fmt.Errorf("invalid private key in keyring for %s: %w", environmentConfig.Name, err)
			}
			return privateKey, nil
		}
	}

	// Try default
	if privateKey, err := common.GetPrivateKey("default"); err == nil {
		// Validate the key format
		if err := common.ValidatePrivateKey(privateKey); err != nil {
			return "", fmt.Errorf("invalid private key in keyring for default: %w", err)
		}
		return privateKey, nil
	}

	// Provide clear instructions on how to provide the key
	return "", fmt.Errorf(`private key required. Please provide it via:
  • Keyring: eigenx auth login
  • Flag: --private-key YOUR_KEY
  • Environment: export PRIVATE_KEY=YOUR_KEY`)
}

// GetDeveloperAddress gets developer address from private key
func GetDeveloperAddress(cCtx *cli.Context) (ethcommon.Address, error) {
	privateKey, err := GetPrivateKeyOrFail(cCtx)
	if err != nil {
		return ethcommon.Address{}, err
	}

	privateKey = strings.TrimPrefix(privateKey, "0x")
	key, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to parse private key: %w", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)

	return addr, nil
}
