package common

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// GlobalConfig contains user-level configuration that persists across all devkit usage
type GlobalConfig struct {
	// FirstRun tracks if this is the user's first time running devkit
	FirstRun bool `yaml:"first_run"`
	// TelemetryEnabled stores the user's global telemetry preference
	TelemetryEnabled *bool `yaml:"telemetry_enabled,omitempty"`
	// The users uuid to identify user across projects
	UserUUID string `yaml:"user_uuid"`
	// DefaultEnvironment stores the user's preferred deployment environment (sepolia, mainnet-alpha, etc.)
	DefaultEnvironment string `yaml:"default_environment,omitempty"`
	// LastVersionCheck stores the timestamp of the last version check
	LastVersionCheck int64 `yaml:"last_version_check,omitempty"`
	// LastKnownVersion stores the last known latest version from the server
	LastKnownVersion string `yaml:"last_known_version,omitempty"`
	// LastProfileCacheUpdate stores the timestamp of the last profile cache update
	LastProfileCacheUpdate int64 `yaml:"last_profile_cache_update,omitempty"`
	// ProfileCache stores app profiles (appID → profile name) per environment
	ProfileCache map[string]map[string]string `yaml:"profile_cache,omitempty"`
}

// GetGlobalConfigDir returns the XDG-compliant directory where global eigenx config should be stored
func GetGlobalConfigDir() (string, error) {
	// First check XDG_CONFIG_HOME
	configHome := os.Getenv("XDG_CONFIG_HOME")

	var baseDir string
	if configHome != "" && filepath.IsAbs(configHome) {
		baseDir = configHome
	} else {
		// Fall back to ~/.config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".config")
	}

	// Use environment-specific config directory
	configDirName := "eigenx" + BuildSuffix

	return filepath.Join(baseDir, configDirName), nil
}

// GetGlobalConfigPath returns the full path to the global config file
func GetGlobalConfigPath() (string, error) {
	configDir, err := GetGlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, GlobalConfigFile), nil
}

// LoadGlobalConfig loads the global configuration, creating defaults if needed
func LoadGlobalConfig() (*GlobalConfig, error) {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		// If we can't determine config path (e.g., no home directory),
		// return first-run defaults so the CLI doesn't fail completely
		return &GlobalConfig{
			FirstRun: true,
		}, nil
	}

	// If file doesn't exist, return defaults for first run
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &GlobalConfig{
			FirstRun: true,
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveGlobalConfig saves the global configuration to disk
func SaveGlobalConfig(config *GlobalConfig) error {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return fmt.Errorf("cannot save global config (unable to determine config directory): %w", err)
	}

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetGlobalTelemetryPreference returns the global telemetry preference
func GetGlobalTelemetryPreference() (*bool, error) {
	config, err := LoadGlobalConfig()
	if err != nil {
		return nil, err
	}
	return config.TelemetryEnabled, nil
}

// SetGlobalTelemetryPreference sets the global telemetry preference
func SetGlobalTelemetryPreference(enabled bool) error {
	config, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	config.TelemetryEnabled = &enabled
	config.FirstRun = false // No longer first run after setting preference

	return SaveGlobalConfig(config)
}

// MarkFirstRunComplete marks that the first run has been completed
func MarkFirstRunComplete() error {
	config, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	config.FirstRun = false

	return SaveGlobalConfig(config)
}

// IsFirstRun checks if this is the user's first time running devkit
func IsFirstRun() (bool, error) {
	config, err := LoadGlobalConfig()
	if err != nil {
		return false, err
	}
	return config.FirstRun, nil
}

// GetDefaultEnvironment returns the user's preferred deployment environment
func GetDefaultEnvironment() (string, error) {
	config, err := LoadGlobalConfig()
	if err != nil {
		return "", err
	}
	return config.DefaultEnvironment, nil
}

// SetDefaultEnvironment sets the user's preferred deployment environment
func SetDefaultEnvironment(environment string) error {
	config, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	config.DefaultEnvironment = environment

	return SaveGlobalConfig(config)
}

// LoadProfileCache loads the cached app profiles for a given environment
// Returns cached profiles and timestamp, using 24-hour TTL
func LoadProfileCache(environment string) (profiles map[string]string, timestamp int64, err error) {
	config, err := LoadGlobalConfig()
	if err != nil {
		return nil, 0, err
	}

	// Initialize cache map if nil
	if config.ProfileCache == nil {
		return make(map[string]string), 0, nil
	}

	// Get profiles for this environment
	envProfiles, exists := config.ProfileCache[environment]
	if !exists {
		return make(map[string]string), 0, nil
	}

	return envProfiles, config.LastProfileCacheUpdate, nil
}

// SaveProfileCache saves app profiles to cache with current timestamp
func SaveProfileCache(environment string, profiles map[string]string) error {
	config, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	// Initialize cache map if nil
	if config.ProfileCache == nil {
		config.ProfileCache = make(map[string]map[string]string)
	}

	// Save profiles for this environment
	config.ProfileCache[environment] = profiles
	config.LastProfileCacheUpdate = time.Now().Unix()

	return SaveGlobalConfig(config)
}

// InvalidateProfileCache clears the profile cache timestamp to force refresh
func InvalidateProfileCache() error {
	config, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	config.LastProfileCacheUpdate = 0
	config.ProfileCache = nil

	return SaveGlobalConfig(config)
}
