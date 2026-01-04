// Package main is the entry point for the vanish CLI application.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bdkmv/vanish/internal/cmd"
	"github.com/bdkmv/vanish/internal/network"
	"github.com/bdkmv/vanish/internal/sync"
	"github.com/bdkmv/vanish/internal/vault"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("vanish - Zero-Footprint Travel Backup CLI")
		fmt.Println("Usage: vanish <command>")
		fmt.Println("Commands:")
		fmt.Println("  setup - Configure Bitwarden access token")
		fmt.Println("  sync  - Perform backup sync via Tailscale")
		os.Exit(1)
	}

	command := os.Args[1]

	// Auto-detect and create appropriate secret store
	store := createSecretStore()

	switch command {
	case "setup":
		if err := cmd.Setup(store); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "sync":
		// Create vault service with store and Bitwarden client
		bwClient := vault.NewBitwardenSDKClient()
		vaultSvc := vault.NewVaultService(store, bwClient)

		// Create network and sync services
		tsProvider := network.NewTsnetProvider()
		syncSvc := sync.NewRcloneService(tsProvider.Dial)

		// Configure sync options
		opts := cmd.SyncOptions{
			ConfigPath: "./config.yaml",
			Timeout:    10 * time.Minute,
		}

		if err := cmd.Sync(vaultSvc, tsProvider, syncSvc, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}

// createSecretStore creates the appropriate secret store based on environment
func createSecretStore() vault.SecretStore {
	// Check if user explicitly wants file-based storage
	if os.Getenv("VANISH_USE_FILE_STORE") == "1" {
		if store, err := vault.NewFileStore(""); err == nil {
			return store
		}
	}

	// Try system keyring first
	store := vault.NewKeyringStore()

	// Test if keyring is available by trying to get a non-existent key
	_, err := store.Get(context.Background(), "vanish-test-key")

	// If error is NOT "not found", it means keyring isn't working
	// (the not-found error is expected and means keyring IS working)
	if err != nil && err != vault.ErrSecretNotFound && !isKeyringNotFoundError(err) {
		// Keyring not available, fall back to file store
		fmt.Fprintln(os.Stderr, "Warning: System keyring not available, using encrypted file storage")
		fmt.Fprintln(os.Stderr, "Set VANISH_USE_FILE_STORE=1 to suppress this warning")

		if fileStore, err := vault.NewFileStore(""); err == nil {
			return fileStore
		}

		// If file store also fails, return keyring anyway and let it fail properly
		fmt.Fprintln(os.Stderr, "Warning: Could not initialize file store either")
	}

	return store
}

func isKeyringNotFoundError(err error) bool {
	// go-keyring returns various "not found" error messages
	errMsg := err.Error()
	return contains(errMsg, "not found") ||
		contains(errMsg, "cannot find") ||
		contains(errMsg, "secret not found")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexSubstring(s, substr) >= 0
}

func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
