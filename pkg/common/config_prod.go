//go:build prod
// +build prod

package common

import (
	"github.com/ethereum/go-ethereum/common"
)

// Build-specific constants for prod environment
const (
	BuildSuffix        = ""
	KeyringServiceName = "eigenx-cli"
	Build              = "prod"
)

// EnvironmentConfigs contains all environments available in prod builds
var EnvironmentConfigs = map[string]EnvironmentConfig{
	"sepolia": {
		Name:                        "sepolia",
		AppControllerAddress:        common.HexToAddress("0x0dd810a6ffba6a9820a10d97b659f07d8d23d4E2"),
		PermissionControllerAddress: ChainAddresses[SepoliaChainID].PermissionController,
		ERC7702DelegatorAddress:     CommonAddresses.ERC7702Delegator,
		KMSServerURL:                "http://10.128.15.203:8080",
		UserApiServerURL:            "https://35.190.43.182",
		DefaultRPCURL:               "https://ethereum-sepolia-rpc.publicnode.com",
	},
	"mainnet-alpha": {
		Name:                        "mainnet-alpha",
		AppControllerAddress:        common.HexToAddress("0xc38d35Fc995e75342A21CBd6D770305b142Fbe67"),
		PermissionControllerAddress: ChainAddresses[MainnetChainID].PermissionController,
		ERC7702DelegatorAddress:     CommonAddresses.ERC7702Delegator,
		KMSServerURL:                "http://10.128.0.2:8080",
		UserApiServerURL:            "https://userapi-compute.eigencloud.xyz",
		DefaultRPCURL:               "https://ethereum-rpc.publicnode.com",
	},
}
