package storage

import (
	"crypto/ecdsa"
	"time"
)

// Bundle represents the local certificate file set
type Bundle struct {
	FullChainPath string
	PrivKeyPath   string
	NotAfter      time.Time
	Issued        bool        // True if newly issued
	Reconstructed bool        // True if key was reconstructed from seed
}

// ChainPEM represents a certificate chain in PEM format
type ChainPEM []byte

// Metadata describes certificate timing information
type Metadata struct {
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// Storage provides certificate persistence operations
type Storage interface {
	// Load retrieves a certificate for the given domain
	//
	// Returns:
	//   - chainPEM and metadata when found (nil chain if not found)
	//   - error only for operational failures (not for missing certs)
	Load(domain string) (ChainPEM, Metadata, error)

	// Store persists a certificate for the given domain
	//
	// The API will extract expiry and issuance dates from the certificate.
	//
	// Params:
	//   - domain: primary domain name
	//   - chain: certificate chain in PEM format
	Store(domain string, chain ChainPEM) error
}

// LocalWriter handles local file system operations for certificates
type LocalWriter interface {
	// WriteChain writes certificate chain to disk
	//
	// Returns the full path to the written file.
	WriteChain(outDir string, chain ChainPEM) (string, error)

	// WriteKey writes private key to disk
	//
	// Returns the full path to the written file.
	WriteKey(outDir string, key *ecdsa.PrivateKey) (string, error)
}
