package commands

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
)

// TelemetryCommand allows users to manage telemetry settings
var TelemetryCommand = &cli.Command{
	Name:  "telemetry",
	Usage: "Manage telemetry settings",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "enable",
			Usage: "Enable telemetry collection",
		},
		&cli.BoolFlag{
			Name:  "disable",
			Usage: "Disable telemetry collection",
		},
		&cli.BoolFlag{
			Name:  "status",
			Usage: "Show current telemetry status",
		},
	},
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		enable := cCtx.Bool("enable")
		disable := cCtx.Bool("disable")
		status := cCtx.Bool("status")

		// Validate flags
		if (enable && disable) || (!enable && !disable && !status) {
			return fmt.Errorf("specify exactly one of --enable, --disable, or --status")
		}

		if status {
			return showTelemetryStatus(logger)
		}

		if enable {
			return enableTelemetry(logger)
		}

		if disable {
			return disableTelemetry(logger)
		}

		return nil
	},
}

// displayGlobalTelemetryStatus shows the global telemetry preference status
func displayGlobalTelemetryStatus(logger iface.Logger, prefix string) error {
	globalPreference, err := common.GetGlobalTelemetryPreference()
	if err != nil {
		return fmt.Errorf("failed to get global telemetry preference: %w", err)
	}

	if globalPreference == nil {
		logger.Info("%s: Not set (defaults to disabled)", prefix)
	} else if *globalPreference {
		logger.Info("%s: Enabled", prefix)
	} else {
		logger.Info("%s: Disabled", prefix)
	}
	return nil
}

func showTelemetryStatus(logger iface.Logger) error {
	return displayGlobalTelemetryStatus(logger, "Telemetry")
}

func enableTelemetry(logger iface.Logger) error {
	// Set global preference
	if err := common.SetGlobalTelemetryPreference(true); err != nil {
		return fmt.Errorf("failed to enable global telemetry: %w", err)
	}

	logger.Info("✅ Telemetry enabled")
	return nil
}

func disableTelemetry(logger iface.Logger) error {
	// Set global preference
	if err := common.SetGlobalTelemetryPreference(false); err != nil {
		return fmt.Errorf("failed to disable global telemetry: %w", err)
	}

	logger.Info("❌ Telemetry disabled")
	return nil
}
