package utils

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
	"github.com/Layr-Labs/eigenx-contracts/pkg/bindings/v1/AppController"
	"github.com/Layr-Labs/eigenx-contracts/pkg/bindings/v2/IPermissionController"
	"github.com/Layr-Labs/eigenx-kms/pkg/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

// GetAppID gets the app id from CLI args or auto-detects from project context. App id is the address of the app contract on L1.
func GetAppID(cCtx *cli.Context, argIndex int) (ethcommon.Address, error) {
	// Check if app_id provided as argument
	if cCtx.Args().Len() > argIndex {
		nameOrID := cCtx.Args().Get(argIndex)

		// Get environment config for context
		environmentConfig, err := GetEnvironmentConfig(cCtx)
		if err != nil {
			return ethcommon.Address{}, fmt.Errorf("failed to get environment config: %w", err)
		}

		// First try to resolve as a name from the registry
		resolvedID, err := common.ResolveAppID(environmentConfig.Name, nameOrID)
		if err == nil {
			return ethcommon.HexToAddress(resolvedID), nil
		}

		// If not a name, check if it's a valid hex address
		if ethcommon.IsHexAddress(nameOrID) {
			return ethcommon.HexToAddress(nameOrID), nil
		}

		return ethcommon.Address{}, fmt.Errorf("invalid app id or name: %s", nameOrID)
	}

	return ethcommon.Address{}, fmt.Errorf("app id or name required. Provide as argument or ensure you're in a project directory with deployment info")
}

func GetAppControllerBinding(cCtx *cli.Context) (*ethclient.Client, *AppController.AppController, error) {
	environmentConfig, err := GetEnvironmentConfig(cCtx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get environment config: %w", err)
	}

	// Get RPC URL from flag or use environment default
	rpcURL, err := getRPCURL(cCtx, &environmentConfig)
	if err != nil {
		return nil, nil, err
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RPC endpoint %s: %w", rpcURL, err)
	}

	appController, err := AppController.NewAppController(environmentConfig.AppControllerAddress, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AppController: %w", err)
	}

	return client, appController, nil
}

