package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// GetAppNameFromLocalRegistry returns the name for a given app ID from the local registry only (legacy fallback).
// Returns empty string if not found. For new code, use utils.GetAppName which checks remote profiles first.
func GetAppNameFromLocalRegistry(context, appID string) string {
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
