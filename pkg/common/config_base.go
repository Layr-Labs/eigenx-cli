package common

import (
	"github.com/ethereum/go-ethereum/common"
)

// EnvironmentConfig defines the configuration for a specific environment
type EnvironmentConfig struct {
	Name                        string
	AppControllerAddress        common.Address
	PermissionControllerAddress common.Address
	ERC7702DelegatorAddress     common.Address
	KMSServerURL                string
	UserApiServerURL            string
	DefaultRPCURL               string
}

type CommonAddr struct {
	ERC7702Delegator common.Address
}

type ChainAddr struct {
	PermissionController common.Address
}

const (
	// Chain IDs
	MainnetChainID uint64 = 1
	SepoliaChainID uint64 = 11155111

	// Fallback environment used if no user-defined default is found
	FallbackEnvironment = "sepolia"
)

var (
	// Common addresses across all chains
	CommonAddresses = CommonAddr{
		ERC7702Delegator: common.HexToAddress("0x63c0c19a282a1b52b07dd5a65b58948a07dae32b"),
	}

	// Addresses specific to each chain
	ChainAddresses = map[uint64]ChainAddr{
		MainnetChainID: {
			PermissionController: common.HexToAddress("0x25E5F8B1E7aDf44518d35D5B2271f114e081f0E5"),
		},
		SepoliaChainID: {
			PermissionController: common.HexToAddress("0x44632dfBdCb6D3E21EF613B0ca8A6A0c618F5a37"),
		},
	}

	// Default environment for each chain ID
	DefaultEnvironmentForChainID = map[uint64]string{
		MainnetChainID: "mainnet-alpha", // Ethereum mainnet
		SepoliaChainID: "sepolia",       // Sepolia testnet
	}
)
