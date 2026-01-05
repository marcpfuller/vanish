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

func TestSync(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(t *testing.T) string
		setupMocks  func(*mocks.MockSecretStore, *mocks.MockTailscaleProvider, *mocks.MockSyncService)
		expectError bool
		errorMsg    string
	}{
		{
			name: "success",
			setupConfig: func(t *testing.T) string {
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
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("testuser:testpass", nil).Once()
				mt.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
					Return(nil).Once()
				mt.EXPECT().LocalAddr().
					Return("100.64.0.1", nil).Once()
				mt.EXPECT().Close().
					Return(nil).Once()
				msy.EXPECT().Sync(mock.Anything, mock.MatchedBy(func(job sync.SyncJob) bool {
					return job.Name == "test-backup" &&
						job.Source == "/tmp/source" &&
						job.Destination == "sftp://testuser:testpass@nas.local:22/backup"
				})).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name: "no enabled jobs",
			setupConfig: func(t *testing.T) string {
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
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("testuser:testpass", nil).Once()
				mt.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
					Return(nil).Once()
				mt.EXPECT().LocalAddr().
					Return("100.64.0.1", nil).Once()
				mt.EXPECT().Close().
					Return(nil).Once()
			},
			expectError: false,
		},
		{
			name: "failed to get auth key",
			setupConfig: func(t *testing.T) string {
				tmpConfig := t.TempDir() + "/config.yaml"
				cfg := &config.Config{SyncJobs: []config.SyncJob{}}
				require.NoError(t, config.Save(tmpConfig, cfg))
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("", fmt.Errorf("keyring error")).Once()
			},
			expectError: true,
			errorMsg:    "failed to get Tailscale auth key",
		},
		{
			name: "failed to get NAS creds",
			setupConfig: func(t *testing.T) string {
				tmpConfig := t.TempDir() + "/config.yaml"
				cfg := &config.Config{SyncJobs: []config.SyncJob{}}
				require.NoError(t, config.Save(tmpConfig, cfg))
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("", fmt.Errorf("creds error")).Once()
			},
			expectError: true,
			errorMsg:    "failed to get NAS credentials",
		},
		{
			name: "tailscale start failed",
			setupConfig: func(t *testing.T) string {
				tmpConfig := t.TempDir() + "/config.yaml"
				cfg := &config.Config{SyncJobs: []config.SyncJob{}}
				require.NoError(t, config.Save(tmpConfig, cfg))
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("testuser:testpass", nil).Once()
				mt.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
					Return(fmt.Errorf("tailscale error")).Once()
				mt.EXPECT().Close().Return(nil).Once()
			},
			expectError: true,
			errorMsg:    "failed to start Tailscale",
		},
		{
			name: "local addr failed",
			setupConfig: func(t *testing.T) string {
				tmpConfig := t.TempDir() + "/config.yaml"
				cfg := &config.Config{SyncJobs: []config.SyncJob{}}
				require.NoError(t, config.Save(tmpConfig, cfg))
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("testuser:testpass", nil).Once()
				mt.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
					Return(nil).Once()
				mt.EXPECT().LocalAddr().
					Return("", fmt.Errorf("no address")).Once()
				mt.EXPECT().Close().Return(nil).Once()
			},
			expectError: true,
			errorMsg:    "failed to get Tailscale IP",
		},
		{
			name: "config load failed",
			setupConfig: func(t *testing.T) string {
				return "/nonexistent/config.yaml"
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("testuser:testpass", nil).Once()
				mt.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
					Return(nil).Once()
				mt.EXPECT().LocalAddr().
					Return("100.64.0.1", nil).Once()
				mt.EXPECT().Close().Return(nil).Once()
			},
			expectError: true,
			errorMsg:    "failed to load config",
		},
		{
			name: "sync job failed",
			setupConfig: func(t *testing.T) string {
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
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("testuser:testpass", nil).Once()
				mt.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
					Return(nil).Once()
				mt.EXPECT().LocalAddr().
					Return("100.64.0.1", nil).Once()
				mt.EXPECT().Close().Return(nil).Once()
				msy.EXPECT().Sync(mock.Anything, mock.Anything).
					Return(fmt.Errorf("rclone error")).Once()
			},
			expectError: true,
			errorMsg:    "sync failed for failing-backup",
		},
		{
			name: "multiple jobs",
			setupConfig: func(t *testing.T) string {
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
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("testuser:testpass", nil).Once()
				mt.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
					Return(nil).Once()
				mt.EXPECT().LocalAddr().
					Return("100.64.0.1", nil).Once()
				mt.EXPECT().Close().Return(nil).Once()
				msy.EXPECT().Sync(mock.Anything, mock.MatchedBy(func(job sync.SyncJob) bool {
					return job.Name == "backup1"
				})).Return(nil).Once()
				msy.EXPECT().Sync(mock.Anything, mock.MatchedBy(func(job sync.SyncJob) bool {
					return job.Name == "backup2"
				})).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name: "close error is logged",
			setupConfig: func(t *testing.T) string {
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
				return tmpConfig
			},
			setupMocks: func(ms *mocks.MockSecretStore, mt *mocks.MockTailscaleProvider, msy *mocks.MockSyncService) {
				ms.EXPECT().Get(mock.Anything, "TS_AUTHKEY").
					Return("tskey-test-12345", nil).Once()
				ms.EXPECT().Get(mock.Anything, "NAS_CREDS").
					Return("testuser:testpass", nil).Once()
				mt.EXPECT().Start(mock.Anything, "tskey-test-12345", "vanish-backup").
					Return(nil).Once()
				mt.EXPECT().LocalAddr().
					Return("100.64.0.1", nil).Once()
				mt.EXPECT().Close().
					Return(fmt.Errorf("close error")).Once()
				msy.EXPECT().Sync(mock.Anything, mock.Anything).
					Return(nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockSecretStore(t)
			mockTS := mocks.NewMockTailscaleProvider(t)
			mockSync := mocks.NewMockSyncService(t)

			vaultSvc := vault.NewVaultService(mockStore)
			tt.setupMocks(mockStore, mockTS, mockSync)

			opts := SyncOptions{
				ConfigPath: tt.setupConfig(t),
				Timeout:    30 * time.Second,
			}

			err := Sync(vaultSvc, mockTS, mockSync, opts)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
