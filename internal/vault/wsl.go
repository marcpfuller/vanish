package vault

import (
	"fmt"
	"os"
	"runtime"
)

// DetectKeychainBackend returns information about which keychain backend will be used
func DetectKeychainBackend() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows Credential Manager"
	case "darwin":
		return "macOS Keychain"
	case "linux":
		// Check if we're in WSL
		if isWSL() {
			return "Linux (WSL - may need Windows Credential Manager access)"
		}
		return "Linux Secret Service (requires DBus)"
	default:
		return fmt.Sprintf("Unknown platform: %s", runtime.GOOS)
	}
}

// isWSL checks if we're running under Windows Subsystem for Linux
func isWSL() bool {
	// Check for WSL-specific environment variables or files
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}

	// Check /proc/version for Microsoft or WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		content := string(data)
		if contains(content, "microsoft") || contains(content, "Microsoft") || contains(content, "WSL") {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(indexSubstring(s, substr) >= 0))
}

func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetWSLKeychainHelp returns helpful information for using keychain in WSL
func GetWSLKeychainHelp() string {
	if !isWSL() {
		return ""
	}

	return `
WSL Detected: The keyring may not work with Linux DBus in WSL.

To use Windows Credential Manager from WSL, you have a few options:

1. Use wsl-keyring helper (recommended):
   Install: https://github.com/bketelsen/wsl-keyring

2. Use pass (password store):
   Install: sudo apt install pass
   Then set: export KEYRING_BACKEND=pass

3. Use file-based storage (less secure):
   Set: export KEYRING_BACKEND=file

4. Access Windows directly:
   Install: sudo apt install wslu
   Use Windows PowerShell to manage credentials

For this application, the simplest approach is to install a DBus session:
   sudo apt install dbus-x11
   eval $(dbus-launch --sh-syntax)

Or use a file-based keyring for development (not recommended for production):
   export KEYRING_BACKEND=file
`
}
