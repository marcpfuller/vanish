package vault

import "context"

// BitwardenClient defines the interface for interacting with Bitwarden vault
type BitwardenClient interface {
	// Authenticate authenticates with Bitwarden using an access token
	Authenticate(ctx context.Context, accessToken string) error

	// GetSecret retrieves a secret by name from the Bitwarden vault
	GetSecret(ctx context.Context, secretName string) (string, error)

	// Close closes the Bitwarden client and cleans up resources
	Close() error
}
