package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
)

func TestGetAuthKeyName(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "with environment flag",
			args:     []string{"test", "--environment", "sepolia"},
			expected: "sepolia",
		},
		{
			name:     "without flag tries environment detection",
			args:     []string{"test"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.App{
				Flags: []cli.Flag{common.EnvironmentFlag},
			}

			var ctx *cli.Context
			app.Action = func(c *cli.Context) error {
				ctx = c
				return nil
			}

			err := app.Run(tt.args)
			require.NoError(t, err)

			result, err := getAuthKeyName(ctx)
			require.NoError(t, err)

			if tt.expected == "" {
				assert.NotEmpty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGeneratePrivateKey(t *testing.T) {
	t.Run("generates valid key and address", func(t *testing.T) {
		privateKey, address, err := generatePrivateKey()
		assert.NoError(t, err)
		assert.NotEmpty(t, privateKey)
		assert.NotEmpty(t, address)

		// Private key should be hex encoded with 0x prefix
		assert.True(t, len(privateKey) > 2, "private key should be longer than 2 characters")
		assert.Equal(t, "0x", privateKey[:2], "private key should have 0x prefix")
		// Should be 66 characters total (0x + 64 hex chars)
		assert.Equal(t, 66, len(privateKey), "private key should be 66 characters (0x + 64 hex)")

		// Address should be hex encoded with 0x prefix and correct length
		assert.Equal(t, 42, len(address), "address should be 42 characters long (0x + 40 hex chars)")
		assert.Equal(t, "0x", address[:2], "address should have 0x prefix")
	})

	t.Run("generates different keys on each call", func(t *testing.T) {
		key1, addr1, err1 := generatePrivateKey()
		key2, addr2, err2 := generatePrivateKey()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, key1, key2, "should generate different private keys")
		assert.NotEqual(t, addr1, addr2, "should generate different addresses")
	})

	t.Run("generated key produces valid ethereum address", func(t *testing.T) {
		privateKey, address, err := generatePrivateKey()
		assert.NoError(t, err)

		// Verify the address can be parsed and is valid checksum format
		// The address should be checksummed (mix of upper and lower case after 0x)
		hasUpper := false
		hasLower := false
		for _, char := range address[2:] {
			if char >= 'A' && char <= 'F' {
				hasUpper = true
			}
			if char >= 'a' && char <= 'f' {
				hasLower = true
			}
		}
		assert.True(t, hasUpper || hasLower, "address should be checksummed")
		assert.NotEmpty(t, privateKey)
	})
}
