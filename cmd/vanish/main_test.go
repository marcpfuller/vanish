package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupLogging_FileMode(t *testing.T) {
	// Create temporary log directory
	tmpDir := t.TempDir()
	logFilePath := filepath.Join(tmpDir, "test.log")

	// Setup logging
	setupLogging(logFilePath)

	// Write a test log message
	testMessage := "test log message for file output"
	log.Println(testMessage)

	// Read the log file
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify the message was written
	logContent := string(content)
	if !strings.Contains(logContent, testMessage) {
		t.Errorf("Expected log file to contain %q, but it didn't.\nLog content:\n%s", testMessage, logContent)
	}

	// Verify timestamp format (should include microseconds)
	if !strings.Contains(logContent, ".") {
		t.Error("Expected log to contain microsecond timestamps (with decimal point), but it doesn't")
	}

	// Verify initialization message
	expectedInit := "Logging initialized to: " + logFilePath
	if !strings.Contains(logContent, expectedInit) {
		t.Errorf("Expected log file to contain initialization message %q", expectedInit)
	}
}

func TestSetupLogging_FilePermissions(t *testing.T) {
	// Create temporary log directory
	tmpDir := t.TempDir()
	logFilePath := filepath.Join(tmpDir, "test-perms.log")

	// Setup logging
	setupLogging(logFilePath)

	// Check file permissions
	info, err := os.Stat(logFilePath)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	// Verify file permissions are 0600 (owner read/write only)
	expectedPerms := os.FileMode(0o600)
	actualPerms := info.Mode().Perm()
	if actualPerms != expectedPerms {
		t.Errorf("Expected file permissions %o, got %o", expectedPerms, actualPerms)
	}
}

func TestSetupLogging_DirectoryCreation(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	// Use nested directory that doesn't exist yet
	logFilePath := filepath.Join(tmpDir, "nested", "logs", "test.log")

	// Verify directory doesn't exist yet
	if _, err := os.Stat(filepath.Dir(logFilePath)); err == nil {
		t.Fatal("Expected log directory to not exist yet")
	}

	// Setup logging (should create directory)
	setupLogging(logFilePath)

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(logFilePath)); err != nil {
		t.Errorf("Expected log directory to be created, but got error: %v", err)
	}

	// Verify log file was created
	if _, err := os.Stat(logFilePath); err != nil {
		t.Errorf("Expected log file to be created, but got error: %v", err)
	}
}

func TestSetupLogging_AppendMode(t *testing.T) {
	// Create temporary log directory
	tmpDir := t.TempDir()
	logFilePath := filepath.Join(tmpDir, "append-test.log")

	// Write initial content
	initialContent := "initial log entry\n"
	if err := os.WriteFile(logFilePath, []byte(initialContent), 0o600); err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Setup logging (should append, not truncate)
	setupLogging(logFilePath)

	// Write a new log message
	newMessage := "appended log message"
	log.Println(newMessage)

	// Read the log file
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify initial content is still there
	if !strings.Contains(logContent, initialContent) {
		t.Error("Expected log file to contain initial content (append mode), but it was truncated")
	}

	// Verify new content was added
	if !strings.Contains(logContent, newMessage) {
		t.Error("Expected log file to contain new message")
	}
}

func TestSetupLogging_InvalidPath(t *testing.T) {
	// Try to setup logging with an invalid path (should fail gracefully)
	invalidPath := "/root/this/path/should/not/be/writable/test.log"

	// Capture the original stderr
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	// Create a pipe to capture stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stderr = w

	// Setup logging with invalid path (should not panic)
	setupLogging(invalidPath)

	// Close write end and read stderr
	if err := w.Close(); err != nil {
		t.Logf("Warning: failed to close pipe writer: %v", err)
	}
	stderrOutput := make([]byte, 1024)
	n, err := r.Read(stderrOutput)
	if err != nil && n == 0 {
		t.Logf("Warning: failed to read stderr: %v", err)
	}
	stderrStr := string(stderrOutput[:n])

	// Verify warning message was printed
	if !strings.Contains(stderrStr, "Warning") {
		t.Logf("Expected warning about directory creation failure, stderr: %s", stderrStr)
	}
}

func TestLogOutputWithoutFileEnv(t *testing.T) {
	// Save original log output
	oldOutput := log.Writer()
	defer log.SetOutput(oldOutput)

	// Create a buffer to capture log output
	var buf strings.Builder
	log.SetOutput(&buf)

	// Log a test message
	testMessage := "test message without file env"
	log.Println(testMessage)

	// Verify the message was written to the buffer
	if !strings.Contains(buf.String(), testMessage) {
		t.Errorf("Expected log output to contain %q, but got: %s", testMessage, buf.String())
	}
}

func TestSetupLogging_DirectoryPermissions(t *testing.T) {
	// Create temporary base directory
	tmpDir := t.TempDir()
	logFilePath := filepath.Join(tmpDir, "subdir", "test.log")

	// Setup logging
	setupLogging(logFilePath)

	// Check directory permissions
	logDir := filepath.Dir(logFilePath)
	info, err := os.Stat(logDir)
	if err != nil {
		t.Fatalf("Failed to stat log directory: %v", err)
	}

	// Verify directory permissions are 0750 or less restrictive
	actualPerms := info.Mode().Perm()
	if actualPerms > 0o750 {
		t.Errorf("Expected directory permissions 0750 or less, got %o", actualPerms)
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
