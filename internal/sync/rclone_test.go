package sync_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/bdkmv/vanish/internal/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRcloneService_Construction(t *testing.T) {
	tests := []struct {
		name   string
		dialer func(ctx context.Context, network, address string) (net.Conn, error)
	}{
		{
			name:   "without dialer",
			dialer: nil,
		},
		{
			name: "with dialer",
			dialer: func(ctx context.Context, network, address string) (net.Conn, error) {
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := sync.NewRcloneService(tt.dialer)
			assert.NotNil(t, service)
		})
	}
}

func TestRcloneService_Sync(t *testing.T) {
	tests := []struct {
		name          string
		setupDirs     func(t *testing.T) (srcDir, dstDir string)
		job           func(srcDir, dstDir string) sync.SyncJob
		expectError   bool
		errorContains string
		validate      func(t *testing.T, srcDir, dstDir string)
	}{
		{
			name: "invalid source",
			setupDirs: func(t *testing.T) (string, string) {
				return "", ""
			},
			job: func(_, _ string) sync.SyncJob {
				return sync.SyncJob{
					Name:        "test",
					Source:      "invalid://source",
					Destination: "/tmp/dest",
				}
			},
			expectError:   true,
			errorContains: "failed to create source filesystem",
		},
		{
			name: "invalid destination",
			setupDirs: func(t *testing.T) (string, string) {
				return "", ""
			},
			job: func(_, _ string) sync.SyncJob {
				return sync.SyncJob{
					Name:        "test",
					Source:      "/tmp/source",
					Destination: "invalid://dest",
				}
			},
			expectError: true,
		},
		{
			name: "empty job",
			setupDirs: func(t *testing.T) (string, string) {
				return "", ""
			},
			job: func(_, _ string) sync.SyncJob {
				return sync.SyncJob{}
			},
			expectError: true,
		},
		{
			name: "cancelled context",
			setupDirs: func(t *testing.T) (string, string) {
				return "", ""
			},
			job: func(_, _ string) sync.SyncJob {
				return sync.SyncJob{
					Name:        "test",
					Source:      "invalid://source",
					Destination: "/tmp/dest",
				}
			},
			expectError: true,
		},
		{
			name: "local path sync",
			setupDirs: func(t *testing.T) (string, string) {
				srcDir := t.TempDir()
				dstDir := t.TempDir()

				testFile := filepath.Join(srcDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test"), 0o600)
				require.NoError(t, err)

				return srcDir, dstDir
			},
			job: func(srcDir, dstDir string) sync.SyncJob {
				return sync.SyncJob{
					Name:        "test",
					Source:      srcDir,
					Destination: dstDir,
				}
			},
			expectError: false,
			validate: func(t *testing.T, srcDir, dstDir string) {
				dstFile := filepath.Join(dstDir, "test.txt")
				data, err := os.ReadFile(dstFile)
				assert.NoError(t, err)
				assert.Equal(t, "test", string(data))
			},
		},
		{
			name: "invalid mode",
			setupDirs: func(t *testing.T) (string, string) {
				return t.TempDir(), t.TempDir()
			},
			job: func(srcDir, dstDir string) sync.SyncJob {
				return sync.SyncJob{
					Name:        "test",
					Source:      srcDir,
					Destination: dstDir,
					Mode:        "invalid-mode",
				}
			},
			expectError:   true,
			errorContains: "invalid mode",
		},
		{
			name: "sync mode",
			setupDirs: func(t *testing.T) (string, string) {
				srcDir := t.TempDir()
				dstDir := t.TempDir()

				testFile := filepath.Join(srcDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test content"), 0o600)
				require.NoError(t, err)

				extraFile := filepath.Join(dstDir, "extra.txt")
				err = os.WriteFile(extraFile, []byte("extra"), 0o600)
				require.NoError(t, err)

				return srcDir, dstDir
			},
			job: func(srcDir, dstDir string) sync.SyncJob {
				return sync.SyncJob{
					Name:        "test",
					Source:      srcDir,
					Destination: dstDir,
					Mode:        "sync",
				}
			},
			expectError: false,
			validate: func(t *testing.T, srcDir, dstDir string) {
				dstFile := filepath.Join(dstDir, "test.txt")
				data, err := os.ReadFile(dstFile)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(data))
			},
		},
		{
			name: "copy mode",
			setupDirs: func(t *testing.T) (string, string) {
				srcDir := t.TempDir()
				dstDir := t.TempDir()

				testFile := filepath.Join(srcDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test content"), 0o600)
				require.NoError(t, err)

				return srcDir, dstDir
			},
			job: func(srcDir, dstDir string) sync.SyncJob {
				return sync.SyncJob{
					Name:        "test",
					Source:      srcDir,
					Destination: dstDir,
					Mode:        "copy",
				}
			},
			expectError: false,
			validate: func(t *testing.T, srcDir, dstDir string) {
				dstFile := filepath.Join(dstDir, "test.txt")
				data, err := os.ReadFile(dstFile)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(data))
			},
		},
		{
			name: "sftp with credentials",
			setupDirs: func(t *testing.T) (string, string) {
				return "", ""
			},
			job: func(_, _ string) sync.SyncJob {
				return sync.SyncJob{
					Name:        "test",
					Source:      "/tmp/source",
					Destination: "sftp://user:pass@host.example.com:22/path",
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := sync.NewRcloneService(nil)
			ctx := context.Background()

			if tt.name == "cancelled context" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			srcDir, dstDir := tt.setupDirs(t)
			job := tt.job(srcDir, dstDir)

			err := service.Sync(ctx, job)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, srcDir, dstDir)
				}
			}
		})
	}
}

func TestRcloneService_SyncAll(t *testing.T) {
	tests := []struct {
		name        string
		jobs        []sync.SyncJob
		expectError bool
	}{
		{
			name:        "empty jobs",
			jobs:        []sync.SyncJob{},
			expectError: false,
		},
		{
			name: "with error",
			jobs: []sync.SyncJob{
				{
					Name:        "test1",
					Source:      "invalid://source",
					Destination: "/tmp/dest",
				},
			},
			expectError: true,
		},
		{
			name: "multiple jobs with error",
			jobs: []sync.SyncJob{
				{
					Name:        "test1",
					Source:      "invalid1://source",
					Destination: "/tmp/dest1",
				},
				{
					Name:        "test2",
					Source:      "invalid2://source",
					Destination: "/tmp/dest2",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := sync.NewRcloneService(nil)
			ctx := context.Background()

			err := service.SyncAll(ctx, tt.jobs)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRcloneService_WithCustomDialer(t *testing.T) {
	dialer := func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, assert.AnError
	}

	service := sync.NewRcloneService(dialer)
	assert.NotNil(t, service)
}
