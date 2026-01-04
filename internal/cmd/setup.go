// Package cmd implements the CLI commands for vanish.
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bdkmv/vanish/internal/vault"
	"golang.org/x/term"
)

// Setup prompts for credentials and stores them securely in the system keychain
func Setup(store vault.SecretStore) error {
	ctx := context.Background()

	log.Println("=== Vanish Setup ===")
	log.Println("This will store your Tailscale and NAS credentials securely.")
	log.Printf("Keychain backend: %s\n", vault.DetectKeychainBackend())

	// Show WSL help if applicable
	if wslHelp := vault.GetWSLKeychainHelp(); wslHelp != "" {
		log.Println(wslHelp)
	}

	log.Println()

	// Prompt for Tailscale auth key
	fmt.Println("Please enter your Tailscale auth key:")
	fmt.Println("(Get one from: https://login.tailscale.com/admin/settings/keys)")
	fmt.Print("Tailscale Auth Key: ")

	tsKeyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // New line after password input
	if err != nil {
		return fmt.Errorf("failed to read Tailscale auth key (terminal may not support password input): %w", err)
	}
	tsKey := strings.TrimSpace(string(tsKeyBytes))

	if tsKey == "" {
		return fmt.Errorf("tailscale auth key cannot be empty")
	}

	// Store the Tailscale key
	if err := store.Set(ctx, vault.TailscaleAuthKeyItem, tsKey); err != nil {
		return fmt.Errorf("failed to store Tailscale auth key in keychain: %w", err)
	}

	log.Println("✓ Tailscale auth key stored successfully")
	log.Println()

	// Prompt for NAS credentials
	log.Println("Please enter your NAS credentials:")
	fmt.Print("NAS Username: ")

	// Use term.ReadPassword for username too (even though it won't echo)
	// This is more reliable on Windows after a previous ReadPassword call
	usernameBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // New line after input
	if err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	username := strings.TrimSpace(string(usernameBytes))

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	fmt.Print("NAS Password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // New line after password input
	if err != nil {
		return fmt.Errorf("failed to read password (terminal may not support password input): %w", err)
	}
	password := strings.TrimSpace(string(passwordBytes))

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Store NAS credentials in "username:password" format
	nasCreds := fmt.Sprintf("%s:%s", username, password)
	if err := store.Set(ctx, vault.NASCredsItem, nasCreds); err != nil {
		return fmt.Errorf("failed to store NAS credentials in keychain: %w", err)
	}

	log.Println("✓ NAS credentials stored successfully")
	log.Println()
	log.Println("✓ Setup complete!")
	log.Println()
	log.Println("You can now run 'vanish sync' to start backing up your files.")
	return nil
}
