package versioncheck

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		// Equal versions
		{"equal versions", "v1.2.3", "v1.2.3", 0},
		{"equal without v prefix", "1.2.3", "1.2.3", 0},
		{"equal mixed format", "v1.2.3", "1.2.3", 0},

		// v1 < v2
		{"major version older", "v1.2.3", "v2.0.0", -1},
		{"minor version older", "v1.2.3", "v1.3.0", -1},
		{"patch version older", "v1.2.3", "v1.2.4", -1},
		{"much older", "v0.0.1", "v1.2.3", -1},

		// v1 > v2
		{"major version newer", "v2.0.0", "v1.2.3", 1},
		{"minor version newer", "v1.3.0", "v1.2.3", 1},
		{"patch version newer", "v1.2.4", "v1.2.3", 1},
		{"much newer", "v2.5.10", "v1.0.0", 1},

		// Special cases
		{"unknown version v1", "unknown", "v1.2.3", -1},
		{"unknown version v2", "v1.2.3", "unknown", 1},
		{"both unknown", "unknown", "unknown", 0},
		{"empty v1", "", "v1.2.3", -1},
		{"empty v2", "v1.2.3", "", 1},

		// Pre-release versions (should compare base version only)
		{"pre-release v1", "v1.2.3-beta", "v1.2.3", 0},
		{"pre-release v2", "v1.2.3", "v1.2.3-rc1", 0},
		{"pre-release both", "v1.2.3-alpha", "v1.2.3-beta", 0},
		{"pre-release different versions", "v1.2.3-beta", "v1.2.4-alpha", -1},

		// Build metadata (should compare base version only)
		{"build metadata", "v1.2.3+build123", "v1.2.3+build456", 0},
		{"build and pre-release", "v1.2.3-beta+build", "v1.2.3", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d; want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected [3]int
	}{
		{"standard version", "1.2.3", [3]int{1, 2, 3}},
		{"version with v prefix", "v1.2.3", [3]int{1, 2, 3}},
		{"version with pre-release", "v1.2.3-beta", [3]int{1, 2, 3}},
		{"version with build metadata", "v1.2.3+build123", [3]int{1, 2, 3}},
		{"version with both", "v1.2.3-alpha+build", [3]int{1, 2, 3}},
		{"large version numbers", "v10.20.30", [3]int{10, 20, 30}},
		{"zero version", "v0.0.0", [3]int{0, 0, 0}},
		{"invalid version", "invalid", [3]int{0, 0, 0}},
		{"partial version", "v1.2", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersion(tt.version)
			if result != tt.expected {
				t.Errorf("parseVersion(%q) = %v; want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestUpdateInfo(t *testing.T) {
	// Test that CompareVersions correctly determines if an update is available
	currentVersion := "v1.2.3"
	latestVersion := "v1.2.4"

	updateAvailable := CompareVersions(currentVersion, latestVersion) < 0
	if !updateAvailable {
		t.Errorf("Expected update to be available when current=%s and latest=%s", currentVersion, latestVersion)
	}

	// Test no update needed
	currentVersion = "v1.2.4"
	latestVersion = "v1.2.4"

	updateAvailable = CompareVersions(currentVersion, latestVersion) < 0
	if updateAvailable {
		t.Errorf("Expected no update when versions are equal: current=%s, latest=%s", currentVersion, latestVersion)
	}

	// Test already on newer version
	currentVersion = "v1.2.5"
	latestVersion = "v1.2.4"

	updateAvailable = CompareVersions(currentVersion, latestVersion) < 0
	if updateAvailable {
		t.Errorf("Expected no update when current version is newer: current=%s, latest=%s", currentVersion, latestVersion)
	}
}
