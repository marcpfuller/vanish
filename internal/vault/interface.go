package vault

import "context"

// SecretStore defines the interface for securely storing and retrieving secrets
// from the system keychain. This interface allows for dependency injection and testing.
type SecretStore interface {
	// Set stores a secret value for the given key in the system keychain
	Set(ctx context.Context, key, value string) error

	// Get retrieves a secret value for the given key from the system keychain
	Get(ctx context.Context, key string) (string, error)

	// Delete removes a secret from the system keychain
	Delete(ctx context.Context, key string) error
}
