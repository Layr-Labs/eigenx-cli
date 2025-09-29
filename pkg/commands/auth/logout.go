package auth

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/urfave/cli/v2"
)

var LogoutCommand = &cli.Command{
	Name:  "logout",
	Usage: "Remove private key from OS keyring",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.ForceFlag,
	}...),
	Action: logoutAction,
}

func logoutAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Determine the key name
	keyName, err := getAuthKeyName(cCtx)
	if err != nil {
		return fmt.Errorf("failed to determine key name: %w", err)
	}

	// Check if key exists
	if _, err := common.GetPrivateKey(keyName); err != nil {
		return fmt.Errorf("no key found for '%s'", keyName)
	}

	// Confirm unless forced
	if !cCtx.Bool(common.ForceFlag.Name) {
		confirmed, err := output.Confirm(fmt.Sprintf("Remove private key for '%s'?", keyName))
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			logger.Info("Logout cancelled")
			return nil
		}
	}

	// Remove from keyring
	if err := common.DeletePrivateKey(keyName); err != nil {
		return fmt.Errorf("failed to remove private key from keyring: %w", err)
	}

	logger.Info("Successfully logged out (removed key: %s)", keyName)

	return nil
}
