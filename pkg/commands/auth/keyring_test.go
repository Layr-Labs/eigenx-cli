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

func TestListStoredKeys(t *testing.T) {
	mock := testutils.SetupMockKeyring(t)

	t.Run("empty store", func(t *testing.T) {
		keys, err := ListStoredKeys()
		require.NoError(t, err)
		assert.Empty(t, keys)
	})

	t.Run("with stored keys", func(t *testing.T) {
		err := mock.StorePrivateKey("sepolia", "0x1234567890123456789012345678901234567890123456789012345678901234")
		require.NoError(t, err)

		keys, err := ListStoredKeys()
		require.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.Equal(t, "0x2e988A386a799F506693793c6A5AF6B54dfAaBfB", keys["sepolia"])
	})

	t.Run("skips invalid keys", func(t *testing.T) {
		mock.Clear()
		err := mock.StorePrivateKey("sepolia", "not-a-valid-key")
		require.NoError(t, err)

		keys, err := ListStoredKeys()
		require.NoError(t, err)
		assert.Empty(t, keys)
	})
}

func TestGetPrivateKeyWithSource(t *testing.T) {
	originalEnv := os.Getenv(common.EigenXPrivateKeyEnvVar)
	defer func() {
		if originalEnv == "" {
			os.Unsetenv(common.EigenXPrivateKeyEnvVar)
		} else {
			os.Setenv(common.EigenXPrivateKeyEnvVar, originalEnv)
		}
	}()

	mock := testutils.SetupMockKeyring(t)

	t.Run("from command flag", func(t *testing.T) {
		os.Unsetenv(common.EigenXPrivateKeyEnvVar)

		app := &cli.App{
			Flags: []cli.Flag{common.PrivateKeyFlag},
		}

		var ctx *cli.Context
		app.Action = func(c *cli.Context) error {
			ctx = c
			return nil
		}

		testKey := "0x1234567890123456789012345678901234567890123456789012345678901234"
		err := app.Run([]string{"test", "--private-key", testKey})
		require.NoError(t, err)

		key, source, err := GetPrivateKeyWithSource(ctx)
		require.NoError(t, err)
		assert.Equal(t, testKey, key)
		assert.Equal(t, "command flag", source)
	})

	t.Run("from environment variable", func(t *testing.T) {
		testKey := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd"
		os.Setenv(common.EigenXPrivateKeyEnvVar, testKey)
		defer os.Unsetenv(common.EigenXPrivateKeyEnvVar)

		app := &cli.App{}
		var ctx *cli.Context
		app.Action = func(c *cli.Context) error {
			ctx = c
			return nil
		}

		err := app.Run([]string{"test"})
		require.NoError(t, err)

		key, source, err := GetPrivateKeyWithSource(ctx)
		require.NoError(t, err)
		assert.Equal(t, testKey, key)
		assert.Equal(t, "environment variable", source)
	})

	t.Run("from stored credentials", func(t *testing.T) {
		os.Unsetenv(common.EigenXPrivateKeyEnvVar)
		mock.Clear()
		err := mock.StorePrivateKey("sepolia", "0x1111111111111111111111111111111111111111111111111111111111111111")
		require.NoError(t, err)

		app := &cli.App{}
		var ctx *cli.Context
		app.Action = func(c *cli.Context) error {
			ctx = c
			return nil
		}

		err = app.Run([]string{"test"})
		require.NoError(t, err)

		key, source, err := GetPrivateKeyWithSource(ctx)
		require.NoError(t, err)
		assert.Equal(t, "0x1111111111111111111111111111111111111111111111111111111111111111", key)
		assert.Equal(t, "stored credentials (sepolia)", source)
	})

	t.Run("priority order", func(t *testing.T) {
		flagKey := "0x1111111111111111111111111111111111111111111111111111111111111111"
		envKey := "0x2222222222222222222222222222222222222222222222222222222222222222"

		os.Setenv(common.EigenXPrivateKeyEnvVar, envKey)
		defer os.Unsetenv(common.EigenXPrivateKeyEnvVar)

		app := &cli.App{
			Flags: []cli.Flag{common.PrivateKeyFlag},
		}
		var ctx *cli.Context
		app.Action = func(c *cli.Context) error {
			ctx = c
			return nil
		}

		err := app.Run([]string{"test", "--private-key", flagKey})
		require.NoError(t, err)

		key, source, err := GetPrivateKeyWithSource(ctx)
		require.NoError(t, err)
		assert.Equal(t, flagKey, key)
		assert.Equal(t, "command flag", source)
	})

	t.Run("no key available", func(t *testing.T) {
		os.Unsetenv(common.EigenXPrivateKeyEnvVar)
		mock.Clear()

		app := &cli.App{}
		var ctx *cli.Context
		app.Action = func(c *cli.Context) error {
			ctx = c
			return nil
		}

		err := app.Run([]string{"test"})
		require.NoError(t, err)

		key, source, err := GetPrivateKeyWithSource(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Private key required")
		assert.Empty(t, key)
		assert.Empty(t, source)
	})
}

func TestGetAddressFromPrivateKey(t *testing.T) {
	tests := []struct {
		name        string
		privateKey  string
		wantAddress string
		wantError   bool
	}{
		{
			name:        "valid key with 0x prefix",
			privateKey:  "0x1234567890123456789012345678901234567890123456789012345678901234",
			wantAddress: "0x2e988A386a799F506693793c6A5AF6B54dfAaBfB",
			wantError:   false,
		},
		{
			name:        "valid key without 0x prefix",
			privateKey:  "1234567890123456789012345678901234567890123456789012345678901234",
			wantAddress: "0x2e988A386a799F506693793c6A5AF6B54dfAaBfB",
			wantError:   false,
		},
		{
			name:        "another valid key",
			privateKey:  "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
			wantAddress: "0x5d46aC553A974ef992A08eeef0A05990802F01F6",
			wantError:   false,
		},
		{
			name:        "invalid key - too short",
			privateKey:  "0x1234",
			wantAddress: "",
			wantError:   true,
		},
		{
			name:        "invalid key - not hex",
			privateKey:  "0xnotahexstring",
			wantAddress: "",
			wantError:   true,
		},
		{
			name:        "empty key",
			privateKey:  "",
			wantAddress: "",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := common.GetAddressFromPrivateKey(tt.privateKey)
			if tt.wantError {
				assert.Error(t, err)
				assert.Empty(t, addr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantAddress, addr)
			}
		})
	}
}

func TestValidatePrivateKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantError bool
	}{
		{
			name:      "valid key with 0x prefix",
			key:       "0x1234567890123456789012345678901234567890123456789012345678901234",
			wantError: false,
		},
		{
			name:      "valid key without 0x prefix",
			key:       "1234567890123456789012345678901234567890123456789012345678901234",
			wantError: false,
		},
		{
			name:      "invalid key",
			key:       "invalid",
			wantError: true,
		},
		{
			name:      "empty key",
			key:       "",
			wantError: true,
		},
		{
			name:      "key too short",
			key:       "0x1234",
			wantError: true,
		},
		{
			name:      "key too long",
			key:       "0x12345678901234567890123456789012345678901234567890123456789012341234",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := common.ValidatePrivateKey(tt.key)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
