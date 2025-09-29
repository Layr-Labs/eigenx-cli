package keys

import (
	"crypto/ecdsa"
	"testing"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

// TestDeriveTLSKeyDeterministic verifies that the same inputs yield identical keys.
func TestDeriveTLSKeyDeterministic(t *testing.T) {
	seed := SeedFromMnemonic(testMnemonic)
	k1 := DeriveTLSKey(seed, "example.com", 0)
	k2 := DeriveTLSKey(seed, "example.com", 0)
	if k1.D.Cmp(k2.D) != 0 {
		t.Fatalf("expected deterministic TLS key; D differs")
	}
}

// TestDeriveAccountKeyDeterministic verifies that the account key is deterministic.
func TestDeriveAccountKeyDeterministic(t *testing.T) {
	seed := SeedFromMnemonic(testMnemonic)
	s1 := DeriveAccountKey(seed)
	s2 := DeriveAccountKey(seed)
	k1 := s1.(*ecdsa.PrivateKey)
	k2 := s2.(*ecdsa.PrivateKey)
	if k1.D.Cmp(k2.D) != 0 {
		t.Fatalf("expected deterministic account key; D differs")
	}
}

// TestDeriveTLSKeyDomainAndVersionAffectKey verifies domain and version rotate the keyspace.
func TestDeriveTLSKeyDomainAndVersionAffectKey(t *testing.T) {
	seed := SeedFromMnemonic(testMnemonic)
	k1 := DeriveTLSKey(seed, "example.com", 0)
	k2 := DeriveTLSKey(seed, "example.org", 0)
	if k1.D.Cmp(k2.D) == 0 {
		t.Fatalf("expected different keys for different domains")
	}
	k3 := DeriveTLSKey(seed, "example.com", 1)
	if k1.D.Cmp(k3.D) == 0 {
		t.Fatalf("expected different keys for different versions")
	}
}