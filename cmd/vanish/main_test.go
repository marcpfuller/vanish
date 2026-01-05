package main

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		validate func(t *testing.T, logPath string)
	}{
		{
			name: "file mode with logging",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "test.log")
			},
			validate: func(t *testing.T, logPath string) {
				testMessage := "test log message for file output"
				log.Println(testMessage)

				content, err := os.ReadFile(logPath)
				require.NoError(t, err)

				logContent := string(content)
				assert.Contains(t, logContent, testMessage)
				assert.Contains(t, logContent, ".", "Expected microsecond timestamps")
				assert.Contains(t, logContent, "Logging initialized to: "+logPath)
			},
		},
		{
			name: "file permissions",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "test-perms.log")
			},
			validate: func(t *testing.T, logPath string) {
				info, err := os.Stat(logPath)
				require.NoError(t, err)

				expectedPerms := os.FileMode(0o600)
				actualPerms := info.Mode().Perm()
				assert.Equal(t, expectedPerms, actualPerms)
			},
		},
		{
			name: "directory creation",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				logPath := filepath.Join(tmpDir, "nested", "logs", "test.log")
				// Verify directory doesn't exist yet
				_, err := os.Stat(filepath.Dir(logPath))
				assert.Error(t, err)
				return logPath
			},
			validate: func(t *testing.T, logPath string) {
				// Verify directory was created
				_, err := os.Stat(filepath.Dir(logPath))
				assert.NoError(t, err)
				// Verify log file was created
				_, err = os.Stat(logPath)
				assert.NoError(t, err)
			},
		},
		{
			name: "append mode",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				logPath := filepath.Join(tmpDir, "append-test.log")
				// Write initial content
				initialContent := "initial log entry\n"
				err := os.WriteFile(logPath, []byte(initialContent), 0o600)
				require.NoError(t, err)
				return logPath
			},
			validate: func(t *testing.T, logPath string) {
				newMessage := "appended log message"
				log.Println(newMessage)

				content, err := os.ReadFile(logPath)
				require.NoError(t, err)

				logContent := string(content)
				assert.Contains(t, logContent, "initial log entry")
				assert.Contains(t, logContent, newMessage)
			},
		},
		{
			name: "directory permissions",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "subdir", "test.log")
			},
			validate: func(t *testing.T, logPath string) {
				logDir := filepath.Dir(logPath)
				info, err := os.Stat(logDir)
				require.NoError(t, err)

				actualPerms := info.Mode().Perm()
				assert.LessOrEqual(t, actualPerms, os.FileMode(0o750))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logPath := tt.setup(t)
			setupLogging(logPath)
			tt.validate(t, logPath)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"substring exists at start", "hello world", "hello", true},
		{"substring exists in middle", "hello world", "lo wo", true},
		{"substring exists at end", "hello world", "world", true},
		{"substring does not exist", "hello world", "xyz", false},
		{"empty substring", "hello world", "", true},
		{"substring longer than string", "hi", "hello", false},
		{"exact match", "test", "test", true},
		{"case sensitive", "Hello", "hello", false},
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

func TestIndexSubstring(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{"found at start", "hello world", "hello", 0},
		{"found in middle", "hello world", "lo", 3},
		{"found at end", "hello world", "world", 6},
		{"not found", "hello world", "xyz", -1},
		{"empty substring", "hello", "", 0},
		{"substring longer than string", "hi", "hello", -1},
		{"exact match", "test", "test", 0},
		{"multiple occurrences returns first", "hello hello", "hello", 0},
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

func TestIsKeyringNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"not found error", &testError{msg: "key not found"}, true},
		{"cannot find error", &testError{msg: "cannot find the key"}, true},
		{"secret not found error", &testError{msg: "secret not found in keyring"}, true},
		{"different error", &testError{msg: "permission denied"}, false},
		{"access denied error", &testError{msg: "access denied"}, false},
		{"file does not exist", os.ErrNotExist, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isKeyringNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("isKeyringNotFoundError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// testError is a custom error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
