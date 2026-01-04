package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bdkmv/vanish/internal/vault"
	"github.com/bdkmv/vanish/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultService_GetBitwardenSecrets(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)
	mockBW := mocks.NewMockBitwardenClient(t)

	service := vault.NewVaultService(mockStore, mockBW)

	// Setup expectations
	accessToken := "test-access-token"
	mockStore.EXPECT().
		Get(ctx, vault.BitwardenTokenKey).
		Return(accessToken, nil).
		Once()

	mockBW.EXPECT().
		Authenticate(ctx, accessToken).
		Return(nil).
		Once()

	secretID := "test-secret-id"
	secretValue := "test-secret-value"
	mockBW.EXPECT().
		GetSecret(ctx, secretID).
		Return(secretValue, nil).
		Once()

	// Execute
	secrets, err := service.GetBitwardenSecrets(ctx, secretID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, secretValue, secrets[secretID])
}

func TestVaultService_GetBitwardenSecrets_NoToken(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)
	mockBW := mocks.NewMockBitwardenClient(t)

	service := vault.NewVaultService(mockStore, mockBW)

	// Setup expectations - token not found
	mockStore.EXPECT().
		Get(ctx, vault.BitwardenTokenKey).
		Return("", vault.ErrSecretNotFound).
		Once()

	// Execute
	_, err := service.GetBitwardenSecrets(ctx, "any-secret")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Bitwarden token")
}

func TestVaultService_GetBitwardenSecrets_AuthFails(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)
	mockBW := mocks.NewMockBitwardenClient(t)

	service := vault.NewVaultService(mockStore, mockBW)

	// Setup expectations
	accessToken := "invalid-token"
	mockStore.EXPECT().
		Get(ctx, vault.BitwardenTokenKey).
		Return(accessToken, nil).
		Once()

	authErr := errors.New("authentication failed")
	mockBW.EXPECT().
		Authenticate(ctx, accessToken).
		Return(authErr).
		Once()

	// Execute
	_, err := service.GetBitwardenSecrets(ctx, "any-secret")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to authenticate")
}

func TestVaultService_GetTailscaleAuthKey(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)
	mockBW := mocks.NewMockBitwardenClient(t)

	service := vault.NewVaultService(mockStore, mockBW)

	// Setup expectations
	accessToken := "test-token"
	authKey := "tskey-test-123"

	mockStore.EXPECT().
		Get(ctx, vault.BitwardenTokenKey).
		Return(accessToken, nil).
		Once()

	mockBW.EXPECT().
		Authenticate(ctx, accessToken).
		Return(nil).
		Once()

	mockBW.EXPECT().
		GetSecret(ctx, vault.TailscaleAuthKeyItem).
		Return(authKey, nil).
		Once()

	// Execute
	result, err := service.GetTailscaleAuthKey(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, authKey, result)
}

func TestVaultService_GetNASCredentials(t *testing.T) {
	ctx := context.Background()
	mockStore := mocks.NewMockSecretStore(t)
	mockBW := mocks.NewMockBitwardenClient(t)

	service := vault.NewVaultService(mockStore, mockBW)

	// Setup expectations
	accessToken := "test-token"
	creds := "admin:SecurePassword123"

	mockStore.EXPECT().
		Get(ctx, vault.BitwardenTokenKey).
		Return(accessToken, nil).
		Once()

	mockBW.EXPECT().
		Authenticate(ctx, accessToken).
		Return(nil).
		Once()

	mockBW.EXPECT().
		GetSecret(ctx, vault.NASCredsItem).
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
	mockBW := mocks.NewMockBitwardenClient(t)

	service := vault.NewVaultService(mockStore, mockBW)

	// Setup expectations
	accessToken := "test-token"
	invalidCreds := "no-colon-separator"

	mockStore.EXPECT().
		Get(ctx, vault.BitwardenTokenKey).
		Return(accessToken, nil).
		Once()

	mockBW.EXPECT().
		Authenticate(ctx, accessToken).
		Return(nil).
		Once()

	mockBW.EXPECT().
		GetSecret(ctx, vault.NASCredsItem).
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
	mockBW := mocks.NewMockBitwardenClient(t)

	service := vault.NewVaultService(mockStore, mockBW)

	// Setup expectations
	mockBW.EXPECT().
		Close().
		Return(nil).
		Once()

	// Execute
	err := service.Close()

	// Assert
	assert.NoError(t, err)
}