// GetContractCaller creates a contract caller from the CLI context
func GetContractCaller(cCtx *cli.Context) (*common.ContractCaller, error) {
	logger := common.LoggerFromContext(cCtx)

	// Get environment config - required for contract addresses
	environmentConfig, err := GetEnvironmentConfig(cCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment config: %w", err)
	}

	// Get RPC URL from flag or environment default
	rpcURL, err := getRPCURL(cCtx, &environmentConfig)
	if err != nil {
		return nil, err
	}

	if rpcURL == environmentConfig.DefaultRPCURL {
		logger.Debug("Using default RPC URL for environment %s: %s", environmentConfig.Name, rpcURL)
	}

	// Get private key from flag or environment
	privateKey, err := GetPrivateKeyOrFail(cCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// Connect to RPC endpoint
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC endpoint %s: %w", rpcURL, err)
	}

	// Get chain ID from the client
	chainID, err := client.ChainID(cCtx.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Create contract caller
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

	return contractCaller, nil
}

// CalculateAndSignApiPermissionDigest calculates the API permission digest using the contract
// and signs it with the user's private key
func CalculateAndSignApiPermissionDigest(
	cCtx *cli.Context,
	permission [4]byte,
	expiry *big.Int,
) ([]byte, error) {
	// Get private key from CLI context
	privateKeyHex, err := GetPrivateKeyOrFail(cCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// Parse private key
	privateKey, err := ethcrypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get AppController binding
	client, appController, err := GetAppControllerBinding(cCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AppController binding: %w", err)
	}
	defer client.Close()

	// Call the contract to calculate the digest hash
	digestHash, err := appController.CalculateApiPermissionDigestHash(&bind.CallOpts{Context: cCtx.Context}, permission, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate digest hash: %w", err)
	}

	// Apply EIP-191 message signing prefix ("\x19Ethereum Signed Message:\n" + length)
	prefixedHash := ethcrypto.Keccak256(
		[]byte("\x19Ethereum Signed Message:\n32"),
		digestHash[:],
	)

	// Sign the EIP-191 prefixed hash
	signature, err := ethcrypto.Sign(prefixedHash, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign digest: %w", err)
	}

	return signature, nil
}

func GetAndPrintAppInfo(cCtx *cli.Context, appID ethcommon.Address, statusOverride ...string) error {
	logger := common.LoggerFromContext(cCtx)

	client, appController, err := GetAppControllerBinding(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get contract caller: %w", err)
	}

	// Get app status and release block number concurrently
	status, releaseBlockNumber, err := common.Parallel(
		func() (uint8, error) { return appController.GetAppStatus(&bind.CallOpts{Context: cCtx.Context}, appID) },
		func() (uint32, error) {
			return appController.GetAppLatestReleaseBlockNumber(&bind.CallOpts{Context: cCtx.Context}, appID)
		},
	)
	if err != nil {
		return fmt.Errorf("failed to get app info: %w", err)
	}

	userApiClient, err := NewUserApiClient(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get userApi client: %w", err)
	}

	count := cCtx.Int(common.AddressCountFlag.Name)
	if count <= 0 {
		count = 1
	}

	// Get environment config for context
	environmentConfig, err := GetEnvironmentConfig(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get environment config: %w", err)
	}

	config := AppController.IAppControllerAppConfig{
		Status:                   status,
		LatestReleaseBlockNumber: releaseBlockNumber,
	}

	info, err := userApiClient.GetInfos(cCtx, []ethcommon.Address{appID}, count)
	if err != nil {
		return fmt.Errorf("failed to get info: %w", err)
	}

	if len(info.Apps) == 0 {
		return fmt.Errorf("no info found for app %s", appID.Hex())
	}

	// Get status override, if provided
	override := ""
	if len(statusOverride) > 0 {
		override = statusOverride[0]
	}
	err = PrintAppInfoWithStatus(cCtx.Context, logger, client, appID, config, info.Apps[0], environmentConfig.Name, override)
	if err != nil {
		return fmt.Errorf("failed to print app info: %w", err)
	}

	return nil
}

func PrintAppInfo(ctx context.Context, logger iface.Logger, client *ethclient.Client, appID ethcommon.Address, config AppController.IAppControllerAppConfig, info AppInfo, environmentName string) error {
	return PrintAppInfoWithStatus(ctx, logger, client, appID, config, info, environmentName, "")
}

func PrintAppInfoWithStatus(ctx context.Context, logger iface.Logger, client *ethclient.Client, appID ethcommon.Address, config AppController.IAppControllerAppConfig, info AppInfo, environmentName string, statusOverride string) error {
	latestReleaseBlockTime := time.Time{}
	if config.LatestReleaseBlockNumber != 0 {
		// get timestamp for block number
		latestReleaseBlock, err := client.BlockByNumber(ctx, big.NewInt(int64(config.LatestReleaseBlockNumber)))
		if err != nil {
			return fmt.Errorf("failed to get BlockNumber: %w", err)
		}
		latestReleaseBlockTime = time.Unix(int64(latestReleaseBlock.Time()), 0)
	}
	fmt.Println()

	// Show app name if available
	if name := common.GetAppName(environmentName, appID.Hex()); name != "" {
		logger.Info("App Name: %s", name)
	}

	logger.Info("App ID: %s", appID.Hex())
	logger.Info("Latest Release Time: %s", latestReleaseBlockTime.Format(time.DateTime))

	// Compare contract and API status to show transition states when they differ
	status := getDisplayStatus(config.Status, info.Status, statusOverride)
	logger.Info("Status: %s", status)
	logger.Info("IP: %s", info.Ip)

	// Display addresses if available
	if len(info.Addresses.Data.EVMAddresses) > 0 {
		printEVMAddresses(logger, info.Addresses.Data.EVMAddresses)
	}
	if len(info.Addresses.Data.SolanaAddresses) > 0 {
		printSolanaAddresses(logger, info.Addresses.Data.SolanaAddresses)
	}

	fmt.Println()
	return nil
}

// printEVMAddresses formats and prints EVM addresses
func printEVMAddresses(logger iface.Logger, addresses []types.EVMAddressAndDerivationPath) {
	if len(addresses) == 1 {
		addr := addresses[0]
		logger.Info("EVM Address: %s (path: %s)", addr.Address.Hex(), addr.DerivationPath)
	} else {
		logger.Info("EVM Addresses:")
		for i, addr := range addresses {
			logger.Info("  [%d] %s (path: %s)", i, addr.Address.Hex(), addr.DerivationPath)
		}
	}
}

// printSolanaAddresses formats and prints Solana addresses
func printSolanaAddresses(logger iface.Logger, addresses []types.SolanaAddressAndDerivationPath) {
	if len(addresses) == 1 {
		addr := addresses[0]
		logger.Info("Solana Address: %s (path: %s)", addr.Address, addr.DerivationPath)
	} else {
		logger.Info("Solana Addresses:")
		for i, addr := range addresses {
			logger.Info("  [%d] %s (path: %s)", i, addr.Address, addr.DerivationPath)
		}
	}
}

// contractStatusToString converts AppStatus enum to string
func contractStatusToString(status uint8) string {
	switch common.AppStatus(status) {
	case common.ContractAppStatusNone:
		return "None"
	case common.ContractAppStatusStarted:
		return "Running"
	case common.ContractAppStatusStopped:
		return "Stopped"
	case common.ContractAppStatusTerminated:
		return "Terminated"
	default:
		return "Unknown"
	}
}

// getDisplayStatus compares contract and API status and returns appropriate display string
func getDisplayStatus(contractStatus uint8, apiStatus string, statusOverride ...string) string {
	// If override provided, use it
	if len(statusOverride) > 0 && statusOverride[0] != "" {
		return statusOverride[0]
	}

	// If no API status, return contract status
	if apiStatus == "" {
		return contractStatusToString(contractStatus)
	}

	// Special API statuses take precedence
	if strings.EqualFold(apiStatus, common.AppStatusExited) {
		return common.AppStatusExited
	}

	contractStatusStr := contractStatusToString(contractStatus)

	// If states match, return API status
	if strings.EqualFold(contractStatusStr, apiStatus) {
		return apiStatus
	}

	// States differ - check if we're in a transition
	transitions := map[string]string{
		"Running":    "Starting",
		"Stopped":    "Stopping",
		"Terminated": "Terminating",
	}

	if transition, exists := transitions[contractStatusStr]; exists {
		return transition
	}

	// Default to API status
	return apiStatus
}

// getRPCURL gets RPC URL from flag or environment default
func getRPCURL(cCtx *cli.Context, environmentConfig *common.EnvironmentConfig) (string, error) {
	rpcURL := cCtx.String(common.RpcUrlFlag.Name)
	if rpcURL == "" && environmentConfig != nil && environmentConfig.DefaultRPCURL != "" {
		rpcURL = environmentConfig.DefaultRPCURL
	}
	if rpcURL == "" {
		return "", fmt.Errorf("rpc-url required. Provide via --rpc-url flag or ensure environment has default RPC URL")
	}
	return rpcURL, nil
}

// CheckAppLogPermission checks if an app currently has public log viewing permissions
func CheckAppLogPermission(cCtx *cli.Context, appAddress ethcommon.Address) (bool, error) {
	// Get environment config
	environmentConfig, err := GetEnvironmentConfig(cCtx)
	if err != nil {
		return false, fmt.Errorf("failed to get environment config: %w", err)
	}

	// Get RPC URL and connect to client
	rpcURL, err := getRPCURL(cCtx, &environmentConfig)
	if err != nil {
		return false, fmt.Errorf("failed to get RPC URL: %w", err)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return false, fmt.Errorf("failed to connect to RPC endpoint %s: %w", rpcURL, err)
	}
	defer client.Close()

	// Create permission controller instance
	permissionController := IPermissionController.NewIPermissionController()

	// Pack the CanCall method
	data := permissionController.PackCanCall(
		appAddress,
		common.AnyoneCanCallAddress,
		common.ApiPermissionsTarget,
		common.CanViewAppLogsPermission)

	// Call the contract
	result, err := client.CallContract(cCtx.Context, ethereum.CallMsg{
		To:   &environmentConfig.PermissionControllerAddress,
		Data: data,
	}, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	// Unpack the result
	canCall, err := permissionController.UnpackCanCall(result)
	if err != nil {
		return false, fmt.Errorf("failed to unpack result: %w", err)
	}

	return canCall, nil
}
