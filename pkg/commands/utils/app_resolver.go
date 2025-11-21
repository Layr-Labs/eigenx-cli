package utils

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// AppResolver provides centralized app name/ID resolution with caching.
// It checks remote profile names first, then falls back to local registry (legacy).
type AppResolver struct {
	cCtx             *cli.Context
	environmentName  string
	apps             []ethcommon.Address
	profileNames     map[string]string // appID.Hex() → profile name
	localRegistry    *common.AppRegistry
	cacheInitialized bool
}

// NewAppResolver creates a new resolver instance for the current environment
func NewAppResolver(cCtx *cli.Context) (*AppResolver, error) {
	environmentConfig, err := GetEnvironmentConfig(cCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment config: %w", err)
	}

	return &AppResolver{
		cCtx:            cCtx,
		environmentName: environmentConfig.Name,
		profileNames:    make(map[string]string),
	}, nil
}

// ResolveAppID resolves an app ID or name by checking remote profile names first,
// then falling back to the local registry (legacy)
func (r *AppResolver) ResolveAppID(nameOrID string) (ethcommon.Address, error) {
	// First check if it's already a valid hex address
	if ethcommon.IsHexAddress(nameOrID) {
		return ethcommon.HexToAddress(nameOrID), nil
	}

	// Ensure cache is initialized
	if err := r.ensureCacheInitialized(); err != nil {
		return ethcommon.Address{}, err
	}

	// Check remote profile names first (case-insensitive)
	nameOrIDLower := strings.ToLower(nameOrID)
	for appHex, profileName := range r.profileNames {
		if strings.ToLower(profileName) == nameOrIDLower {
			return ethcommon.HexToAddress(appHex), nil
		}
	}

	// Not found remotely, fall back to local registry
	if app, exists := r.localRegistry.Apps[nameOrID]; exists {
		return ethcommon.HexToAddress(app.AppID), nil
	}

	return ethcommon.Address{}, fmt.Errorf("app not found: %s", nameOrID)
}

// GetAppName returns the app name by checking remote profile first,
// then falling back to local registry
func (r *AppResolver) GetAppName(appID ethcommon.Address) string {
	// Ensure cache is initialized
	if err := r.ensureCacheInitialized(); err != nil {
		return ""
	}

	// Check remote profile first
	if profileName, exists := r.profileNames[appID.Hex()]; exists && profileName != "" {
		return profileName
	}

	// Fall back to local registry (case-insensitive lookup)
	appIDLower := strings.ToLower(appID.Hex())
	for name, app := range r.localRegistry.Apps {
		if strings.ToLower(app.AppID) == appIDLower {
			return name
		}
	}

	return ""
}

// IsNameAvailable checks if a name is available by checking both remote profiles
// and local registry
func (r *AppResolver) IsNameAvailable(name string) bool {
	// Ensure cache is initialized
	if err := r.ensureCacheInitialized(); err != nil {
		return false // If we can't check, assume not available to be safe
	}

	// Check remote profile names (case-insensitive)
	nameLower := strings.ToLower(name)
	for _, profileName := range r.profileNames {
		if strings.ToLower(profileName) == nameLower {
			return false // Name is taken
		}
	}

	// Check local registry
	_, existsInLocal := r.localRegistry.Apps[name]
	return !existsInLocal
}

// FindAvailableName finds an available variant of the base name by appending numbers
func (r *AppResolver) FindAvailableName(baseName string) string {
	if r.IsNameAvailable(baseName) {
		return baseName
	}

	// Try appending numbers until we find an available name
	for i := 2; i <= 100; i++ {
		candidate := fmt.Sprintf("%s-%d", baseName, i)
		if r.IsNameAvailable(candidate) {
			return candidate
		}
	}

	// Fallback: return base name with large random-ish suffix
	return fmt.Sprintf("%s-new", baseName)
}

// GetAllApps returns the cached list of app addresses
func (r *AppResolver) GetAllApps() ([]ethcommon.Address, error) {
	if err := r.ensureCacheInitialized(); err != nil {
		return nil, err
	}
	return r.apps, nil
}

// FormatAppDisplay returns a user-friendly display string for an app
// Returns "name (0x123...)" if name exists, or just "0x123..." if no name
// Checks remote profile first, then falls back to local registry
func (r *AppResolver) FormatAppDisplay(appID ethcommon.Address) string {
	name := r.GetAppName(appID)
	if name != "" {
		return fmt.Sprintf("%s (%s)", name, appID.Hex())
	}
	return appID.Hex()
}

// ensureCacheInitialized lazily loads app list and profile names once per resolver instance
func (r *AppResolver) ensureCacheInitialized() error {
	if r.cacheInitialized {
		return nil
	}

	// Fetch apps from contract
	client, appController, err := GetAppControllerBinding(r.cCtx)
	if err != nil {
		return fmt.Errorf("failed to get app controller binding: %w", err)
	}
	defer client.Close()

	developerAddr, err := GetDeveloperAddress(r.cCtx)
	if err != nil {
		return fmt.Errorf("failed to get developer address: %w", err)
	}

	result, err := appController.GetAppsByDeveloper(&bind.CallOpts{Context: r.cCtx.Context}, developerAddr, big.NewInt(0), big.NewInt(100))
	if err != nil {
		return fmt.Errorf("failed to get apps: %w", err)
	}

	r.apps = result.Apps

	// Try to load cached profile names (24-hour TTL)
	cachedProfiles, cacheTimestamp, err := common.LoadProfileCache(r.environmentName)
	if err != nil {
		cachedProfiles = make(map[string]string)
		cacheTimestamp = 0
	}

	// Check if cache is fresh (< 24 hours old)
	now := time.Now().Unix()
	cacheAge := time.Duration(now-cacheTimestamp) * time.Second
	cacheFresh := cacheTimestamp > 0 && cacheAge < 24*time.Hour

	if cacheFresh {
		// Use cached profiles
		r.profileNames = cachedProfiles
	} else {
		// Cache is stale or missing, fetch from API
		r.profileNames = getProfileNamesForApps(r.cCtx, r.apps)

		// Save to cache
		_ = common.SaveProfileCache(r.environmentName, r.profileNames)
	}

	// Load local registry
	registry, err := common.LoadAppRegistry(r.environmentName)
	if err != nil {
		// Local registry might not exist yet, that's okay
		r.localRegistry = &common.AppRegistry{Apps: make(map[string]common.App)}
	} else {
		r.localRegistry = registry
	}

	r.cacheInitialized = true
	return nil
}
