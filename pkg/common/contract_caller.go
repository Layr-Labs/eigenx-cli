package common

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	erc7702delegatorV2 "github.com/Layr-Labs/eigenx-cli/internal/bindings/EIP7702StatelessDeleGator"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	appcontrollerV1 "github.com/Layr-Labs/eigenx-contracts/pkg/bindings/v1/AppController"
	appcontrollerV2 "github.com/Layr-Labs/eigenx-contracts/pkg/bindings/v2/AppController"
	permissioncontrollerV2 "github.com/Layr-Labs/eigenx-contracts/pkg/bindings/v2/IPermissionController"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
)

const (
	gasLimitOverestimationPercentage = 20  // 20%
	gasPriceOverestimationPercentage = 100 // 100%
)

var (
	eip7702CodePrefix = []byte{0xef, 0x01, 0x00}
	executeBatchMode  = [32]byte{0x01} // Mode for ExecuteBatch for the ERC7702 delegator
)

// ContractCaller provides a high-level interface for interacting with contracts
type ContractCaller struct {
	ethclient                   *ethclient.Client
	privateKey                  *ecdsa.PrivateKey
	chainID                     *big.Int
	logger                      iface.Logger
	environmentConfig           EnvironmentConfig
	appControllerBinding        *appcontrollerV2.AppController
	permissionControllerBinding *permissioncontrollerV2.IPermissionController
	erc7702DelegatorBinding     *erc7702delegatorV2.EIP7702StatelessDeleGator
	SelfAddress                 common.Address
}

func NewContractCaller(privateKeyHex string, chainID *big.Int, environmentConfig EnvironmentConfig, client *ethclient.Client, logger iface.Logger) (*ContractCaller, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	SelfAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	return &ContractCaller{
		ethclient:                   client,
		privateKey:                  privateKey,
		chainID:                     chainID,
		logger:                      logger,
		environmentConfig:           environmentConfig,
		appControllerBinding:        appcontrollerV2.NewAppController(),
		permissionControllerBinding: permissioncontrollerV2.NewIPermissionController(),
		erc7702DelegatorBinding:     erc7702delegatorV2.NewEIP7702StatelessDeleGator(),
		SelfAddress:                 SelfAddress,
	}, nil
}

// DeployApp creates a new app via AppController contract, accepts admin permissions, and upgrades the app
func (cc *ContractCaller) DeployApp(ctx context.Context, salt [32]byte, release appcontrollerV2.IAppControllerRelease, publicLogs bool, imageRef string) (appID common.Address, err error) {
	createData, err := cc.appControllerBinding.TryPackCreateApp(salt, release)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack create app: %w", err)
	}

	appController, err := appcontrollerV1.NewAppController(cc.environmentConfig.AppControllerAddress, cc.ethclient)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to create app controller: %w", err)
	}

	appAddress, err := appController.CalculateAppId(&bind.CallOpts{Context: ctx}, cc.SelfAddress, salt)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to calculate app id: %w", err)
	}

	acceptAdminData, err := cc.permissionControllerBinding.TryPackAcceptAdmin(appAddress)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack accept admin: %w", err)
	}

	// assemble executions
	executions := []erc7702delegatorV2.Execution{
		{
			Target:   cc.environmentConfig.AppControllerAddress,
			Value:    big.NewInt(0),
			CallData: createData,
		},
		{
			Target:   cc.environmentConfig.PermissionControllerAddress,
			Value:    big.NewInt(0),
			CallData: acceptAdminData,
		},
	}

	// Only add public logs permission if requested
	if publicLogs {
		anyoneCanViewLogsData, err := cc.permissionControllerBinding.TryPackSetAppointee(appAddress, AnyoneCanCallAddress, ApiPermissionsTarget, CanViewAppLogsPermission)
		if err != nil {
			return common.Address{}, fmt.Errorf("failed to pack anyone can view logs: %w", err)
		}
		executions = append(executions, erc7702delegatorV2.Execution{
			Target:   cc.environmentConfig.PermissionControllerAddress,
			Value:    big.NewInt(0),
			CallData: anyoneCanViewLogsData,
		})
	}

	// Prepare confirmation and pending messages
	confirmationPrompt := fmt.Sprintf("Deploy new app with image: %s", imageRef)
	pendingMessage := "Deploying new app..."

	return appAddress, cc.ExecuteBatch(ctx, executions, cc.isMainnet(), confirmationPrompt, pendingMessage)
}

