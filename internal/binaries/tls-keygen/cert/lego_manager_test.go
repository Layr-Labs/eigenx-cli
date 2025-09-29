package cert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/config"
	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/keys"
	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/storage"
)

// Mock storage for testing
type mockLegoStorage struct {
	chain    storage.ChainPEM
	meta     storage.Metadata
	loadErr  error
	storeErr error
}

func (m *mockLegoStorage) Load(domain string) (storage.ChainPEM, storage.Metadata, error) {
	return m.chain, m.meta, m.loadErr
}

func (m *mockLegoStorage) Store(domain string, chain storage.ChainPEM) error {
	return m.storeErr
}

// Mock local writer
type mockLegoLocalWriter struct {
	chainPath string
	keyPath   string
	err       error
}

func (m *mockLegoLocalWriter) WriteChain(dir string, chain storage.ChainPEM) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.chainPath == "" {
		m.chainPath = dir + "/fullchain.pem"
	}
	return m.chainPath, nil
}

func (m *mockLegoLocalWriter) WriteKey(dir string, key *ecdsa.PrivateKey) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.keyPath == "" {
		m.keyPath = dir + "/privkey.pem"
	}
	return m.keyPath, nil
}

// TestNewLegoManager tests manager creation
func TestNewLegoManager(t *testing.T) {
	storage := &mockLegoStorage{}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)
	if manager == nil {
		t.Fatal("NewLegoManager returned nil")
	}
	if manager.storage != storage {
		t.Error("storage not set correctly")
	}
	if manager.local != localWriter {
		t.Error("local writer not set correctly")
	}
	if manager.log != logger {
		t.Error("logger not set correctly")
	}
}

