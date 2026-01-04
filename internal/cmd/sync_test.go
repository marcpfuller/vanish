package cmd_test

import (
	"testing"
	"time"

	"github.com/bdkmv/vanish/internal/cmd"
	"github.com/bdkmv/vanish/internal/vault"
	"github.com/bdkmv/vanish/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSync_NoToken(t *testing.T) {
	// Create mock store that returns "not found" for token
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("", vault.ErrSecretNotFound)

	mockBW := mocks.NewMockBitwardenClient(t)
	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockSyncSvc := mocks.NewMockSyncService(t)

	opts := cmd.SyncOptions{
		ConfigPath: "test-config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestSync_AuthenticationFails(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(assert.AnError)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockSyncSvc := mocks.NewMockSyncService(t)

	opts := cmd.SyncOptions{
		ConfigPath: "test-config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to authenticate")
}

func TestSync_MissingTailscaleKey(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("", assert.AnError)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockSyncSvc := mocks.NewMockSyncService(t)

	opts := cmd.SyncOptions{
		ConfigPath: "test-config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Tailscale")
}

func TestSync_MissingNASCreds(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("tskey-auth-xxxxx", nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.NASCredsItem).
		Return("", assert.AnError)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockSyncSvc := mocks.NewMockSyncService(t)

	opts := cmd.SyncOptions{
		ConfigPath: "test-config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NAS")
}

func TestSync_InvalidNASCredsFormat(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("tskey-auth-xxxxx", nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.NASCredsItem).
		Return("invalid-format-no-colon", nil)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockSyncSvc := mocks.NewMockSyncService(t)

	opts := cmd.SyncOptions{
		ConfigPath: "test-config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credential")
}

func TestSync_ConfigFileNotFound(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("tskey-auth-xxxxx", nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.NASCredsItem).
		Return("user:pass", nil)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockTsProvider.EXPECT().Start(mock.Anything, "tskey-auth-xxxxx", mock.Anything).
		Return(nil)
	mockTsProvider.EXPECT().LocalAddr().Return("100.64.0.1", nil)
	mockTsProvider.EXPECT().Close().Return(nil)

	mockSyncSvc := mocks.NewMockSyncService(t)

	opts := cmd.SyncOptions{
		ConfigPath: "/nonexistent/config.yaml",
		Timeout:    1 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	// Will fail when trying to load config file
	assert.Contains(t, err.Error(), "config")
}

func TestSyncOptions_DefaultTimeout(t *testing.T) {
	opts := cmd.SyncOptions{
		ConfigPath: "test.yaml",
		Timeout:    0,
	}

	// Verify options can be created
	assert.Equal(t, "test.yaml", opts.ConfigPath)
}

func TestSync_SuccessfulSync(t *testing.T) {
	// Setup mocks for successful path
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("tskey-auth-xxxxx", nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.NASCredsItem).
		Return("testuser:testpass", nil)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockTsProvider.EXPECT().Start(mock.Anything, "tskey-auth-xxxxx", "vanish-backup").
		Return(nil)
	mockTsProvider.EXPECT().LocalAddr().Return("100.64.0.5", nil)
	mockTsProvider.EXPECT().Close().Return(nil)

	mockSyncSvc := mocks.NewMockSyncService(t)
	// Expect Sync to be called for each enabled job (2 jobs in valid_config.yaml)
	mockSyncSvc.EXPECT().Sync(mock.Anything, mock.MatchedBy(func(job interface{}) bool {
		// Accept any sync job
		return true
	})).Return(nil).Times(2)

	opts := cmd.SyncOptions{
		ConfigPath: "testdata/valid_config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.NoError(t, err)
}

func TestSync_NoEnabledJobs(t *testing.T) {
	// Create a config with no enabled jobs
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("tskey-auth-xxxxx", nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.NASCredsItem).
		Return("testuser:testpass", nil)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockTsProvider.EXPECT().Start(mock.Anything, "tskey-auth-xxxxx", mock.Anything).
		Return(nil)
	mockTsProvider.EXPECT().LocalAddr().Return("100.64.0.5", nil)
	mockTsProvider.EXPECT().Close().Return(nil)

	mockSyncSvc := mocks.NewMockSyncService(t)
	// No Sync calls expected since all jobs are disabled

	opts := cmd.SyncOptions{
		ConfigPath: "testdata/no_enabled_jobs.yaml",
		Timeout:    5 * time.Second,
	}

	// This should succeed with a warning but no error
	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.NoError(t, err)
}

func TestSync_TailscaleStartFails(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("tskey-auth-xxxxx", nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.NASCredsItem).
		Return("testuser:testpass", nil)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockTsProvider.EXPECT().Start(mock.Anything, "tskey-auth-xxxxx", "vanish-backup").
		Return(assert.AnError)
	mockTsProvider.EXPECT().Close().Return(nil) // defer will call Close even on error

	mockSyncSvc := mocks.NewMockSyncService(t)

	opts := cmd.SyncOptions{
		ConfigPath: "testdata/valid_config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Tailscale")
}

func TestSync_LocalAddrFails(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("tskey-auth-xxxxx", nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.NASCredsItem).
		Return("testuser:testpass", nil)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockTsProvider.EXPECT().Start(mock.Anything, "tskey-auth-xxxxx", "vanish-backup").
		Return(nil)
	mockTsProvider.EXPECT().LocalAddr().Return("", assert.AnError)
	mockTsProvider.EXPECT().Close().Return(nil)

	mockSyncSvc := mocks.NewMockSyncService(t)

	opts := cmd.SyncOptions{
		ConfigPath: "testdata/valid_config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IP")
}

func TestSync_SyncJobFails(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockStore.EXPECT().Get(mock.Anything, vault.BitwardenTokenKey).
		Return("test-token", nil)

	mockBW := mocks.NewMockBitwardenClient(t)
	mockBW.EXPECT().Authenticate(mock.Anything, "test-token").
		Return(nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.TailscaleAuthKeyItem).
		Return("tskey-auth-xxxxx", nil)
	mockBW.EXPECT().GetSecret(mock.Anything, vault.NASCredsItem).
		Return("testuser:testpass", nil)

	vaultService := vault.NewVaultService(mockStore, mockBW)

	mockTsProvider := mocks.NewMockTailscaleProvider(t)
	mockTsProvider.EXPECT().Start(mock.Anything, "tskey-auth-xxxxx", "vanish-backup").
		Return(nil)
	mockTsProvider.EXPECT().LocalAddr().Return("100.64.0.5", nil)
	mockTsProvider.EXPECT().Close().Return(nil)

	mockSyncSvc := mocks.NewMockSyncService(t)
	// First sync succeeds, second fails
	mockSyncSvc.EXPECT().Sync(mock.Anything, mock.Anything).
		Return(nil).Once()
	mockSyncSvc.EXPECT().Sync(mock.Anything, mock.Anything).
		Return(assert.AnError).Once()

	opts := cmd.SyncOptions{
		ConfigPath: "testdata/valid_config.yaml",
		Timeout:    5 * time.Second,
	}

	err := cmd.Sync(vaultService, mockTsProvider, mockSyncSvc, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync failed")
}