// UpgradeApp upgrades an app via AppController contract
func (cc *ContractCaller) UpgradeApp(ctx context.Context, appAddress common.Address, release appcontrollerV2.IAppControllerRelease, publicLogs bool, needsPermissionChange bool, imageRef string) error {
	upgradeData, err := cc.appControllerBinding.TryPackUpgradeApp(appAddress, release)
	if err != nil {
		return fmt.Errorf("failed to pack upgrade app: %w", err)
	}

	// Start with upgrade execution
	executions := []erc7702delegatorV2.Execution{
		{
			Target:   cc.environmentConfig.AppControllerAddress,
			Value:    big.NewInt(0),
			CallData: upgradeData,
		},
	}

	// Add permission transaction if needed
	if needsPermissionChange {
		if publicLogs {
			// Add public permission (private→public)
			addLogsData, err := cc.permissionControllerBinding.TryPackSetAppointee(appAddress, AnyoneCanCallAddress, ApiPermissionsTarget, CanViewAppLogsPermission)
			if err != nil {
				return fmt.Errorf("failed to pack add logs permission: %w", err)
			}
			executions = append(executions, erc7702delegatorV2.Execution{
				Target:   cc.environmentConfig.PermissionControllerAddress,
				Value:    big.NewInt(0),
				CallData: addLogsData,
			})
		} else {
			// Remove public permission (public→private)
			removeLogsData, err := cc.permissionControllerBinding.TryPackRemoveAppointee(appAddress, AnyoneCanCallAddress, ApiPermissionsTarget, CanViewAppLogsPermission)
			if err != nil {
				return fmt.Errorf("failed to pack remove logs permission: %w", err)
			}
			executions = append(executions, erc7702delegatorV2.Execution{
				Target:   cc.environmentConfig.PermissionControllerAddress,
				Value:    big.NewInt(0),
				CallData: removeLogsData,
			})
		}
	}

	// Prepare confirmation and pending messages
	appName := GetAppName(cc.environmentConfig.Name, appAddress.Hex())

	confirmationPrompt := "Upgrade app"
	pendingMessage := "Upgrading app..."
	if appName != "" {
		confirmationPrompt = fmt.Sprintf("%s '%s'", confirmationPrompt, appName)
		pendingMessage = fmt.Sprintf("Upgrading app '%s'...", appName)
	}
	confirmationPrompt = fmt.Sprintf("%s with image: %s", confirmationPrompt, imageRef)

	return cc.ExecuteBatch(ctx, executions, cc.isMainnet(), confirmationPrompt, pendingMessage)
}

// StartApp starts a stopped app via AppController contract
func (cc *ContractCaller) StartApp(ctx context.Context, appAddress common.Address) error {
	data, err := cc.appControllerBinding.TryPackStartApp(appAddress)
	if err != nil {
		return fmt.Errorf("failed to pack start app: %w", err)
	}

	// Create the CallMsg
	callMsg := &ethereum.CallMsg{
		To:   &cc.environmentConfig.AppControllerAddress,
		Data: data,
	}

	// Prepare confirmation and pending messages
	appName := GetAppName(cc.environmentConfig.Name, appAddress.Hex())

	confirmationPrompt := "Start app"
	pendingMessage := "Starting app..."
	if appName != "" {
		confirmationPrompt = fmt.Sprintf("%s '%s'", confirmationPrompt, appName)
		pendingMessage = fmt.Sprintf("Starting app '%s'...", appName)
	}

	return cc.SendAndWaitForTransaction(ctx, "StartApp", callMsg, cc.isMainnet(), confirmationPrompt, pendingMessage)
}

