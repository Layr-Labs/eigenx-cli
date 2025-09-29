package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "generate", GenerateCommand.Name)
		assert.Contains(t, GenerateCommand.Aliases, "gen")
		assert.Contains(t, GenerateCommand.Aliases, "new")
		assert.Equal(t, "Generate a new private key and optionally store it in OS keyring", GenerateCommand.Usage)
		assert.NotNil(t, GenerateCommand.Action)

		flagNames := make([]string, 0)
		for _, flag := range GenerateCommand.Flags {
			flagNames = append(flagNames, flag.Names()...)
		}

		assert.Contains(t, flagNames, "environment")
		assert.Contains(t, flagNames, "store")
		assert.Contains(t, flagNames, "s") // store alias
	})
}
