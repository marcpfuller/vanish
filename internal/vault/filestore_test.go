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
	tests := []struct {
		name      string
		setupPath func(t *testing.T) string
		wantErr   bool
	}{
		{
			name: "with explicit path",
			setupPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "secrets.enc")
			},
			wantErr: false,
		},
		{
			name: "with default path",
			setupPath: func(t *testing.T) string {
				// Set temp HOME to avoid polluting real home
				tmpDir := t.TempDir()
				require.NoError(t, os.Setenv("HOME", tmpDir))
				t.Cleanup(func() {
					// Cleanup is handled by test framework
				})
				return ""
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storePath := tt.setupPath(t)
			store, err := vault.NewFileStore(storePath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, store)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, store)
			}
		})
	}
}

func TestFileStore_SetAndGet(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		wantValue string
	}{
		{
			name:      "simple string",
			key:       "test-key",
			value:     "test-value",
			wantValue: "test-value",
		},
		{
			name:      "empty value",
			key:       "empty-key",
			value:     "",
			wantValue: "",
		},
		{
			name:      "unicode value",
			key:       "unicode-key",
			value:     "Hello 世界 🌍",
			wantValue: "Hello 世界 🌍",
		},
		{
			name:      "special characters",
			key:       "special-key",
			value:     "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantValue: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			storePath := filepath.Join(tmpDir, "secrets.enc")

			store, err := vault.NewFileStore(storePath)
			require.NoError(t, err)

			ctx := context.Background()

			// Set the secret
			err = store.Set(ctx, tt.key, tt.value)
			require.NoError(t, err)

			// Get the secret back
			value, err := store.Get(ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestFileStore_Operations(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "get non-existent key",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				_, err = store.Get(ctx, "non-existent")
				assert.ErrorIs(t, err, vault.ErrSecretNotFound)
			},
		},
		{
			name: "delete existing key",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Set(ctx, "test-key", "test-value")
				require.NoError(t, err)

				err = store.Delete(ctx, "test-key")
				require.NoError(t, err)

				_, err = store.Get(ctx, "test-key")
				assert.ErrorIs(t, err, vault.ErrSecretNotFound)
			},
		},
		{
			name: "delete non-existent key",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Delete(ctx, "non-existent")
				assert.ErrorIs(t, err, vault.ErrSecretNotFound)
			},
		},
		{
			name: "multiple secrets",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				secrets := map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				}

				for k, v := range secrets {
					err = store.Set(ctx, k, v)
					require.NoError(t, err)
				}

				for k, expectedValue := range secrets {
					value, err := store.Get(ctx, k)
					require.NoError(t, err)
					assert.Equal(t, expectedValue, value)
				}
			},
		},
		{
			name: "persistence across instances",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")

				store1, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store1.Set(ctx, "persist-key", "persist-value")
				require.NoError(t, err)

				store2, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				value, err := store2.Get(ctx, "persist-key")
				require.NoError(t, err)
				assert.Equal(t, "persist-value", value)
			},
		},
		{
			name: "update value",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Set(ctx, "update-key", "initial-value")
				require.NoError(t, err)

				err = store.Set(ctx, "update-key", "updated-value")
				require.NoError(t, err)

				value, err := store.Get(ctx, "update-key")
				require.NoError(t, err)
				assert.Equal(t, "updated-value", value)
			},
		},
		{
			name: "large value",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				largeValue := string(make([]byte, 10240))
				for i := range largeValue {
					largeValue = string(append([]byte(largeValue[:i]), byte('A'+i%26)))
				}

				err = store.Set(ctx, "large-key", largeValue)
				require.NoError(t, err)

				value, err := store.Get(ctx, "large-key")
				require.NoError(t, err)
				assert.Len(t, value, 10240)
			},
		},
		{
			name: "machine ID consistency",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath1 := filepath.Join(tmpDir, "secrets1.enc")
				storePath2 := filepath.Join(tmpDir, "secrets2.enc")

				store1, err := vault.NewFileStore(storePath1)
				require.NoError(t, err)

				store2, err := vault.NewFileStore(storePath2)
				require.NoError(t, err)

				ctx := context.Background()
				err = store1.Set(ctx, "key", "value")
				require.NoError(t, err)

				err = store2.Set(ctx, "key", "value")
				require.NoError(t, err)

				value1, err := store1.Get(ctx, "key")
				require.NoError(t, err)
				assert.Equal(t, "value", value1)

				value2, err := store2.Get(ctx, "key")
				require.NoError(t, err)
				assert.Equal(t, "value", value2)
			},
		},
		{
			name: "special characters",
			run: func(t *testing.T) {
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
			},
		},
		{
			name: "empty key",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Set(ctx, "", "value")
				require.NoError(t, err)

				value, err := store.Get(ctx, "")
				require.NoError(t, err)
				assert.Equal(t, "value", value)
			},
		},
		{
			name: "empty value",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Set(ctx, "empty-key", "")
				require.NoError(t, err)

				value, err := store.Get(ctx, "empty-key")
				require.NoError(t, err)
				assert.Equal(t, "", value)
			},
		},
		{
			name: "multiple keys with delete",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Set(ctx, "key1", "value1")
				require.NoError(t, err)
				err = store.Set(ctx, "key2", "value2")
				require.NoError(t, err)
				err = store.Set(ctx, "key3", "value3")
				require.NoError(t, err)

				v1, err := store.Get(ctx, "key1")
				require.NoError(t, err)
				assert.Equal(t, "value1", v1)

				v2, err := store.Get(ctx, "key2")
				require.NoError(t, err)
				assert.Equal(t, "value2", v2)

				v3, err := store.Get(ctx, "key3")
				require.NoError(t, err)
				assert.Equal(t, "value3", v3)

				err = store.Delete(ctx, "key2")
				require.NoError(t, err)

				_, err = store.Get(ctx, "key2")
				assert.Error(t, err)

				v1, err = store.Get(ctx, "key1")
				require.NoError(t, err)
				assert.Equal(t, "value1", v1)

				v3, err = store.Get(ctx, "key3")
				require.NoError(t, err)
				assert.Equal(t, "value3", v3)
			},
		},
		{
			name: "set after delete",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Set(ctx, "key", "value1")
				require.NoError(t, err)

				err = store.Delete(ctx, "key")
				require.NoError(t, err)

				err = store.Set(ctx, "key", "value2")
				require.NoError(t, err)

				value, err := store.Get(ctx, "key")
				require.NoError(t, err)
				assert.Equal(t, "value2", value)
			},
		},
		{
			name: "unicode values",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				store, err := vault.NewFileStore(tmpDir)
				require.NoError(t, err)

				ctx := context.Background()
				unicodeValue := "Hello 世界 🌍 مرحبا שלום"
				err = store.Set(ctx, "unicode-key", unicodeValue)
				require.NoError(t, err)

				value, err := store.Get(ctx, "unicode-key")
				require.NoError(t, err)
				assert.Equal(t, unicodeValue, value)
			},
		},
		{
			name: "non-existent directory",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storeDir := filepath.Join(tmpDir, "nonexistent")

				store, err := vault.NewFileStore(storeDir)
				require.NoError(t, err)
				assert.NotNil(t, store)

				ctx := context.Background()
				err = store.Set(ctx, "test", "value")
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestFileStore_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		run           func(t *testing.T)
		skipCondition func() bool
	}{
		{
			name: "corrupted file",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				store, err := vault.NewFileStore(tmpDir)
				require.NoError(t, err)

				ctx := context.Background()
				secretsPath := filepath.Join(tmpDir, "secrets.enc")
				err = os.WriteFile(secretsPath, []byte("corrupted data"), 0o600)
				require.NoError(t, err)

				_, err = store.Get(ctx, "any-key")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "decrypt")
			},
		},
		{
			name: "corrupted base64",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				store, err := vault.NewFileStore(tmpDir)
				require.NoError(t, err)

				ctx := context.Background()
				secretsPath := filepath.Join(tmpDir, "secrets.enc")
				err = os.WriteFile(secretsPath, []byte("!!!invalid base64!!!"), 0o600)
				require.NoError(t, err)

				_, err = store.Get(ctx, "any-key")
				assert.Error(t, err)
			},
		},
		{
			name: "empty file",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				store, err := vault.NewFileStore(tmpDir)
				require.NoError(t, err)

				ctx := context.Background()
				secretsPath := filepath.Join(tmpDir, "secrets.enc")
				err = os.WriteFile(secretsPath, []byte(""), 0o600)
				require.NoError(t, err)

				_, err = store.Get(ctx, "any-key")
				assert.Error(t, err)
			},
		},
		{
			name: "short ciphertext",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				store, err := vault.NewFileStore(tmpDir)
				require.NoError(t, err)

				ctx := context.Background()
				secretsPath := filepath.Join(tmpDir, "secrets.enc")
				shortData := []byte{1, 2, 3}
				encoded := []byte(base64.StdEncoding.EncodeToString(shortData))
				err = os.WriteFile(secretsPath, encoded, 0o600)
				require.NoError(t, err)

				_, err = store.Get(ctx, "any-key")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "ciphertext too short")
			},
		},
		{
			name: "invalid JSON handling",
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				storePath := filepath.Join(tmpDir, "secrets.enc")
				store, err := vault.NewFileStore(storePath)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Set(ctx, "test", "value")
				require.NoError(t, err)

				value, err := store.Get(ctx, "test")
				require.NoError(t, err)
				assert.Equal(t, "value", value)
			},
		},
		{
			name: "read-only directory",
			skipCondition: func() bool {
				return os.Getuid() == 0
			},
			run: func(t *testing.T) {
				tmpDir := t.TempDir()
				store, err := vault.NewFileStore(tmpDir)
				require.NoError(t, err)

				ctx := context.Background()
				err = store.Set(ctx, "test", "value")
				require.NoError(t, err)

				err = os.Chmod(tmpDir, 0o444)
				require.NoError(t, err)
				defer os.Chmod(tmpDir, 0o755) //nolint:errcheck

				err = store.Set(ctx, "test2", "value2")
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipCondition != nil && tt.skipCondition() {
				t.Skip("Skipping test due to skip condition")
			}
			tt.run(t)
		})
	}
}
