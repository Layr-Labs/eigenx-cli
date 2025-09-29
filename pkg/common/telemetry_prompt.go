package common

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
)

// TelemetryPromptOptions controls how the telemetry prompt behaves
type TelemetryPromptOptions struct {
	// EnableTelemetry automatically enables telemetry without prompting (for --enable-telemetry flag)
	EnableTelemetry bool
	// DisableTelemetry automatically disables telemetry without prompting (for --disable-telemetry flag)
	DisableTelemetry bool
	// SkipPromptInCI skips the prompt in CI environments (defaults to disabled)
	SkipPromptInCI bool
}

// ShowTelemetryNotice displays telemetry information notice without prompting
func ShowTelemetryNotice(logger iface.Logger, opts TelemetryPromptOptions) bool {
	// Handle explicit enable/disable flags first (they take precedence over everything)
	if opts.EnableTelemetry {
		fmt.Println("✅ Telemetry enabled via --enable-telemetry flag")
		return true
	}

	if opts.DisableTelemetry {
		fmt.Println("❌ Telemetry disabled via --disable-telemetry flag")
		return false
	}

	// Check if we're in a CI environment - disable by default
	if opts.SkipPromptInCI && isCI() {
		logger.Debug("CI environment detected, telemetry disabled by default")
		return false
	}

	// Show brief telemetry notice (not the full info)
	showTelemetryOptOutNotice()

	// Default to enabled
	return true
}

// showTelemetryOptOutNotice shows a brief telemetry notice for first run
func showTelemetryOptOutNotice() {
	fmt.Println()
	fmt.Println("📊 Telemetry is enabled to help us improve EigenX")
	fmt.Println("   We collect anonymous usage data only. No personal information or private keys.")
	fmt.Println()
	fmt.Println("   To opt out: eigenx telemetry --disable")
	fmt.Println("   View status: eigenx telemetry --status")
	fmt.Println()
}
