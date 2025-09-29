package cert

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

// LeafCertificateExpiry extracts the expiration time from the leaf certificate in a PEM chain
//
// The leaf certificate is the first certificate in the chain, which is the server's
// own certificate (as opposed to intermediate or root CA certificates).
//
// Returns the expiration time (NotAfter field) of the leaf certificate,
// or zero time if the PEM data is invalid or contains no certificates.
func LeafCertificateExpiry(pemData []byte) time.Time {
	rest := pemData
	for {
		block, r := pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			if c, err := x509.ParseCertificate(block.Bytes); err == nil {
				return c.NotAfter
			}
		}
		rest = r
	}
	return time.Time{}
}

// LeafPubMatches checks if the first cert in chain matches the given public key
//
// Returns true if the leaf certificate's public key matches the provided ECDSA key.
func LeafPubMatches(chainPEM []byte, pub *ecdsa.PublicKey) bool {
	block, _ := pem.Decode(chainPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	certPub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	return ok && certPub.Equal(pub)
}

// NormalizeSANs normalizes and deduplicates domain names for certificate SANs
//
// Params:
//   - domain: primary domain (will be CN)
//   - extras: additional domain names
//
// Returns:
//   - sans: normalized list with primary domain first
//   - primary: the normalized primary domain
//   - error: if normalization fails
func NormalizeSANs(domain string, extras []string) (sans []string, primary string, err error) {
	normalize := func(s string) (string, error) {
		s = strings.ToLower(strings.TrimSpace(s))
		// Convert to ASCII using IDNA encoding for international domains
		return idna.ToASCII(s)
	}

	cn, err := normalize(domain)
	if err != nil {
		return nil, "", fmt.Errorf("invalid domain %q: %w", domain, err)
	}

	// Use map to track unique domains
	seen := map[string]struct{}{cn: {}}
	sans = append(sans, cn)

	for _, dn := range extras {
		normalized, err := normalize(dn)
		if err != nil {
			return nil, "", fmt.Errorf("invalid domain %q: %w", dn, err)
		}
		if _, exists := seen[normalized]; !exists {
			seen[normalized] = struct{}{}
			sans = append(sans, normalized)
		}
	}

	return sans, cn, nil
}
