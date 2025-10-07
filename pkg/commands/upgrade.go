package commands

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Layr-Labs/eigenx-cli/internal/version"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
)

// UpgradeCommand defines the CLI command to upgrade the eigenx binary
var UpgradeCommand = &cli.Command{
	Name:  "upgrade",
	Usage: "Upgrade eigenx binary",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "version",
			Usage: "Version to upgrade to (e.g. v0.0.8)",
			Value: "latest",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		return UpgradeEigenX(cCtx)
	},
}

// UpgradeEigenX resolves the latest version if needed and invokes PerformUpgrade to install the new version
func UpgradeEigenX(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Get current version
	currentVersion := version.GetVersion()
	currentCommit := version.GetCommit()

	// Get the version to be installed
	requestedVersion := cCtx.String("version")
	// Default requestedVersion to "latest"
	if requestedVersion == "" {
		requestedVersion = "latest"
	}

	// Get target version from S3
	targetVersion, err := common.GetLatestVersionFromS3(requestedVersion)
	if err != nil {
		return fmt.Errorf("failed to get version %s: %w", requestedVersion, err)
	}

	// Log upgrade
	bucketType := "stable"
	if common.BuildSuffix != "" {
		bucketType = "dev"
	}
	logger.Info("Upgrading eigenx from %s (%s) to %s [%s bucket]...", currentVersion, currentCommit, targetVersion, bucketType)

	// Avoid redundant upgrade
	if currentVersion == targetVersion {
		return fmt.Errorf("already on version: %s", currentVersion)
	}

	// Determine install location
	var path string

	// Try to locate the current eigenx binary, considering Windows .exe extension
	if runtime.GOOS == "windows" {
		// On Windows, try both eigenx and eigenx.exe
		path, err = exec.LookPath("eigenx.exe")
		if err != nil {
			path, err = exec.LookPath("eigenx")
		}
	} else {
		path, err = exec.LookPath("eigenx")
	}

	if err != nil {
		return fmt.Errorf("could not locate current eigenx binary: %w", err)
	}
	binDir := filepath.Dir(path)

	// Perform the upgrade and source
	return PerformUpgrade(targetVersion, binDir, logger)
}

// PerformUpgrade downloads and installs the target version of the eigenx binary.
// It supports both .tar.gz and raw .tar archive formats.
func PerformUpgrade(version, binDir string, logger iface.Logger) error {
	arch := strings.ToLower(runtime.GOARCH)
	distro := strings.ToLower(runtime.GOOS)

	url := common.BuildDownloadURL(version, arch, distro)

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin dir: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response from server: %s", resp.Status)
	}

	// Extract archive based on format
	if strings.HasSuffix(url, ".zip") || strings.Contains(resp.Header.Get("Content-Type"), "application/zip") {
		return extractZipArchive(resp.Body, binDir, logger)
	}
	return extractTarArchive(resp.Body, resp.Header.Get("Content-Type"), binDir, logger)

}

// extractZipArchive extracts a ZIP archive to the specified directory
func extractZipArchive(body io.Reader, binDir string, logger iface.Logger) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("error reading ZIP data: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("error parsing ZIP archive: %w", err)
	}

	for _, file := range zipReader.File {
		if err := extractFileFromZip(file, binDir, logger); err != nil {
			return err
		}
	}

	logger.Info("Upgrade complete.")
	return nil
}

// extractFileFromZip extracts a single file from a ZIP archive with security validation
func extractFileFromZip(file *zip.File, binDir string, logger iface.Logger) error {
	targetPath, err := validateAndResolvePath(file.Name, binDir)
	if err != nil {
		return err
	}

	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("error opening file in ZIP: %w", err)
	}
	defer rc.Close()

	return writeFileWithPermissions(rc, targetPath, logger)
}

// extractTarArchive extracts a tar archive (with optional gzip compression) to the specified directory
func extractTarArchive(body io.Reader, contentType string, binDir string, logger iface.Logger) error {
	var tr *tar.Reader
	var gzr *gzip.Reader

	switch {
	case strings.Contains(contentType, "gzip"):
		var err error
		gzr, err = gzip.NewReader(body)
		if err != nil {
			return fmt.Errorf("gzip parse error: %w", err)
		}
		defer gzr.Close()
		tr = tar.NewReader(gzr)

	case strings.Contains(contentType, "x-tar"), strings.Contains(contentType, "application/octet-stream"):
		tr = tar.NewReader(body)

	default:
		bodyBytes, _ := io.ReadAll(body)
		return fmt.Errorf("unexpected content type: %s\nBody: %s", contentType, string(bodyBytes))
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tarball: %w", err)
		}

		targetPath, err := validateAndResolvePath(hdr.Name, binDir)
		if err != nil {
			return err
		}

		if err := writeFileWithPermissions(tr, targetPath, logger); err != nil {
			return err
		}
	}

	logger.Info("Upgrade complete.")
	return nil
}

// validateAndResolvePath validates and resolves a file path for archive extraction
func validateAndResolvePath(archivePath, binDir string) (string, error) {
	cleanName := filepath.Clean(archivePath)
	if strings.Contains(cleanName, "..") || filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("invalid file path in archive: %s", archivePath)
	}

	targetPath := filepath.Join(binDir, cleanName)
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("error resolving file path: %w", err)
	}

	absBinDir, err := filepath.Abs(binDir)
	if err != nil {
		return "", fmt.Errorf("error resolving binDir path: %w", err)
	}

	if !strings.HasPrefix(absTargetPath, absBinDir) {
		return "", fmt.Errorf("file path escapes target directory: %s", absTargetPath)
	}

	return absTargetPath, nil
}

// writeFileWithPermissions writes data from a reader to a file with executable permissions
func writeFileWithPermissions(src io.Reader, targetPath string, logger iface.Logger) error {
	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, src); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("error setting permissions: %w", err)
	}

	logger.Info("Installed: %s", targetPath)
	return nil
}
