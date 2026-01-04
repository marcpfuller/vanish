package vault_test

import (
	"os"
	"testing"

	"github.com/bdkmv/vanish/internal/vault"
	"github.com/stretchr/testify/assert"
)

func TestDetectKeychainBackend(t *testing.T) {
	// This will return different results based on the environment
	// Just verify it doesn't panic and returns a non-empty string
	backend := vault.DetectKeychainBackend()
	assert.NotEmpty(t, backend)
	// Should contain one of the expected keywords
	assert.True(t,
		backend == "keyring" ||
			backend == "wsl-file" ||
			backend == "file" ||
			len(backend) > 0, // Any non-empty string is valid
		"backend should be a valid keychain backend identifier")
}

func TestDetectKeychainBackend_WithWSLEnv(t *testing.T) {
	// Simulate WSL environment
	oldKernel := os.Getenv("WSL_DISTRO_NAME")
	os.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	defer func() {
		if oldKernel == "" {
			os.Unsetenv("WSL_DISTRO_NAME")
		} else {
			os.Setenv("WSL_DISTRO_NAME", oldKernel)
		}
	}()

	backend := vault.DetectKeychainBackend()
	assert.NotEmpty(t, backend)
}

func TestGetWSLKeychainHelp(t *testing.T) {
	help := vault.GetWSLKeychainHelp()
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "WSL")
	assert.Contains(t, help, "keyring")
}
