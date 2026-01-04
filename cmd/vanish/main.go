// Package main is the entry point for the vanish CLI application.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bdkmv/vanish/internal/cmd"
	"github.com/bdkmv/vanish/internal/network"
	"github.com/bdkmv/vanish/internal/scheduler"
	"github.com/bdkmv/vanish/internal/sync"
	"github.com/bdkmv/vanish/internal/vault"
	"github.com/bdkmv/vanish/pkg/config"
)

func main() {
	// Set up logging to file if VANISH_LOG_FILE is set
	logFile := os.Getenv("VANISH_LOG_FILE")
	if logFile != "" {
		setupLogging(logFile)
	} else {
		// For interactive mode, configure stderr with timestamps
		log.SetOutput(os.Stderr)
		log.SetFlags(log.Ldate | log.Ltime)
	}

	log.Println("vanish starting...")

	// Catch any panics and print them
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal error: %v\n", r)
			os.Exit(1)
		}
	}()

	if len(os.Args) < 2 {
		log.Println("vanish - Zero-Footprint Travel Backup CLI")
		log.Println("Usage: vanish <command>")
		log.Println("Commands:")
		log.Println("  setup   - Store Tailscale and NAS credentials")
		log.Println("  sync    - Perform backup sync via Tailscale")
		log.Println("  service - Run as a scheduled service (use VANISH_LOG_FILE for logging)")
		os.Exit(1)
	}

	command := os.Args[1]

	// Auto-detect and create appropriate secret store
	store := createSecretStore()

	switch command {
	case "setup":
		log.Println("Running setup command...")
		if err := cmd.Setup(store); err != nil {
			log.Printf("Setup error: %v\n", err)
			os.Exit(1)
		}
	case "sync":
		log.Println("Running sync command...")
		// Create vault service
		vaultSvc := vault.NewVaultService(store)

		// Create network and sync services
		tsProvider := network.NewTsnetProvider()
		syncSvc := sync.NewRcloneService(tsProvider.Dial)

		// Configure sync options
		opts := cmd.SyncOptions{
			ConfigPath: "./config.yaml",
			Timeout:    10 * time.Minute,
		}

		if err := cmd.Sync(vaultSvc, tsProvider, syncSvc, opts); err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "service":
		log.Println("Running service command...")
		if err := runService(store); err != nil {
			log.Printf("Service error: %v\n", err)
			os.Exit(1)
		}
	default:
		log.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

// createSecretStore creates the appropriate secret store based on environment
func createSecretStore() vault.SecretStore {
	// Check if user explicitly wants file-based storage
	if os.Getenv("VANISH_USE_FILE_STORE") == "1" {
		store, err := vault.NewFileStore("")
		if err == nil {
			log.Println("Using file-based secret storage")
			return store
		}
		log.Printf("Warning: Failed to create file store: %v\n", err)
	}

	// Try system keyring first
	log.Println("Attempting to initialize system keyring...")
	store := vault.NewKeyringStore()

	// Test if keyring is available by trying to get a non-existent key
	_, err := store.Get(context.Background(), "vanish-test-key")

	// If error is NOT "not found", it means keyring isn't working
	// (the not-found error is expected and means keyring IS working)
	if err != nil && err != vault.ErrSecretNotFound && !isKeyringNotFoundError(err) {
		// Keyring not available, fall back to file store
		log.Printf("Warning: System keyring not available (%v), using encrypted file storage\n", err)
		log.Println("Set VANISH_USE_FILE_STORE=1 to suppress this warning")

		if fileStore, err := vault.NewFileStore(""); err == nil {
			return fileStore
		} else {
			// If file store also fails, return keyring anyway and let it fail properly
			log.Printf("Warning: Could not initialize file store either: %v\n", err)
		}
	} else {
		log.Println("System keyring initialized successfully")
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

// runService runs vanish as a scheduled service
func runService(store vault.SecretStore) error {
	ctx := context.Background()

	// Get config path from args or use default
	configPath := "./config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if scheduling is enabled
	if !cfg.Schedule.Enabled {
		return fmt.Errorf("scheduling is not enabled in config (set schedule.enabled: true)")
	}

	if cfg.Schedule.Cron == "" {
		return fmt.Errorf("cron schedule is empty in config")
	}

	log.Println("🚀 Starting vanish service...")

	// Create vault service
	vaultSvc := vault.NewVaultService(store)
	defer func() {
		if err := vaultSvc.Close(); err != nil {
			log.Printf("Warning: failed to close vault service: %v", err)
		}
	}()

	// Create network and sync services
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	// Create scheduler
	sched := scheduler.NewScheduler(vaultSvc, tsProvider, syncSvc, configPath)

	// Start scheduler
	if err := sched.Start(ctx, cfg.Schedule.Cron); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	defer sched.Stop()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("✅ Service is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("\n🛑 Received shutdown signal, stopping service...")
	return nil
}

// setupLogging configures logging to write to a file with timestamps
func setupLogging(logFilePath string) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create log directory: %v\n", err)
		return
	}

	// Open log file (create if doesn't exist, append if exists)
	// #nosec G304 - logFilePath comes from VANISH_LOG_FILE environment variable, not untrusted user input
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open log file: %v\n", err)
		return
	}

	// Write to both file and stderr for visibility
	multiWriter := io.MultiWriter(logFile, os.Stderr)

	// Configure log package with timestamps
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("Logging initialized to: %s", logFilePath)
}
