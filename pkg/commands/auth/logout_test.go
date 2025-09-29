package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/testutils"
)

func TestLogoutCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "logout", LogoutCommand.Name)
		assert.Equal(t, "Remove private key from OS keyring", LogoutCommand.Usage)
		assert.NotNil(t, LogoutCommand.Action)

		flagNames := make([]string, len(LogoutCommand.Flags))
		for i, flag := range LogoutCommand.Flags {
			flagNames[i] = flag.Names()[0]
		}

		assert.Contains(t, flagNames, "environment")
		assert.Contains(t, flagNames, "force")
	})
}

func TestLogoutAction(t *testing.T) {
	mock := testutils.SetupMockKeyring(t)

	t.Run("key not found", func(t *testing.T) {
		mock.Clear()

		flags := append(common.GlobalFlags, common.EnvironmentFlag, common.ForceFlag)
		app, noopLogger := testutils.CreateTestAppWithNoopLoggerAndAccess("test-app", flags, func(cCtx *cli.Context) error {
			return logoutAction(cCtx)
		})

		err := app.Run([]string{"test-app"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no key found")
		assert.Empty(t, noopLogger.GetEntries())
	})

	t.Run("successful logout", func(t *testing.T) {
		mock.Clear()
		err := mock.StorePrivateKey("sepolia", "0x1234567890123456789012345678901234567890123456789012345678901234")
		require.NoError(t, err)

		flags := append(common.GlobalFlags, common.EnvironmentFlag, common.ForceFlag)
		app, noopLogger := testutils.CreateTestAppWithNoopLoggerAndAccess("test-app", flags, func(cCtx *cli.Context) error {
			return logoutAction(cCtx)
		})

		err = app.Run([]string{"test-app", "--force"})
		require.NoError(t, err)

		logs := noopLogger.GetEntries()
		assert.Len(t, logs, 1)
		assert.Contains(t, logs[0].Message, "Successfully logged out")
	})
}
