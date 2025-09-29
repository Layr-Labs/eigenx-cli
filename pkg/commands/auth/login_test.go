package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "login", LoginCommand.Name)
		assert.Equal(t, "Store an existing private key in OS keyring", LoginCommand.Usage)
		assert.NotNil(t, LoginCommand.Action)

		flagNames := make([]string, len(LoginCommand.Flags))
		for i, flag := range LoginCommand.Flags {
			flagNames[i] = flag.Names()[0]
		}

		assert.Contains(t, flagNames, "environment")
	})
}
