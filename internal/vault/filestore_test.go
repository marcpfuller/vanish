package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bdkmv/vanish/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestNewFileStore_DefaultPath(t *testing.T) {
	// Create in temp dir to avoid polluting home
	oldHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	store, err := vault.NewFileStore("")
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestFileStore_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Set a secret
	err = store.Set(ctx, "test-key", "test-value")
	require.NoError(t, err)

	// Get the secret back
	value, err := store.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", value)
}

func TestFileStore_GetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Try to get non-existent key
	_, err = store.Get(ctx, "non-existent")
	assert.ErrorIs(t, err, vault.ErrSecretNotFound)
}

func TestFileStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Set a secret
	err = store.Set(ctx, "test-key", "test-value")
	require.NoError(t, err)

	// Delete it
	err = store.Delete(ctx, "test-key")
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.Get(ctx, "test-key")
	assert.ErrorIs(t, err, vault.ErrSecretNotFound)
}

func TestFileStore_DeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Delete non-existent key - file store returns error for this
	err = store.Delete(ctx, "non-existent")
	// FileStore returns ErrSecretNotFound when deleting non-existent key
	assert.ErrorIs(t, err, vault.ErrSecretNotFound)
}

func TestFileStore_MultipleSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Set multiple secrets
	secrets := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range secrets {
		err = store.Set(ctx, k, v)
		require.NoError(t, err)
	}

	// Verify all can be retrieved
	for k, expectedValue := range secrets {
		value, err := store.Get(ctx, k)
		require.NoError(t, err)
		assert.Equal(t, expectedValue, value)
	}
}

func TestFileStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	// Create first store and save data
	store1, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()
	err = store1.Set(ctx, "persist-key", "persist-value")
	require.NoError(t, err)

	// Create second store with same path
	store2, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	// Should be able to read the data
	value, err := store2.Get(ctx, "persist-key")
	require.NoError(t, err)
	assert.Equal(t, "persist-value", value)
}

func TestFileStore_UpdateValue(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Set initial value
	err = store.Set(ctx, "update-key", "initial-value")
	require.NoError(t, err)

	// Update value
	err = store.Set(ctx, "update-key", "updated-value")
	require.NoError(t, err)

	// Verify updated value
	value, err := store.Get(ctx, "update-key")
	require.NoError(t, err)
	assert.Equal(t, "updated-value", value)
}

func TestFileStore_LargeValue(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a large value (10KB)
	largeValue := string(make([]byte, 10240))
	for i := range largeValue {
		largeValue = string(append([]byte(largeValue[:i]), byte('A'+i%26)))
	}

	err = store.Set(ctx, "large-key", largeValue)
	require.NoError(t, err)

	value, err := store.Get(ctx, "large-key")
	require.NoError(t, err)
	assert.Equal(t, len(largeValue), len(value))
}

func TestFileStore_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	specialValue := "Special!@#$%^&*()_+-={}[]|:;<>?,./~`\n\t\r"
	err = store.Set(ctx, "special-key", specialValue)
	require.NoError(t, err)

	value, err := store.Get(ctx, "special-key")
	require.NoError(t, err)
	assert.Equal(t, specialValue, value)
}

func TestFileStore_EmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Empty key should still work
	err = store.Set(ctx, "", "value")
	require.NoError(t, err)

	value, err := store.Get(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "value", value)
}

func TestFileStore_EmptyValue(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Empty value should work
	err = store.Set(ctx, "empty-key", "")
	require.NoError(t, err)

	value, err := store.Get(ctx, "empty-key")
	require.NoError(t, err)
	assert.Equal(t, "", value)
}

func TestFileStore_MultipleKeysOperations(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Set multiple keys
	err = store.Set(ctx, "key1", "value1")
	require.NoError(t, err)
	err = store.Set(ctx, "key2", "value2")
	require.NoError(t, err)
	err = store.Set(ctx, "key3", "value3")
	require.NoError(t, err)

	// Get all
	v1, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", v1)

	v2, err := store.Get(ctx, "key2")
	require.NoError(t, err)
	assert.Equal(t, "value2", v2)

	v3, err := store.Get(ctx, "key3")
	require.NoError(t, err)
	assert.Equal(t, "value3", v3)

	// Delete one
	err = store.Delete(ctx, "key2")
	require.NoError(t, err)

	// Verify key2 is gone
	_, err = store.Get(ctx, "key2")
	assert.Error(t, err)

	// But others still exist
	v1, err = store.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", v1)

	v3, err = store.Get(ctx, "key3")
	require.NoError(t, err)
	assert.Equal(t, "value3", v3)
}

func TestFileStore_SetAfterDelete(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Set, delete, set again
	err = store.Set(ctx, "key", "value1")
	require.NoError(t, err)

	err = store.Delete(ctx, "key")
	require.NoError(t, err)

	err = store.Set(ctx, "key", "value2")
	require.NoError(t, err)

	value, err := store.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, "value2", value)
}