// StopApp stops a running app via AppController contract
func (cc *ContractCaller) StopApp(ctx context.Context, appAddress common.Address) error {
	data, err := cc.appControllerBinding.TryPackStopApp(appAddress)
	if err != nil {
		return fmt.Errorf("failed to pack stop app: %w", err)
	}

	// Create the CallMsg
	callMsg := &ethereum.CallMsg{
		To:   &cc.environmentConfig.AppControllerAddress,
		Data: data,
	}

	// Prepare confirmation and pending messages
	appName := GetAppName(cc.environmentConfig.Name, appAddress.Hex())

	confirmationPrompt := "Stop app"
	pendingMessage := "Stopping app..."
	if appName != "" {
		confirmationPrompt = fmt.Sprintf("%s '%s'", confirmationPrompt, appName)
		pendingMessage = fmt.Sprintf("Stopping app '%s'...", appName)
	}

	return cc.SendAndWaitForTransaction(ctx, "StopApp", callMsg, cc.isMainnet(), confirmationPrompt, pendingMessage)
}

// TerminateApp terminates an app permanently via AppController contract
func (cc *ContractCaller) TerminateApp(ctx context.Context, appAddress common.Address, force bool) error {
	data, err := cc.appControllerBinding.TryPackTerminateApp(appAddress)
	if err != nil {
		return fmt.Errorf("failed to pack terminate app: %w", err)
	}

	// Create the CallMsg
	callMsg := &ethereum.CallMsg{
		To:   &cc.environmentConfig.AppControllerAddress,
		Data: data,
	}

	// Prepare confirmation and pending messages
	appName := GetAppName(cc.environmentConfig.Name, appAddress.Hex())

	confirmationPrompt := "⚠️  \033[1mPermanently\033[0m destroy app"
	pendingMessage := "Terminating app..."
	if appName != "" {
		confirmationPrompt = fmt.Sprintf("%s '%s'", confirmationPrompt, appName)
		pendingMessage = fmt.Sprintf("Terminating app '%s'...", appName)
	}

	// Note: Terminate always needs confirmation unless force is specified
	return cc.SendAndWaitForTransaction(ctx, "TerminateApp", callMsg, !force, confirmationPrompt, pendingMessage)
}

// GetActiveAppCount returns the number of active apps (STARTED or STOPPED) for a user
func (cc *ContractCaller) GetActiveAppCount(ctx context.Context, user common.Address) (uint32, error) {
	appController, err := appcontrollerV1.NewAppController(cc.environmentConfig.AppControllerAddress, cc.ethclient)
	if err != nil {
		return 0, fmt.Errorf("failed to create app controller: %w", err)
	}

	count, err := appController.GetActiveAppCount(&bind.CallOpts{Context: ctx}, user)
	if err != nil {
		return 0, fmt.Errorf("failed to get active app count: %w", err)
	}

	return count, nil
}

// GetMaxActiveAppsPerUser returns the quota limit for a user
func (cc *ContractCaller) GetMaxActiveAppsPerUser(ctx context.Context, user common.Address) (uint32, error) {
	appController, err := appcontrollerV1.NewAppController(cc.environmentConfig.AppControllerAddress, cc.ethclient)
	if err != nil {
		return 0, fmt.Errorf("failed to create app controller: %w", err)
	}

	quota, err := appController.GetMaxActiveAppsPerUser(&bind.CallOpts{Context: ctx}, user)
	if err != nil {
		return 0, fmt.Errorf("failed to get max active apps: %w", err)
	}

	return quota, nil
}

// GetAppsByCreator retrieves a paginated list of apps created by the specified address
func (cc *ContractCaller) GetAppsByCreator(ctx context.Context, creator common.Address, offset uint64, limit uint64) ([]common.Address, any, error) {
	appController, err := appcontrollerV1.NewAppController(cc.environmentConfig.AppControllerAddress, cc.ethclient)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create app controller: %w", err)
	}

	result, err := appController.GetAppsByCreator(
		&bind.CallOpts{Context: ctx},
		creator,
		new(big.Int).SetUint64(offset),
		new(big.Int).SetUint64(limit),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get apps by creator: %w", err)
	}

	return result.Apps, result.AppConfigsMem, nil
}

