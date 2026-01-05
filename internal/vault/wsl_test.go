package vault

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectKeychainBackend_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}
	backend := DetectKeychainBackend()
	if backend != "Windows Credential Manager" {
		t.Errorf("Expected 'Windows Credential Manager', got %q", backend)
	}
}

func TestDetectKeychainBackend_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}
	backend := DetectKeychainBackend()
	if backend != "macOS Keychain" {
		t.Errorf("Expected 'macOS Keychain', got %q", backend)
	}
}

func TestDetectKeychainBackend_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}
	backend := DetectKeychainBackend()
	// Should contain "Linux" in the result
	if !contains(backend, "Linux") {
		t.Errorf("Expected backend to contain 'Linux', got %q", backend)
	}
}

func TestIsWSL_EnvVars(t *testing.T) {
	// Save original env vars
	origDistro := os.Getenv("WSL_DISTRO_NAME")
	origInterop := os.Getenv("WSL_INTEROP")
	defer func() {
		if origDistro != "" {
			require.NoError(t, os.Setenv("WSL_DISTRO_NAME", origDistro))
		} else {
			require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
		}
		if origInterop != "" {
			require.NoError(t, os.Setenv("WSL_INTEROP", origInterop))
		} else {
			require.NoError(t, os.Unsetenv("WSL_INTEROP"))
		}
	}()

	// Test with WSL_DISTRO_NAME
	require.NoError(t, os.Setenv("WSL_DISTRO_NAME", "Ubuntu"))
	require.NoError(t, os.Unsetenv("WSL_INTEROP"))
	if !isWSL() {
		t.Error("Expected isWSL to return true with WSL_DISTRO_NAME set")
	}

	// Test with WSL_INTEROP
	require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
	require.NoError(t, os.Setenv("WSL_INTEROP", "/run/WSL/8_interop"))
	if !isWSL() {
		t.Error("Expected isWSL to return true with WSL_INTEROP set")
	}

	// Test with both unset (will check /proc/version if on Linux)
	require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
	require.NoError(t, os.Unsetenv("WSL_INTEROP"))
	// Just verify it doesn't panic
	_ = isWSL()
}

func TestGetWSLKeychainHelp_NotWSL(t *testing.T) {
	// Save original env vars
	origDistro := os.Getenv("WSL_DISTRO_NAME")
	origInterop := os.Getenv("WSL_INTEROP")
	defer func() {
		if origDistro != "" {
			require.NoError(t, os.Setenv("WSL_DISTRO_NAME", origDistro))
		}
		if origInterop != "" {
			require.NoError(t, os.Setenv("WSL_INTEROP", origInterop))
		}
	}()

	// Unset WSL env vars
	require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
	require.NoError(t, os.Unsetenv("WSL_INTEROP"))

	// On non-WSL systems, should return empty string (unless /proc/version says otherwise)
	help := GetWSLKeychainHelp()
	// We can't assert much here since it depends on the actual environment
	// Just verify it doesn't panic
	_ = help
}

func TestGetWSLKeychainHelp_WithWSL(t *testing.T) {
	// Save original env vars
	origDistro := os.Getenv("WSL_DISTRO_NAME")
	defer func() {
		if origDistro != "" {
			require.NoError(t, os.Setenv("WSL_DISTRO_NAME", origDistro))
		} else {
			require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
		}
	}()

	// Set WSL env var
	require.NoError(t, os.Setenv("WSL_DISTRO_NAME", "Ubuntu"))

	help := GetWSLKeychainHelp()
	if help == "" {
		t.Error("Expected non-empty help string for WSL")
	}

	// Should contain helpful information
	if !contains(help, "WSL") {
		t.Error("Expected help string to mention WSL")
	}
}

func TestContains_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"both empty", "", "", true},
		{"empty substr", "hello", "", true},
		{"empty string", "", "test", false},
		{"single char match", "a", "a", true},
		{"single char no match", "a", "b", false},
		{"substr at end", "hello world", "world", true},
		{"substr at start", "hello world", "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, expected %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestIndexSubstring_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{"both empty", "", "", 0},
		{"empty substr in non-empty", "hello", "", 0},
		{"empty string", "", "test", -1},
		{"single char found", "a", "a", 0},
		{"single char not found", "a", "b", -1},
		{"at end", "abc", "c", 2},
		{"at start", "abc", "a", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexSubstring(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("indexSubstring(%q, %q) = %d, expected %d", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}
