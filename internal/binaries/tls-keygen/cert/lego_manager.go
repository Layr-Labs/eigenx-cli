package cert

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/config"
	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/keys"
	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/storage"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// LegoUser implements the lego.User interface
type LegoUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *LegoUser) GetEmail() string                        { return u.Email }
func (u *LegoUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *LegoUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// LegoManager handles certificate management using Lego library
type LegoManager struct {
	storage storage.Storage     // Remote storage
	local   storage.LocalWriter // Local file writer
	clock   func() time.Time    // Time provider (defaults to time.Now)
	log     *slog.Logger
}

// NewLegoManager creates a new certificate manager using Lego
func NewLegoManager(remoteStorage storage.Storage, localWriter storage.LocalWriter, logger *slog.Logger) *LegoManager {
	return &LegoManager{
		storage: remoteStorage,
		local:   localWriter,
		clock:   time.Now,
		log:     logger,
	}
}

// SetClock sets the time provider (mainly for testing)
func (m *LegoManager) SetClock(clock func() time.Time) {
	m.clock = clock
}

// EnsureCertificate ensures a valid certificate exists locally
func (m *LegoManager) EnsureCertificate(ctx context.Context, opts config.Config) (storage.Bundle, error) {
	// Use injected clock
	now := m.clock

	// Normalize and deduplicate SANs
	sans, primary, err := NormalizeSANs(opts.Domain, opts.AltNames)
	if err != nil {
		return storage.Bundle{}, err
	}

	// Derive deterministic keys
	seed := keys.SeedFromMnemonic(opts.Mnemonic)
	acctKey := keys.DeriveAccountKey(seed)
	tlsKey := keys.DeriveTLSKey(seed, primary, opts.Version)

	// Fast path: forced re-issue
	if opts.ForceIssue {
		m.log.Info("force issue flag set, obtaining new certificate")
		return m.issueAndPersist(ctx, opts, primary, sans, tlsKey, acctKey)
	}

	// Check remote store
	m.log.Info("checking remote store", "domain", primary)
	chain, meta, err := m.storage.Load(primary)
	if err != nil {
		m.log.Warn("error checking remote store", "error", err)
		// Issue new cert on error
		return m.issueAndPersist(ctx, opts, primary, sans, tlsKey, acctKey)
	}

	if chain == nil {
		// No certificate found - issue new one
		m.log.Info("no certificate found in remote store")
		return m.issueAndPersist(ctx, opts, primary, sans, tlsKey, acctKey)
	}

	// Certificate found - check validity and renewal window
	expiry := meta.ExpiresAt
	if expiry.IsZero() {
		// Parse expiry from certificate if not in metadata
		expiry = LeafCertificateExpiry(chain)
	}

	if expiry.IsZero() {
		// Can't determine expiry - issue new cert to be safe
		m.log.Warn("unable to determine certificate expiry, issuing new certificate")
		return m.issueAndPersist(ctx, opts, primary, sans, tlsKey, acctKey)
	}

	// Check if expired or in renewal window
	if now().After(expiry) {
		m.log.Warn("remote certificate expired", "at", expiry.Format(time.RFC3339))
		return m.issueAndPersist(ctx, opts, primary, sans, tlsKey, acctKey)
	}

	if now().After(expiry.Add(-opts.RenewalWindow)) {
		m.log.Info("remote certificate in renewal window", "expires", expiry.Format(time.RFC3339))
		return m.issueAndPersist(ctx, opts, primary, sans, tlsKey, acctKey)
	}

	// Valid and not in renewal window - install locally
	return m.installFromRemote(opts.OutDir, chain, tlsKey, expiry)
}

// issueAndPersist obtains a new certificate and persists it
func (m *LegoManager) issueAndPersist(ctx context.Context, opts config.Config, primary string, sans []string, tlsKey *ecdsa.PrivateKey, acctKey crypto.Signer) (storage.Bundle, error) {
	m.log.Info("obtaining new certificate", "SANs", sans)

	// Create Lego user
	user := &LegoUser{
		Email: opts.Email,
		key:   acctKey,
	}

	// Create Lego config
	legoConfig := lego.NewConfig(user)
	legoConfig.CADirURL = opts.CADir
	legoConfig.UserAgent = opts.UserAgent

	// Create Lego client
	client, err := lego.NewClient(legoConfig)
	if err != nil {
		return storage.Bundle{}, fmt.Errorf("create lego client: %w", err)
	}

	// Setup challenge solver based on type
	switch opts.Challenge {
	case config.HTTP01:
		provider := http01.NewProviderServer("", "80")
		err = client.Challenge.SetHTTP01Provider(provider)
	case config.TLSALPN01:
		provider := tlsalpn01.NewProviderServer("", "443")
		err = client.Challenge.SetTLSALPN01Provider(provider)
	default:
		return storage.Bundle{}, fmt.Errorf("unsupported challenge type: %s", opts.Challenge)
	}
	if err != nil {
		return storage.Bundle{}, fmt.Errorf("set challenge provider: %w", err)
	}

	// Register account
	regOpts := registration.RegisterOptions{
		TermsOfServiceAgreed: true,
	}
	reg, err := client.Registration.Register(regOpts)
	if err != nil {
		// Try to retrieve existing registration
		reg, err = client.Registration.ResolveAccountByKey()
		if err != nil {
			return storage.Bundle{}, fmt.Errorf("register account: %w", err)
		}
	}
	user.Registration = reg
	m.log.Info("registered ACME account", "location", reg.URI)

	// Create certificate request with our derived TLS key
	request := certificate.ObtainRequest{
		Domains:    sans,
		Bundle:     true,
		PrivateKey: tlsKey, // Use our deterministically derived key!
	}

	// Obtain certificate (Lego v4 doesn't have ObtainWithContext)
	certResource, err := client.Certificate.Obtain(request)
	if err != nil {
		return storage.Bundle{}, fmt.Errorf("obtain certificate: %w", err)
	}

	// Write certificate and key locally
	fullPath, keyPath, err := m.writeCertificateFiles(opts.OutDir, certResource.Certificate, tlsKey)
	if err != nil {
		return storage.Bundle{}, err
	}

	expiry := LeafCertificateExpiry(certResource.Certificate)
	m.log.Info("certificate obtained", "expires", expiry.Format(time.RFC3339))

	// Store remotely (API will extract expiry from certificate)
	if err := m.storage.Store(primary, certResource.Certificate); err != nil {
		m.log.Warn("failed to store certificate remotely", "error", err)
		// Don't fail the operation - local files are written
	}

	return storage.Bundle{
		FullChainPath: fullPath,
		PrivKeyPath:   keyPath,
		NotAfter:      expiry,
		Issued:        true,
		Reconstructed: false,
	}, nil
}

// writeCertificateFiles writes certificate chain and private key to local filesystem
func (m *LegoManager) writeCertificateFiles(outDir string, chain storage.ChainPEM, tlsKey *ecdsa.PrivateKey) (fullChainPath, privKeyPath string, err error) {
	// Write certificate chain
	fullChainPath, err = m.local.WriteChain(outDir, chain)
	if err != nil {
		return "", "", fmt.Errorf("write certificate chain: %w", err)
	}

	// Write private key
	privKeyPath, err = m.local.WriteKey(outDir, tlsKey)
	if err != nil {
		return "", "", fmt.Errorf("write private key: %w", err)
	}

	return fullChainPath, privKeyPath, nil
}

// installFromRemote installs a certificate from remote storage
func (m *LegoManager) installFromRemote(outDir string, chain storage.ChainPEM, tlsKey *ecdsa.PrivateKey, expiry time.Time) (storage.Bundle, error) {
	// Verify the certificate matches our key
	if !LeafPubMatches(chain, &tlsKey.PublicKey) {
		return storage.Bundle{}, errors.New("remote certificate does not match derived key")
	}

	// Write certificate and key files
	fullPath, keyPath, err := m.writeCertificateFiles(outDir, chain, tlsKey)
	if err != nil {
		return storage.Bundle{}, err
	}

	m.log.Info("installed certificate from remote store", "expires", expiry.Format(time.RFC3339))

	return storage.Bundle{
		FullChainPath: fullPath,
		PrivKeyPath:   keyPath,
		NotAfter:      expiry,
		Issued:        false,
		Reconstructed: true,
	}, nil
}