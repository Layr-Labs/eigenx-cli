package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// genLeafCertForKey creates a minimal self-signed leaf for the provided key.
func genLeafCertForKey(t *testing.T, key *ecdsa.PrivateKey, cn string, notAfter time.Time) []byte {
	t.Helper()
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// TestLeafCertificateExpiry ensures expiration time can be extracted from the leaf cert in the PEM chain.
func TestLeafCertificateExpiry(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	exp := time.Now().Add(48 * time.Hour).Round(time.Second)
	p := genLeafCertForKey(t, key, "example.com", exp)
	got := LeafCertificateExpiry(p).Round(time.Second)
	if !got.Equal(exp) {
		t.Fatalf("LeafCertificateExpiry mismatch: got %v want %v", got, exp)
	}
}

// TestLeafPubMatches ensures the leaf's pubkey matches/non-matches expectations.
func TestLeafPubMatches(t *testing.T) {
	k1, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	k2, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	p := genLeafCertForKey(t, k1, "example.com", time.Now().Add(24*time.Hour))
	if !LeafPubMatches(p, &k1.PublicKey) {
		t.Fatalf("expected pubkey to match")
	}
	if LeafPubMatches(p, &k2.PublicKey) {
		t.Fatalf("expected pubkey mismatch to return false")
	}
}

// TestNormalizeSANs_UnicodeAndDedup validates IDNA conversion and deduplication.
func TestNormalizeSANs_UnicodeAndDedup(t *testing.T) {
	sans, primary, err := NormalizeSANs("bücher.example", []string{
		"example.com",
		"EXAMPLE.com",
		"xn--bcher-kva.example", // duplicate of the primary in punycode
	})
	if err != nil {
		t.Fatalf("NormalizeSANs error: %v", err)
	}
	wantPrimary := "xn--bcher-kva.example"
	if primary != wantPrimary {
		t.Fatalf("primary punycode mismatch: got %q want %q", primary, wantPrimary)
	}
	// Expect 2 unique SANs: primary + example.com
	if len(sans) != 2 {
		t.Fatalf("expected 2 unique SANs, got %d: %v", len(sans), sans)
	}
	if sans[0] != wantPrimary || sans[1] != "example.com" {
		t.Fatalf("unexpected SAN order/values: %v", sans)
	}
}

// TestNormalizeSANs_CaseFolding ensures case is normalized
func TestNormalizeSANs_CaseFolding(t *testing.T) {
	sans, primary, err := NormalizeSANs("EXAMPLE.COM", []string{
		"Example.Com",
		"example.com",
		"EXAMPLE.COM",
	})
	if err != nil {
		t.Fatalf("NormalizeSANs error: %v", err)
	}
	if primary != "example.com" {
		t.Fatalf("expected lowercase primary, got %q", primary)
	}
	// Should have deduplicated to just one domain
	if len(sans) != 1 {
		t.Fatalf("expected 1 unique SAN after dedup, got %d: %v", len(sans), sans)
	}
}