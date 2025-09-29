package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/testutils"
)

func TestListAction(t *testing.T) {
	mock := testutils.SetupMockKeyring(t)

	t.Run("empty keyring", func(t *testing.T) {
		app, noopLogger := testutils.CreateTestAppWithNoopLoggerAndAccess("test-app", common.GlobalFlags, func(cCtx *cli.Context) error {
			return listAction(cCtx)
		})

		stdout, stderr := testutils.CaptureOutput(func() {
			err := app.Run([]string{"test-app"})
			require.NoError(t, err)
		})

		assert.Contains(t, stdout, "No keys stored in keyring")
		assert.Contains(t, stdout, "To store a key, use:")
		assert.Contains(t, stdout, "eigenx auth login")
		assert.Empty(t, stderr)
		assert.Empty(t, noopLogger.GetEntries())
	})

	t.Run("with stored keys", func(t *testing.T) {
		mock.Clear()
		err := mock.StorePrivateKey("sepolia", "0x1234567890123456789012345678901234567890123456789012345678901234")
		require.NoError(t, err)

		app, noopLogger := testutils.CreateTestAppWithNoopLoggerAndAccess("test-app", common.GlobalFlags, func(cCtx *cli.Context) error {
			return listAction(cCtx)
		})

		stdout, stderr := testutils.CaptureOutput(func() {
			err := app.Run([]string{"test-app"})
			require.NoError(t, err)
		})

		assert.Contains(t, stdout, "Stored private keys:")
		assert.Contains(t, stdout, "sepolia")
		assert.Contains(t, stdout, "0x2e988A386a799F506693793c6A5AF6B54dfAaBfB")
		assert.Contains(t, stdout, "Usage:")
		assert.Empty(t, stderr)
		assert.Empty(t, noopLogger.GetEntries())
	})

	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "list", ListCommand.Name)
		assert.Equal(t, "List all stored private keys by deployment environment", ListCommand.Usage)
		assert.Equal(t, common.GlobalFlags, ListCommand.Flags)
		assert.NotNil(t, ListCommand.Action)
	})
}
