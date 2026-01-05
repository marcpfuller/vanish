package cmd

import (
	"context"
	"fmt"
	"log"
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
// This implements the core workflow:
// 1. Fetch TS_AUTHKEY and NAS_CREDS from keyring or environment variables
// 2. Initialize tsnet.Server (Ephemeral)
// 3. Bridge rclone SFTP to tsnet dialer
// 4. Execute sync.Sync for jobs defined in config.yaml
func Sync(vaultService *vault.VaultService, tsProvider network.TailscaleProvider, syncService sync.SyncService, opts SyncOptions) error {
	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	log.Println("=== Vanish Sync ===")
	log.Println()

	// Step 1: Fetch secrets from keyring or environment variables
	log.Println("📡 Fetching credentials...")
	authKey, err := vaultService.GetTailscaleAuthKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Tailscale auth key: %w", err)
	}

	username, password, err := vaultService.GetNASCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get NAS credentials: %w", err)
	}

	log.Println("✓ Credentials retrieved")
	log.Printf("✓ NAS user: %s\n", username)
	log.Println()

	// Step 3: Initialize ephemeral Tailscale server
	log.Println("🔐 Starting ephemeral Tailscale VPN...")
	defer func() {
		if err := tsProvider.Close(); err != nil {
			log.Printf("Warning: failed to close Tailscale: %v\n", err)
		}
	}()

	if err := tsProvider.Start(ctx, authKey, network.DefaultTailscaleHostname); err != nil {
		return fmt.Errorf("failed to start Tailscale: %w", err)
	}

	localIP, err := tsProvider.LocalAddr()
	if err != nil {
		return fmt.Errorf("failed to get Tailscale IP: %w", err)
	}

	log.Printf("✓ Tailscale connected: %s\n", localIP)
	log.Println()

	// Step 4: Use injected sync service
	log.Println("🔄 Preparing sync service...")

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
				Mode:        cfgJob.Mode,
			})
		}
	}

	if len(jobs) == 0 {
		log.Println("⚠️  No enabled sync jobs found in config")
		return nil
	}

	// Check for destructive sync operations and warn user
	hasSyncMode := false
	for _, job := range jobs {
		if job.Mode == "sync" {
			hasSyncMode = true
			break
		}
	}

	if hasSyncMode {
		log.Println()
		log.Println("⚠️  WARNING: Some jobs use 'sync' mode which is DESTRUCTIVE!")
		log.Println("   This will DELETE files from destination that don't exist in source.")
		fmt.Print("   Continue? (yes/no): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			log.Println("❌ Sync cancelled (read error)")
			return nil
		}
		if response != "yes" && response != "y" && response != "YES" && response != "Y" {
			log.Println("❌ Sync cancelled by user")
			return nil
		}
		log.Println()
	}

	log.Printf("✓ Found %d enabled sync job(s)\n", len(jobs))
	log.Println()

	// Step 5: Execute sync jobs
	log.Println("📦 Starting backup sync...")
	for i, job := range jobs {
		mode := job.Mode
		if mode == "" {
			mode = "copy"
		}
		log.Printf("[%d/%d] %s (%s mode)\n", i+1, len(jobs), job.Name, mode)
		log.Printf("  Source: %s\n", job.Source)
		log.Printf("  Destination: %s\n", maskCredentials(job.Destination))

		if err := syncService.Sync(ctx, job); err != nil {
			return fmt.Errorf("sync failed for %s: %w", job.Name, err)
		}

		log.Println("  ✓ Complete")
		log.Println()
	}

	log.Println("✓ All sync jobs completed successfully!")
	return nil
}

// injectCredentials adds username and password to SFTP URLs if not already present
func injectCredentials(destination, username, password string) string {
	// Simple implementation: assumes sftp://host:port/path format
	// In production, use proper URL parsing
	if len(destination) > 7 && destination[:7] == "sftp://" {
		// Check if credentials are already present (look for @ symbol before any / or port number)
		rest := destination[7:]
		foundAt := false
		for i := 0; i < len(rest); i++ {
			if rest[i] == '@' {
				// Credentials already present, return as-is
				foundAt = true
				break
			}
			if rest[i] == '/' {
				// Reached path without finding @, no credentials present
				break
			}
			// Check if this is a port number (: followed by digits)
			if rest[i] == ':' && i+1 < len(rest) {
				// Look ahead to see if this is a port (digits) or password (anything else)
				if rest[i+1] >= '0' && rest[i+1] <= '9' {
					// This is a port number, no credentials present
					break
				}
				// This is likely a password separator, keep looking for @
			}
		}
		if foundAt {
			return destination
		}
		// No credentials found, inject them
		return fmt.Sprintf("sftp://%s:%s@%s", username, password, rest)
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
