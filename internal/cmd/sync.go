package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bdkmv/vanish/internal/network"
	"github.com/bdkmv/vanish/internal/sync"
	"github.com/bdkmv/vanish/internal/vault"
	"github.com/bdkmv/vanish/pkg/config"
)

// SyncOptions contains options for the sync command
type SyncOptions struct {
	ConfigPath string
	Timeout    time.Duration
}

// Sync executes the complete backup sync workflow
// This implements Feature B from AGENT.md:
// 1. Authenticate with Bitwarden using keychain token
// 2. Fetch TS_AUTHKEY and NAS_CREDS from vault
// 3. Initialize tsnet.Server (Ephemeral)
// 4. Bridge rclone SFTP to tsnet dialer
// 5. Execute sync.Sync for jobs defined in config.yaml
func Sync(vaultService *vault.VaultService, tsProvider network.TailscaleProvider, syncService sync.SyncService, opts SyncOptions) error {
	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	fmt.Println("=== Vanish Sync ===")
	fmt.Println()

	// Step 1 & 2: Authenticate with Bitwarden and fetch secrets
	fmt.Println("📡 Fetching secrets from Bitwarden...")
	authKey, err := vaultService.GetTailscaleAuthKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Tailscale auth key: %w", err)
	}

	username, password, err := vaultService.GetNASCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get NAS credentials: %w", err)
	}

	fmt.Println("✓ Secrets retrieved")
	fmt.Printf("✓ NAS user: %s\n", username)
	fmt.Println()

	// Step 3: Initialize ephemeral Tailscale server
	fmt.Println("🔐 Starting ephemeral Tailscale VPN...")
	defer func() {
		if err := tsProvider.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Tailscale: %v\n", err)
		}
	}()

	if err := tsProvider.Start(ctx, authKey, network.DefaultTailscaleHostname); err != nil {
		return fmt.Errorf("failed to start Tailscale: %w", err)
	}

	localIP, err := tsProvider.LocalAddr()
	if err != nil {
		return fmt.Errorf("failed to get Tailscale IP: %w", err)
	}

	fmt.Printf("✓ Tailscale connected: %s\n", localIP)
	fmt.Println()

	// Step 4: Use injected sync service
	fmt.Println("🔄 Preparing sync service...")

	// Load sync configuration
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Filter enabled jobs and convert to sync jobs
	var jobs []sync.SyncJob
	for _, cfgJob := range cfg.SyncJobs {
		if cfgJob.Enabled {
			jobs = append(jobs, sync.SyncJob{
				Name:        cfgJob.Name,
				Source:      cfgJob.Source,
				Destination: injectCredentials(cfgJob.Destination, username, password),
			})
		}
	}

	if len(jobs) == 0 {
		fmt.Println("⚠️  No enabled sync jobs found in config")
		return nil
	}

	fmt.Printf("✓ Found %d enabled sync job(s)\n", len(jobs))
	fmt.Println()

	// Step 5: Execute sync jobs
	fmt.Println("📦 Starting backup sync...")
	for i, job := range jobs {
		fmt.Printf("[%d/%d] %s\n", i+1, len(jobs), job.Name)
		fmt.Printf("  Source: %s\n", job.Source)
		fmt.Printf("  Destination: %s\n", maskCredentials(job.Destination))

		if err := syncService.Sync(ctx, job); err != nil {
			return fmt.Errorf("sync failed for %s: %w", job.Name, err)
		}

		fmt.Println("  ✓ Complete")
		fmt.Println()
	}

	fmt.Println("✓ All sync jobs completed successfully!")
	return nil
}

// injectCredentials adds username and password to SFTP URLs
func injectCredentials(destination, username, password string) string {
	// Simple implementation: assumes sftp://host:port/path format
	// In production, use proper URL parsing
	if len(destination) > 7 && destination[:7] == "sftp://" {
		return fmt.Sprintf("sftp://%s:%s@%s", username, password, destination[7:])
	}
	return destination
}

// maskCredentials hides passwords in URLs for display
func maskCredentials(destination string) string {
	// Simple masking for display purposes
	// In production, use proper URL parsing
	for i := 0; i < len(destination); i++ {
		if destination[i] == ':' && i+1 < len(destination) && i > 7 {
			// Found potential password after username
			for j := i + 1; j < len(destination); j++ {
				if destination[j] == '@' {
					return destination[:i+1] + "****" + destination[j:]
				}
			}
		}
	}
	return destination
}
