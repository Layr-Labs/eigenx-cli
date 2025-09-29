package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

const (
	AppRegistryVersion = "1.0.0"
	AppRegistryFile    = "apps.yaml"
)

type AppRegistry struct {
	Version string         `yaml:"version"`
	Apps    map[string]App `yaml:"apps"`
}

type App struct {
	AppID     string    `yaml:"app_id"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

// GetAppRegistryPath returns the path to the app registry file for a specific context
func GetAppRegistryPath(context string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".eigenx", "apps", fmt.Sprintf("%s.yaml", context)), nil
}

// LoadAppRegistry loads the app registry from disk
func LoadAppRegistry(context string) (*AppRegistry, error) {
	path, err := GetAppRegistryPath(context)
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty registry
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &AppRegistry{
			Version: AppRegistryVersion,
			Apps:    make(map[string]App),
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read app registry: %w", err)
	}

	var registry AppRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse app registry: %w", err)
	}

	// Initialize apps map if nil
	if registry.Apps == nil {
		registry.Apps = make(map[string]App)
	}

	return &registry, nil
}

// SaveAppRegistry saves the app registry to disk
func SaveAppRegistry(context string, registry *AppRegistry) error {
	path, err := GetAppRegistryPath(context)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(registry)
	if err != nil {
		return fmt.Errorf("failed to marshal app registry: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write app registry: %w", err)
	}

	return nil
}

// SetAppName sets or updates a name for an app
func SetAppName(context, appIDOrName, newName string) error {
	registry, err := LoadAppRegistry(context)
	if err != nil {
		return err
	}

	// Resolve the target app ID and find any existing name
	targetAppID, err := ResolveAppID(context, appIDOrName)
	if err != nil {
		// If can't resolve, check if it's a valid app ID
		if !common.IsHexAddress(appIDOrName) {
			return fmt.Errorf("invalid app ID or name: %s", appIDOrName)
		}
		targetAppID = appIDOrName
	}

	// Normalize app ID for comparison
	targetAppIDLower := strings.ToLower(targetAppID)

	// Find and remove any existing names for this app ID
	for name, app := range registry.Apps {
		if strings.ToLower(app.AppID) == targetAppIDLower {
			delete(registry.Apps, name)
		}
	}

	// If newName is empty, we're just removing the name
	if newName == "" {
		return SaveAppRegistry(context, registry)
	}

	// Add the new name entry
	registry.Apps[newName] = App{
		AppID:     targetAppID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return SaveAppRegistry(context, registry)
}

// ResolveAppID resolves a name or app ID to an app ID
func ResolveAppID(context, nameOrID string) (string, error) {
	// First check if it's already a valid hex address
	if common.IsHexAddress(nameOrID) {
		return nameOrID, nil
	}

	// Try to load from registry
	registry, err := LoadAppRegistry(context)
	if err != nil {
		return "", err
	}

	// Look up by name
	if app, exists := registry.Apps[nameOrID]; exists {
		return app.AppID, nil
	}

	return "", fmt.Errorf("app not found: %s", nameOrID)
}

// GetAppName returns the name for a given app ID, or empty string if not found
func GetAppName(context, appID string) string {
	registry, err := LoadAppRegistry(context)
	if err != nil {
		return ""
	}

	// Normalize the app ID
	appID = strings.ToLower(appID)

	for name, app := range registry.Apps {
		if strings.ToLower(app.AppID) == appID {
			return name
		}
	}

	return ""
}

// ListApps returns all apps in the registry
func ListApps(context string) (map[string]App, error) {
	registry, err := LoadAppRegistry(context)
	if err != nil {
		return nil, err
	}
	return registry.Apps, nil
}

// FormatAppDisplay returns a user-friendly display string for an app
// Returns "name (0x123...)" if name exists, or just "0x123..." if no name
func FormatAppDisplay(context string, appID common.Address) string {
	if name := GetAppName(context, appID.Hex()); name != "" {
		return fmt.Sprintf("%s (%s)", name, appID.Hex())
	}
	return appID.Hex()
}
