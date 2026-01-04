package vault_test

import (
	"context"
	"testing"

	"github.com/bdkmv/vanish/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyringStore_SetAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyring test in short mode")
	}

	ctx := context.Background()
	store := vault.NewKeyringStore()

	// Test data
	testKey := "test-key"
	testValue := "test-secret-value"

	// Set a value
	err := store.Set(ctx, testKey, testValue)
	if err != nil {
		t.Skipf("Skipping test - keyring not available: %v", err)
	}
	require.NoError(t, err, "Set should not return an error")

	// Get the value back
	retrievedValue, err := store.Get(ctx, testKey)
	require.NoError(t, err, "Get should not return an error")
	assert.Equal(t, testValue, retrievedValue, "Retrieved value should match set value")

	// Clean up
	err = store.Delete(ctx, testKey)
	require.NoError(t, err, "Delete should not return an error")
}

func TestKeyringStore_GetNonExistent(t *testing.T) {
	ctx := context.Background()
	store := vault.NewKeyringStore()

	// Try to get a non-existent key
	_, err := store.Get(ctx, "non-existent-key")
	assert.Error(t, err, "Get should return an error for non-existent key")
}

func TestKeyringStore_DeleteNonExistent(t *testing.T) {
	ctx := context.Background()
	store := vault.NewKeyringStore()

	// Try to delete a non-existent key (should not error according to go-keyring behavior)
	err := store.Delete(ctx, "non-existent-key")
	// go-keyring returns an error for non-existent keys, so we expect an error
	assert.Error(t, err, "Delete should return an error for non-existent key")
}

func TestKeyringStore_BiwardenTokenStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyring test in short mode")
	}

	ctx := context.Background()
	store := vault.NewKeyringStore()

	// Test storing Bitwarden token
	token := "bw_test_token_12345"
	err := store.Set(ctx, vault.BitwardenTokenKey, token)
	if err != nil {
		t.Skipf("Skipping test - keyring not available: %v", err)
	}
	require.NoError(t, err, "Should be able to store Bitwarden token")

	// Retrieve it
	retrievedToken, err := store.Get(ctx, vault.BitwardenTokenKey)
	require.NoError(t, err, "Should be able to retrieve Bitwarden token")
	assert.Equal(t, token, retrievedToken, "Token should match")

	// Clean up
	err = store.Delete(ctx, vault.BitwardenTokenKey)
	require.NoError(t, err, "Should be able to delete Bitwarden token")
}
