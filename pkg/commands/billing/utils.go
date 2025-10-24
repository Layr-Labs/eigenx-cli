package billing

import (
	"context"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

// NetworkAppInfo contains app count and optionally the ContractCaller for a network
type NetworkAppInfo struct {
	Count  uint32
	Caller *common.ContractCaller // nil if keepCallers=false
}

// GetActiveAppCountsForAllNetworks queries active app counts across all configured environments.
// It iterates over all environments in common.EnvironmentConfigs and returns a map of results.
// If keepCallers is true, the ContractCaller is kept in the result (for later use like in cancel command).
// If keepCallers is false, clients are closed immediately after getting the count (for status display).
// Individual environment failures are logged as warnings and don't fail the entire operation.
func GetActiveAppCountsForAllNetworks(
	ctx context.Context,
	cCtx *cli.Context,
	developerAddr ethcommon.Address,
	privateKeyHex string,
	keepCallers bool,
) (map[string]NetworkAppInfo, error) {
	logger := common.LoggerFromContext(cCtx)
	results := make(map[string]NetworkAppInfo)

	for env, envConfig := range common.EnvironmentConfigs {
		// Get RPC URL for this environment
		rpcURL := envConfig.DefaultRPCURL
		if customRPC := cCtx.String(common.RpcUrlFlag.Name); customRPC != "" {
			rpcURL = customRPC
		}

		// Connect to RPC
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			logger.Warn("Failed to connect to %s: %v", env, err)
			continue
		}

		// Get chain ID
		chainID, err := client.ChainID(ctx)
		if err != nil {
			client.Close()
			logger.Warn("Failed to get chain ID for %s: %v", env, err)
			continue
		}

		// Create contract caller for this environment
		caller, err := common.NewContractCaller(privateKeyHex, chainID, envConfig, client, logger)
		if err != nil {
			client.Close()
			logger.Warn("Failed to create contract caller for %s: %v", env, err)
			continue
		}

		// Get active app count for this network
		count, err := caller.GetActiveAppCount(ctx, developerAddr)
		if err != nil {
			if !keepCallers {
				client.Close()
			}
			logger.Warn("Failed to get app count for %s: %v", env, err)
			continue
		}

		// Store result
		info := NetworkAppInfo{
			Count: count,
		}
		if keepCallers {
			info.Caller = caller
		} else {
			client.Close()
		}
		results[env] = info
	}

	return results, nil
}
