package vault_test

import (
	"context"
	"testing"

	"github.com/bdkmv/vanish/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBitwardenSDKClient_Authenticate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bitwarden integration test in short mode")
	}

	client := vault.NewBitwardenSDKClient()
	defer func() { _ = client.Close() }() //nolint:errcheck // Intentionally discarding error in test cleanup

	ctx := context.Background()

	// Test with invalid token (should fail)
	err := client.Authenticate(ctx, "invalid-token")
	assert.Error(t, err, "Authentication with invalid token should fail")
}

func TestBitwardenSDKClient_GetSecret_NotAuthenticated(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bitwarden integration test in short mode")
	}

	client := vault.NewBitwardenSDKClient()
	defer func() { _ = client.Close() }() //nolint:errcheck // Intentionally discarding error in test cleanup

	ctx := context.Background()

	// Try to get secret without authentication
	_, err := client.GetSecret(ctx, "some-secret")
	assert.Error(t, err, "Getting secret without authentication should fail")
}

func TestBitwardenSDKClient_GetSecret_WithValidAuth(t *testing.T) {
	// This test requires a valid Bitwarden token and secret
	// Skip by default since it requires real credentials
	t.Skip("Requires valid Bitwarden credentials - for manual testing only")

	client := vault.NewBitwardenSDKClient()
	defer func() { _ = client.Close() }() //nolint:errcheck // Intentionally discarding error in test cleanup

	ctx := context.Background()

	// Replace with actual token for manual testing
	token := "your-test-token-here"
	err := client.Authenticate(ctx, token)
	require.NoError(t, err)

	// Replace with actual secret ID
	secretValue, err := client.GetSecret(ctx, "secret-id-here")
	require.NoError(t, err)
	assert.NotEmpty(t, secretValue)
}

func TestBitwardenSDKClient_Close(t *testing.T) {
	client := vault.NewBitwardenSDKClient()

	// Close should not error even if not authenticated
	err := client.Close()
	assert.NoError(t, err, "Close should not error")

	// Calling Close again should also not error
	err = client.Close()
	assert.NoError(t, err, "Multiple Close calls should not error")
}

func TestBitwardenSDKClient_GetSecret_AfterClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bitwarden integration test in short mode")
	}

	client := vault.NewBitwardenSDKClient()
	err := client.Close()
	assert.NoError(t, err)

	ctx := context.Background()

	// Try to use client after close
	_, err = client.GetSecret(ctx, "some-secret")
	assert.Error(t, err, "Using client after Close should fail")
}
