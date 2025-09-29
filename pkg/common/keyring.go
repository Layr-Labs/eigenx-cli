package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/zalando/go-keyring"
)

const (
	KeyPrefix = "eigenx-"
)

var ErrKeyNotFound = errors.New("key not found")

// wrapKeyringError converts keyring backend errors to our standard error types
func wrapKeyringError(err error, environment string) error {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "secret not found") {
		return fmt.Errorf("%w: %s", ErrKeyNotFound, environment)
	}

	return err
}

type KeyringStore interface {
	StorePrivateKey(environment, privateKey string) error
	GetPrivateKey(environment string) (string, error)
	DeletePrivateKey(environment string) error
}

type OSKeyringStore struct{}

func (o *OSKeyringStore) StorePrivateKey(environment, privateKey string) error {
	account := KeyPrefix + environment
	return keyring.Set(KeyringServiceName, account, privateKey)
}

func (o *OSKeyringStore) GetPrivateKey(environment string) (string, error) {
	account := KeyPrefix + environment
	key, err := keyring.Get(KeyringServiceName, account)
	return key, wrapKeyringError(err, environment)
}

func (o *OSKeyringStore) DeletePrivateKey(environment string) error {
	account := KeyPrefix + environment
	err := keyring.Delete(KeyringServiceName, account)
	return wrapKeyringError(err, environment)
}

var DefaultKeyringStore KeyringStore = &OSKeyringStore{}

func StorePrivateKey(environment, privateKey string) error {
	return DefaultKeyringStore.StorePrivateKey(environment, privateKey)
}

func GetPrivateKey(environment string) (string, error) {
	return DefaultKeyringStore.GetPrivateKey(environment)
}

func DeletePrivateKey(environment string) error {
	return DefaultKeyringStore.DeletePrivateKey(environment)
}

// ValidatePrivateKey validates that a private key is in the correct format
func ValidatePrivateKey(key string) error {
	_, err := GetAddressFromPrivateKey(key)
	return err
}

// GetAddressFromPrivateKey validates a private key and returns the corresponding address
func GetAddressFromPrivateKey(privateKeyHex string) (string, error) {
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", err
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	return address.Hex(), nil
}
