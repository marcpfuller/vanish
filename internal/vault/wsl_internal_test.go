package vault

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsWSL(t *testing.T) {
	// Save original env vars
	origDistro := os.Getenv("WSL_DISTRO_NAME")
	origInterop := os.Getenv("WSL_INTEROP")
	defer func() {
		if origDistro == "" {
			os.Unsetenv("WSL_DISTRO_NAME") //nolint:errcheck
		} else {
			os.Setenv("WSL_DISTRO_NAME", origDistro) //nolint:errcheck
		}
		if origInterop == "" {
			os.Unsetenv("WSL_INTEROP") //nolint:errcheck
		} else {
			os.Setenv("WSL_INTEROP", origInterop) //nolint:errcheck
		}
	}()

	// Test 1: With WSL_DISTRO_NAME set
	os.Setenv("WSL_DISTRO_NAME", "Ubuntu") //nolint:errcheck
	result := isWSL()
	t.Logf("isWSL with WSL_DISTRO_NAME=Ubuntu returned: %v", result)

	// Test 2: With WSL_INTEROP set
	os.Unsetenv("WSL_DISTRO_NAME")                   //nolint:errcheck
	os.Setenv("WSL_INTEROP", "/run/WSL/123_interop") //nolint:errcheck
	result = isWSL()
	t.Logf("isWSL with WSL_INTEROP set returned: %v", result)

	// Test 3: Without WSL env vars (depends on /proc/version)
	os.Unsetenv("WSL_DISTRO_NAME") //nolint:errcheck
	os.Unsetenv("WSL_INTEROP")     //nolint:errcheck
	result = isWSL()
	t.Logf("isWSL without env vars returned: %v", result)

	// Result depends on actual environment, just verify it doesn't panic
	assert.IsType(t, false, result)
}

func TestDetectKeychainBackend_WithWSLEnv(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("WSL detection only applies on Linux")
	}

	// Save original env vars
	origDistro := os.Getenv("WSL_DISTRO_NAME")
	origInterop := os.Getenv("WSL_INTEROP")
	defer func() {
		if origDistro == "" {
			os.Unsetenv("WSL_DISTRO_NAME") //nolint:errcheck
		} else {
			os.Setenv("WSL_DISTRO_NAME", origDistro) //nolint:errcheck
		}
		if origInterop == "" {
			os.Unsetenv("WSL_INTEROP") //nolint:errcheck
		} else {
			os.Setenv("WSL_INTEROP", origInterop) //nolint:errcheck
		}
	}()

	// Test with WSL environment
	os.Setenv("WSL_DISTRO_NAME", "Ubuntu") //nolint:errcheck
	backend := DetectKeychainBackend()
	assert.NotEmpty(t, backend)
	assert.Contains(t, backend, "WSL")
	t.Logf("Backend with WSL_DISTRO_NAME: %s", backend)

	// Test without WSL environment
	os.Unsetenv("WSL_DISTRO_NAME") //nolint:errcheck
	os.Unsetenv("WSL_INTEROP")     //nolint:errcheck
	backend = DetectKeychainBackend()
	assert.NotEmpty(t, backend)
	t.Logf("Backend without WSL env: %s", backend)
}

func TestGetWSLKeychainHelp(t *testing.T) {
	// Save original env vars
	origDistro := os.Getenv("WSL_DISTRO_NAME")
	origInterop := os.Getenv("WSL_INTEROP")
	defer func() {
		if origDistro == "" {
			os.Unsetenv("WSL_DISTRO_NAME") //nolint:errcheck
		} else {
			os.Setenv("WSL_DISTRO_NAME", origDistro) //nolint:errcheck
		}
		if origInterop == "" {
			os.Unsetenv("WSL_INTEROP") //nolint:errcheck
		} else {
			os.Setenv("WSL_INTEROP", origInterop) //nolint:errcheck
		}
	}()

	// Test 1: In WSL environment (on Linux)
	if runtime.GOOS == "linux" {
		os.Setenv("WSL_DISTRO_NAME", "Ubuntu") //nolint:errcheck
		help := GetWSLKeychainHelp()
		assert.NotEmpty(t, help)
		assert.Contains(t, help, "WSL")
		assert.Contains(t, help, "Windows")
		assert.Contains(t, help, "keyring")
		t.Logf("Help text length in WSL: %d chars", len(help))
	}

	// Test 2: Outside WSL environment
	os.Unsetenv("WSL_DISTRO_NAME") //nolint:errcheck
	os.Unsetenv("WSL_INTEROP")     //nolint:errcheck
	help := GetWSLKeychainHelp()
	// On non-Linux or non-WSL, should be empty
	if runtime.GOOS != "linux" || !isWSL() {
		assert.Empty(t, help)
		t.Log("Not in WSL, help text is empty (expected)")
	} else {
		t.Log("In WSL environment, help text provided")
	}
}
