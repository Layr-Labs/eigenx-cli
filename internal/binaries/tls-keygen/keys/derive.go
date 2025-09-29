package keys

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/binary"

	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/hkdf"
)

// Key derivation constants
const (
	keyInfoAccount   = "eigenx/acme-account/v1"
	keyInfoTLSPrefix = "eigenx/tls-key/v1"
)

// SeedFromMnemonic converts a BIP-39 mnemonic into a seed
//
// Uses the mnemonic with no passphrase to generate a deterministic seed.
func SeedFromMnemonic(mnemonic string) []byte {
	return bip39.NewSeed(mnemonic, "")
}

// DeriveAccountKey deterministically derives an ACME account key from seed
//
// Returns a P-256 ECDSA key suitable for ACME account operations.
func DeriveAccountKey(seed []byte) crypto.Signer {
	return deriveP256(seed, []byte(keyInfoAccount))
}

// DeriveTLSKey deterministically derives a P-256 ECDSA key for domain/version
//
// Params:
//   - seed: base seed from mnemonic
//   - domain: domain name for key derivation
//   - version: rotation version number
//
// Returns P-256 ECDSA private key for TLS certificate.
func DeriveTLSKey(seed []byte, domain string, version uint32) *ecdsa.PrivateKey {
	info := append([]byte(keyInfoTLSPrefix), []byte(domain)...)
	var v [4]byte
	binary.BigEndian.PutUint32(v[:], version)
	info = append(info, v[:]...)
	return deriveP256(seed, info).(*ecdsa.PrivateKey)
}

// deriveP256 derives a P-256 ECDSA key using HKDF
func deriveP256(seed, info []byte) crypto.Signer {
	out := hkdfExpand(seed, info)
	for i := 0; i < 8; i++ {
		sk, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), out[:])
		if err == nil {
			return sk
		}
		h := sha256.Sum256(out[:])
		out = h
	}
	panic("failed to map to P-256 scalar")
}

// hkdfExpand performs HKDF expansion with SHA-256
func hkdfExpand(seed, info []byte) [32]byte {
	rd := hkdf.New(sha256.New, seed, nil, info) // salt=nil; domain separation via info
	var out [32]byte
	_, _ = rd.Read(out[:])
	return out
}

