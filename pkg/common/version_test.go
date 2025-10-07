package common

import (
	"testing"

	"golang.org/x/mod/semver"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"version with v prefix", "v1.2.3", "v1.2.3"},
		{"version without v prefix", "1.2.3", "v1.2.3"},
		{"unknown version", "unknown", "v0.0.0"},
		{"empty version", "", "v0.0.0"},
		{"pre-release version", "v1.2.3-beta", "v1.2.3-beta"},
		{"version without v and pre-release", "1.2.3-rc1", "v1.2.3-rc1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSemverCompare(t *testing.T) {
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

		// Special cases (unknown/empty treated as v0.0.0)
		{"unknown version v1", "unknown", "v1.2.3", -1},
		{"unknown version v2", "v1.2.3", "unknown", 1},
		{"both unknown", "unknown", "unknown", 0},
		{"empty v1", "", "v1.2.3", -1},
		{"empty v2", "v1.2.3", "", 1},

		// Pre-release versions (semver treats pre-release as less than release)
		{"pre-release vs release", "v1.2.3-beta", "v1.2.3", -1},
		{"release vs pre-release", "v1.2.3", "v1.2.3-rc1", 1},
		{"alpha vs beta", "v1.2.3-alpha", "v1.2.3-beta", -1},
		{"beta vs rc", "v1.2.3-beta", "v1.2.3-rc", -1},
		{"rc vs release", "v1.2.3-rc1", "v1.2.3", -1},
		{"pre-release different versions", "v1.2.3-beta", "v1.2.4-alpha", -1},
		{"same pre-release", "v1.2.3-beta.1", "v1.2.3-beta.1", 0},

		// Build metadata (ignored by semver.Compare)
		{"build metadata ignored", "v1.2.3+build123", "v1.2.3+build456", 0},
		{"build metadata vs clean", "v1.2.3+build", "v1.2.3", 0},
		{"pre-release with build", "v1.2.3-beta+build", "v1.2.3-beta", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := semver.Compare(normalizeVersion(tt.v1), normalizeVersion(tt.v2))
			if result != tt.expected {
				t.Errorf("semver.Compare(normalizeVersion(%q), normalizeVersion(%q)) = %d; want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestUpdateAvailability(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		wantUpdate     bool
	}{
		{"update available", "v1.2.3", "v1.2.4", true},
		{"no update needed", "v1.2.4", "v1.2.4", false},
		{"already on newer version", "v1.2.5", "v1.2.4", false},
		{"major version update", "v1.0.0", "v2.0.0", true},
		{"minor version update", "v1.2.0", "v1.3.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateAvailable := semver.Compare(normalizeVersion(tt.currentVersion), normalizeVersion(tt.latestVersion)) < 0
			if updateAvailable != tt.wantUpdate {
				t.Errorf("For current=%s, latest=%s: got update=%v, want %v", tt.currentVersion, tt.latestVersion, updateAvailable, tt.wantUpdate)
			}
		})
	}
}
