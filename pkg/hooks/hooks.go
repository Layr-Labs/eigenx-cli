package hooks

import (
	"fmt"
	"os"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/logger"
	"github.com/Layr-Labs/eigenx-cli/pkg/telemetry"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// EnvFile is the name of the environment file
const EnvFile = ".env"
const namespace = "EigenX"

type ActionChain struct {
	Processors []func(action cli.ActionFunc) cli.ActionFunc
}

// NewActionChain creates a new action chain
func NewActionChain() *ActionChain {
	return &ActionChain{
		Processors: make([]func(action cli.ActionFunc) cli.ActionFunc, 0),
	}
}

// Use appends a new processor to the chain
func (ac *ActionChain) Use(processor func(action cli.ActionFunc) cli.ActionFunc) {
	ac.Processors = append(ac.Processors, processor)
}

func (ac *ActionChain) Wrap(action cli.ActionFunc) cli.ActionFunc {
	for i := len(ac.Processors) - 1; i >= 0; i-- {
		action = ac.Processors[i](action)
	}
	return action
}

func ApplyMiddleware(commands []*cli.Command, chain *ActionChain) {
	for _, cmd := range commands {
		if cmd.Action != nil {
			cmd.Action = chain.Wrap(cmd.Action)
		}
		if len(cmd.Subcommands) > 0 {
			ApplyMiddleware(cmd.Subcommands, chain)
		}
	}
}

func getFlagValue(ctx *cli.Context, name string) interface{} {
	if !ctx.IsSet(name) {
		return nil
	}

	if ctx.Bool(name) {
		return ctx.Bool(name)
	}
	if ctx.String(name) != "" {
		return ctx.String(name)
	}
	if ctx.Int(name) != 0 {
		return ctx.Int(name)
	}
	if ctx.Float64(name) != 0 {
		return ctx.Float64(name)
	}
	return nil
}

func collectFlagValues(ctx *cli.Context) map[string]interface{} {
	flags := make(map[string]interface{})

	// App-level flags
	for _, flag := range ctx.App.Flags {
		flagName := flag.Names()[0]
		if ctx.IsSet(flagName) {
			flags[flagName] = getFlagValue(ctx, flagName)
		}
	}

	// Command-level flags
	for _, flag := range ctx.Command.Flags {
		flagName := flag.Names()[0]
		if ctx.IsSet(flagName) {
			flags[flagName] = getFlagValue(ctx, flagName)
		}
	}

	return flags
}

func setupTelemetry(cCtx *cli.Context) telemetry.Client {
	logger := common.LoggerFromContext(cCtx)

	// Get global telemetry preference
	globalPref, err := common.GetGlobalTelemetryPreference()
	if err != nil {
		logger.Debug("Failed to get telemetry preference: %v", err)
		return telemetry.NewNoopClient()
	}

	// If telemetry is disabled or not set, return noop client
	telemetryEnabled := globalPref != nil && *globalPref
	if !telemetryEnabled {
		return telemetry.NewNoopClient()
	}

	appEnv, ok := common.AppEnvironmentFromContext(cCtx.Context)
	if !ok {
		return telemetry.NewNoopClient()
	}

	// Ensure UserUUID is saved for consistent tracking across sessions
	globalConfig, err := common.LoadGlobalConfig()
	if err != nil || globalConfig == nil || globalConfig.UserUUID == "" {
		// No saved UserUUID, save the current one for consistent tracking
		if appEnv.UserUUID != "" {
			if err := common.SaveUserId(appEnv.UserUUID); err != nil {
				logger.Debug("Failed to save user UUID: %v", err)
			}
		}
	}

	phClient, err := telemetry.NewPostHogClient(appEnv, namespace)
	if err != nil {
		return telemetry.NewNoopClient()
	}

	return phClient
}

// WithFirstRunSetup handles first-run environment and telemetry setup
func WithFirstRunSetup(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Check if this is the first run
	isFirstRun, err := common.IsFirstRun()
	if err != nil {
		logger.Debug("Failed to check first run status: %v", err)
		return nil // Don't fail the command, just skip the setup
	}

	if !isFirstRun {
		return nil // Not first run, continue normally
	}

	fmt.Println()
	fmt.Println("Welcome to EigenX CLI!")
	fmt.Println()

	// 1. Set default deployment environment first
	if err := setDeploymentEnvironment(logger); err != nil {
		logger.Debug("Setting default deployment environment failed: %v", err)
		// Continue with telemetry even if setting default environment fails
	}

	// 2. Handle telemetry setup
	if err := handleTelemetrySetup(cCtx, logger); err != nil {
		logger.Debug("Telemetry setup failed: %v", err)
		// Continue even if telemetry setup fails
	}

	// 3. Mark first run as complete
	if err := common.MarkFirstRunComplete(); err != nil {
		logger.Debug("Failed to mark first run complete: %v", err)
	}

	return nil
}

// setDeploymentEnvironment sets the default deployment environment for new users based on build type
func setDeploymentEnvironment(logger iface.Logger) error {
	// Check if environment is already set
	if defaultEnv, err := common.GetDefaultEnvironment(); err == nil && defaultEnv != "" {
		return nil // Environment already set, continue normally
	}

	// Choose default based on build type
	var defaultEnv string
	if common.Build == "prod" {
		defaultEnv = common.DefaultEnvironmentForChainID[common.MainnetChainID]
	} else {
		defaultEnv = common.DefaultEnvironmentForChainID[common.SepoliaChainID]
	}

	// Save the selected environment
	if err := common.SetDefaultEnvironment(defaultEnv); err != nil {
		logger.Debug("Failed to save default environment: %v", err)
		return nil // Don't fail the command
	}

	fmt.Printf("✅ Deployment environment: \033[1m%s\033[0m\n", defaultEnv)
	fmt.Println("You can change this later with: eigenx environment set <env>")

	return nil
}

// handleTelemetrySetup handles the telemetry setup part of first-run setup
func handleTelemetrySetup(cCtx *cli.Context, logger iface.Logger) error {

	// Check for global flags that control telemetry behavior
	opts := common.TelemetryPromptOptions{
		EnableTelemetry:  cCtx.Bool("enable-telemetry"),
		DisableTelemetry: cCtx.Bool("disable-telemetry"),
		SkipPromptInCI:   true, // Always skip in CI environments
	}

	// Show telemetry notice and set default preference
	choice := common.ShowTelemetryNotice(logger, opts)

	// Save the preference globally
	if err := common.SetGlobalTelemetryPreference(choice); err != nil {
		logger.Debug("Failed to save telemetry preference: %v", err)
	}

	// First run handling complete
	logger.Debug("First run telemetry setup completed with preference: %v", choice)
	return nil
}

func WithMetricEmission(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// Run command action
		err := action(ctx)

		client := setupTelemetry(ctx)
		ctx.Context = telemetry.ContextWithClient(ctx.Context, client)
		// emit result metrics
		emitTelemetryMetrics(ctx, err)

		return err
	}
}

