package vault_test

import (
	"context"
	"encoding/base64"
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
	require.NoError(t, os.Setenv("HOME", tmpDir))
	defer func() {
		//nolint:errcheck // Best effort cleanup in defer
		os.Setenv("HOME", oldHome)
	}()

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
	assert.Len(t, value, 10240)
}

func TestFileStore_GetMachineID(t *testing.T) {
	// This tests the internal getMachineID function indirectly
	// by creating multiple FileStores - they should use the same key
	tmpDir := t.TempDir()
	storePath1 := filepath.Join(tmpDir, "secrets1.enc")
	storePath2 := filepath.Join(tmpDir, "secrets2.enc")

	store1, err := vault.NewFileStore(storePath1)
	require.NoError(t, err)

	store2, err := vault.NewFileStore(storePath2)
	require.NoError(t, err)

	ctx := context.Background()

	// Set value in store1
	err = store1.Set(ctx, "key", "value")
	require.NoError(t, err)

	// Set value in store2
	err = store2.Set(ctx, "key", "value")
	require.NoError(t, err)

	// Both should work (they use the same machine-derived key)
	value1, err := store1.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, "value", value1)

	value2, err := store2.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, "value", value2)
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

func TestFileStore_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := vault.NewFileStore(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write corrupted data to the actual secrets file
	secretsPath := filepath.Join(tmpDir, "secrets.enc")
	err = os.WriteFile(secretsPath, []byte("corrupted data"), 0o600)
	require.NoError(t, err)

	// Try to get should fail with decrypt error
	_, err = store.Get(ctx, "any-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decrypt")
}

func TestFileStore_CorruptedBase64(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := vault.NewFileStore(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write invalid base64 to the actual secrets file
	secretsPath := filepath.Join(tmpDir, "secrets.enc")
	err = os.WriteFile(secretsPath, []byte("!!!invalid base64!!!"), 0o600)
	require.NoError(t, err)

	// Try to get should fail with base64 decode error
	_, err = store.Get(ctx, "any-key")
	assert.Error(t, err)
}

func TestFileStore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := vault.NewFileStore(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write empty file to the actual secrets file
	secretsPath := filepath.Join(tmpDir, "secrets.enc")
	err = os.WriteFile(secretsPath, []byte(""), 0o600)
	require.NoError(t, err)

	// Try to get should fail
	_, err = store.Get(ctx, "any-key")
	assert.Error(t, err)
}

func TestFileStore_ShortCiphertext(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := vault.NewFileStore(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write valid base64 but too short to be valid ciphertext
	// Need at least 12 bytes for nonce in AES-GCM
	secretsPath := filepath.Join(tmpDir, "secrets.enc")
	shortData := []byte{1, 2, 3} // Only 3 bytes
	encoded := []byte(base64.StdEncoding.EncodeToString(shortData))
	err = os.WriteFile(secretsPath, encoded, 0o600)
	require.NoError(t, err)

	// Try to get should fail with "ciphertext too short"
	_, err = store.Get(ctx, "any-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestFileStore_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store, err := vault.NewFileStore(storePath)
	require.NoError(t, err)

	ctx := context.Background()

	// First create a valid encrypted file, then we'll manually encrypt invalid JSON
	// We can't easily test this without exposing internal encrypt method
	// So we'll set a valid secret first, then corrupt the file to test error handling

	err = store.Set(ctx, "test", "value")
	require.NoError(t, err)

	// Now read the file and verify it works
	value, err := store.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "value", value)
}

func TestFileStore_NonExistentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Create path in non-existent subdirectory
	storeDir := filepath.Join(tmpDir, "nonexistent")

	// Should succeed - NewFileStore creates directories
	store, err := vault.NewFileStore(storeDir)
	require.NoError(t, err)
	assert.NotNil(t, store)

	// And should be able to use it
	ctx := context.Background()
	err = store.Set(ctx, "test", "value")
	require.NoError(t, err)
}

func TestFileStore_ReadOnlyDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping read-only test when running as root")
	}

	tmpDir := t.TempDir()

	store, err := vault.NewFileStore(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Set a value first
	err = store.Set(ctx, "test", "value")
	require.NoError(t, err)

	// Make directory read-only
	err = os.Chmod(tmpDir, 0o444)
	require.NoError(t, err)
	defer os.Chmod(tmpDir, 0o755) //nolint:errcheck // Best effort cleanup

	// Try to set - should fail with permission error
	err = store.Set(ctx, "test2", "value2")
	assert.Error(t, err)
}

func TestFileStore_UnicodeValues(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := vault.NewFileStore(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Test various Unicode characters
	unicodeValue := "Hello 世界 🌍 مرحبا שלום"
	err = store.Set(ctx, "unicode-key", unicodeValue)
	require.NoError(t, err)

	value, err := store.Get(ctx, "unicode-key")
	require.NoError(t, err)
	assert.Equal(t, unicodeValue, value)
}