// TestLegoManager_SetClock tests clock injection
func TestLegoManager_SetClock(t *testing.T) {
	manager := NewLegoManager(&mockLegoStorage{}, &mockLegoLocalWriter{}, slog.Default())

	customTime := func() time.Time {
		return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	manager.SetClock(customTime)
	if manager.clock() != customTime() {
		t.Error("clock not set correctly")
	}
}

// TestLegoManager_RemoteValid tests when a valid cert exists in remote storage
func TestLegoManager_RemoteValid(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	// Derive the same key that the manager will derive
	seed := keys.SeedFromMnemonic(opts.Mnemonic)
	tlsKey := keys.DeriveTLSKey(seed, "example.com", 0)

	// Create test certificate with the correct key
	expiry := time.Now().Add(60 * 24 * time.Hour)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &tlsKey.PublicKey, tlsKey)
	testChain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	storage := &mockLegoStorage{
		chain: testChain,
		meta: storage.Metadata{
			ExpiresAt: expiry,
		},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	bundle, err := manager.EnsureCertificate(ctx, opts)
	if err != nil {
		t.Fatalf("EnsureCertificate() error = %v", err)
	}

	// Should install from remote without issuing new cert
	if bundle.Issued {
		t.Error("Should not have issued new certificate")
	}
	if !bundle.Reconstructed {
		t.Error("Should have reconstructed from remote")
	}
}

// TestLegoManager_RemoteLoadError tests when remote storage returns an error
func TestLegoManager_RemoteLoadError(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	storage := &mockLegoStorage{
		loadErr: context.DeadlineExceeded, // Simulate timeout error
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	// Should try to issue new cert when Load returns error
	// This will fail because we don't have a real ACME server
	_, err := manager.EnsureCertificate(ctx, opts)
	if err == nil {
		t.Error("Expected error when issuing cert without ACME server")
	}
}

// TestLegoManager_RemoteNoCert tests when remote storage has no certificate
func TestLegoManager_RemoteNoCert(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	storage := &mockLegoStorage{
		chain: nil, // No certificate found
		meta:  storage.Metadata{},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	// Should try to issue new cert
	// This will fail because we don't have a real ACME server
	_, err := manager.EnsureCertificate(ctx, opts)
	if err == nil {
		t.Error("Expected error when issuing cert without ACME server")
	}
}

// TestLegoManager_RemoteExpired tests when remote certificate is expired
func TestLegoManager_RemoteExpired(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	// Derive the same key that the manager will derive
	seed := keys.SeedFromMnemonic(opts.Mnemonic)
	tlsKey := keys.DeriveTLSKey(seed, "example.com", 0)

	// Create expired test certificate
	expiry := time.Now().Add(-24 * time.Hour) // Expired yesterday
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             expiry.Add(-30 * 24 * time.Hour),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &tlsKey.PublicKey, tlsKey)
	testChain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	storage := &mockLegoStorage{
		chain: testChain,
		meta: storage.Metadata{
			ExpiresAt: expiry,
		},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	// Should try to issue new cert because existing is expired
	// This will fail because we don't have a real ACME server
	_, err := manager.EnsureCertificate(ctx, opts)
	if err == nil {
		t.Error("Expected error when issuing cert without ACME server")
	}
}

// TestLegoManager_RenewalWindow tests when certificate is in renewal window
func TestLegoManager_RenewalWindow(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	// Derive the same key that the manager will derive
	seed := keys.SeedFromMnemonic(opts.Mnemonic)
	tlsKey := keys.DeriveTLSKey(seed, "example.com", 0)

	// Create certificate expiring in 20 days (within 30 day renewal window)
	expiry := time.Now().Add(20 * 24 * time.Hour)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &tlsKey.PublicKey, tlsKey)
	testChain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	storage := &mockLegoStorage{
		chain: testChain,
		meta: storage.Metadata{
			ExpiresAt: expiry,
		},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	// Should try to renew cert because it's in renewal window
	// This will fail because we don't have a real ACME server
	_, err := manager.EnsureCertificate(ctx, opts)
	if err == nil {
		t.Error("Expected error when issuing cert without ACME server")
	}
}

// TestLegoManager_ForceIssue tests force issue flag
func TestLegoManager_ForceIssue(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
		ForceIssue:    true, // Force new certificate
	}

	// Derive the same key that the manager will derive
	seed := keys.SeedFromMnemonic(opts.Mnemonic)
	tlsKey := keys.DeriveTLSKey(seed, "example.com", 0)

	// Create valid test certificate
	expiry := time.Now().Add(60 * 24 * time.Hour)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &tlsKey.PublicKey, tlsKey)
	testChain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	storage := &mockLegoStorage{
		chain: testChain,
		meta: storage.Metadata{
			ExpiresAt: expiry,
		},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	// Should issue new cert despite having a valid one
	// This will fail because we don't have a real ACME server
	_, err := manager.EnsureCertificate(ctx, opts)
	if err == nil {
		t.Error("Expected error when issuing cert without ACME server")
	}
}

// TestLegoManager_NoExpiryInMeta tests when metadata doesn't have expiry
func TestLegoManager_NoExpiryInMeta(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	// Derive the same key that the manager will derive
	seed := keys.SeedFromMnemonic(opts.Mnemonic)
	tlsKey := keys.DeriveTLSKey(seed, "example.com", 0)

	// Create test certificate
	expiry := time.Now().Add(60 * 24 * time.Hour)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &tlsKey.PublicKey, tlsKey)
	testChain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	storage := &mockLegoStorage{
		chain: testChain,
		meta: storage.Metadata{
			// ExpiresAt is zero - should parse from cert
		},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	bundle, err := manager.EnsureCertificate(ctx, opts)
	if err != nil {
		t.Fatalf("EnsureCertificate() error = %v", err)
	}

	// Should install from remote without issuing new cert
	if bundle.Issued {
		t.Error("Should not have issued new certificate")
	}
	if !bundle.Reconstructed {
		t.Error("Should have reconstructed from remote")
	}
}

// TestLegoManager_InvalidCertData tests when certificate data can't be parsed
func TestLegoManager_InvalidCertData(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	storage := &mockLegoStorage{
		chain: []byte("invalid cert data"),
		meta: storage.Metadata{
			// No expiry, and cert data is invalid
		},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	// Should issue new cert because can't determine expiry
	// This will fail because we don't have a real ACME server
	_, err := manager.EnsureCertificate(ctx, opts)
	if err == nil {
		t.Error("Expected error when issuing cert without ACME server")
	}
}

// TestLegoManager_WrongKey tests when remote cert doesn't match derived key
func TestLegoManager_WrongKey(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	// Create test certificate with WRONG key
	wrongKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	expiry := time.Now().Add(60 * 24 * time.Hour)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &wrongKey.PublicKey, wrongKey)
	testChain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	storage := &mockLegoStorage{
		chain: testChain,
		meta: storage.Metadata{
			ExpiresAt: expiry,
		},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	_, err := manager.EnsureCertificate(ctx, opts)
	if err == nil {
		t.Error("Expected error when certificate doesn't match derived key")
	}
	if err.Error() != "remote certificate does not match derived key" {
		t.Errorf("Expected key mismatch error, got: %v", err)
	}
}

// TestLegoManager_ClockOverride tests clock injection for time-based testing
func TestLegoManager_ClockOverride(t *testing.T) {
	ctx := context.Background()

	opts := config.Config{
		Mnemonic:      "test test test test test test test test test test test test",
		Domain:        "example.com",
		OutDir:        "/tmp/test",
		CADir:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		Challenge:     config.HTTP01,
		RenewalWindow: 30 * 24 * time.Hour,
		APIURL:        "https://api.example.com",
	}

	// Derive the same key that the manager will derive
	seed := keys.SeedFromMnemonic(opts.Mnemonic)
	tlsKey := keys.DeriveTLSKey(seed, "example.com", 0)

	// Create certificate that expires in 60 days from real time
	realExpiry := time.Now().Add(60 * 24 * time.Hour)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              realExpiry,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &tlsKey.PublicKey, tlsKey)
	testChain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	storage := &mockLegoStorage{
		chain: testChain,
		meta: storage.Metadata{
			ExpiresAt: realExpiry,
		},
	}
	localWriter := &mockLegoLocalWriter{}
	logger := slog.Default()

	manager := NewLegoManager(storage, localWriter, logger)

	// Set clock to 45 days in future (cert will be in renewal window)
	futureTime := func() time.Time {
		return time.Now().Add(45 * 24 * time.Hour)
	}
	manager.SetClock(futureTime)

	// Should try to renew because from clock's perspective, cert is in renewal window
	// This will fail because we don't have a real ACME server
	_, err := manager.EnsureCertificate(ctx, opts)
	if err == nil {
		t.Error("Expected error when issuing cert without ACME server")
	}
}