// Suspend suspends all active apps for an account and sets their max active apps to 0
func (cc *ContractCaller) Suspend(ctx context.Context, account common.Address, apps []common.Address) error {
	data, err := cc.appControllerBinding.TryPackSuspend(account, apps)
	if err != nil {
		return fmt.Errorf("failed to pack suspend: %w", err)
	}

	// Create the CallMsg
	callMsg := &ethereum.CallMsg{
		To:   &cc.environmentConfig.AppControllerAddress,
		Data: data,
	}

	// Prepare messages
	pendingMessage := fmt.Sprintf("Suspending %d app(s)...", len(apps))
	confirmationPrompt := fmt.Sprintf("Suspend %d app(s) for account %s", len(apps), account.Hex())

	return cc.SendAndWaitForTransaction(ctx, "Suspend", callMsg, cc.isMainnet(), confirmationPrompt, pendingMessage)
}

// EIP 7702 Utility Functions

// CheckERC7702Delegation checks if the given account already delegates to the ERC-7702 delegator
func (cc *ContractCaller) CheckERC7702Delegation(ctx context.Context, account common.Address) (bool, error) {
	code, err := cc.ethclient.CodeAt(ctx, account, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get account code: %w", err)
	}

	// Check if code matches EIP-7702 delegation pattern: 0xef0100 || delegator_address
	expectedCode := append(eip7702CodePrefix, cc.environmentConfig.ERC7702DelegatorAddress.Bytes()...)

	return bytes.Equal(code, expectedCode), nil
}

func (cc *ContractCaller) Undelegate(ctx context.Context) error {
	signedAuth, err := cc.createAuthorization(ctx, common.Address{})
	if err != nil {
		return fmt.Errorf("failed to create authorization: %w", err)
	}

	callMsg := &ethereum.CallMsg{
		To:   &cc.SelfAddress,
		Data: make([]byte, 0),
	}
	callMsg.AuthorizationList = []types.SetCodeAuthorization{signedAuth}

	// Prepare confirmation and pending messages
	confirmationPrompt := "Undelegate account (removes EIP-7702 delegation)"
	pendingMessage := "Undelegating account..."

	return cc.SendAndWaitForTransaction(ctx, "Undelegate", callMsg, cc.isMainnet(), confirmationPrompt, pendingMessage)
}

// ExecuteBatch executes a batch of executions. It sets the code of the EOA to the delegator contract if not already set.
func (cc *ContractCaller) ExecuteBatch(ctx context.Context, executions []erc7702delegatorV2.Execution, needsConfirmation bool, confirmationPrompt string, pendingMessage string) error {
	encodedExecutions, err := EncodeExecutions(executions)
	if err != nil {
		return fmt.Errorf("failed to encode executions: %w", err)
	}
	data := cc.erc7702DelegatorBinding.PackExecute0(executeBatchMode, encodedExecutions)
	callMsg := ethereum.CallMsg{
		To:   &cc.SelfAddress, // eip7702 txs send to themselves
		Data: data,
	}

	isDelegated, err := cc.CheckERC7702Delegation(ctx, cc.SelfAddress)
	if err != nil {
		return fmt.Errorf("failed to check delegation status: %w", err)
	}

	// If not delegated, set the authorization list
	if !isDelegated {
		signedAuth, err := cc.createAuthorization(ctx, cc.environmentConfig.ERC7702DelegatorAddress)
		if err != nil {
			return fmt.Errorf("failed to create authorization: %w", err)
		}

		// Set the authorization list
		callMsg.AuthorizationList = []types.SetCodeAuthorization{signedAuth}
	}

	return cc.SendAndWaitForTransaction(ctx, "ExecuteBatch", &callMsg, needsConfirmation, confirmationPrompt, pendingMessage)
}

func (cc *ContractCaller) createAuthorization(ctx context.Context, delegator common.Address) (types.SetCodeAuthorization, error) {
	// Get current nonce for the account
	nonce, err := cc.ethclient.PendingNonceAt(ctx, cc.SelfAddress)
	if err != nil {
		return types.SetCodeAuthorization{}, fmt.Errorf("failed to get account nonce: %w", err)
	}
	// Increment nonce for authorization
	authorizationNonce := nonce + 1

	// Create authorization tuple for ERC-7702 delegation
	authorization := types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(cc.chainID),
		Address: delegator,
		Nonce:   authorizationNonce,
	}

	// Sign the authorization
	signedAuth, err := types.SignSetCode(cc.privateKey, authorization)
	if err != nil {
		return types.SetCodeAuthorization{}, fmt.Errorf("failed to sign authorization: %w", err)
	}
	return signedAuth, nil
}

