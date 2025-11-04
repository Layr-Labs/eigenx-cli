package common

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"

	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/logger"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/progress"
	"github.com/urfave/cli/v2"
)

// loggerContextKey is used to store the logger in the context
type loggerContextKey struct{}

// progressTrackerContextKey is used to store the progress tracker in the context
type progressTrackerContextKey struct{}

// GetLoggerFromCLIContext creates a logger based on the CLI context
// It checks the verbose flag and returns the appropriate logger
func GetLoggerFromCLIContext(cCtx *cli.Context) (iface.Logger, iface.ProgressTracker) {
	verbose := cCtx.Bool("verbose")
	return GetLogger(verbose)
}

// Get logger for the env we're in
func GetLogger(verbose bool) (iface.Logger, iface.ProgressTracker) {

	var log iface.Logger
	var tracker iface.ProgressTracker

	if progress.IsTTY() {
		log = logger.NewLogger(verbose)
		tracker = progress.NewTTYProgressTracker(10, os.Stdout)
	} else {
		log = logger.NewZapLogger(verbose)
		tracker = progress.NewLogProgressTracker(10, log)
	}

	return log, tracker
}

// isCI checks if the code is running in a CI environment like GitHub Actions.
func isCI() bool {
	return os.Getenv("CI") == "true"
}

// WithLogger stores the logger in the context
func WithLogger(ctx context.Context, logger iface.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// WithProgressTracker stores the progress tracker in the context
func WithProgressTracker(ctx context.Context, tracker iface.ProgressTracker) context.Context {
	return context.WithValue(ctx, progressTrackerContextKey{}, tracker)
}

// LoggerFromContext retrieves the logger from the context
// If no logger is found, it returns a non-verbose logger as fallback
func LoggerFromContext(cCtx *cli.Context) iface.Logger {
	if logger, ok := cCtx.Context.Value(loggerContextKey{}).(iface.Logger); ok {
		return logger
	}
	// Fallback to logger according to verbose flag if not found in context
	log, _ := GetLoggerFromCLIContext(cCtx)
	return log
}

// ProgressTrackerFromContext retrieves the progress tracker from the context
// If no tracker is found, it returns a non-verbose tracker as fallback
func ProgressTrackerFromContext(ctx context.Context) iface.ProgressTracker {
	if tracker, ok := ctx.Value(progressTrackerContextKey{}).(iface.ProgressTracker); ok {
		return tracker
	}
	// Fallback to non-verbose tracker if not found in context
	_, tracker := GetLogger(false)
	return tracker
}

// PeelBoolFromFlags reports whether a boolean CLI flag is set anywhere in args,
// It supports these forms:
//
//	--verbose
//	--verbose=true|false|1|0|yes|no|t|f
//	--verbose true|false|1|0|yes|no|t|f
//	-v
//	-v=true|false|1|0|yes|no|t|f
//	-v true|false|1|0|yes|no|t|f
//
// The last occurrence wins. If a flag is present without an explicit value, it is treated as true.
func PeelBoolFromFlags(args []string, longFlag, shortFlag string) bool {
	// isBoolLiteral parses common truthy and falsy string literals.
	isBoolLiteral := func(s string) (ok bool, value bool) {
		switch strings.ToLower(s) {
		case "1", "t", "true", "yes", "y":
			return true, true
		case "0", "f", "false", "no", "n":
			return true, false
		default:
			return false, false
		}
	}

	value := false

	for i := 0; i < len(args); i++ {
		token := args[i]

		switch {
		// Exact long or short flag, possibly followed by a separate value token.
		case token == longFlag || token == shortFlag:
			// If the next token exists and is a boolean literal, consume it.
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				if ok, v := isBoolLiteral(args[i+1]); ok {
					value = v
					i++ // consume the value token
					continue
				}
			}
			// No explicit value provided. Presence implies true.
			value = true

		// Equals form for the long flag, for example --verbose=true.
		case strings.HasPrefix(token, longFlag+"="):
			if ok, v := isBoolLiteral(strings.TrimPrefix(token, longFlag+"=")); ok {
				value = v
			} else {
				// Treat unknown values as presence implies true.
				value = true
			}

		// Equals form for the short flag, for example -v=false.
		case strings.HasPrefix(token, shortFlag+"="):
			if ok, v := isBoolLiteral(strings.TrimPrefix(token, shortFlag+"=")); ok {
				value = v
			} else {
				value = true
			}
		}
	}

	return value
}

// ValidateAppName validates that an app name follows Docker image naming restrictions
func ValidateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("app name cannot be empty")
	}

	// Check length constraints (2-255 characters)
	if len(name) < 2 {
		return fmt.Errorf("app name must be at least 2 characters long")
	}
	if len(name) > 255 {
		return fmt.Errorf("app name must be 255 characters or less")
	}

	// Check character constraints (lowercase letters, numbers, hyphens, underscores)
	validChars := regexp.MustCompile(`^[a-z0-9_-]+$`)
	if !validChars.MatchString(name) {
		return fmt.Errorf("app name can only contain lowercase letters, numbers, hyphens (-), and underscores (_)")
	}

	return nil
}

// Parallel executes two functions concurrently and returns both results
func Parallel[T1, T2 any](fn1 func() (T1, error), fn2 func() (T2, error)) (T1, T2, error) {
	var res1 T1
	var res2 T2
	var err1, err2 error
	done := make(chan struct{})

	go func() {
		res1, err1 = fn1()
		done <- struct{}{}
	}()

	go func() {
		res2, err2 = fn2()
		done <- struct{}{}
	}()

	<-done
	<-done

	if err1 != nil {
		return res1, res2, err1
	}
	return res1, res2, err2
}

// IsMainnetEnvironment checks if the given environment is a mainnet environment
func IsMainnetEnvironment(env string) bool {
	return strings.Contains(env, "mainnet")
}

// FormatETH converts wei amount to ETH and formats it as a readable string
func FormatETH(weiAmount *big.Int) string {
	ethAmount := new(big.Float).Quo(new(big.Float).SetInt(weiAmount), big.NewFloat(1e18))
	costStr := ethAmount.Text('f', 6)

	// Remove trailing zeros and decimal point if needed
	trimmed := strings.TrimRight(strings.TrimRight(costStr, "0"), ".")

	// If result is "0", show "<0.000001" for small amounts
	if trimmed == "0" && weiAmount.Cmp(big.NewInt(0)) > 0 {
		return "<0.000001"
	}
	return trimmed
}

// CreateTempDir creates a temporary directory with fallback to ~/.eigenx/tmp if system temp fails
func CreateTempDir(prefix string) (string, error) {
	// First try the system temp directory
	tempDir, err := os.MkdirTemp(os.TempDir(), prefix)
	if err != nil {
		// If that fails, try `~/.eigenx/tmp`
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("failed to create temp directory and couldn't find home dir: %w (home error: %v)", err, homeErr)
		}

		// Create the fallback directory if it doesn't exist
		fallbackBase := fmt.Sprintf("%s/.eigenx/tmp", homeDir)
		if mkErr := os.MkdirAll(fallbackBase, 0755); mkErr != nil {
			return "", fmt.Errorf("failed to create temp directory in system temp (%w) and fallback location (%v)", err, mkErr)
		}

		// Create temp directory in fallback location
		tempDir, err = os.MkdirTemp(fallbackBase, prefix)
		if err != nil {
			return "", fmt.Errorf("failed to create temp directory in both system temp and fallback location: %w", err)
		}
	}
	return tempDir, nil
}
