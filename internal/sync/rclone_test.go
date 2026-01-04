package sync_test

import (
	"context"
	"net"
	"testing"

	"github.com/bdkmv/vanish/internal/sync"
	"github.com/stretchr/testify/assert"
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
		Source:      "/tmp/source",
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
