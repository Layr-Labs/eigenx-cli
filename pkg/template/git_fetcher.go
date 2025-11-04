package template

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/logger"
)

// GitFetcherConfig holds options; we only care about Verbose here
type GitFetcherConfig struct {
	Verbose bool
}

// TODO: implement metric transport
type GitMetrics interface {
	CloneStarted(repo string)
	CloneFinished(repo string, err error)
}

// GitFetcher wraps clone with metrics and reporting
type GitFetcher struct {
	Client  *GitClient
	Metrics GitMetrics
	Config  GitFetcherConfig
	Logger  logger.ProgressLogger
}

func (f *GitFetcher) Fetch(ctx context.Context, repoURL, ref, targetDir string) error {
	if repoURL == "" {
		return fmt.Errorf("repoURL is required")
	}

	// Print job initiation
	f.Logger.Info("\nCloning repo: %s → %s\n\n", repoURL, targetDir)

	// Report to metrics
	if f.Metrics != nil {
		f.Metrics.CloneStarted(repoURL)
	}

	// Build a reporter that knows how to drive our ProgressLogger
	var reporter Reporter
	if !f.Config.Verbose {
		reporter = NewCloneReporter(repoURL, f.Logger, f.Metrics)
	}

	// Initiate clone
	err := f.Client.Clone(ctx, repoURL, ref, targetDir, f.Config, reporter)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// Print job completion
	f.Logger.Info("Clone repo complete: %s\n\n", repoURL)
	return nil
}

// FetchSubdirectory clones only a specific subdirectory using sparse-checkout for efficiency
func (f *GitFetcher) FetchSubdirectory(ctx context.Context, repoURL, ref, subPath, targetDir string) error {
	if repoURL == "" {
		return fmt.Errorf("repoURL is required")
	}
	if subPath == "" {
		return fmt.Errorf("subPath is required")
	}

	// Create temporary directory for sparse clone
	tempDir, err := common.CreateTempDir("eigenx-template-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Use sparse clone to fetch only the needed subdirectory
	f.Logger.Info("\nCloning template: %s → extracting %s\n\n", repoURL, subPath)

	if f.Metrics != nil {
		f.Metrics.CloneStarted(repoURL)
	}

	var reporter Reporter
	if !f.Config.Verbose {
		reporter = NewCloneReporter(repoURL, f.Logger, f.Metrics)
	}

	err = f.Client.CloneSparse(ctx, repoURL, ref, subPath, tempDir, f.Config, reporter)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// Verify subdirectory exists in sparse checkout
	srcPath := filepath.Join(tempDir, subPath)
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("template subdirectory %s not found in %s", subPath, repoURL)
	}

	// Copy subdirectory contents to target
	err = copyDirectory(srcPath, targetDir)
	if err != nil {
		return fmt.Errorf("failed to copy template: %w", err)
	}

	f.Logger.Info("Template extraction complete: %s\n\n", subPath)
	return nil
}

// copyDirectory recursively copies all files and directories from src to dest
func copyDirectory(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the relative path from src
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Calculate destination path
		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		return copyFile(path, destPath, info.Mode())
	})
}

// copyFile copies a single file from src to dest with the given mode
func copyFile(src, dest string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file contents
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	// Set file mode
	return os.Chmod(dest, mode)
}
