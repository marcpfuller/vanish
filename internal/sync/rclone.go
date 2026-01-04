package sync

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	_ "github.com/rclone/rclone/backend/local" // Register local backend
	_ "github.com/rclone/rclone/backend/sftp"  // Register SFTP backend
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/obscure"
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
	// Parse source and destination with proper configuration
	fsrc, err := r.newFs(ctx, job.Source)
	if err != nil {
		return fmt.Errorf("failed to create source filesystem: %w", err)
	}

	fdst, err := r.newFs(ctx, job.Destination)
	if err != nil {
		return fmt.Errorf("failed to create destination filesystem: %w", err)
	}

	// Determine mode (default to "copy" if not specified)
	mode := job.Mode
	if mode == "" {
		mode = "copy"
	}

	// Execute based on mode
	switch mode {
	case "copy":
		// Copy mode: only adds/updates files, never deletes
		if err := sync.CopyDir(ctx, fdst, fsrc, false); err != nil {
			return fmt.Errorf("copy failed for job %s: %w", job.Name, err)
		}
	case "sync":
		// Sync mode: makes destination match source exactly (destructive)
		if err := sync.Sync(ctx, fdst, fsrc, false); err != nil {
			return fmt.Errorf("sync failed for job %s: %w", job.Name, err)
		}
	default:
		return fmt.Errorf("invalid mode %q for job %s: must be 'copy' or 'sync'", mode, job.Name)
	}

	return nil
}

// newFs creates a new fs.Fs from a path, handling SFTP URLs with embedded credentials
func (r *RcloneService) newFs(ctx context.Context, path string) (fs.Fs, error) {
	// Check if it's an SFTP URL with credentials
	if strings.HasPrefix(path, "sftp://") {
		u, err := url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %w", err)
		}

		// If URL contains user info, configure SFTP connection dynamically
		if u.User != nil {
			username := u.User.Username()
			password, _ := u.User.Password()
			port := u.Port()
			if port == "" {
				port = "22"
			}

			// Create the remote path (after the hostname:port)
			remotePath := u.Path
			if remotePath == "" {
				remotePath = "/"
			}

			// Use connection string format with parameters
			// Format: :sftp,host=...,user=...,pass=...,port=...:path
			connectionString := fmt.Sprintf(
				":sftp,host=%s,user=%s,pass=%s,port=%s:%s",
				u.Hostname(),
				username,
				obscure.MustObscure(password),
				port,
				remotePath,
			)

			return fs.NewFs(ctx, connectionString)
		}
	}

	// For local paths or URLs without credentials, use standard parsing
	return fs.NewFs(ctx, path)
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