var (
	executionArrayType, _ = abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "target", Type: "address"},
		{Name: "value", Type: "uint256"},
		{Name: "callData", Type: "bytes"},
	})
)

func EncodeExecutions(executions []erc7702delegatorV2.Execution) ([]byte, error) {
	arguments := abi.Arguments{
		{
			Type: executionArrayType,
		},
	}

	encodedExecutions, err := arguments.Pack(executions)
	if err != nil {
		return nil, err
	}

	return encodedExecutions, nil
}

/// TX SENDING

func (cc *ContractCaller) SendAndWaitForTransaction(ctx context.Context, txDescription string, callMsg *ethereum.CallMsg, needsConfirmation bool, confirmationPrompt string, pendingMessage string) error {
	// if from is not set, use self address
	if callMsg.From.Cmp(common.Address{}) == 0 {
		callMsg.From = cc.SelfAddress
	}

	nonce, gasTipCap, gasPrice, gasEstimate, err := cc.getTxParams(ctx, *callMsg)
	if err != nil {
		return err
	}

	// Handle confirmation if needed
	if needsConfirmation {
		// Calculate cost for confirmation
		maxCostWei := new(big.Int).Mul(big.NewInt(int64(gasEstimate)), gasPrice)
		cost := FormatETH(maxCostWei)
		err = cc.showConfirmationPrompt(confirmationPrompt, cost)
		if err != nil {
			return err
		}
	}

	// Show pending message if provided
	if pendingMessage != "" {
		cc.logger.Info(pendingMessage)
	}

	var tx *types.Transaction
	if len(callMsg.AuthorizationList) == 0 {
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:    cc.chainID,
			Nonce:      nonce,
			GasTipCap:  gasTipCap,
			GasFeeCap:  gasPrice,
			Gas:        gasEstimate,
			To:         callMsg.To,
			Value:      callMsg.Value,
			Data:       callMsg.Data,
			AccessList: callMsg.AccessList,
		})
	} else {
		tx = types.NewTx(&types.SetCodeTx{
			ChainID:    uint256.MustFromBig(cc.chainID),
			Nonce:      nonce,
			GasTipCap:  uint256.MustFromBig(gasTipCap),
			GasFeeCap:  uint256.MustFromBig(gasPrice),
			Gas:        gasEstimate,
			To:         *callMsg.To,
			Value:      uint256.MustFromBig(callMsg.Value),
			Data:       callMsg.Data,
			AccessList: callMsg.AccessList,
			AuthList:   callMsg.AuthorizationList,
		})
	}

	err = cc.sendAndWaitForTransaction(ctx, txDescription, tx)
	if err != nil {
		return fmt.Errorf("failed to send and wait for transaction: %w", err)
	}
	return nil
}

// showConfirmationPrompt displays a simplified confirmation dialog
func (cc *ContractCaller) showConfirmationPrompt(confirmationPrompt string, cost string) error {
	fmt.Println()
	fmt.Printf("%s on \033[1m%s\033[0m (max cost: %s ETH)\n", confirmationPrompt, cc.environmentConfig.Name, cost)
	fmt.Println()

	confirmed, err := output.Confirm("Continue?")
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return fmt.Errorf("operation cancelled")
	}

	fmt.Println()
	return nil
}

func (cc *ContractCaller) sendAndWaitForTransaction(
	ctx context.Context,
	txDescription string,
	tx *types.Transaction,
) error {
	// sign the transaction
	signer := types.LatestSignerForChainID(cc.chainID)
	signedTx, err := types.SignTx(tx, signer, cc.privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = cc.ethclient.SendTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, cc.ethclient, signedTx)
	if err != nil {
		cc.logger.Error("Waiting for %s transaction (hash: %s) failed: %v", txDescription, tx.Hash().Hex(), err)
		return fmt.Errorf("waiting for %s transaction (hash: %s): %w", txDescription, tx.Hash().Hex(), err)
	}
	if receipt.Status == 0 {
		cc.logger.Error("%s transaction (hash: %s) reverted", txDescription, tx.Hash().Hex())
		return fmt.Errorf("%s transaction (hash: %s) reverted", txDescription, tx.Hash().Hex())
	}
	return nil
}

