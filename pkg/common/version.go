package common

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
)

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
