package cmd

import (
	"fmt"
	"testing"
	"time"

	"github.com/bdkmv/vanish/internal/sync"
	"github.com/bdkmv/vanish/internal/vault"
	"github.com/bdkmv/vanish/mocks"
	"github.com/bdkmv/vanish/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSync_Success(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	// Setup vault service
	vaultSvc := vault.NewVaultService(mockStore)

	// Mock expectations - credentials from keyring
	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("testuser:testpass", nil).Once()

	// Mock Tailscale expectations
	mockTS.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
		Return(nil).Once()
	mockTS.EXPECT().LocalAddr().
		Return("100.64.0.1", nil).Once()
	mockTS.EXPECT().Close().
		Return(nil).Once()

	// Mock sync expectations
	mockSync.EXPECT().Sync(mock.Anything, mock.MatchedBy(func(job sync.SyncJob) bool {
		return job.Name == "test-backup" &&
			job.Source == "/tmp/source" &&
			job.Destination == "sftp://testuser:testpass@nas.local:22/backup"
	})).Return(nil).Once()

	// Create temporary config file
	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{
		SyncJobs: []config.SyncJob{
			{
				Name:        "test-backup",
				Source:      "/tmp/source",
				Destination: "sftp://nas.local:22/backup",
				Enabled:     true,
			},
		},
	}
	require.NoError(t, config.Save(tmpConfig, cfg))

	// Execute sync
	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	// Assert
	assert.NoError(t, err)
}

func TestSync_NoEnabledJobs(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	// Mock expectations
	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("testuser:testpass", nil).Once()
	mockTS.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
		Return(nil).Once()
	mockTS.EXPECT().LocalAddr().
		Return("100.64.0.1", nil).Once()
	mockTS.EXPECT().Close().
		Return(nil).Once()

	// Create config with disabled jobs
	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{
		SyncJobs: []config.SyncJob{
			{
				Name:        "disabled-backup",
				Source:      "/tmp/source",
				Destination: "sftp://nas.local:22/backup",
				Enabled:     false,
			},
		},
	}
	require.NoError(t, config.Save(tmpConfig, cfg))

	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	// Should succeed with no jobs run
	assert.NoError(t, err)
}

func TestSync_FailedToGetAuthKey(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	// Mock auth key failure
	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("", fmt.Errorf("keyring error")).Once()

	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{SyncJobs: []config.SyncJob{}}
	require.NoError(t, config.Save(tmpConfig, cfg))

	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Tailscale auth key")
}

func TestSync_FailedToGetNASCreds(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	// Mock successful auth key but failed NAS creds
	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("", fmt.Errorf("creds error")).Once()

	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{SyncJobs: []config.SyncJob{}}
	require.NoError(t, config.Save(tmpConfig, cfg))

	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get NAS credentials")
}

func TestSync_TailscaleStartFailed(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("testuser:testpass", nil).Once()
	mockTS.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
		Return(fmt.Errorf("tailscale error")).Once()
	mockTS.EXPECT().Close().Return(nil).Once()

	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{SyncJobs: []config.SyncJob{}}
	require.NoError(t, config.Save(tmpConfig, cfg))

	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start Tailscale")
}

func TestSync_LocalAddrFailed(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("testuser:testpass", nil).Once()
	mockTS.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
		Return(nil).Once()
	mockTS.EXPECT().LocalAddr().
		Return("", fmt.Errorf("no address")).Once()
	mockTS.EXPECT().Close().Return(nil).Once()

	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{SyncJobs: []config.SyncJob{}}
	require.NoError(t, config.Save(tmpConfig, cfg))

	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Tailscale IP")
}

func TestSync_ConfigLoadFailed(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("testuser:testpass", nil).Once()
	mockTS.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
		Return(nil).Once()
	mockTS.EXPECT().LocalAddr().
		Return("100.64.0.1", nil).Once()
	mockTS.EXPECT().Close().Return(nil).Once()

	opts := SyncOptions{
		ConfigPath: "/nonexistent/config.yaml",
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestSync_SyncJobFailed(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("testuser:testpass", nil).Once()
	mockTS.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
		Return(nil).Once()
	mockTS.EXPECT().LocalAddr().
		Return("100.64.0.1", nil).Once()
	mockTS.EXPECT().Close().Return(nil).Once()

	// Mock sync failure
	mockSync.EXPECT().Sync(mock.Anything, mock.Anything).
		Return(fmt.Errorf("rclone error")).Once()

	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{
		SyncJobs: []config.SyncJob{
			{
				Name:        "failing-backup",
				Source:      "/tmp/source",
				Destination: "sftp://nas.local:22/backup",
				Enabled:     true,
			},
		},
	}
	require.NoError(t, config.Save(tmpConfig, cfg))

	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync failed for failing-backup")
}

func TestSync_MultipleJobs(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("testuser:testpass", nil).Once()
	mockTS.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
		Return(nil).Once()
	mockTS.EXPECT().LocalAddr().
		Return("100.64.0.1", nil).Once()
	mockTS.EXPECT().Close().Return(nil).Once()

	// Expect two sync calls
	mockSync.EXPECT().Sync(mock.Anything, mock.MatchedBy(func(job sync.SyncJob) bool {
		return job.Name == "backup1"
	})).Return(nil).Once()
	mockSync.EXPECT().Sync(mock.Anything, mock.MatchedBy(func(job sync.SyncJob) bool {
		return job.Name == "backup2"
	})).Return(nil).Once()

	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{
		SyncJobs: []config.SyncJob{
			{
				Name:        "backup1",
				Source:      "/tmp/source1",
				Destination: "sftp://nas.local:22/backup1",
				Enabled:     true,
			},
			{
				Name:        "backup2",
				Source:      "/tmp/source2",
				Destination: "sftp://nas.local:22/backup2",
				Enabled:     true,
			},
			{
				Name:        "disabled",
				Source:      "/tmp/source3",
				Destination: "sftp://nas.local:22/backup3",
				Enabled:     false,
			},
		},
	}
	require.NoError(t, config.Save(tmpConfig, cfg))

	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	assert.NoError(t, err)
}

func TestSync_CloseErrorIsLogged(t *testing.T) {
	mockStore := mocks.NewMockSecretStore(t)
	mockTS := mocks.NewMockTailscaleProvider(t)
	mockSync := mocks.NewMockSyncService(t)

	vaultSvc := vault.NewVaultService(mockStore)

	mockStore.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
		Return("tskey-test-12345", nil).Once()
	mockStore.EXPECT().Get(mock.Anything, "NAS_CREDS").
		Return("testuser:testpass", nil).Once()
	mockTS.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
		Return(nil).Once()
	mockTS.EXPECT().LocalAddr().
		Return("100.64.0.1", nil).Once()
	mockTS.EXPECT().Close().
		Return(fmt.Errorf("close error")).Once()

	mockSync.EXPECT().Sync(mock.Anything, mock.Anything).
		Return(nil).Once()

	tmpConfig := t.TempDir() + "/config.yaml"
	cfg := &config.Config{
		SyncJobs: []config.SyncJob{
			{
				Name:        "test",
				Source:      "/tmp/source",
				Destination: "sftp://nas.local:22/backup",
				Enabled:     true,
			},
		},
	}
	require.NoError(t, config.Save(tmpConfig, cfg))

	opts := SyncOptions{
		ConfigPath: tmpConfig,
		Timeout:    30 * time.Second,
	}

	err := Sync(vaultSvc, mockTS, mockSync, opts)

	// Should succeed despite close error (which is just logged)
	assert.NoError(t, err)
}
