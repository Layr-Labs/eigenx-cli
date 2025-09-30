package app

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-contracts/pkg/bindings/v1/AppController"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

const (
	LOG_ROTATION_THRESHOLD = 1024
	LOG_POLL_INTERVAL      = 2 * time.Second
)

var ListCommand = &cli.Command{
	Name:  "list",
	Usage: "List all deployed apps",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		common.PrivateKeyFlag,
		common.AllFlag,
		common.AddressCountFlag,
	}...),
	Action: listAction,
}

var InfoCommand = &cli.Command{
	Name:      "info",
	Usage:     "Show detailed instance info",
	ArgsUsage: "[app-id|name]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		common.AddressCountFlag,
	}...),
	Action: infoAction,
}

var LogsCommand = &cli.Command{
	Name:      "logs",
	Usage:     "View app logs",
	ArgsUsage: "[app-id|name]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.EnvironmentFlag,
		common.RpcUrlFlag,
		common.FollowFlag,
	}...),
	Action: logsAction,
}

func listAction(cCtx *cli.Context) error {
	ctx := cCtx.Context
	logger := common.LoggerFromContext(cCtx)

	// Get contract caller from context
	client, appController, err := utils.GetAppControllerBinding(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get contract caller: %w", err)
	}

	developerAddr, err := utils.GetDeveloperAddress(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get developer address: %w", err)
	}

	// List apps with pagination (start with first 50)
	result, err := appController.GetAppsByDeveloper(&bind.CallOpts{Context: ctx}, developerAddr, big.NewInt(0), big.NewInt(50))
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	if len(result.Apps) == 0 {
		logger.Info("No apps found for developer %s", developerAddr.Hex())
		return nil
	}

	showAll := cCtx.Bool(common.AllFlag.Name)
	var filteredApps []ethcommon.Address
	var filteredConfigs []AppController.IAppControllerAppConfig

	// Filter out terminated apps unless --all flag is used
	for i, appAddr := range result.Apps {
		config := result.AppConfigsMem[i]
		if !showAll && common.AppStatus(config.Status) == common.ContractAppStatusTerminated {
			continue
		}
		filteredApps = append(filteredApps, appAddr)
		filteredConfigs = append(filteredConfigs, config)
	}

	if len(filteredApps) == 0 {
		if showAll {
			logger.Info("No apps found for developer %s", developerAddr.Hex())
		} else {
			logger.Info("No active apps found for developer %s (use --all to show terminated apps)", developerAddr.Hex())
		}
		return nil
	}

	userApiClient, err := utils.NewUserApiClient(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get KMS client: %w", err)
	}

	// Get environment config for context
	environmentConfig, err := utils.GetEnvironmentConfig(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get environment config: %w", err)
	}

	count := cCtx.Int(common.AddressCountFlag.Name)
	if count <= 0 {
		count = 1
	}

	infos, err := userApiClient.GetInfos(cCtx, filteredApps, count)
	if err != nil {
		return fmt.Errorf("failed to get info: %w", err)
	}

	if len(infos.Apps) != len(filteredApps) {
		return fmt.Errorf("expected %d app infos but got %d", len(filteredApps), len(infos.Apps))
	}

	for i, appAddr := range filteredApps {
		err = utils.PrintAppInfo(ctx, logger, client, appAddr, filteredConfigs[i], infos.Apps[i], environmentConfig.Name)
		if err != nil {
			return fmt.Errorf("failed to print app info: %w", err)
		}
		if i < len(filteredApps)-1 {
			fmt.Println("----------------------------------------------------------------------")
		}
	}

	return nil
}

func infoAction(cCtx *cli.Context) error {
	// Get app address from args or interactive selection
	appID, err := utils.GetAppIDInteractive(cCtx, 0, "view")
	if err != nil {
		return fmt.Errorf("failed to get app address: %w", err)
	}

	return utils.GetAndPrintAppInfo(cCtx, appID)
}

func logsAction(cCtx *cli.Context) error {
	fmt.Println()

	appID, err := utils.GetAppIDInteractive(cCtx, 0, "view logs for")
	if err != nil {
		return fmt.Errorf("failed to get app address: %w", err)
	}

	userApiClient, err := utils.NewUserApiClient(cCtx)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	follow := cCtx.Bool(common.FollowFlag.Name)

	// Track previously seen logs to only show new content
	var previousLogs string

	for {
		logs, err := userApiClient.GetLogs(cCtx, appID)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}
		// Print only new logs
		if logs != previousLogs {
			if previousLogs == "" {
				// First fetch: print all logs
				fmt.Print(logs)
			} else {
				// Find where the new logs start by looking for the longest suffix
				// of previousLogs that is a prefix of logs
				newContent := findNewLogContent(previousLogs, logs)
				if newContent != "" {
					fmt.Print(newContent)
				}
			}
			previousLogs = logs
		}

		// If not following, exit after first fetch
		if !follow {
			fmt.Println()
			return nil
		}

		// Wait before next poll
		time.Sleep(LOG_POLL_INTERVAL)
	}
}

// findNewLogContent finds the new content in logs that wasn't in previousLogs.
// It handles cases where logs may have been rotated (previousLogs suffix matches logs prefix).
func findNewLogContent(previousLogs, logs string) string {
	// If logs start with all of previousLogs, return just the new part
	if strings.HasPrefix(logs, previousLogs) {
		return strings.TrimPrefix(logs, previousLogs)
	}

	// Find the longest suffix of previousLogs that is a prefix of logs
	// This handles log rotation where some old logs are included
	for i := 1; i < len(previousLogs); i++ {
		suffix := previousLogs[i:]
		if strings.HasPrefix(logs, suffix) {
			// Found overlap, return only the new content
			return strings.TrimPrefix(logs, suffix)
		}
	}

	// must be longer than the rotation threshold
	if len(logs) < LOG_ROTATION_THRESHOLD {
		return logs
	}

	// No overlap found, logs were completely replaced
	return logs
}
