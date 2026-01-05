package vault

import (
	"context"
	"fmt"
	"os"
)

// VaultService provides high-level vault operations using SecretStore
//
//nolint:revive // VaultService stuttering is intentional for clarity
type VaultService struct {
	store SecretStore
}

// NewVaultService creates a new VaultService
func NewVaultService(store SecretStore) *VaultService {
	return &VaultService{
		store: store,
	}
}

// GetTailscaleAuthKey retrieves the Tailscale auth key from keyring or environment variable
func (v *VaultService) GetTailscaleAuthKey(ctx context.Context) (string, error) {
	// Check environment variable first (for testing/development)
	if envKey := os.Getenv("TS_AUTHKEY"); envKey != "" {
		return envKey, nil
	}

	// Try to get from keyring
	key, err := v.store.Get(ctx, TailscaleAuthKeyItem)
	if err != nil {
		return "", fmt.Errorf("tailscale auth key not found in keyring (run 'vanish setup' or set TS_AUTHKEY environment variable): %w", err)
	}
	return key, nil
}

// GetNASCredentials retrieves the NAS credentials from keyring or environment variable
// Returns username and password separately
func (v *VaultService) GetNASCredentials(ctx context.Context) (username, password string, err error) {
	// Check environment variable first (for testing/development)
	if envCreds := os.Getenv("NAS_CREDS"); envCreds != "" {
		username, password, err := parseCredentials(envCreds)
		if err != nil {
			return "", "", fmt.Errorf("failed to parse NAS_CREDS environment variable: %w", err)
		}
		return username, password, nil
	}

	// Try to get from keyring
	creds, err := v.store.Get(ctx, NASCredsItem)
	if err != nil {
		return "", "", fmt.Errorf("NAS credentials not found in keyring (run 'vanish setup' or set NAS_CREDS environment variable): %w", err)
	}

	// Parse the credentials in format "username:password"
	username, password, err = parseCredentials(creds)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse NAS credentials: %w", err)
	}

	return username, password, nil
}

// parseCredentials parses credentials in "username:password" format
func parseCredentials(creds string) (username, password string, err error) {
	// Find the first colon separator
	for i := 0; i < len(creds); i++ {
		if creds[i] == ':' {
			return creds[:i], creds[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid credential format, expected 'username:password'")
}

// Close closes the vault service and cleans up resources
func (v *VaultService) Close() error {
	return nil
}
