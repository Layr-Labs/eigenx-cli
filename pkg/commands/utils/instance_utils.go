package utils

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// ConfigureInstanceType handles environment file creation or update based on instance type selection.
// It implements smart logic:
// - No env file + default instance type: no action (uses deployment defaults)
// - No env file + non-default instance type: creates .env file with instance type
// - Has env file: updates existing file with instance type
// Returns the final envFilePath (which may be updated if created).
func ConfigureInstanceType(cCtx *cli.Context, envFilePath, instanceType string) (string, error) {
	// Validate instance type
	if !common.ValidateInstanceType(instanceType) {
		return "", fmt.Errorf("invalid instance type: %s", instanceType)
	}

	logger := common.LoggerFromContext(cCtx)

	// If env file specified, set the variable
	if envFilePath != "" {
		if err := SetEnvFileVariable(envFilePath, common.EigenMachineTypeEnvVar, instanceType); err != nil {
			return envFilePath, fmt.Errorf("failed to set instance type in env file: %w", err)
		}
		logger.Info("Instance type set to: %s", instanceType)
		return envFilePath, nil
	}

	// No env file - check if we need to create one
	defaultInstanceType := common.GetDefaultInstanceType().Value
	if instanceType == defaultInstanceType {
		// Default instance type + no env file = use deployment defaults
		return "", nil
	}

	// Non-default instance type requires env file
	envFilePath = common.DefaultEnvFileName
	logger.Info("Creating %s to configure instance type: %s", envFilePath, instanceType)
	if err := SetEnvFileVariable(envFilePath, common.EigenMachineTypeEnvVar, instanceType); err != nil {
		return envFilePath, fmt.Errorf("failed to set instance type in env file: %w", err)
	}

	return envFilePath, nil
}
