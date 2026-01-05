// Package sync provides file synchronization services using rclone.
package sync

import "context"

// SyncJob represents a single sync operation
//
//nolint:revive // SyncJob stuttering is intentional for clarity
type SyncJob struct {
	Name        string
	Source      string
	Destination string
	Mode        string // "copy" (default) or "sync" (destructive)
}

// SyncService defines the interface for executing sync operations
//
//nolint:revive // SyncService stuttering is intentional for clarity
type SyncService interface {
	// Sync executes a sync job from source to destination
	Sync(ctx context.Context, job SyncJob) error

	// SyncAll executes multiple sync jobs
	SyncAll(ctx context.Context, jobs []SyncJob) error
}
