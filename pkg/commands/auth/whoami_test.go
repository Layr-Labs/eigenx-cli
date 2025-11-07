package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/testutils"
)

func TestWhoamiCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "whoami", WhoamiCommand.Name)
		assert.Equal(t, "Show current authentication status and address", WhoamiCommand.Usage)
		assert.NotNil(t, WhoamiCommand.Action)

		flagNames := make([]string, len(WhoamiCommand.Flags))
		for i, flag := range WhoamiCommand.Flags {
			flagNames[i] = flag.Names()[0]
		}

		assert.Contains(t, flagNames, "environment")
	})
}

func TestWhoamiAction(t *testing.T) {
	originalEnv := os.Getenv(common.EigenXPrivateKeyEnvVar)
	defer func() {
		if originalEnv == "" {
			os.Unsetenv(common.EigenXPrivateKeyEnvVar)
		} else {
			os.Setenv(common.EigenXPrivateKeyEnvVar, originalEnv)
		}
	}()

	mock := testutils.SetupMockKeyring(t)

	t.Run("with environment variable", func(t *testing.T) {
		testKey := "0x1234567890123456789012345678901234567890123456789012345678901234"
		os.Setenv(common.EigenXPrivateKeyEnvVar, testKey)
		defer os.Unsetenv(common.EigenXPrivateKeyEnvVar)

		app, noopLogger := testutils.CreateTestAppWithNoopLoggerAndAccess("test-app", common.GlobalFlags, func(cCtx *cli.Context) error {
			return whoamiAction(cCtx)
		})

		stdout, stderr := testutils.CaptureOutput(func() {
			err := app.Run([]string{"test-app"})
			require.NoError(t, err)
		})

		assert.Contains(t, stdout, "Address: 0x2e988A386a799F506693793c6A5AF6B54dfAaBfB")
		assert.Contains(t, stdout, "Source:  environment variable")
		assert.Empty(t, stderr)
		assert.Empty(t, noopLogger.GetEntries())
	})

	t.Run("with stored credentials", func(t *testing.T) {
		os.Unsetenv(common.EigenXPrivateKeyEnvVar)
		mock.Clear()
		err := mock.StorePrivateKey("sepolia", "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd")
		require.NoError(t, err)

		app, noopLogger := testutils.CreateTestAppWithNoopLoggerAndAccess("test-app", common.GlobalFlags, func(cCtx *cli.Context) error {
			return whoamiAction(cCtx)
		})

		stdout, stderr := testutils.CaptureOutput(func() {
			err := app.Run([]string{"test-app"})
			require.NoError(t, err)
		})

		assert.Contains(t, stdout, "Address: 0x5d46aC553A974ef992A08eeef0A05990802F01F6")
		assert.Contains(t, stdout, "Source:  stored credentials (sepolia)")
		assert.Empty(t, stderr)
		assert.Empty(t, noopLogger.GetEntries())
	})

	t.Run("not authenticated", func(t *testing.T) {
		os.Unsetenv(common.EigenXPrivateKeyEnvVar)
		mock.Clear()

		app, noopLogger := testutils.CreateTestAppWithNoopLoggerAndAccess("test-app", common.GlobalFlags, func(cCtx *cli.Context) error {
			return whoamiAction(cCtx)
		})

		stdout, stderr := testutils.CaptureOutput(func() {
			err := app.Run([]string{"test-app"})
			require.NoError(t, err)
		})

		assert.Contains(t, stdout, "Not authenticated")
		assert.Contains(t, stdout, "To authenticate, use one of:")
		assert.Contains(t, stdout, "eigenx auth login")
		assert.Empty(t, stderr)
		assert.Empty(t, noopLogger.GetEntries())
	})

	t.Run("with environment flag", func(t *testing.T) {
		os.Unsetenv(common.EigenXPrivateKeyEnvVar)
		mock.Clear()
		err := mock.StorePrivateKey("sepolia", "0x2222222222222222222222222222222222222222222222222222222222222222")
		require.NoError(t, err)

		flags := append(common.GlobalFlags, common.EnvironmentFlag)
		app, noopLogger := testutils.CreateTestAppWithNoopLoggerAndAccess("test-app", flags, func(cCtx *cli.Context) error {
			return whoamiAction(cCtx)
		})

		stdout, stderr := testutils.CaptureOutput(func() {
			err := app.Run([]string{"test-app", "--environment", "sepolia"})
			require.NoError(t, err)
		})

		assert.Contains(t, stdout, "Address: 0x1563915e194D8CfBA1943570603F7606A3115508")
		assert.Contains(t, stdout, "Source:  stored credentials (sepolia)")
		assert.Empty(t, stderr)
		assert.Empty(t, noopLogger.GetEntries())
	})
}
