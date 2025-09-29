package auth

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var WhoamiCommand = &cli.Command{
	Name:  "whoami",
	Usage: "Show current authentication status and address",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
	}...),
	Action: whoamiAction,
}

func whoamiAction(cCtx *cli.Context) error {
	// Try to get private key from any source
	privateKey, source, err := GetPrivateKeyWithSource(cCtx)
	if err != nil {
		fmt.Println("Not authenticated")
		fmt.Println("")
		fmt.Println("To authenticate, use one of:")
		fmt.Println("  eigenx auth login                    # Store key in keyring")
		fmt.Println("  export PRIVATE_KEY=0x...             # Use environment variable")
		fmt.Println("  eigenx <command> --private-key 0x... # Use flag")
		return nil
	}

	// Get address from private key
	address, err := common.GetAddressFromPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to get address from private key: %w", err)
	}

	// Display authentication info
	fmt.Printf("Address: %s\n", address)
	fmt.Printf("Source:  %s\n", source)

	// Show note if there's a different key available for current environment
	if environmentConfig, err := utils.GetEnvironmentConfig(cCtx); err == nil {
		if envKey, err := common.GetPrivateKey(environmentConfig.Name); err == nil {
			envAddress, _ := common.GetAddressFromPrivateKey(envKey)
			if envAddress != address {
				fmt.Printf("Note: Different key available for %s: %s\n", environmentConfig.Name, envAddress)
			}
		}
	}

	return nil
}
