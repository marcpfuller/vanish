// Package cmd implements the CLI commands for vanish.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bdkmv/vanish/internal/vault"
	"golang.org/x/term"
)

// Setup prompts for the Bitwarden access token and stores it securely in the system keychain
func Setup(store vault.SecretStore) error {
	ctx := context.Background()

	fmt.Println("=== Vanish Setup ===")
	fmt.Println("This will configure your Bitwarden access token for secure backup operations.")
	fmt.Printf("Keychain backend: %s\n", vault.DetectKeychainBackend())

	// Show WSL help if applicable
	if wslHelp := vault.GetWSLKeychainHelp(); wslHelp != "" {
		fmt.Println(wslHelp)
	}

	fmt.Println()

	// Check if token already exists
	existingToken, err := store.Get(ctx, vault.BitwardenTokenKey)
	if err == nil && existingToken != "" {
		fmt.Println("A Bitwarden token is already configured.")
		fmt.Print("Do you want to replace it? (yes/no): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			fmt.Println("Setup cancelled.")
			return nil
		}
	}

	// Prompt for Bitwarden token
	fmt.Println()
	fmt.Println("Please enter your Bitwarden Access Token:")
	fmt.Print("Token: ")

	// Read token securely (without echo)
	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	fmt.Println() // New line after password input

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Store the token in the keychain
	if err := store.Set(ctx, vault.BitwardenTokenKey, token); err != nil {
		return fmt.Errorf("failed to store token in keychain: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Successfully stored Bitwarden token in system keychain")
	fmt.Println("✓ Setup complete!")
	return nil
}
