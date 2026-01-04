// Package vault provides secure secret storage and Bitwarden integration.
package vault

import (
	"context"
	"errors"
	"fmt"
	"sync"

	sdk "github.com/bitwarden/sdk-go"
)

var (
	// ErrNotAuthenticated is returned when attempting operations without authentication
	ErrNotAuthenticated = errors.New("not authenticated with Bitwarden")

	// ErrClientClosed is returned when attempting to use a closed client
	ErrClientClosed = errors.New("Bitwarden client has been closed")
)

// BitwardenSDKClient implements BitwardenClient using the official Bitwarden SDK
type BitwardenSDKClient struct {
	client        sdk.BitwardenClientInterface
	authenticated bool
	closed        bool
	mu            sync.RWMutex
}

// NewBitwardenSDKClient creates a new Bitwarden SDK client
func NewBitwardenSDKClient() *BitwardenSDKClient {
	// Create the Bitwarden client
	client, err := sdk.NewBitwardenClient(nil, nil)
	if err != nil {
		// If we can't create the client, return a client that will error on use
		return &BitwardenSDKClient{
			client: nil,
			closed: true,
		}
	}

	return &BitwardenSDKClient{
		client:        client,
		authenticated: false,
		closed:        false,
	}
}

// Authenticate authenticates with Bitwarden using an access token
func (b *BitwardenSDKClient) Authenticate(ctx context.Context, accessToken string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrClientClosed
	}

	if b.client == nil {
		return fmt.Errorf("Bitwarden client not initialized")
	}

	// Authenticate using access token
	err := b.client.AccessTokenLogin(accessToken, nil)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Bitwarden: %w", err)
	}

	b.authenticated = true
	return nil
}

// GetSecret retrieves a secret by ID from the Bitwarden vault
func (b *BitwardenSDKClient) GetSecret(ctx context.Context, secretID string) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return "", ErrClientClosed
	}

	if !b.authenticated {
		return "", ErrNotAuthenticated
	}

	if b.client == nil {
		return "", fmt.Errorf("Bitwarden client not initialized")
	}

	// Get the secret from Bitwarden
	secret, err := b.client.Secrets().Get(secretID)
	if err != nil {
		return "", fmt.Errorf("failed to get secret from Bitwarden: %w", err)
	}

	if secret == nil {
		return "", fmt.Errorf("secret not found: %s", secretID)
	}

	return secret.Value, nil
}

// Close closes the Bitwarden client and cleans up resources
func (b *BitwardenSDKClient) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil // Already closed, not an error
	}

	b.closed = true
	b.authenticated = false

	if b.client != nil {
		// The SDK doesn't have an explicit Close method, but we mark it as closed
		// to prevent further use
		b.client = nil
	}

	return nil
}
