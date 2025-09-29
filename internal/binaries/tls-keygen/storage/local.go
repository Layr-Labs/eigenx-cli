package storage

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

const (
	CertFullChainFileName = "fullchain.pem"
	CertPrivKeyFileName   = "privkey.pem"
)

// LocalFileWriter implements LocalWriter for file system operations
type LocalFileWriter struct{}

// WriteChain writes certificate chain to disk
func (LocalFileWriter) WriteChain(outDir string, chain ChainPEM) (string, error) {
	path, _ := CertPaths(outDir)
	// Ensure directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}
	if err := os.WriteFile(path, chain, 0644); err != nil {
		return "", fmt.Errorf("write certificate: %w", err)
	}
	return path, nil
}

// WriteKey writes private key to disk
func (LocalFileWriter) WriteKey(outDir string, key *ecdsa.PrivateKey) (string, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("marshal EC key: %w", err)
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})

	_, path := CertPaths(outDir)
	// Ensure directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("write private key: %w", err)
	}
	return path, nil
}

// CertPaths returns the standard paths for certificate and key files
func CertPaths(outDir string) (fullChainPath, privKeyPath string) {
	return filepath.Join(outDir, CertFullChainFileName), filepath.Join(outDir, CertPrivKeyFileName)
}


