package commands

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Layr-Labs/eigenx-cli/pkg/common/logger"
	"github.com/stretchr/testify/assert"
)

func TestUpgrade_PerformUpgrade(t *testing.T) {
	// Create a fake tar.gz containing a single dummy binary file
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	content := []byte("#!/bin/sh\necho EigenX CLI upgraded\n")
	hdr := &tar.Header{
		Name: "eigenx",
		Mode: 0755,
		Size: int64(len(content)),
	}
	err := tw.WriteHeader(hdr)
	assert.NoError(t, err)
	_, err = tw.Write(content)
	assert.NoError(t, err)
	tw.Close()
	gz.Close()

	// Start a test server that returns the tarball
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	// Patch the URL builder temporarily (for testing)
	oldURLBuilder := buildDownloadURL
	buildDownloadURL = func(version, arch, distro string) string {
		return ts.URL // fake URL instead of real S3
	}
	defer func() { buildDownloadURL = oldURLBuilder }()

	tmpDir := t.TempDir()
	log := logger.NewNoopLogger()

	err = PerformUpgrade("v0.0.1", tmpDir, log)
	assert.NoError(t, err)

	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "eigenx", files[0].Name())

	path := filepath.Join(tmpDir, "eigenx")
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "echo EigenX CLI upgraded")
}

func TestUpgrade_GetLatestVersionFromS3(t *testing.T) {
	// Test server to mock S3
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle both HEAD and GET requests
		switch {
		case r.URL.Path == "/VERSION" && r.Method == "GET":
			_, _ = w.Write([]byte("v9.9.9"))
		case strings.Contains(r.URL.Path, "v9.9.9") && r.Method == "HEAD":
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "v1.2.3") && r.Method == "HEAD":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// Patch S3 URL to use mock server
	originalVersionURL := s3VersionURL
	originalBuildURL := buildDownloadURL
	s3VersionURL = ts.URL + "/VERSION"
	buildDownloadURL = func(version, arch, distro string) string {
		return ts.URL + "/" + version + "/eigenx-cli-" + distro + "-" + arch + "-" + version + ".tar.gz"
	}
	defer func() {
		s3VersionURL = originalVersionURL
		buildDownloadURL = originalBuildURL
	}()

	// Test fetching latest version
	version, err := GetLatestVersionFromS3("latest")
	assert.NoError(t, err)
	assert.Equal(t, "v9.9.9", version)

	// Test explicit version
	version, err = GetLatestVersionFromS3("v1.2.3")
	assert.NoError(t, err)
	assert.Equal(t, "v1.2.3", version)

	// Test non-existent version
	_, err = GetLatestVersionFromS3("v0.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
