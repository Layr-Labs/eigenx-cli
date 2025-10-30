//go:build !prod
// +build !prod

package common

import (
	"github.com/ethereum/go-ethereum/common"
)

// Build-specific constants for dev environment
const (
	BuildSuffix        = "-dev"
	KeyringServiceName = "eigenx-cli-dev"
	Build              = "dev"
)

// EnvironmentConfigs contains all environments available in dev builds
var EnvironmentConfigs = map[string]EnvironmentConfig{
	"sepolia": {
		Name:                        "sepolia",
		AppControllerAddress:        common.HexToAddress("0xa86DC1C47cb2518327fB4f9A1627F51966c83B92"),
		PermissionControllerAddress: ChainAddresses[SepoliaChainID].PermissionController,
		ERC7702DelegatorAddress:     CommonAddresses.ERC7702Delegator,
		KMSServerURL:                "http://10.128.0.57:8080",
		UserApiServerURL:            "https://userapi-compute-sepolia-dev.eigencloud.xyz",
		DefaultRPCURL:               "https://ethereum-sepolia-rpc.publicnode.com",
	},
}
