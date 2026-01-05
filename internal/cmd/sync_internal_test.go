package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInjectCredentials(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		username    string
		password    string
		expected    string
	}{
		{
			name:        "SFTP without credentials",
			destination: "sftp://host.example.com:22/path",
			username:    "user",
			password:    "pass",
			expected:    "sftp://user:pass@host.example.com:22/path",
		},
		{
			name:        "SFTP with credentials already present",
			destination: "sftp://user:pass@host.example.com:22/path",
			username:    "newuser",
			password:    "newpass",
			expected:    "sftp://user:pass@host.example.com:22/path", // Should not change
		},
		{
			name:        "SFTP with password containing special chars",
			destination: "sftp://user:p@ss@host.example.com:22/path",
			username:    "newuser",
			password:    "newpass",
			expected:    "sftp://user:p@ss@host.example.com:22/path", // Should not change
		},
		{
			name:        "Non-SFTP URL",
			destination: "https://example.com/path",
			username:    "user",
			password:    "pass",
			expected:    "https://example.com/path", // Should not change
		},
		{
			name:        "Local path",
			destination: "/local/path",
			username:    "user",
			password:    "pass",
			expected:    "/local/path", // Should not change
		},
		{
			name:        "SFTP without port",
			destination: "sftp://host.example.com/path",
			username:    "user",
			password:    "pass",
			expected:    "sftp://user:pass@host.example.com/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectCredentials(tt.destination, tt.username, tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskCredentials(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		expected    string
	}{
		{
			name:        "SFTP with password",
			destination: "sftp://user:password123@host.example.com:22/path",
			expected:    "sftp://user:****@host.example.com:22/path",
		},
		{
			name:        "SFTP with short password",
			destination: "sftp://user:p@host.example.com:22/path",
			expected:    "sftp://user:****@host.example.com:22/path",
		},
		{
			name:        "SFTP without credentials",
			destination: "sftp://host.example.com:22/path",
			expected:    "sftp://host.example.com:22/path",
		},
		{
			name:        "Local path",
			destination: "/local/path",
			expected:    "/local/path",
		},
		{
			name:        "URL without password",
			destination: "sftp://user@host.example.com:22/path",
			expected:    "sftp://user@host.example.com:22/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskCredentials(tt.destination)
			assert.Equal(t, tt.expected, result)
		})
	}
}
