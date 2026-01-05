package vault

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectKeychainBackend(t *testing.T) {
	tests := []struct {
		name         string
		goos         string
		wantContains string
		skip         bool
	}{
		{
			name:         "windows",
			goos:         "windows",
			wantContains: "Windows Credential Manager",
			skip:         runtime.GOOS != "windows",
		},
		{
			name:         "darwin",
			goos:         "darwin",
			wantContains: "macOS Keychain",
			skip:         runtime.GOOS != "darwin",
		},
		{
			name:         "linux",
			goos:         "linux",
			wantContains: "Linux",
			skip:         runtime.GOOS != "linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skipf("Skipping %s-specific test on %s", tt.goos, runtime.GOOS)
			}

			backend := DetectKeychainBackend()
			assert.Contains(t, backend, tt.wantContains)
		})
	}
}

func TestIsWSL_EnvVars(t *testing.T) {
	tests := []struct {
		name       string
		distroName string
		interop    string
		wantWSL    bool
	}{
		{
			name:       "WSL_DISTRO_NAME set",
			distroName: "Ubuntu",
			interop:    "",
			wantWSL:    true,
		},
		{
			name:       "WSL_INTEROP set",
			distroName: "",
			interop:    "/run/WSL/8_interop",
			wantWSL:    true,
		},
		{
			name:       "both set",
			distroName: "Ubuntu",
			interop:    "/run/WSL/8_interop",
			wantWSL:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			origDistro := os.Getenv("WSL_DISTRO_NAME")
			origInterop := os.Getenv("WSL_INTEROP")
			t.Cleanup(func() {
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
			})

			// Set test env vars
			if tt.distroName != "" {
				require.NoError(t, os.Setenv("WSL_DISTRO_NAME", tt.distroName))
			} else {
				require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
			}
			if tt.interop != "" {
				require.NoError(t, os.Setenv("WSL_INTEROP", tt.interop))
			} else {
				require.NoError(t, os.Unsetenv("WSL_INTEROP"))
			}

			result := isWSL()
			assert.Equal(t, tt.wantWSL, result)
		})
	}
}

func TestIsWSL_BothUnset(t *testing.T) {
	// Save original env vars
	origDistro := os.Getenv("WSL_DISTRO_NAME")
	origInterop := os.Getenv("WSL_INTEROP")
	t.Cleanup(func() {
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
	})

	// Test with both unset (will check /proc/version if on Linux)
	require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
	require.NoError(t, os.Unsetenv("WSL_INTEROP"))
	// Just verify it doesn't panic
	_ = isWSL()
}

func TestGetWSLKeychainHelp(t *testing.T) {
	tests := []struct {
		name         string
		distroName   string
		wantContains string
		wantEmpty    bool
	}{
		{
			name:         "with WSL",
			distroName:   "Ubuntu",
			wantContains: "WSL",
			wantEmpty:    false,
		},
		{
			name:       "without WSL",
			distroName: "",
			wantEmpty:  true, // On non-WSL/non-Linux systems
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			origDistro := os.Getenv("WSL_DISTRO_NAME")
			t.Cleanup(func() {
				if origDistro != "" {
					require.NoError(t, os.Setenv("WSL_DISTRO_NAME", origDistro))
				} else {
					require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
				}
			})

			// Set test env
			if tt.distroName != "" {
				require.NoError(t, os.Setenv("WSL_DISTRO_NAME", tt.distroName))
			} else {
				require.NoError(t, os.Unsetenv("WSL_DISTRO_NAME"))
			}

			help := GetWSLKeychainHelp()

			if tt.wantEmpty && runtime.GOOS != "linux" {
				assert.Empty(t, help)
			} else if tt.wantContains != "" {
				assert.Contains(t, help, tt.wantContains)
			}
		})
	}
}

func TestStringHelpers(t *testing.T) {
	t.Run("contains", func(t *testing.T) {
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
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("indexSubstring", func(t *testing.T) {
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
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}