func (cc *ContractCaller) getTxParams(ctx context.Context, callMsg ethereum.CallMsg) (uint64, *big.Int, *big.Int, uint64, error) {
	nonce, err := cc.ethclient.PendingNonceAt(ctx, cc.SelfAddress)
	if err != nil {
		return 0, nil, nil, 0, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasTipCap, err := cc.ethclient.SuggestGasTipCap(ctx)
	if err != nil {
		return 0, nil, nil, 0, fmt.Errorf("failed to suggest gas tip cap: %w", err)
	}

	head, err := cc.ethclient.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, nil, nil, 0, fmt.Errorf("failed to get block by number: %w", err)
	}
	gasPrice := new(big.Int).Add(head.BaseFee, gasTipCap)
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(100+gasPriceOverestimationPercentage))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(100))

	gasEstimate, err := cc.ethclient.EstimateGas(ctx, callMsg)
	if err != nil {
		// Try to parse custom contract errors
		if parsedErr := cc.parseEstimateGasError(err); parsedErr != nil {
			return 0, nil, nil, 0, parsedErr
		}
		return 0, nil, nil, 0, fmt.Errorf("failed to estimate gas: %w", err)
	}
	gasEstimate = gasEstimate * (100 + gasLimitOverestimationPercentage) / 100

	return nonce, gasTipCap, gasPrice, gasEstimate, nil
}

// parseEstimateGasError attempts to parse custom contract errors from EstimateGas failures
func (cc *ContractCaller) parseEstimateGasError(err error) error {
	if err == nil {
		return nil
	}

	// Check if error has ErrorData method (go-ethereum RPC errors)
	rpcErr, ok := err.(interface{ ErrorData() interface{} })
	if !ok {
		return nil
	}

	data := rpcErr.ErrorData()
	if data == nil {
		return nil
	}

	// Convert data to bytes
	var hexData []byte
	switch v := data.(type) {
	case string:
		hexData = common.FromHex(v)
	case []byte:
		hexData = v
	default:
		return nil
	}

	if len(hexData) < 4 {
		return nil
	}

	// Parse and format the error
	parsedError, err := cc.appControllerBinding.UnpackError(hexData)
	if err != nil {
		return nil
	}

	return formatAppControllerError(parsedError)
}

// isMainnet checks if the chain ID corresponds to Ethereum mainnet
func (cc *ContractCaller) isMainnet() bool {
	return cc.chainID.Cmp(big.NewInt(int64(MainnetChainID))) == 0
}

// formatAppControllerError converts parsed contract errors to user-friendly messages
func formatAppControllerError(parsedError any) error {
	switch parsedError.(type) {
	case *appcontrollerV2.AppControllerMaxActiveAppsExceeded:
		return fmt.Errorf("you have reached your app deployment limit. To request access or increase your limit, please visit https://onboarding.eigencloud.xyz/ or reach out to the Eigen team")
	case *appcontrollerV2.AppControllerGlobalMaxActiveAppsExceeded:
		return fmt.Errorf("the platform has reached the maximum number of active apps. please try again later")
	case *appcontrollerV2.AppControllerInvalidPermissions:
		return fmt.Errorf("you don't have permission to perform this operation")
	case *appcontrollerV2.AppControllerAppAlreadyExists:
		return fmt.Errorf("an app with this owner and salt already exists")
	case *appcontrollerV2.AppControllerAppDoesNotExist:
		return fmt.Errorf("the specified app does not exist")
	case *appcontrollerV2.AppControllerInvalidAppStatus:
		return fmt.Errorf("the app is in an invalid state for this operation")
	case *appcontrollerV2.AppControllerMoreThanOneArtifact:
		return fmt.Errorf("only one artifact is allowed per release")
	case *appcontrollerV2.AppControllerInvalidSignature:
		return fmt.Errorf("invalid signature provided")
	case *appcontrollerV2.AppControllerSignatureExpired:
		return fmt.Errorf("the provided signature has expired")
	case *appcontrollerV2.AppControllerInvalidReleaseMetadataURI:
		return fmt.Errorf("invalid release metadata URI provided")
	case *appcontrollerV2.AppControllerInvalidShortString:
		return fmt.Errorf("invalid short string format")
	default:
		return nil
	}
}
