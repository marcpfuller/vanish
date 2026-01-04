package vault

import (
	"context"
	"fmt"
)

// VaultService provides high-level vault operations combining SecretStore and BitwardenClient
//
//nolint:revive // VaultService stuttering is intentional for clarity
type VaultService struct {
	store     SecretStore
	bitwarden BitwardenClient
}

// NewVaultService creates a new VaultService
func NewVaultService(store SecretStore, bitwarden BitwardenClient) *VaultService {
	return &VaultService{
		store:     store,
		bitwarden: bitwarden,
	}
}

// GetBitwardenSecrets retrieves secrets from Bitwarden vault
// It authenticates using the stored access token and fetches the specified secrets
func (v *VaultService) GetBitwardenSecrets(ctx context.Context, secretNames ...string) (map[string]string, error) {
	// Get Bitwarden access token from keychain
	accessToken, err := v.store.Get(ctx, BitwardenTokenKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get Bitwarden token from keychain: %w", err)
	}

	// Authenticate with Bitwarden
	if err := v.bitwarden.Authenticate(ctx, accessToken); err != nil {
		return nil, fmt.Errorf("failed to authenticate with Bitwarden: %w", err)
	}

	// Fetch all requested secrets
	secrets := make(map[string]string)
	for _, name := range secretNames {
		value, err := v.bitwarden.GetSecret(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret %s: %w", name, err)
		}
		secrets[name] = value
	}

	return secrets, nil
}

// GetTailscaleAuthKey retrieves the Tailscale authentication key from Bitwarden
func (v *VaultService) GetTailscaleAuthKey(ctx context.Context) (string, error) {
	secrets, err := v.GetBitwardenSecrets(ctx, TailscaleAuthKeyItem)
	if err != nil {
		return "", err
	}
	return secrets[TailscaleAuthKeyItem], nil
}

// GetNASCredentials retrieves the NAS credentials from Bitwarden
// Returns username and password separately
func (v *VaultService) GetNASCredentials(ctx context.Context) (username, password string, err error) {
	secrets, err := v.GetBitwardenSecrets(ctx, NASCredsItem)
	if err != nil {
		return "", "", err
	}

	creds := secrets[NASCredsItem]
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

// Close closes the Bitwarden client
func (v *VaultService) Close() error {
	return v.bitwarden.Close()
}
