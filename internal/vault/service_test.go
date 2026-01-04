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
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)

	service := vault.NewVaultService(mockStore)

	// Setup expectations
	expectedKey := "tskey-auth-test123"
	mockStore.EXPECT().
		Get(ctx, vault.TailscaleAuthKeyItem).
		Return(expectedKey, nil).
		Once()

	// Execute
	key, err := service.GetTailscaleAuthKey(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedKey, key)
}

func TestVaultService_GetNASCredentials(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)

	service := vault.NewVaultService(mockStore)

	// Setup expectations
	creds := "admin:SecurePassword123"
	mockStore.EXPECT().
		Get(ctx, vault.NASCredsItem).
		Return(creds, nil).
		Once()

	// Execute
	username, password, err := service.GetNASCredentials(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "admin", username)
	assert.Equal(t, "SecurePassword123", password)
}

func TestVaultService_GetNASCredentials_InvalidFormat(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)

	service := vault.NewVaultService(mockStore)

	// Setup expectations - invalid format (no colon)
	invalidCreds := "no-colon-separator"
	mockStore.EXPECT().
		Get(ctx, vault.NASCredsItem).
		Return(invalidCreds, nil).
		Once()

	// Execute
	_, _, err := service.GetNASCredentials(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credential format")
}

func TestVaultService_Close(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)

	service := vault.NewVaultService(mockStore)

	// Execute
	err := service.Close()

	// Assert - should be no-op and return nil
	assert.NoError(t, err)
}

func TestVaultService_GetTailscaleAuthKey_FromEnv(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)

	service := vault.NewVaultService(mockStore)

	// Set environment variable
	expectedKey := "tskey-from-env-123"
	require.NoError(t, os.Setenv("TS_AUTHKEY", expectedKey))
	defer func() { _ = os.Unsetenv("TS_AUTHKEY") }() //nolint:errcheck // test cleanup

	// Execute - should use env var, no keyring access
	key, err := service.GetTailscaleAuthKey(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedKey, key)
}

func TestVaultService_GetNASCredentials_FromEnv(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)

	service := vault.NewVaultService(mockStore)

	// Set environment variable
	require.NoError(t, os.Setenv("NAS_CREDS", "testuser:testpass"))
	defer func() { _ = os.Unsetenv("NAS_CREDS") }() //nolint:errcheck // test cleanup

	// Execute - should use env var, no keyring access
	username, password, err := service.GetNASCredentials(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "testuser", username)
	assert.Equal(t, "testpass", password)
}

func TestVaultService_GetNASCredentials_FromEnv_InvalidFormat(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)

	service := vault.NewVaultService(mockStore)

	// Set invalid environment variable
	require.NoError(t, os.Setenv("NAS_CREDS", "invalid-no-colon"))
	defer func() { _ = os.Unsetenv("NAS_CREDS") }() //nolint:errcheck // test cleanup

	// Execute
	_, _, err := service.GetNASCredentials(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credential format")
}
