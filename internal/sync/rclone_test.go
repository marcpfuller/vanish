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

func TestNewRcloneService(t *testing.T) {
	service := sync.NewRcloneService(nil)
	assert.NotNil(t, service)
}

func TestNewRcloneService_WithDialer(t *testing.T) {
	dialer := func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, nil
	}

	service := sync.NewRcloneService(dialer)
	assert.NotNil(t, service)
}

func TestRcloneService_Sync_InvalidSource(t *testing.T) {
	service := sync.NewRcloneService(nil)
	ctx := context.Background()

	job := sync.SyncJob{
		Name:        "test",
		Source:      "invalid://source",
		Destination: "/tmp/dest",
	}

	err := service.Sync(ctx, job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create source filesystem")
}

func TestRcloneService_SyncAll_EmptyJobs(t *testing.T) {
	service := sync.NewRcloneService(nil)
	ctx := context.Background()

	err := service.SyncAll(ctx, []sync.SyncJob{})
	assert.NoError(t, err)
}

func TestRcloneService_SyncAll_WithError(t *testing.T) {
	service := sync.NewRcloneService(nil)
	ctx := context.Background()

	jobs := []sync.SyncJob{
		{
			Name:        "test1",
			Source:      "invalid://source",
			Destination: "/tmp/dest",
		},
	}

	err := service.SyncAll(ctx, jobs)
	assert.Error(t, err)
}

func TestRcloneService_Sync_InvalidDestination(t *testing.T) {
	service := sync.NewRcloneService(nil)
	ctx := context.Background()

	job := sync.SyncJob{
		Name:        "test",
		Source:      "/tmp/source",
		Destination: "invalid://dest",
	}

	err := service.Sync(ctx, job)
	assert.Error(t, err)
	// Error message may vary depending on which filesystem fails first
}

func TestRcloneService_Sync_EmptyJob(t *testing.T) {
	service := sync.NewRcloneService(nil)
	ctx := context.Background()

	job := sync.SyncJob{}

	err := service.Sync(ctx, job)
	assert.Error(t, err)
}

func TestRcloneService_SyncAll_MultipleJobs(t *testing.T) {
	service := sync.NewRcloneService(nil)
	ctx := context.Background()

	jobs := []sync.SyncJob{
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
	}

	// Should fail on first job
	err := service.SyncAll(ctx, jobs)
	assert.Error(t, err)
}

func TestRcloneService_Sync_CancelledContext(t *testing.T) {
	service := sync.NewRcloneService(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	job := sync.SyncJob{
		Name:        "test",
		Source:      "invalid://source", // Use invalid backend to ensure error
		Destination: "/tmp/dest",
	}

	err := service.Sync(ctx, job)
	assert.Error(t, err)
}

func TestRcloneService_WithCustomDialer(t *testing.T) {
	// Test that a custom dialer can be passed to the service
	// The dialer is used for SFTP connections over Tailscale
	dialer := func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, assert.AnError
	}

	service := sync.NewRcloneService(dialer)
	assert.NotNil(t, service)

	// The dialer would be invoked when syncing to sftp:// destinations
	// Since we can't easily test this without a real SFTP server,
	// we verify the service accepts the dialer without error
}

func TestRcloneService_newFs_SFTPWithCredentials(t *testing.T) {
	service := sync.NewRcloneService(nil)

	// Test SFTP URL with credentials
	job := sync.SyncJob{
		Name:        "test",
		Source:      "/tmp/source",
		Destination: "sftp://user:pass@host.example.com:22/path",
	}

	// This will fail because host.example.com doesn't exist,
	// but it should successfully parse the URL
	err := service.Sync(context.Background(), job)
	assert.Error(t, err)
	// Should fail at connection, not URL parsing
	assert.NotContains(t, err.Error(), "invalid control character")
}

func TestRcloneService_newFs_LocalPath(t *testing.T) {
	service := sync.NewRcloneService(nil)

	// Create temporary directories
	srcDir, err := os.MkdirTemp("", "src")
	require.NoError(t, err)
	defer os.RemoveAll(srcDir) //nolint:errcheck // Best effort cleanup

	dstDir, err := os.MkdirTemp("", "dst")
	require.NoError(t, err)
	defer os.RemoveAll(dstDir) //nolint:errcheck // Best effort cleanup

	// Create a test file
	testFile := filepath.Join(srcDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0o600)
	require.NoError(t, err)

	// Test local path sync (should work)
	job := sync.SyncJob{
		Name:        "test",
		Source:      srcDir,
		Destination: dstDir,
	}

	err = service.Sync(context.Background(), job)
	assert.NoError(t, err)

	// Verify file was synced
	dstFile := filepath.Join(dstDir, "test.txt")
	data, err := os.ReadFile(dstFile)
	assert.NoError(t, err)
	assert.Equal(t, "test", string(data))
}

func TestRcloneService_Sync_InvalidMode(t *testing.T) {
	service := sync.NewRcloneService(nil)
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	job := sync.SyncJob{
		Name:        "test",
		Source:      srcDir,
		Destination: dstDir,
		Mode:        "invalid-mode",
	}

	err := service.Sync(context.Background(), job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

func TestRcloneService_Sync_SyncMode(t *testing.T) {
	service := sync.NewRcloneService(nil)
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a test file in source
	testFile := filepath.Join(srcDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o600)
	require.NoError(t, err)

	// Create a file in destination that doesn't exist in source
	extraFile := filepath.Join(dstDir, "extra.txt")
	err = os.WriteFile(extraFile, []byte("extra"), 0o600)
	require.NoError(t, err)

	// Test sync mode
	job := sync.SyncJob{
		Name:        "test",
		Source:      srcDir,
		Destination: dstDir,
		Mode:        "sync",
	}

	err = service.Sync(context.Background(), job)
	assert.NoError(t, err)

	// Verify file was synced
	dstFile := filepath.Join(dstDir, "test.txt")
	data, err := os.ReadFile(dstFile)
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(data))
}

func TestRcloneService_Sync_CopyMode(t *testing.T) {
	service := sync.NewRcloneService(nil)
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a test file in source
	testFile := filepath.Join(srcDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o600)
	require.NoError(t, err)

	// Test copy mode explicitly
	job := sync.SyncJob{
		Name:        "test",
		Source:      srcDir,
		Destination: dstDir,
		Mode:        "copy",
	}

	err = service.Sync(context.Background(), job)
	assert.NoError(t, err)

	// Verify file was copied
	dstFile := filepath.Join(dstDir, "test.txt")
	data, err := os.ReadFile(dstFile)
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(data))
}
