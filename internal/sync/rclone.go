package sync

import (
	"context"
	"fmt"
	"net"

	_ "github.com/rclone/rclone/backend/sftp" // Register SFTP backend
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/sync"
)

// RcloneService implements SyncService using rclone for file synchronization
type RcloneService struct {
	dialer func(ctx context.Context, network, address string) (net.Conn, error)
}

// NewRcloneService creates a new rclone-based sync service
// The dialer parameter allows injecting a custom dialer (e.g., through Tailscale)
func NewRcloneService(dialer func(ctx context.Context, network, address string) (net.Conn, error)) *RcloneService {
	return &RcloneService{
		dialer: dialer,
	}
}

// Sync executes a sync job from source to destination
func (r *RcloneService) Sync(ctx context.Context, job SyncJob) error {
	// Custom dialer support would require rclone configuration
	// For now, we'll use standard SFTP
	// In production, you'd configure fs.Config or use rclone's config system

	// Parse source and destination
	fsrc, err := fs.NewFs(ctx, job.Source)
	if err != nil {
		return fmt.Errorf("failed to create source filesystem: %w", err)
	}

	fdst, err := fs.NewFs(ctx, job.Destination)
	if err != nil {
		return fmt.Errorf("failed to create destination filesystem: %w", err)
	}

	// Execute sync
	if err := sync.Sync(ctx, fdst, fsrc, false); err != nil {
		return fmt.Errorf("sync failed for job %s: %w", job.Name, err)
	}

	return nil
}

// SyncAll executes multiple sync jobs sequentially
func (r *RcloneService) SyncAll(ctx context.Context, jobs []SyncJob) error {
	for _, job := range jobs {
		if err := r.Sync(ctx, job); err != nil {
			return fmt.Errorf("failed at job %s: %w", job.Name, err)
		}
	}
	return nil
}
