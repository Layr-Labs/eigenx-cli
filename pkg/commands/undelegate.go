package commands

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// UndelegateCommand defines the CLI command to undelegate an account from the EIP7702 delegator
var UndelegateCommand = &cli.Command{
	Name:   "undelegate",
	Usage:  "Undelegate your account from the EIP7702 delegator",
	Flags:  common.GlobalFlags,
	Action: undelegateAction,
}

// UndelegateAccount undelegates the account from the EIP7702 delegator
func undelegateAction(cCtx *cli.Context) error {
	// 1. Do preflight checks (auth, network, etc.) first
	preflightCtx, err := utils.DoPreflightChecks(cCtx)
	if err != nil {
		return err
	}

	logger := common.LoggerFromContext(cCtx)

	// 2. Check if account is currently delegated
	isDelegated, err := preflightCtx.Caller.CheckERC7702Delegation(cCtx.Context, preflightCtx.Caller.SelfAddress)
	if err != nil {
		return fmt.Errorf("failed to check delegation status: %w", err)
	}

	if !isDelegated {
		logger.Info(fmt.Sprintf("Account %s is not currently delegated", preflightCtx.Caller.SelfAddress))
		return nil
	}

	logger.Info(fmt.Sprintf("Undelegating account %s...", preflightCtx.Caller.SelfAddress))

	// 3. Undelegate
	err = preflightCtx.Caller.Undelegate(cCtx.Context)
	if err != nil {
		return fmt.Errorf("failed to undelegate: %w", err)
	}

	logger.Info("Account undelegated successfully")

	return nil
}
