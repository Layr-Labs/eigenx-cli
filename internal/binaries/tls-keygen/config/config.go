package config

import (
	"errors"
	"fmt"
	"time"
)

// Challenge represents the ACME challenge type
type Challenge string

const (
	HTTP01    Challenge = "http-01"
	TLSALPN01 Challenge = "tls-alpn-01"

	// Let's Encrypt CA URLs
	LEProd    = "https://acme-v02.api.letsencrypt.org/directory"
	LEStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

// Config holds all configuration for the TLS keygen tool
type Config struct {
	// Core parameters
	Mnemonic string
	Domain   string
	AltNames []string
	OutDir   string
	Email    string
	CADir    string

	// Force reissue even if cert exists
	ForceIssue bool
	// Renewal window
	RenewalWindow time.Duration
	// Certificate expiry override
	NotAfter time.Time

	// Challenge type for ACME
	Challenge Challenge

	// Operation timeout
	Timeout time.Duration

	// Remote persistence API
	APIURL string

	// Token audience for GCE identity tokens
	TokenAudience string

	// Environment toggles
	Staging bool

	// Deterministic key rotation version
	Version uint32

	// User agent for ACME requests
	UserAgent string
}

// Validate checks structural constraints
//
// Returns error if mnemonic or domain is missing, or if challenge type is invalid.
func (o *Config) Validate() error {
	if o.Mnemonic == "" {
		return errors.New("mnemonic is required")
	}
	if o.Domain == "" {
		return errors.New("domain is required")
	}
	if o.Challenge != HTTP01 && o.Challenge != TLSALPN01 {
		return fmt.Errorf("invalid challenge type: %s", o.Challenge)
	}
	return nil
}