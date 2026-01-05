package vault_test

import (
	"context"
	"testing"

	"github.com/bdkmv/vanish/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyringStore(t *testing.T) {
	tests := []struct {
		name      string
		skipShort bool
		run       func(t *testing.T)
	}{
		{
			name:      "set and get",
			skipShort: true,
			run: func(t *testing.T) {
				ctx := context.Background()
				store := vault.NewKeyringStore()

				testKey := "test-key"
				testValue := "test-secret-value"

				err := store.Set(ctx, testKey, testValue)
				if err != nil {
					t.Skipf("Skipping test - keyring not available: %v", err)
				}
				require.NoError(t, err)

				retrievedValue, err := store.Get(ctx, testKey)
				require.NoError(t, err)
				assert.Equal(t, testValue, retrievedValue)

				err = store.Delete(ctx, testKey)
				require.NoError(t, err)
			},
		},
		{
			name:      "get non-existent key",
			skipShort: false,
			run: func(t *testing.T) {
				ctx := context.Background()
				store := vault.NewKeyringStore()

				_, err := store.Get(ctx, "non-existent-key")
				assert.Error(t, err)
			},
		},
		{
			name:      "delete non-existent key",
			skipShort: false,
			run: func(t *testing.T) {
				ctx := context.Background()
				store := vault.NewKeyringStore()

				err := store.Delete(ctx, "non-existent-key")
				assert.Error(t, err)
			},
		},
		{
			name:      "credentials storage",
			skipShort: true,
			run: func(t *testing.T) {
				ctx := context.Background()
				store := vault.NewKeyringStore()

				tsKey := "tskey-auth-test12345"
				err := store.Set(ctx, vault.TailscaleAuthKeyItem, tsKey)
				if err != nil {
					t.Skipf("Skipping test - keyring not available: %v", err)
				}
				require.NoError(t, err)

				retrievedKey, err := store.Get(ctx, vault.TailscaleAuthKeyItem)
				require.NoError(t, err)
				assert.Equal(t, tsKey, retrievedKey)

				err = store.Delete(ctx, vault.TailscaleAuthKeyItem)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipShort && testing.Short() {
				t.Skip("Skipping keyring test in short mode")
			}
			tt.run(t)
		})
	}
}
