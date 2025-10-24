package common

import ethcommon "github.com/ethereum/go-ethereum/common"

// Project structure constants
const (
	// L1 is the name of the L1 chain
	L1 = "l1"

	// ContractsDir is the subdirectory name for contract components
	ContractsDir = "contracts"

	// Makefile is the name of the makefile used for root level operations
	Makefile = "Makefile"

	// ContractsMakefile is the name of the makefile used for contract level operations
	ContractsMakefile = "Makefile"

	// GlobalConfigFile is the name of the global YAML used to store global config details (eg, user_id)
	GlobalConfigFile = "config.yaml"

	// Docker open timeout
	DockerOpenTimeoutSeconds = 10

	// Docker open retry interval in milliseconds
	DockerOpenRetryIntervalMilliseconds = 500

	// WatchPollIntervalSeconds is the interval between watch loop polls in seconds
	WatchPollIntervalSeconds = 5

	// Environment variable names
	MnemonicEnvVar         = "MNEMONIC"                  // Filtered out, overridden by protocol
	EigenMachineTypeEnvVar = "EIGEN_MACHINE_TYPE_PUBLIC" // Instance type configuration
)

// API permissions constants
var (
	// The permission to view app logs
	// bytes4(keccak256("CAN_VIEW_APP_LOGS()"))
	CanViewAppLogsPermission = [4]byte{0x2f, 0xd3, 0xf2, 0xfe}

	// The permission to view sensitive app info (including real IPs)
	// bytes4(keccak256("CAN_VIEW_SENSITIVE_APP_INFO()"))
	CanViewSensitiveAppInfoPermission = [4]byte{0x0e, 0x67, 0xb2, 0x2f}

	// The permission to manage billing and subscriptions
	// bytes4(keccak256("CAN_MANAGE_BILLING()"))
	CanManageBillingPermission = [4]byte{0xd6, 0xb8, 0x55, 0xa1}

	// The address that is used to allow auth to be bypassed for certain permissions
	// address(bytes20(keccak256("PermissionController:AnyoneCanCall")))
	AnyoneCanCallAddress = ethcommon.HexToAddress("0x493219d9949348178af1f58740655951a8cd110c")

	// The address that is permissioned onchain for calls
	// address(bytes20(keccak256("PermissionController:ApiPermissions")))
	ApiPermissionsTarget = ethcommon.HexToAddress("0x57ee1fb74c1087e26446abc4fb87fd8f07c43d8d")
)

type AppStatus uint8

const (
	ContractAppStatusNone AppStatus = iota
	ContractAppStatusStarted
	ContractAppStatusStopped
	ContractAppStatusTerminated
	ContractAppStatusSuspended
)

// App status strings from API
const (
	AppStatusCreated     = "Created"
	AppStatusDeploying   = "Deploying"
	AppStatusUpgrading   = "Upgrading"
	AppStatusResuming    = "Resuming"
	AppStatusRunning     = "Running"
	AppStatusStopping    = "Stopping"
	AppStatusStopped     = "Stopped"
	AppStatusTerminating = "Terminating"
	AppStatusTerminated  = "Terminated"
	AppStatusSuspended   = "Suspended"
	AppStatusFailed      = "Failed"
	AppStatusExited      = "Exited"
)
