package versioncheck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenx-cli/internal/version"
	"github.com/Layr-Labs/eigenx-cli/pkg/commands"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
)

const (
	// CheckInterval is how often to check for updates (24 hours)
	CheckInterval = 24 * time.Hour
)

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
}

// CheckForUpdate checks if a new version is available, using cached results when possible
func CheckForUpdate(logger iface.Logger) (*UpdateInfo, error) {
	currentVersion := version.GetVersion()

	// Load global config to check cache
	config, err := common.LoadGlobalConfig()
	if err != nil {
		logger.Debug("Failed to load config for version check: %v", err)
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: currentVersion,
		}, nil
	}

	// Check if we need to perform a fresh check
	now := time.Now().Unix()
	lastCheck := config.LastVersionCheck
	timeSinceCheck := time.Duration(now-lastCheck) * time.Second

	// If we checked recently, use cached result
	if lastCheck > 0 && timeSinceCheck < CheckInterval && config.LastKnownVersion != "" {
		logger.Debug("Using cached version check result (last checked %v ago)", timeSinceCheck)
		updateAvailable := CompareVersions(currentVersion, config.LastKnownVersion) < 0
		return &UpdateInfo{
			Available:      updateAvailable,
			CurrentVersion: currentVersion,
			LatestVersion:  config.LastKnownVersion,
		}, nil
	}

	// Perform fresh check
	logger.Debug("Performing fresh version check...")
	latestVersion, err := commands.GetLatestVersionFromS3("latest")
	if err != nil {
		logger.Debug("Failed to fetch latest version: %v", err)
		// Return cached result if available, otherwise no update
		if config.LastKnownVersion != "" {
			updateAvailable := CompareVersions(currentVersion, config.LastKnownVersion) < 0
			return &UpdateInfo{
				Available:      updateAvailable,
				CurrentVersion: currentVersion,
				LatestVersion:  config.LastKnownVersion,
			}, nil
		}
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: currentVersion,
		}, nil
	}

	// Update cache
	config.LastVersionCheck = now
	config.LastKnownVersion = latestVersion
	if err := common.SaveGlobalConfig(config); err != nil {
		logger.Debug("Failed to save version check cache: %v", err)
	}

	updateAvailable := CompareVersions(currentVersion, latestVersion) < 0
	return &UpdateInfo{
		Available:      updateAvailable,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
	}, nil
}

// CompareVersions compares two semantic version strings
// Returns -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
// Handles versions with 'v' prefix (e.g., "v1.2.3") and without
func CompareVersions(v1, v2 string) int {
	// Handle special cases
	if v1 == v2 {
		return 0
	}
	if v1 == "unknown" || v1 == "" {
		return -1 // Treat unknown/empty as older
	}
	if v2 == "unknown" || v2 == "" {
		return 1 // Treat unknown/empty as older
	}

	// Parse versions
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	// Compare major, minor, patch
	for i := range 3 {
		if parts1[i] < parts2[i] {
			return -1
		}
		if parts1[i] > parts2[i] {
			return 1
		}
	}

	return 0
}

// parseVersion extracts major.minor.patch from a version string
// Handles formats like "v1.2.3", "1.2.3", "v1.2.3-beta", etc.
func parseVersion(v string) [3]int {
	// Remove 'v' prefix if present
	v = strings.TrimPrefix(v, "v")

	// Extract numeric version (before any - or +)
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(v)

	if len(matches) != 4 {
		// If parsing fails, return [0, 0, 0]
		return [3]int{0, 0, 0}
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return [3]int{major, minor, patch}
}

// PrintUpdateNotification prints a user-friendly notification about an available update
func PrintUpdateNotification(info *UpdateInfo) {
	if !info.Available {
		return
	}

	// Build version line with proper padding (61 chars total visible width)
	versionText := fmt.Sprintf("  Current: %s  Latest: %s", info.CurrentVersion, info.LatestVersion)
	padding := strings.Repeat(" ", 61-len(versionText))
	versionLine := fmt.Sprintf("  Current: \033[2m%s\033[0m  Latest: \033[1;32m%s\033[0m%s",
		info.CurrentVersion, info.LatestVersion, padding)

	fmt.Println()
	fmt.Printf("\033[33m╭─────────────────────────────────────────────────────────────╮\033[0m\n")
	fmt.Printf("\033[33m│\033[0m  \033[1mA new version of eigenx is available!\033[0m                      \033[33m│\033[0m\n")
	fmt.Printf("\033[33m│\033[0m                                                             \033[33m│\033[0m\n")
	fmt.Printf("\033[33m│\033[0m%s\033[33m│\033[0m\n", versionLine)
	fmt.Printf("\033[33m│\033[0m                                                             \033[33m│\033[0m\n")
	fmt.Printf("\033[33m│\033[0m  Run \033[1;36meigenx upgrade\033[0m to update                               \033[33m│\033[0m\n")
	fmt.Printf("\033[33m╰─────────────────────────────────────────────────────────────╯\033[0m\n")
	fmt.Println()
}