func emitTelemetryMetrics(ctx *cli.Context, actionError error) {
	metrics, err := telemetry.MetricsFromContext(ctx.Context)
	if err != nil {
		return
	}
	metrics.Properties["command"] = ctx.Command.HelpName
	result := "Success"
	dimensions := map[string]string{}
	if actionError != nil {
		result = "Failure"
		dimensions["error"] = actionError.Error()
	}
	metrics.AddMetricWithDimensions(result, 1, dimensions)

	duration := time.Since(metrics.StartTime).Milliseconds()
	metrics.AddMetric("DurationMilliseconds", float64(duration))

	client, ok := telemetry.ClientFromContext(ctx.Context)
	if !ok {
		return
	}
	defer client.Close()

	l := logger.NewZapLogger(false)
	for _, metric := range metrics.Metrics {
		mDimensions := metric.Dimensions
		for k, v := range metrics.Properties {
			mDimensions[k] = v
		}
		err = client.AddMetric(ctx.Context, metric)
		if err != nil {
			l.Error("failed to add metric", "error", err.Error())
		}
	}
}

func LoadEnvFile(ctx *cli.Context) error {
	// Skip loading .env for the create command
	if ctx.Command.Name != "create" {
		if err := loadEnvFile(); err != nil {
			return err
		}
	}
	return nil
}

// loadEnvFile loads environment variables from .env file if it exists
// Silently succeeds if no .env file is found
func loadEnvFile() error {
	// Check if .env file exists in current directory
	if _, err := os.Stat(EnvFile); os.IsNotExist(err) {
		return nil // .env doesn't exist, just return without error
	}

	// Load .env file
	return godotenv.Load(EnvFile)
}

// getEnvironmentForMetrics returns the environment name for telemetry metrics
// without heavy operations like RPC detection
func getEnvironmentForMetrics(ctx *cli.Context) string {
	if env := ctx.String(common.EnvironmentFlag.Name); env != "" {
		return env
	}
	if env, _ := common.GetDefaultEnvironment(); env != "" {
		return env
	}
	return common.FallbackEnvironment
}

// getDeployerForMetrics attempts to get the deployer address from private key if available
func getDeployerForMetrics(ctx *cli.Context) string {
	// Use the existing GetDeveloperAddress function which handles all sources
	addr, err := utils.GetDeveloperAddress(ctx)
	if err != nil {
		return ""
	}
	return addr.Hex()
}

func WithCommandMetricsContext(ctx *cli.Context) error {
	metrics := telemetry.NewMetricsContext()
	ctx.Context = telemetry.WithMetricsContext(ctx.Context, metrics)

	// Get environment name for metrics
	environment := getEnvironmentForMetrics(ctx)

	// Set environment in metrics
	metrics.Properties["environment"] = environment

	// Set appEnv details in metrics
	if appEnv, ok := common.AppEnvironmentFromContext(ctx.Context); ok {
		metrics.Properties["cli_version"] = appEnv.CLIVersion
		metrics.Properties["os"] = appEnv.OS
		metrics.Properties["arch"] = appEnv.Arch
		metrics.Properties["user_uuid"] = appEnv.UserUUID
	}

	// Collect deployer address if available
	if deployer := getDeployerForMetrics(ctx); deployer != "" {
		metrics.Properties["deployer_address"] = deployer
	}

	// Set flags in metrics
	for k, v := range collectFlagValues(ctx) {
		metrics.Properties[k] = fmt.Sprintf("%v", v)
	}

	metrics.AddMetric("Count", 1)
	return nil
}
