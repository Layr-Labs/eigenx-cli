package testutils

import (
	"testing"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/zalando/go-keyring"
)

// MockKeyring provides a test helper for keyring operations using the native mock
type MockKeyring struct {
	t *testing.T
}

// SetupMockKeyring initializes the native keyring mock for testing
func SetupMockKeyring(t *testing.T) *MockKeyring {
	keyring.MockInit()

	// Clear any existing keys at start
	mock := &MockKeyring{t: t}
	mock.Clear()

	t.Cleanup(func() {
		mock.Clear()
	})

	return mock
}

// StorePrivateKey stores a key using the real implementation (which talks to the mock)
func (m *MockKeyring) StorePrivateKey(environment, privateKey string) error {
	return common.StorePrivateKey(environment, privateKey)
}

// Clear removes all stored keys from the mock
func (m *MockKeyring) Clear() {
	_ = keyring.DeleteAll(common.KeyringServiceName)
}
