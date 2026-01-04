package vault

import (
	"context"

	"github.com/zalando/go-keyring"
)

// KeyringStore implements SecretStore using the system keychain via go-keyring
type KeyringStore struct{}

// NewKeyringStore creates a new KeyringStore instance
func NewKeyringStore() *KeyringStore {
	return &KeyringStore{}
}

// Set stores a secret value in the system keychain
func (k *KeyringStore) Set(_ context.Context, key, value string) error {
	return keyring.Set(KeychainServiceName, key, value)
}

// Get retrieves a secret value from the system keychain
func (k *KeyringStore) Get(_ context.Context, key string) (string, error) {
	return keyring.Get(KeychainServiceName, key)
}

// Delete removes a secret from the system keychain
func (k *KeyringStore) Delete(_ context.Context, key string) error {
	return keyring.Delete(KeychainServiceName, key)
}
