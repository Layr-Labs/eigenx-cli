package iface

import common "github.com/ethereum/go-ethereum/common"

// AppNameResolver resolves app IDs to display names.
type AppNameResolver interface {
	// GetAppName returns the app name by checking remote profile first,
	// then falling back to local registry. Returns empty string if not found.
	GetAppName(appID common.Address) string
}
