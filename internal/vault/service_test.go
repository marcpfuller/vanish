package vault_test

import (
	"context"
	"os"
	"testing"

	"github.com/bdkmv/vanish/internal/vault"
	"github.com/bdkmv/vanish/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultService_GetTailscaleAuthKey(t *testing.T) {
	tests := []struct {
		name        string
		envVar      string
		envValue    string
		setupMock   func(*mocks.MockSecretStore)
		expectedKey string
		expectError bool
	}{
		{
			name:   "from keyring",
			envVar: "",
			setupMock: func(m *mocks.MockSecretStore) {
				m.EXPECT().
					Get(context.Background(), vault.TailscaleAuthKeyItem).
					Return("tskey-auth-test123", nil).
					Once()
			},
			expectedKey: "tskey-auth-test123",
			expectError: false,
		},
		{
			name:        "from environment",
			envVar:      "TS_AUTHKEY",
			envValue:    "tskey-from-env-123",
			setupMock:   func(m *mocks.MockSecretStore) {},
			expectedKey: "tskey-from-env-123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockStore := mocks.NewMockSecretStore(t)

			if tt.envVar != "" {
				require.NoError(t, os.Setenv(tt.envVar, tt.envValue))
				defer func() { _ = os.Unsetenv(tt.envVar) }() //nolint:errcheck
			}

			tt.setupMock(mockStore)
			service := vault.NewVaultService(mockStore)

			key, err := service.GetTailscaleAuthKey(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedKey, key)
			}
		})
	}
}

func TestVaultService_GetNASCredentials(t *testing.T) {
	tests := []struct {
		name             string
		envVar           string
		envValue         string
		setupMock        func(*mocks.MockSecretStore)
		expectedUsername string
		expectedPassword string
		expectError      bool
		errorContains    string
	}{
		{
			name:   "from keyring",
			envVar: "",
			setupMock: func(m *mocks.MockSecretStore) {
				m.EXPECT().
					Get(context.Background(), vault.NASCredsItem).
					Return("admin:SecurePassword123", nil).
					Once()
			},
			expectedUsername: "admin",
			expectedPassword: "SecurePassword123",
			expectError:      false,
		},
		{
			name:   "from keyring invalid format",
			envVar: "",
			setupMock: func(m *mocks.MockSecretStore) {
				m.EXPECT().
					Get(context.Background(), vault.NASCredsItem).
					Return("no-colon-separator", nil).
					Once()
			},
			expectError:   true,
			errorContains: "invalid credential format",
		},
		{
			name:             "from environment",
			envVar:           "NAS_CREDS",
			envValue:         "testuser:testpass",
			setupMock:        func(m *mocks.MockSecretStore) {},
			expectedUsername: "testuser",
			expectedPassword: "testpass",
			expectError:      false,
		},
		{
			name:          "from environment invalid format",
			envVar:        "NAS_CREDS",
			envValue:      "invalid-no-colon",
			setupMock:     func(m *mocks.MockSecretStore) {},
			expectError:   true,
			errorContains: "invalid credential format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockStore := mocks.NewMockSecretStore(t)

			if tt.envVar != "" {
				require.NoError(t, os.Setenv(tt.envVar, tt.envValue))
				defer func() { _ = os.Unsetenv(tt.envVar) }() //nolint:errcheck
			}

			tt.setupMock(mockStore)
			service := vault.NewVaultService(mockStore)

			username, password, err := service.GetNASCredentials(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedUsername, username)
				assert.Equal(t, tt.expectedPassword, password)
			}
		})
	}
}

func TestVaultService_Close(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	service := vault.NewVaultService(mockStore)

	err := service.Close()
	assert.NoError(t, err)
}
