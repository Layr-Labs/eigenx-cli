package common

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/Layr-Labs/eigenx-cli/internal/version"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
)

const (
	// VersionCheckInterval is how often to check for updates (24 hours)
	VersionCheckInterval = 24 * time.Hour
)

// UpdateInfo contains information about an available version update
type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
}

// GetS3VersionURL returns the S3 URL for the VERSION file
var GetS3VersionURL = func() string {
	return "https://s3.amazonaws.com/eigenlayer-eigenx-releases" + BuildSuffix + "/VERSION"
}

// BuildDownloadURL constructs the S3 download URL for a specific version and platform
var BuildDownloadURL = func(version, arch, distro string) string {
	ext := ".tar.gz"
	if strings.Contains(distro, "windows") {
		ext = ".zip"
	}
	return "https://s3.amazonaws.com/eigenlayer-eigenx-releases" + BuildSuffix + "/" +
		version + "/eigenx-cli-" + distro + "-" + arch + "-" + version + ext
}

// GetLatestVersionFromS3 fetches the latest version from the S3 bucket
// If version is "latest", it fetches from the VERSION file
// Otherwise, it returns the specified version (for explicit version upgrades)
func GetLatestVersionFromS3(version string) (string, error) {
	if version == "latest" {
		resp, err := http.Get(GetS3VersionURL())
		if err != nil {
			return "", fmt.Errorf("failed to fetch latest version: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to fetch latest version: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read version: %w", err)
		}

		version = strings.TrimSpace(string(body))
		if version == "" {
			return "", fmt.Errorf("empty version from S3")
		}
	}

	// Verify the version exists by checking if the download URL is accessible
	arch := strings.ToLower(runtime.GOARCH)
	distro := strings.ToLower(runtime.GOOS)
	url := BuildDownloadURL(version, arch, distro)

	resp, err := http.Head(url)
	if err != nil {
		return "", fmt.Errorf("failed to verify version %s exists: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("version %s not found (status %s)", version, resp.Status)
	}

	return version, nil
}

// CheckForUpdate checks if a new version is available, using cached results when possible
func CheckForUpdate(logger iface.Logger) (*UpdateInfo, error) {
	currentVersion := version.GetVersion()

	// Load global config to check cache
	config, err := LoadGlobalConfig()
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
	if lastCheck > 0 && timeSinceCheck < VersionCheckInterval && config.LastKnownVersion != "" {
		logger.Debug("Using cached version check result (last checked %v ago)", timeSinceCheck)
		updateAvailable := semver.Compare(normalizeVersion(currentVersion), normalizeVersion(config.LastKnownVersion)) < 0
		return &UpdateInfo{
			Available:      updateAvailable,
			CurrentVersion: currentVersion,
			LatestVersion:  config.LastKnownVersion,
		}, nil
	}

	// Perform fresh check
	logger.Debug("Performing fresh version check...")
	latestVersion, err := GetLatestVersionFromS3("latest")
	if err != nil {
		logger.Debug("Failed to fetch latest version: %v", err)
		// Return cached result if available, otherwise no update
		if config.LastKnownVersion != "" {
			updateAvailable := semver.Compare(normalizeVersion(currentVersion), normalizeVersion(config.LastKnownVersion)) < 0
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
	if err := SaveGlobalConfig(config); err != nil {
		logger.Debug("Failed to save version check cache: %v", err)
	}

	updateAvailable := semver.Compare(normalizeVersion(currentVersion), normalizeVersion(latestVersion)) < 0
	return &UpdateInfo{
		Available:      updateAvailable,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
	}, nil
}

// PrintUpdateNotification prints a user-friendly notification about an available update
func PrintUpdateNotification(info *UpdateInfo) {
	if info == nil || !info.Available {
		return
	}

	// Build version line with proper padding (59 chars total visible width to match other lines)
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

// normalizeVersion ensures the version has a 'v' prefix for semver.Compare
// Also handles special cases like "unknown" or empty strings
func normalizeVersion(v string) string {
	// Handle special cases
	if v == "unknown" || v == "" {
		return "v0.0.0"
	}

	// Ensure 'v' prefix
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}

	return v
}
